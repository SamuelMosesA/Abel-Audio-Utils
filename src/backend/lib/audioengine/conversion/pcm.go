package conversion

import "encoding/binary"

const (
	// DefaultWavSampleRate is the fallback standard audio sampling rate in Hz.
	DefaultWavSampleRate = 48000
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
		if sample > 1.0 {
			sample = 1.0
		} else if sample < -1.0 {
			sample = -1.0
		}
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(int16(sample*32767)))
	}
	return pcm
}

// PCM16ToFloat32 converts signed 16-bit little-endian PCM bytes into normalized float32 samples.
func PCM16ToFloat32(data []byte) []float32 {
	numSamples := len(data) / 2
	floats := make([]float32, numSamples)
	for i := 0; i < numSamples; i++ {
		s := int16(binary.LittleEndian.Uint16(data[i*2 : (i+1)*2]))
		floats[i] = float32(s) / 32767.0
	}
	return floats
}
