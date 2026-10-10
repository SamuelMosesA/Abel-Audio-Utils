package openai

import (
	"testing"
	"time"

	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sync_map "github.com/zolstein/sync-map"
)

func TestPendingAudioBuffer(t *testing.T) {
	t.Run("bounded capacity evicts oldest chunks", func(t *testing.T) {
		buf := NewPendingAudioBuffer(10) // 10 bytes max

		buf.Push([]byte{1, 2, 3, 4})
		assert.Equal(t, 4, buf.Bytes())
		assert.Equal(t, 1, buf.Len())

		buf.Push([]byte{5, 6, 7, 8})
		assert.Equal(t, 8, buf.Bytes())
		assert.Equal(t, 2, buf.Len())

		// Adding 4 bytes exceeds 10 bytes (8 + 4 = 12 > 10), so first chunk {1,2,3,4} is dropped
		buf.Push([]byte{9, 10, 11, 12})
		assert.Equal(t, 8, buf.Bytes())
		assert.Equal(t, 2, buf.Len())

		chunk := buf.Pop()
		assert.Equal(t, []byte{5, 6, 7, 8}, chunk)
		assert.Equal(t, 1, buf.Len())

		chunk = buf.Pop()
		assert.Equal(t, []byte{9, 10, 11, 12}, chunk)
		assert.Equal(t, 0, buf.Len())

		assert.Nil(t, buf.Pop())
	})

	t.Run("clear resets buffer", func(t *testing.T) {
		buf := NewPendingAudioBuffer(100)
		buf.Push([]byte{1, 2, 3})
		assert.Equal(t, 3, buf.Bytes())
		buf.Clear()
		assert.Equal(t, 0, buf.Bytes())
		assert.Equal(t, 0, buf.Len())
	})
}

func TestBroadcastSubtitle(t *testing.T) {
	var subs sync_map.Map[string, []chan string]
	ch1 := make(chan string, 5)
	ch2 := make(chan string, 5)

	subs.Store("es", []chan string{ch1, ch2})

	BroadcastSubtitle(&subs, "es", "Hola Mundo")

	select {
	case msg := <-ch1:
		assert.Contains(t, msg, "Hola Mundo")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("ch1 did not receive subtitle")
	}

	select {
	case msg := <-ch2:
		assert.Contains(t, msg, "Hola Mundo")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("ch2 did not receive subtitle")
	}

	// Broadcasting to non-existent language should not panic
	BroadcastSubtitle(&subs, "fr", "Bonjour")
}

func TestAudioUtil(t *testing.T) {
	appState := state.NewAppState("/tmp/storage", "/tmp/cloud")
	cfg := &config.Config{SampleRate: 48000}

	rate := ResolveSampleRate(appState, cfg)
	assert.Equal(t, 48000, rate)

	chunk := []float32{0.5, 0.5, -0.5, -0.5}
	pcm := DownsampleChunkForAI(chunk, appState, cfg)
	require.NotEmpty(t, pcm)

	// Decode delta with 0 sample rate should default cleanly
	floats, err := DecodeAIDelta("", appState, cfg)
	assert.NoError(t, err)
	assert.Empty(t, floats)
}
