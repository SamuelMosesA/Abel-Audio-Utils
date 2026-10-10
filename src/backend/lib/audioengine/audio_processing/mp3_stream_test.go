package audio_processing

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateStereoSineChunk(sampleRate int, durationMs int, freq float64) []float32 {
	numSamples := (sampleRate * durationMs / 1000) * 2 // stereo
	chunk := make([]float32, numSamples)
	for i := 0; i < numSamples; i += 2 {
		t := float64(i/2) / float64(sampleRate)
		val := float32(0.5 * math.Sin(2*math.Pi*freq*t))
		chunk[i] = val   // L
		chunk[i+1] = val // R
	}
	return chunk
}

func TestLiveAudioBroadcaster_Lifecycle(t *testing.T) {
	broadcaster, err := NewLiveAudioBroadcaster()
	require.NoError(t, err)
	defer broadcaster.Close()

	sampleRate := 48000
	ch, unsubscribe, err := broadcaster.Subscribe("default", sampleRate, nil)
	require.NoError(t, err)
	require.NotNil(t, ch)
	defer unsubscribe()

	// Feed 500ms of stereo audio
	chunk := generateStereoSineChunk(sampleRate, 100, 440)
	for i := 0; i < 5; i++ {
		err := broadcaster.Publish("default", sampleRate, chunk)
		require.NoError(t, err)
		time.Sleep(20 * time.Millisecond)
	}

	// Verify we receive MP3 frames
	select {
	case data, ok := <-ch:
		require.True(t, ok)
		assert.NotEmpty(t, data)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for MP3 frames from live audio broadcaster")
	}
}

func TestLiveAudioBroadcaster_InvalidStreamName(t *testing.T) {
	broadcaster, err := NewLiveAudioBroadcaster()
	require.NoError(t, err)
	defer broadcaster.Close()

	assert.ErrorIs(t, broadcaster.EnsureStream("../default", 48000, nil), ErrInvalidStreamName)
	assert.ErrorIs(t, broadcaster.EnsureStream("default/stream", 48000, nil), ErrInvalidStreamName)

	_, _, err = broadcaster.Subscribe("invalid/name", 48000, nil)
	assert.ErrorIs(t, err, ErrInvalidStreamName)
}

func TestLiveAudioBroadcaster_SlowListenerNonBlocking(t *testing.T) {
	broadcaster, err := NewLiveAudioBroadcaster()
	require.NoError(t, err)
	defer broadcaster.Close()

	sampleRate := 48000
	// Slow listener (never drains channel)
	slowCh, unsubSlow, err := broadcaster.Subscribe("testlang", sampleRate, nil)
	require.NoError(t, err)
	defer unsubSlow()
	_ = slowCh

	// Fast listener
	fastCh, unsubFast, err := broadcaster.Subscribe("testlang", sampleRate, nil)
	require.NoError(t, err)
	defer unsubFast()

	chunk := generateStereoSineChunk(sampleRate, 100, 440)
	// Push many chunks to fill slow buffer
	for i := 0; i < 70; i++ {
		err := broadcaster.Publish("testlang", sampleRate, chunk)
		require.NoError(t, err)
	}

	// Verify fast listener still receives packets without blocking broadcaster
	received := 0
	deadline := time.After(2 * time.Second)
	for received < 3 {
		select {
		case data, ok := <-fastCh:
			if !ok {
				t.Fatal("fast channel closed unexpectedly")
			}
			if len(data) > 0 {
				received++
			}
		case <-deadline:
			t.Fatalf("fast listener timed out; only received %d chunks", received)
		}
	}
}
