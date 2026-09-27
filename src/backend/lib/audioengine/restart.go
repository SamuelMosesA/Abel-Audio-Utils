package audioengine

import (
	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"

	pa "github.com/gordonklaus/portaudio"
)

var (
	ErrRecordingInProgress = errors.New("cannot restart the engine while recording")
	ErrRestartInProgress   = errors.New("an engine restart is already in progress")
)

// RestartResult is what the admin UI gets back after a restart.
type RestartResult struct {
	Devices  []AudioDevice `json:"devices"`
	DeviceID int           `json:"deviceID"`
	// Name of the device the engine was reconnected to, if it was running before.
	RestoredDevice string `json:"restoredDevice,omitempty"`
	// Name of the previously running device when it could not be reconnected.
	MissingDevice    string   `json:"missingDevice,omitempty"`
	RestoreError     string   `json:"restoreError,omitempty"`
	ConfigReloaded   bool     `json:"configReloaded"`
	ConfigError      string   `json:"configError,omitempty"`
	NeedsFullRestart []string `json:"needsFullRestart,omitempty"`
}

// Restarter stops the audio engine, re-initializes PortAudio so newly connected
// devices are detected, reloads config.yaml and reconnects to the previous device by name.
type Restarter struct {
	AppState   *state.AppState
	Cfg        *config.Config
	ConfigPath string
	Streamer   AudioStreamer

	// Hooks default to PortAudio and config.LoadConfig; tests replace them.
	ReinitAudio func() error
	ListDevices func() ([]*pa.DeviceInfo, error)
	LoadConfig  func(path string) (*config.Config, error)
}

func NewRestarter(appState *state.AppState, cfg *config.Config, configPath string) *Restarter {
	return &Restarter{
		AppState:    appState,
		Cfg:         cfg,
		ConfigPath:  configPath,
		ReinitAudio: reinitPortAudio,
		ListDevices: pa.Devices,
		LoadConfig:  config.LoadConfig,
	}
}

// PortAudio only enumerates devices in Initialize, so a full cycle is needed to see new hardware.
func reinitPortAudio() error {
	if err := pa.Terminate(); err != nil {
		return err
	}
	return pa.Initialize()
}

func (r *Restarter) Restart() (RestartResult, error) {
	appState := r.AppState
	logger := slog.With("component", "engine")

	// Mark the restart before checking recording state; CreateRecording refuses to
	// start while IsRestarting, so a recording can't slip in between.
	if !appState.Engine().BeginRestart() {
		return RestartResult{}, ErrRestartInProgress
	}
	defer appState.Engine().EndRestart()

	if appState.IsRecording() {
		return RestartResult{}, ErrRecordingInProgress
	}

	appState.EngineLifecycle.Lock()
	defer appState.EngineLifecycle.Unlock()

	logger.Info("Engine restart requested")

	conf := appState.Config()
	previousDevice := ""
	if oldDevices := appState.Devices(); conf.IsRunning() && conf.DeviceID() >= 0 && int(conf.DeviceID()) < len(oldDevices) {
		previousDevice = oldDevices[conf.DeviceID()].Name
	}

	// The stream must be fully closed before PortAudio is terminated.
	if err := stopAudioEngineLocked(appState); err != nil {
		return RestartResult{}, err
	}
	state.Update[state.InterfaceConfig](appState, state.SectionInterface, func(s *state.InterfaceConfig) {
		s.SetIsRunning(false)
		s.SetDeviceID(-1)
	})

	if err := r.ReinitAudio(); err != nil {
		return RestartResult{}, fmt.Errorf("reinitialize audio: %w", err)
	}
	all, err := r.ListDevices()
	if err != nil {
		return RestartResult{}, fmt.Errorf("list audio devices: %w", err)
	}
	devices := InputDevices(all)
	appState.SetDevices(devices)

	result := RestartResult{Devices: GetDevices(devices), DeviceID: -1}
	r.reloadConfig(&result)

	// Device IDs are list positions and shift when hardware changes, so match by name.
	if previousDevice != "" {
		idx := findDeviceByName(devices, previousDevice)
		if idx < 0 {
			result.MissingDevice = previousDevice
			logger.Warn("Previous audio device not found after restart", slog.String("audio.device", previousDevice))
		} else if err := startAudioEngineLocked(r.Streamer, appState, r.Cfg, idx, appState.RecordChan, appState.PlaybackChan); err != nil {
			result.MissingDevice = previousDevice
			result.RestoreError = err.Error()
			logger.Error("Failed to reconnect audio device after restart", slog.String("audio.device", previousDevice), slog.Any("audio.error", err))
		} else {
			state.Update[state.InterfaceConfig](appState, state.SectionInterface, func(s *state.InterfaceConfig) {
				s.SetIsRunning(true)
				s.SetDeviceID(int32(idx))
			})
			result.DeviceID = idx
			result.RestoredDevice = previousDevice
		}
	}

	logger.Info("Engine restart complete",
		slog.Int("audio.device_count", len(devices)),
		slog.String("audio.restored_device", result.RestoredDevice),
		slog.Bool("config.reloaded", result.ConfigReloaded),
	)
	return result, nil
}

// reloadConfig re-reads config.yaml and the credentials file. A broken file is
// reported but does not fail the restart; the previous config stays in effect.
func (r *Restarter) reloadConfig(result *RestartResult) {
	if r.ConfigPath == "" || r.LoadConfig == nil {
		return
	}
	fresh, err := r.LoadConfig(r.ConfigPath)
	if err != nil {
		result.ConfigError = err.Error()
		slog.With("component", "engine").Warn("Config reload failed, keeping previous config", slog.Any("error", err))
		return
	}
	// main resolves these to absolute paths at startup; do the same so they compare equal.
	if abs, err := filepath.Abs(fresh.StorageLocation); err == nil {
		fresh.StorageLocation = abs
	}
	if abs, err := filepath.Abs(fresh.CloudDriveLocation); err == nil {
		fresh.CloudDriveLocation = abs
	}

	changes := r.Cfg.ApplyReload(fresh)
	result.ConfigReloaded = true
	result.NeedsFullRestart = changes.NeedsFullRestart

	// Only overwrite live routing/gain when the defaults in the file changed,
	// so values tuned in the admin UI survive a plain device re-scan.
	state.Update[state.InterfaceConfig](r.AppState, state.SectionInterface, func(s *state.InterfaceConfig) {
		if changes.DefaultChL {
			s.SetChL(int32(fresh.DefaultChL))
		}
		if changes.DefaultChR {
			s.SetChR(int32(fresh.DefaultChR))
		}
		if changes.DefaultBoost {
			s.SetBoost(fresh.DefaultBoost)
		}
		s.SetSampleRate(int32(fresh.SampleRate))
	})
}

func findDeviceByName(devices []*pa.DeviceInfo, name string) int {
	for i, d := range devices {
		if d.Name == name {
			return i
		}
	}
	return -1
}
