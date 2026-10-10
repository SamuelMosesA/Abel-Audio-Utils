package audioengine

import (
	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"
	"abel/src/backend/lib/telemetry"
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	pa "github.com/gordonklaus/portaudio"
)

// engineMu serializes all PortAudio initialization, termination, stream creation,
// and engine restart operations across the entire application.
var engineMu sync.Mutex

// PortAudio hooks, replaced in tests.
var (
	reinitPortAudio = func() error {
		if err := pa.Terminate(); err != nil {
			return err
		}
		return pa.Initialize()
	}
	listPortAudioDevices = pa.Devices
)

// RestartResult is returned to the admin UI after a restart.
type RestartResult struct {
	Devices     []AudioDevice `json:"devices"`
	DeviceID    int           `json:"deviceID"`
	Reconnected string        `json:"reconnected,omitempty"`
	ConfigError string        `json:"configError,omitempty"`
}

// StopAudioEngine stops the active audio engine goroutine and awaits stream teardown.
func StopAudioEngine(appState *state.AppState) error {
	if q := appState.QuitAudio; q != nil {
		close(q)
		appState.QuitAudio = nil
	}

	done := appState.DoneAudio
	if done != nil {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			slog.Warn("Audio engine did not terminate within timeout")
		}
		appState.DoneAudio = nil
	}

	deadline := time.Now().Add(2 * time.Second)
	for appState.Engine().IsRunning() {
		if time.Now().After(deadline) {
			return fmt.Errorf("audio engine did not stop in time")
		}
		time.Sleep(10 * time.Millisecond)
	}

	return nil
}

// RestartEngine re-scans audio devices so hardware connected after startup appears,
// reloads config.yaml and the credentials file, and reconnects to the previous device.
func RestartEngine(streamer AudioStreamer, appState *state.AppState, cfg *config.Config) (RestartResult, error) {
	engineMu.Lock()
	defer engineMu.Unlock()

	// Device IDs are list positions that shift when hardware changes, so remember the name.
	previous := ""
	if conf := appState.Config(); conf.IsRunning() && conf.DeviceID() >= 0 && int(conf.DeviceID()) < len(appState.Devices) {
		previous = appState.Devices[conf.DeviceID()].Name
	}

	if err := refreshDevices(appState); err != nil {
		state.Update[state.InterfaceConfig](appState, state.SectionInterface, func(s *state.InterfaceConfig) {
			s.SetIsRunning(false)
			s.SetDeviceID(-1)
		})
		return RestartResult{DeviceID: -1}, err
	}

	result := RestartResult{DeviceID: -1}
	defaultsChanged := false
	if cfg.Path != "" {
		// A broken file is reported but keeps the previous config.
		if fresh, err := config.LoadConfig(cfg.Path); err != nil {
			result.ConfigError = err.Error()
		} else {
			defaultsChanged = fresh.DefaultChL != cfg.DefaultChL || fresh.DefaultChR != cfg.DefaultChR || fresh.DefaultBoost != cfg.DefaultBoost
			cfg.Credentials = fresh.Credentials
			cfg.DefaultChL, cfg.DefaultChR, cfg.DefaultBoost = fresh.DefaultChL, fresh.DefaultChR, fresh.DefaultBoost
		}
	}

	for i, d := range appState.Devices {
		if previous != "" && d.Name == previous {
			if err := startAudioEngineLocked(streamer, appState, cfg, i, appState.RecordChan, appState.PlaybackChan); err == nil {
				result.DeviceID, result.Reconnected = i, previous
			}
			break
		}
	}

	state.Update[state.InterfaceConfig](appState, state.SectionInterface, func(s *state.InterfaceConfig) {
		s.SetIsRunning(result.DeviceID >= 0)
		s.SetDeviceID(int32(result.DeviceID))
		// Only overwrite live routing/gain when the file's defaults changed,
		// so values tuned in the admin UI survive a plain device re-scan.
		if defaultsChanged {
			s.SetChL(int32(cfg.DefaultChL))
			s.SetChR(int32(cfg.DefaultChR))
			s.SetBoost(cfg.DefaultBoost)
		}
	})

	result.Devices = GetDevices(appState.Devices)
	return result, nil
}

// refreshDevices stops the engine and re-initializes PortAudio, which only
// enumerates devices in Initialize. Caller must hold engineMu.
func refreshDevices(appState *state.AppState) error {
	if err := StopAudioEngine(appState); err != nil {
		return err
	}

	if err := reinitPortAudio(); err != nil {
		return fmt.Errorf("reinitialize audio: %w", err)
	}
	all, err := listPortAudioDevices()
	if err != nil {
		return fmt.Errorf("list audio devices: %w", err)
	}
	var devices []*pa.DeviceInfo
	for _, d := range all {
		if d.MaxInputChannels > 0 {
			devices = append(devices, d)
		}
	}
	appState.Devices = devices
	return nil
}

func StartAudioEngine(streamer AudioStreamer, appState *state.AppState, cfg *config.Config, deviceID int, recordChan chan<- []float32, playbackChan chan<- []float32) error {
	engineMu.Lock()
	defer engineMu.Unlock()
	return startAudioEngineLocked(streamer, appState, cfg, deviceID, recordChan, playbackChan)
}

func startAudioEngineLocked(streamer AudioStreamer, appState *state.AppState, cfg *config.Config, deviceID int, recordChan chan<- []float32, playbackChan chan<- []float32) error {
	if streamer == nil {
		streamer = &PADriver{}
	}

	devices := appState.Devices
	if deviceID < 0 || deviceID >= len(devices) {
		return fmt.Errorf("invalid device")
	}
	dev := devices[deviceID]

	if err := StopAudioEngine(appState); err != nil {
		return err
	}

	quit := make(chan bool)
	done := make(chan struct{})
	appState.QuitAudio = quit
	appState.DoneAudio = done
	appState.Engine().SetRunning(true)

	// Engine GoRoutine
	logger := slog.With("component", "audio")
	go func() {
		logger.Info("Audio engine started", slog.String("device", dev.Name))
		defer close(done)
		defer logger.Info("Audio engine stopped")
		defer func() { appState.Engine().SetRunning(false) }()
		defer func() {
			if appState.Translator != nil {
				appState.Translator.CloseAll()
			}
		}()

		var in []float32
		var stream PortAudioStream
		var err error
		openedSampleRate := cfg.SampleRate
		openedChannels := dev.MaxInputChannels

		logger.Info("Opening stream",
			slog.String("audio.device", dev.Name),
			slog.Int("audio.channels", openedChannels),
			slog.Int("audio.sample_rate", openedSampleRate),
		)

		// Try 1: Configured sample rate and MaxInputChannels
		in = make([]float32, cfg.BufferSize*openedChannels)
		stream, err = streamer.OpenStream(pa.StreamParameters{
			Input:      pa.StreamDeviceParameters{Device: dev, Channels: openedChannels, Latency: dev.DefaultLowInputLatency},
			SampleRate: float64(openedSampleRate), FramesPerBuffer: cfg.BufferSize,
		}, in)

		// Try 2: If configured sample rate failed, fallback directly to DefaultSampleRate of the device
		if err != nil {
			logger.Warn("Failed to open stream at requested sample rate, falling back to device default",
				slog.Int("audio.requested_rate", cfg.SampleRate),
				slog.Float64("audio.default_rate", dev.DefaultSampleRate),
				slog.Any("audio.error", err),
			)
			openedSampleRate = int(dev.DefaultSampleRate)
			if openedSampleRate <= 0 {
				openedSampleRate = 44100 // Safe default fallback
			}
			in = make([]float32, cfg.BufferSize*openedChannels)
			stream, err = streamer.OpenStream(pa.StreamParameters{
				Input:      pa.StreamDeviceParameters{Device: dev, Channels: openedChannels, Latency: dev.DefaultLowInputLatency},
				SampleRate: float64(openedSampleRate), FramesPerBuffer: cfg.BufferSize,
			}, in)
		}

		// Try 3: If that still failed, try fallback to 2 channels
		if err != nil && openedChannels > 2 {
			logger.Warn("Failed to open stream with max channels, trying fallback with 2 channels",
				slog.Int("audio.requested_channels", openedChannels),
				slog.Any("audio.error", err),
			)
			openedChannels = 2
			in = make([]float32, cfg.BufferSize*openedChannels)
			stream, err = streamer.OpenStream(pa.StreamParameters{
				Input:      pa.StreamDeviceParameters{Device: dev, Channels: openedChannels, Latency: dev.DefaultLowInputLatency},
				SampleRate: float64(openedSampleRate), FramesPerBuffer: cfg.BufferSize,
			}, in)
		}

		if err != nil {
			logger.Error("Error opening stream: all configurations failed", slog.Any("audio.error", err))
			return
		}

		// Update state with the actually opened sample rate!
		state.Update[state.InterfaceConfig](appState, state.SectionInterface, func(s *state.InterfaceConfig) {
			s.SetSampleRate(int32(openedSampleRate))
		})

		stream.Start()
		defer stream.Stop()
		defer stream.Close()

		for {
			select {
			case <-quit:
				return
			default:
			}

			startTime := time.Now()

			if err := stream.Read(); err != nil {
				select {
				case <-quit:
					return
				default:
					continue
				}
			}

			// Read current interface config once per loop
			conf := appState.Config()
			chL := int(conf.ChL())
			chR := int(conf.ChR())
			boost := float32(conf.Boost())

			if boost == 0 {
				boost = 1.0
			}

			stereoChunk := make([]float32, cfg.BufferSize*2)
			for i := 0; i < cfg.BufferSize; i++ {
				idxL := (i * openedChannels) + chL
				idxR := (i * openedChannels) + chR

				var sL, sR float32
				if idxL < len(in) {
					sL = in[idxL]
				}
				if idxR < len(in) {
					sR = in[idxR]
				}

				sL *= boost
				sR *= boost
				if sL > 1.0 {
					sL = 1.0
				} else if sL < -1.0 {
					sL = -1.0
				}
				if sR > 1.0 {
					sR = 1.0
				} else if sR < -1.0 {
					sR = -1.0
				}

				stereoChunk[i*2] = sL
				stereoChunk[i*2+1] = sR
			}

			// Fan-out to consumers
			select {
			case recordChan <- stereoChunk:
			default:
				if appState.IsRecording() {
					logger.Warn("Audio engine dropped recording chunk: buffer full")
					if telemetry.DroppedAudioChunks != nil {
						telemetry.DroppedAudioChunks.Add(context.Background(), 1)
					}
				}
			}
			select {
			case playbackChan <- stereoChunk:
			default:
			}

			if telemetry.AudioLoopLatency != nil {
				telemetry.AudioLoopLatency.Record(context.Background(), float64(time.Since(startTime).Nanoseconds())/1e6)
			}
		}
	}()

	return nil
}
