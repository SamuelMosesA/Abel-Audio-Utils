package web

import (
	"behringerRecorder/lib/config"
	"behringerRecorder/lib/types"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCalculatePeakMeters(t *testing.T) {
	buffer := []float32{0.5, -0.3, 0.1, 0.2, -0.8, 0.4}
	maxL, maxR := CalculatePeakMeters(buffer)
	assert.Equal(t, float32(0.8), maxL)
	assert.Equal(t, float32(0.4), maxR)
}

func TestStartAudioBroadcaster(t *testing.T) {
	state := &types.AppState{}
	cfg := &config.Config{
		BufferSize:             512,
		GeminiChunkMultiplier: 2,
	}
	playbackChan := make(chan []float32, 1)

	StartAudioBroadcaster(state, cfg, playbackChan)

	// Register a stream channel
	ch := make(chan []float32, 1)
	state.StreamChannels.Store(ch, true)

	chunk := []float32{0.1, 0.2}
	playbackChan <- chunk

	select {
	case received := <-ch:
		assert.Equal(t, chunk, received)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Audio not broadcasted to stream channel")
	}

	close(playbackChan)
}
