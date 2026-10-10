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

// ExtractStereoChunk extracts left and right channels from multi-channel interleaved audio,
// applies digital gain boost, and clamps samples strictly to [-1.0, 1.0].
// If boost <= 0, unity gain (1.0) is used.
func ExtractStereoChunk(in []float32, bufferSize int, openedChannels int, chL int, chR int, boost float32) []float32 {
	if boost <= 0 {
		boost = 1.0
	}
	stereoChunk := make([]float32, bufferSize*2)
	if openedChannels <= 0 {
		return stereoChunk
	}
	for i := 0; i < bufferSize; i++ {
		idxL := (i * openedChannels) + chL
		idxR := (i * openedChannels) + chR

		var sL, sR float32
		if idxL >= 0 && idxL < len(in) {
			sL = in[idxL]
		}
		if idxR >= 0 && idxR < len(in) {
			sR = in[idxR]
		}

		sL *= boost
		sR *= boost
		if sL > 1.0 {
			sL = 1.0
		} else if sL < -1.0 {
			sL = -1.0
		}
		if sR > 1.0 {
			sR = 1.0
		} else if sR < -1.0 {
			sR = -1.0
		}

		stereoChunk[i*2] = sL
		stereoChunk[i*2+1] = sR
	}
	return stereoChunk
}
