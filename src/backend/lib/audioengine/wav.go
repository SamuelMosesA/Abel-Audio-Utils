package audioengine

import (
	"os"

	"abel/src/backend/lib/audioengine/audio_processing"
)

const (
	wavHeaderSize        = audio_processing.WavHeaderSize
	wavBitsPerSample     = audio_processing.WavBitsPerSample
	wavBytesPerSample    = audio_processing.WavBytesPerSample
	defaultWavSampleRate = audio_processing.DefaultWavSampleRate
)

// GenerateWavHeader constructs the 44-byte standard PCM WAV header in memory.
func GenerateWavHeader(ch uint16, dataSize uint32, sampleRate int) [wavHeaderSize]byte {
	return audio_processing.GenerateWavHeader(ch, dataSize, sampleRate)
}

// WritePlaceholderHeader delegates to audio_processing.WritePlaceholderWavHeader.
func WritePlaceholderHeader(f *os.File, ch uint16, sampleRate int) error {
	if f == nil {
		return nil
	}
	return audio_processing.WritePlaceholderWavHeader(f, ch, sampleRate)
}

// FinalizeWavHeader seeks back to the beginning of f and finalizes the header fields.
func FinalizeWavHeader(f *os.File, ch uint16, s int64, sampleRate int) error {
	if f == nil {
		return nil
	}
	dataBytes := s * int64(ch) * wavBytesPerSample
	return audio_processing.FinalizeWavHeader(f, ch, dataBytes, sampleRate)
}
