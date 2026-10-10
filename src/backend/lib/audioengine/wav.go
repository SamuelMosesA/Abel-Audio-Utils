package audioengine

import (
	"os"

	"abel/src/backend/lib/audioengine/conversion"
)

const (
	wavHeaderSize        = conversion.WavHeaderSize
	wavBitsPerSample     = conversion.WavBitsPerSample
	wavBytesPerSample    = conversion.WavBytesPerSample
	defaultWavSampleRate = conversion.DefaultWavSampleRate
)

// GenerateWavHeader constructs the 44-byte standard PCM WAV header in memory.
func GenerateWavHeader(ch uint16, dataSize uint32, sampleRate int) [wavHeaderSize]byte {
	return conversion.GenerateWavHeader(ch, dataSize, sampleRate)
}

// WritePlaceholderHeader delegates to conversion.WritePlaceholderWavHeader.
func WritePlaceholderHeader(f *os.File, ch uint16, sampleRate int) error {
	if f == nil {
		return nil
	}
	return conversion.WritePlaceholderWavHeader(f, ch, sampleRate)
}

// FinalizeWavHeader seeks back to the beginning of f and finalizes the header fields.
func FinalizeWavHeader(f *os.File, ch uint16, s int64, sampleRate int) error {
	if f == nil {
		return nil
	}
	dataBytes := s * int64(ch) * wavBytesPerSample
	return conversion.FinalizeWavHeader(f, ch, dataBytes, sampleRate)
}
