package conversion

import "encoding/binary"

const (
	// DefaultWavSampleRate is the fallback standard audio sampling rate in Hz.
	DefaultWavSampleRate = 44100
	// OpenAIRate is the target sampling rate for OpenAI Realtime audio.
	OpenAIRate = 24000

	WavHeaderSize     = 44
	WavBitsPerSample  = 16
	WavBytesPerSample = WavBitsPerSample / 8
)

// Float32ToPCM16 converts a slice of [-1.0, 1.0] float32 audio samples
// into signed 16-bit little-endian PCM bytes with hard-limiting clamping.
func Float32ToPCM16(chunk []float32) []byte {
	pcm := make([]byte, len(chunk)*2)
	for i, sample := range chunk {
		val := sample
		if val > 1.0 {
			val = 1.0
		} else if val < -1.0 {
			val = -1.0
		}
		raw := int16(val * 32767.0)
		binary.LittleEndian.PutUint16(pcm[i*2:(i+1)*2], uint16(raw))
	}
	return pcm
}
