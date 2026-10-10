package audioengine

import (
	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	pa "github.com/gordonklaus/portaudio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockStream struct {
	ReadFunc func() error
}

func (m *MockStream) Start() error { return nil }
func (m *MockStream) Stop() error  { return nil }
func (m *MockStream) Close() error { return nil }
func (m *MockStream) Read() error {
	if m.ReadFunc != nil {
		return m.ReadFunc()
	}
	return nil
}

type MockStreamer struct {
	OpenStreamFunc func(params pa.StreamParameters, args ...interface{}) (PortAudioStream, error)
}

func (m *MockStreamer) OpenStream(params pa.StreamParameters, args ...interface{}) (PortAudioStream, error) {
	if m.OpenStreamFunc != nil {
		return m.OpenStreamFunc(params, args...)
	}
	return &MockStream{}, nil
}

func TestEngineAudioProcessing(t *testing.T) {
	appState := state.NewAppState("", "")
	state.Update[state.InterfaceConfig](appState, state.SectionInterface, func(s *state.InterfaceConfig) {
		s.SetChL(0)
		s.SetChR(1)
		s.SetBoost(1.0)
	})
	appState.Devices = []*pa.DeviceInfo{{Name: "Test", MaxInputChannels: 2}}
	cfg := &config.Config{BufferSize: 2, SampleRate: 44100}

	recordChan := make(chan []float32, 1)
	playbackChan := make(chan []float32, 1)

	mockStreamer := &MockStreamer{
		OpenStreamFunc: func(params pa.StreamParameters, args ...interface{}) (PortAudioStream, error) {
			in := args[0].([]float32)
			// Mock reading 2 frames
			return &MockStream{
				ReadFunc: func() error {
					in[0], in[1] = 0.5, -0.5 // Frame 0
					in[2], in[3] = 0.1, -0.1 // Frame 1
					return nil
				},
			}, nil
		},
	}

	err := StartAudioEngine(mockStreamer, appState, cfg, 0, recordChan, playbackChan)
	assert.NoError(t, err)

	// Wait for processing
	select {
	case chunk := <-recordChan:
		assert.Equal(t, 4, len(chunk))
		assert.Equal(t, float32(0.5), chunk[0])
		assert.Equal(t, float32(-0.5), chunk[1])
		assert.Equal(t, float32(0.1), chunk[2])
		assert.Equal(t, float32(-0.1), chunk[3])
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for audio chunk")
	}

	// Close engine cleanly
	err = StopAudioEngine(appState)
	assert.NoError(t, err)
	assert.False(t, appState.Engine().IsRunning())
}

func TestStartAudioEngineRejectsInvalidDevice(t *testing.T) {
	appState := state.NewAppState("", "")
	appState.Devices = []*pa.DeviceInfo{{Name: "Test", MaxInputChannels: 2}}
	cfg := &config.Config{BufferSize: 2, SampleRate: 44100}

	for _, id := range []int{-1, 1, 99} {
		err := StartAudioEngine(&MockStreamer{}, appState, cfg, id, nil, nil)
		assert.Error(t, err, "device %d", id)
		assert.False(t, appState.Engine().IsRunning(), "device %d must not mark the engine running", id)
		assert.Nil(t, appState.QuitAudio, "device %d must not start an engine", id)
	}
}

// stubPortAudio replaces the PortAudio hooks for one test.
func stubPortAudio(t *testing.T, reinit func() error, devices []*pa.DeviceInfo) {
	t.Helper()
	origReinit, origList := reinitPortAudio, listPortAudioDevices
	t.Cleanup(func() { reinitPortAudio, listPortAudioDevices = origReinit, origList })
	reinitPortAudio = reinit
	listPortAudioDevices = func() ([]*pa.DeviceInfo, error) { return devices, nil }
}

// writeConfig writes a config.yaml plus credentials file and returns the config path.
func writeConfig(t *testing.T, boost, password string) string {
	t.Helper()
	dir := t.TempDir()
	creds := filepath.Join(dir, "creds.json")
	require.NoError(t, os.WriteFile(creds, []byte(`[{"username":"admin","password":"`+password+`"}]`), 0o600))
	path := filepath.Join(dir, "config.yaml")
	yaml := "default_ch_l: 1\ndefault_ch_r: 0\ndefault_boost: " + boost + "\nadmin_user_credentials: \"" + filepath.ToSlash(creds) + "\"\n"
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o600))
	return path
}

func TestRestartEngineReconnectsByNameAndReloadsConfig(t *testing.T) {
	appState := state.NewAppState("", "")
	appState.Devices = []*pa.DeviceInfo{
		{Name: "USB Headset", MaxInputChannels: 1},
		{Name: "Behringer UMC404HD", MaxInputChannels: 4},
	}
	cfg := &config.Config{BufferSize: 2, SampleRate: 44100, DefaultChL: 1, DefaultBoost: 1.0,
		Credentials: map[string]string{"admin": "old"}, Path: writeConfig(t, "1.5", "new")}
	state.Update[state.InterfaceConfig](appState, state.SectionInterface, func(s *state.InterfaceConfig) {
		s.SetIsRunning(true)
		s.SetDeviceID(1)
		s.SetChL(1)
	})

	var streamClosed atomic.Bool
	streamer := &MockStreamer{
		OpenStreamFunc: func(params pa.StreamParameters, args ...interface{}) (PortAudioStream, error) {
			return &closeTrackingStream{closed: &streamClosed}, nil
		},
	}
	require.NoError(t, StartAudioEngine(streamer, appState, cfg, 1, make(chan []float32, 1), make(chan []float32, 1)))

	// After the re-scan the headset is gone and the mixer moved from #1 to #0.
	stubPortAudio(t, func() error {
		assert.True(t, streamClosed.Load(), "stream must be closed before PortAudio is re-initialized")
		return nil
	}, []*pa.DeviceInfo{
		{Name: "Behringer UMC404HD", MaxInputChannels: 4},
		{Name: "Speakers", MaxInputChannels: 0},
		{Name: "Built-in Microphone", MaxInputChannels: 2},
	})

	result, err := RestartEngine(streamer, appState, cfg)
	require.NoError(t, err)
	defer func() {
		_ = StopAudioEngine(appState)
	}()

	assert.Equal(t, []AudioDevice{
		{ID: 0, Name: "Behringer UMC404HD", In: 4},
		{ID: 1, Name: "Built-in Microphone", In: 2},
	}, result.Devices, "output-only devices are skipped")
	assert.Equal(t, "Behringer UMC404HD", result.Reconnected)
	assert.Equal(t, 0, result.DeviceID)
	assert.Empty(t, result.ConfigError)
	assert.True(t, appState.Config().IsRunning())
	assert.Equal(t, int32(0), appState.Config().DeviceID())

	assert.Equal(t, "new", cfg.Credentials["admin"], "password reloaded")
	assert.Equal(t, 1.5, appState.Config().Boost(), "changed default boost applied")
}

func TestRestartEngineStaysStoppedWhenDeviceMissing(t *testing.T) {
	appState := state.NewAppState("", "")
	appState.Devices = []*pa.DeviceInfo{{Name: "Behringer UMC404HD", MaxInputChannels: 4}}
	cfg := &config.Config{BufferSize: 2, SampleRate: 44100,
		Credentials: map[string]string{"admin": "pw"}, Path: filepath.Join(t.TempDir(), "missing.yaml")}
	state.Update[state.InterfaceConfig](appState, state.SectionInterface, func(s *state.InterfaceConfig) {
		s.SetIsRunning(true)
		s.SetDeviceID(0)
		s.SetChL(3)
	})
	stubPortAudio(t, func() error { return nil }, []*pa.DeviceInfo{{Name: "Built-in Microphone", MaxInputChannels: 2}})

	result, err := RestartEngine(&MockStreamer{}, appState, cfg)
	require.NoError(t, err, "a missing config file must not fail the device re-scan")

	assert.Empty(t, result.Reconnected)
	assert.Equal(t, -1, result.DeviceID)
	assert.False(t, appState.Config().IsRunning(), "must not fall back to another device")
	assert.Equal(t, int32(-1), appState.Config().DeviceID())
	assert.Nil(t, appState.QuitAudio)
	assert.NotEmpty(t, result.ConfigError)
	assert.Equal(t, "pw", cfg.Credentials["admin"], "previous config kept")
	assert.Equal(t, int32(3), appState.Config().ChL(), "UI-tuned value kept")
}

func TestRestartEngineFailsWhenAudioReinitFails(t *testing.T) {
	appState := state.NewAppState("", "")
	stubPortAudio(t, func() error { return errors.New("host error") }, nil)

	_, err := RestartEngine(&MockStreamer{}, appState, &config.Config{})
	assert.ErrorContains(t, err, "host error")
	assert.False(t, appState.Config().IsRunning(), "engine state must be stopped on reinit error")
	assert.Equal(t, int32(-1), appState.Config().DeviceID())
}

func TestConcurrentRestartAndStopEngine(t *testing.T) {
	appState := state.NewAppState("", "")
	appState.Devices = []*pa.DeviceInfo{
		{Name: "Test Device", MaxInputChannels: 2},
	}
	cfg := &config.Config{BufferSize: 2, SampleRate: 44100}

	stubPortAudio(t, func() error {
		time.Sleep(5 * time.Millisecond)
		return nil
	}, []*pa.DeviceInfo{{Name: "Test Device", MaxInputChannels: 2}})

	streamer := &MockStreamer{
		OpenStreamFunc: func(params pa.StreamParameters, args ...interface{}) (PortAudioStream, error) {
			return &MockStream{}, nil
		},
	}

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = RestartEngine(streamer, appState, cfg)
		}()
	}
	wg.Wait()

	_ = StopAudioEngine(appState)
	assert.False(t, appState.Engine().IsRunning())
}

type closeTrackingStream struct {
	MockStream
	closed *atomic.Bool
}

func (s *closeTrackingStream) Read() error  { time.Sleep(time.Millisecond); return nil }
func (s *closeTrackingStream) Close() error { s.closed.Store(true); return nil }
