package audioengine

import (
	"abel/src/backend/lib/config"
	"abel/src/backend/lib/state"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculatePeakMeters(t *testing.T) {
	buffer := []float32{0.5, -0.1, 0.2, 0.8}
	maxL, maxR := CalculatePeakMeters(buffer)
	assert.Equal(t, float32(0.5), maxL)
	assert.Equal(t, float32(0.8), maxR)
}

type capturePublisher struct {
	language   string
	sampleRate int
	chunk      []float32
	done       chan struct{}
}

func (p *capturePublisher) Publish(language string, sampleRate int, chunk []float32) error {
	p.language = language
	p.sampleRate = sampleRate
	p.chunk = append([]float32(nil), chunk...)
	close(p.done)
	return nil
}

func TestStartAudioBroadcasterPublishesOriginalAudio(t *testing.T) {
	appState := state.NewAppState("", "")
	cfg := &config.Config{SampleRate: 48000}
	playback := make(chan []float32, 1)
	publisher := &capturePublisher{done: make(chan struct{})}

	StartAudioBroadcaster(appState, cfg, playback, publisher)
	playback <- []float32{0.25, -0.25}
	close(playback)

	select {
	case <-publisher.done:
	case <-time.After(time.Second):
		require.FailNow(t, "publisher did not receive audio")
	}
	assert.Equal(t, "default", publisher.language)
	assert.Equal(t, 48000, publisher.sampleRate)
	assert.Equal(t, []float32{0.25, -0.25}, publisher.chunk)
}
