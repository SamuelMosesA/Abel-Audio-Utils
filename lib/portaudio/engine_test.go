package portaudio

import (
	"behringerRecorder/lib/config"
	"behringerRecorder/lib/types"
	"testing"
	"time"

	pa "github.com/gordonklaus/portaudio"
	"github.com/stretchr/testify/assert"
)

type MockStream struct {
	ReadFunc func() error
}

func (m *MockStream) Start() error { return nil }
func (m *MockStream) Stop() error { return nil }
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
	state := &types.AppState{
		ChLeft:  0,
		ChRight: 1,
		Boost:   1.0,
		Devices: []*pa.DeviceInfo{{Name: "Test", MaxInputChannels: 2}},
	}
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

	err := StartAudioEngine(mockStreamer, state, cfg, 0, recordChan, playbackChan)
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
	
	// Close engine
	close(state.QuitAudio)
	time.Sleep(100 * time.Millisecond)
}

