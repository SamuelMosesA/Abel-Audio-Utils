package audioengine

import (
	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	pa "github.com/gordonklaus/portaudio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// trackingStream records whether the engine closed it.
type trackingStream struct {
	closed atomic.Bool
}

func (s *trackingStream) Start() error { return nil }
func (s *trackingStream) Stop() error  { return nil }
func (s *trackingStream) Close() error { s.closed.Store(true); return nil }
func (s *trackingStream) Read() error {
	time.Sleep(time.Millisecond)
	return nil
}

// trackingStreamer records which devices were opened. OpenStream runs on the engine goroutine.
type trackingStreamer struct {
	mu     sync.Mutex
	opened []string
	last   *trackingStream
}

func (m *trackingStreamer) OpenStream(params pa.StreamParameters, args ...interface{}) (PortAudioStream, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.opened = append(m.opened, params.Input.Device.Name)
	m.last = &trackingStream{}
	return m.last, nil
}

func (m *trackingStreamer) Opened() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.opened...)
}

func (m *trackingStreamer) Last() *trackingStream {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.last
}

// waitOpened waits until the engine goroutine has opened n streams.
func (m *trackingStreamer) waitOpened(t *testing.T, n int) {
	t.Helper()
	require.Eventually(t, func() bool { return len(m.Opened()) >= n }, time.Second, 5*time.Millisecond)
}

func newTestRestarter(t *testing.T, appState *state.AppState, cfg *config.Config, next []*pa.DeviceInfo) (*Restarter, *trackingStreamer) {
	t.Helper()
	streamer := &trackingStreamer{}
	r := &Restarter{
		AppState:    appState,
		Cfg:         cfg,
		Streamer:    streamer,
		ReinitAudio: func() error { return nil },
		ListDevices: func() ([]*pa.DeviceInfo, error) { return next, nil },
	}
	return r, streamer
}

func testConfig(t *testing.T) *config.Config {
	dir := t.TempDir()
	return &config.Config{BufferSize: 4, SampleRate: 48000, StorageLocation: dir, CloudDriveLocation: dir}
}

// startOn starts the engine on deviceID and records it in the interface state, like UpdateAudioConfig.
func startOn(t *testing.T, appState *state.AppState, cfg *config.Config, streamer AudioStreamer, deviceID int) {
	t.Helper()
	require.NoError(t, StartAudioEngine(streamer, appState, cfg, deviceID, appState.RecordChan, appState.PlaybackChan))
	state.Update[state.InterfaceConfig](appState, state.SectionInterface, func(s *state.InterfaceConfig) {
		s.SetIsRunning(true)
		s.SetDeviceID(int32(deviceID))
	})
}

func TestRestartRefusedWhileRecording(t *testing.T) {
	appState := state.NewAppState("", "")
	state.Update[state.RecordIntent](appState, state.SectionRecording, func(s *state.RecordIntent) { s.SetRecording(true) })

	reinitCalled := false
	r, _ := newTestRestarter(t, appState, testConfig(t), nil)
	r.ReinitAudio = func() error { reinitCalled = true; return nil }

	_, err := r.Restart()
	assert.ErrorIs(t, err, ErrRecordingInProgress)
	assert.False(t, reinitCalled, "audio must not be touched while recording")
	assert.False(t, appState.Engine().IsRestarting(), "restart flag must be cleared")
}

func TestRestartRefusedWhenAlreadyRestarting(t *testing.T) {
	appState := state.NewAppState("", "")
	require.True(t, appState.Engine().BeginRestart())

	r, _ := newTestRestarter(t, appState, testConfig(t), nil)
	_, err := r.Restart()
	assert.ErrorIs(t, err, ErrRestartInProgress)
	assert.True(t, appState.Engine().IsRestarting(), "must not clear another restart's flag")
}

func TestRestartDetectsNewDevicesAndReconnectsByName(t *testing.T) {
	appState := state.NewAppState("", "")
	cfg := testConfig(t)
	appState.SetDevices([]*pa.DeviceInfo{
		{Name: "USB Headset", MaxInputChannels: 1},
		{Name: "Behringer UMC404HD", MaxInputChannels: 4},
	})
	streamer := &trackingStreamer{}
	startOn(t, appState, cfg, streamer, 1)
	streamer.waitOpened(t, 1)
	firstStream := streamer.Last()

	// After the re-scan the headset is gone and the mixer moved from #1 to #0.
	next := []*pa.DeviceInfo{
		{Name: "Behringer UMC404HD", MaxInputChannels: 4},
		{Name: "Speakers", MaxInputChannels: 0},
		{Name: "Built-in Microphone", MaxInputChannels: 2},
	}
	r, _ := newTestRestarter(t, appState, cfg, next)
	r.Streamer = streamer
	r.ReinitAudio = func() error {
		assert.True(t, firstStream.closed.Load(), "stream must be closed before PortAudio is re-initialized")
		return nil
	}

	result, err := r.Restart()
	require.NoError(t, err)

	assert.Equal(t, []AudioDevice{
		{ID: 0, Name: "Behringer UMC404HD", In: 4},
		{ID: 1, Name: "Built-in Microphone", In: 2},
	}, result.Devices, "output-only devices are filtered out")
	assert.Equal(t, "Behringer UMC404HD", result.RestoredDevice)
	assert.Equal(t, 0, result.DeviceID)
	streamer.waitOpened(t, 2)
	assert.Equal(t, []string{"Behringer UMC404HD", "Behringer UMC404HD"}, streamer.Opened())
	assert.Equal(t, int32(0), appState.Config().DeviceID())
	assert.True(t, appState.Config().IsRunning())
	assert.False(t, appState.Engine().IsRestarting())

	require.NoError(t, StopAudioEngine(appState))
}

func TestRestartLeavesEngineStoppedWhenDeviceMissing(t *testing.T) {
	appState := state.NewAppState("", "")
	cfg := testConfig(t)
	appState.SetDevices([]*pa.DeviceInfo{{Name: "Behringer UMC404HD", MaxInputChannels: 4}})
	streamer := &trackingStreamer{}
	startOn(t, appState, cfg, streamer, 0)
	streamer.waitOpened(t, 1)

	r, _ := newTestRestarter(t, appState, cfg, []*pa.DeviceInfo{{Name: "Built-in Microphone", MaxInputChannels: 2}})
	r.Streamer = streamer

	result, err := r.Restart()
	require.NoError(t, err)

	assert.Equal(t, "Behringer UMC404HD", result.MissingDevice)
	assert.Empty(t, result.RestoredDevice)
	assert.Equal(t, -1, result.DeviceID)
	assert.Equal(t, int32(-1), appState.Config().DeviceID())
	assert.False(t, appState.Config().IsRunning(), "must not fall back to another device")
	assert.Len(t, streamer.Opened(), 1, "no new stream should be opened")
	assert.Nil(t, appState.QuitAudio)
}

func TestRestartWithEngineStoppedOnlyRescans(t *testing.T) {
	appState := state.NewAppState("", "")
	r, streamer := newTestRestarter(t, appState, testConfig(t), []*pa.DeviceInfo{{Name: "Behringer UMC404HD", MaxInputChannels: 4}})

	result, err := r.Restart()
	require.NoError(t, err)
	assert.Len(t, result.Devices, 1)
	assert.Equal(t, -1, result.DeviceID)
	assert.Empty(t, streamer.Opened())
	assert.Len(t, appState.Devices(), 1)
}

func TestRestartReloadsConfig(t *testing.T) {
	appState := state.NewAppState("", "")
	cfg := testConfig(t)
	cfg.Port = "8080"
	cfg.DefaultChL, cfg.DefaultChR, cfg.DefaultBoost = 1, 0, 1.0
	cfg.Credentials = map[string]string{"admin": "old"}
	// Channel tuned in the admin UI; the file's default_ch_l did not change, so it must survive.
	state.Update[state.InterfaceConfig](appState, state.SectionInterface, func(s *state.InterfaceConfig) {
		s.SetChL(3)
		s.SetChR(0)
		s.SetBoost(1.0)
	})

	fresh := testConfigFields(cfg)
	fresh.Port = "9090"
	fresh.DefaultBoost = 1.5
	fresh.Credentials = map[string]string{"admin": "new"}

	r, _ := newTestRestarter(t, appState, cfg, nil)
	r.ConfigPath = "config.yaml"
	r.LoadConfig = func(path string) (*config.Config, error) { return fresh, nil }

	result, err := r.Restart()
	require.NoError(t, err)

	assert.True(t, result.ConfigReloaded)
	assert.Equal(t, []string{"port"}, result.NeedsFullRestart)
	assert.Equal(t, "8080", cfg.Port, "port is not applied live")
	assert.True(t, cfg.CheckCredentials("admin", "new"))
	assert.False(t, cfg.CheckCredentials("admin", "old"))
	assert.Equal(t, 1.5, appState.Config().Boost(), "changed default is applied")
	assert.Equal(t, int32(3), appState.Config().ChL(), "UI-tuned value kept when default unchanged")
}

func TestRestartKeepsConfigWhenReloadFails(t *testing.T) {
	appState := state.NewAppState("", "")
	cfg := testConfig(t)
	cfg.Credentials = map[string]string{"admin": "pw"}

	r, _ := newTestRestarter(t, appState, cfg, nil)
	r.ConfigPath = "config.yaml"
	r.LoadConfig = func(path string) (*config.Config, error) { return nil, errors.New("yaml: bad indent") }

	result, err := r.Restart()
	require.NoError(t, err, "a broken config file must not fail the device re-scan")
	assert.False(t, result.ConfigReloaded)
	assert.Contains(t, result.ConfigError, "bad indent")
	assert.True(t, cfg.CheckCredentials("admin", "pw"))
}

func TestRestartFailsWhenAudioReinitFails(t *testing.T) {
	appState := state.NewAppState("", "")
	r, _ := newTestRestarter(t, appState, testConfig(t), nil)
	r.ReinitAudio = func() error { return errors.New("host error") }

	_, err := r.Restart()
	assert.ErrorContains(t, err, "host error")
	assert.False(t, appState.Engine().IsRestarting())
}

// testConfigFields copies the plain fields of cfg (Config holds a mutex, so it can't be copied directly).
func testConfigFields(cfg *config.Config) *config.Config {
	return &config.Config{
		Port:               cfg.Port,
		SampleRate:         cfg.SampleRate,
		BufferSize:         cfg.BufferSize,
		StorageLocation:    cfg.StorageLocation,
		CloudDriveLocation: cfg.CloudDriveLocation,
		DefaultChL:         cfg.DefaultChL,
		DefaultChR:         cfg.DefaultChR,
		DefaultBoost:       cfg.DefaultBoost,
	}
}
