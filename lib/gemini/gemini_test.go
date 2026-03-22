package gemini

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/genai"
)

func TestDownsampling(t *testing.T) {
	tm := &TranslationManager{}
	tm.Enabled.Store(true)
	
	// Input: 12 samples (2 frames of 48kHz stereo = 6 samples each? No, 1 sample pair = 2 samples)
	// 48kHz stereo -> 16kHz mono
	// Every 6 samples (3 pairs) -> 1 sample
	
	// 12 samples = 6 pairs. Should result in 2 samples.
	chunk := make([]float32, 12)
	for i := 0; i < 12; i++ {
		chunk[i] = 0.5
	}
	
	// We need to capture the output. PushAudio sends to sessions.
	// But we can test the logic by extracting it or testing PushAudio with a mock session.
}

func TestConvertInt16ToFloat32(t *testing.T) {
	// Gemini outputs 24kHz Mono. System expects 48kHz Stereo.
	// 1 sample -> 4 samples (repeats 2x for 48kHz, 2x for stereo)
	
	data := []byte{0x00, 0x40} // 16384 (0.5 approx)
	floats := convertInt16ToFloat32(data)
	
	assert.Equal(t, 4, len(floats))
	expected := float32(16384) / 32767.0
	assert.InDelta(t, expected, floats[0], 0.0001)
	assert.InDelta(t, expected, floats[1], 0.0001)
	assert.InDelta(t, expected, floats[2], 0.0001)
	assert.InDelta(t, expected, floats[3], 0.0001)
}

type MockGeminiSession struct {
	SendFunc    func(genai.LiveRealtimeInput) error
	ReceiveFunc func() (*genai.LiveServerMessage, error)
}

func (m *MockGeminiSession) SendRealtimeInput(input genai.LiveRealtimeInput) error {
	if m.SendFunc != nil {
		return m.SendFunc(input)
	}
	return nil
}

func (m *MockGeminiSession) Receive() (*genai.LiveServerMessage, error) {
	if m.ReceiveFunc != nil {
		return m.ReceiveFunc()
	}
	return nil, nil // Block or return error to stop loop
}

func (m *MockGeminiSession) Close() error { return nil }

type MockGeminiClient struct {
	ConnectFunc func(ctx context.Context, model string, config *genai.LiveConnectConfig) (GeminiSession, error)
}

func (m *MockGeminiClient) Connect(ctx context.Context, model string, config *genai.LiveConnectConfig) (GeminiSession, error) {
	if m.ConnectFunc != nil {
		return m.ConnectFunc(ctx, model, config)
	}
	return &MockGeminiSession{}, nil
}

func TestPushAudio(t *testing.T) {
	tm := &TranslationManager{}
	tm.Enabled.Store(true)

	session := &TranslationSession{
		Language: "English",
		AudioIn:  make(chan []byte, 10),
	}
	tm.sessions.Store("English", session)

	// 48kHz stereo -> 16kHz mono. 6 samples per frame in 48kHz stereo?
	// 48000 Hz, stereo = 2 samples per frame.
	// 16000 Hz, mono = 1 sample per frame.
	// Ratio: 3 frames of 48kHz stereo -> 1 frame of 16kHz mono.
	// 3 frames = 6 samples. 
	
	chunk := make([]float32, 12) // 2 output mono samples
	for i := 0; i < 12; i++ {
		chunk[i] = 0.5
	}

	tm.PushAudio(chunk)

	select {
	case data := <-session.AudioIn:
		// 2 samples * 2 bytes = 4 bytes
		assert.Equal(t, 4, len(data))
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Audio not received")
	}
}

func TestTranslationLifecycle(t *testing.T) {
	mockClient := &MockGeminiClient{
		ConnectFunc: func(ctx context.Context, model string, config *genai.LiveConnectConfig) (GeminiSession, error) {
			return &MockGeminiSession{
				ReceiveFunc: func() (*genai.LiveServerMessage, error) {
					// Return a dummy message or block
					time.Sleep(10 * time.Millisecond)
					return &genai.LiveServerMessage{
						ServerContent: &genai.LiveServerContent{
							OutputTranscription: &genai.Transcription{Text: "Hello"},
						},
					}, nil
				},
			}, nil
		},
	}

	tm := &TranslationManager{
		client: mockClient,
		model:  "gemini-2.0-flash-exp",
	}
	tm.Enabled.Store(true)

	ch := tm.GetChannel("English")
	assert.NotNil(t, ch)

	// Wait for session to start and receive one message
	time.Sleep(100 * time.Millisecond)
	
	tm.CloseAll()
}

