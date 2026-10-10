package audioengine

import (
	"abel/src/backend/lib/audioengine/audio_processing"
	"context"
	"time"
)

// HLSPublisher aliases the consolidated audio_processing.HLSPublisher.
type HLSPublisher = audio_processing.HLSPublisher

var (
	ErrHLSNotReady       = audio_processing.ErrHLSNotReady
	ErrInvalidStreamName = audio_processing.ErrInvalidStreamName
)

// NewHLSPublisher creates an HLSPublisher via audio_processing.
func NewHLSPublisher() (*HLSPublisher, error) {
	return audio_processing.NewHLSPublisher()
}

func newHLSPublisher(ffmpegPath, root string) *HLSPublisher {
	return audio_processing.NewHLSPublisherWithRoot(ffmpegPath, root)
}

func float32ToPCM16(chunk []float32) []byte {
	return audio_processing.Float32ToPCM16(chunk)
}

// WaitForHLSPlaylist waits for the HLS playlist to become ready.
func WaitForHLSPlaylist(ctx context.Context, p *HLSPublisher, language string, timeout time.Duration) ([]byte, error) {
	return p.WaitForPlaylist(ctx, language, timeout)
}
