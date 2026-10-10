package conversion

// DownsampleStereoToMonoPCM24k converts stereo float32 samples at srcRate
// into 24kHz mono 16-bit little-endian PCM bytes suitable for speech recognition / OpenAI Realtime.
func DownsampleStereoToMonoPCM24k(chunk []float32, srcRate int) []byte {
	if len(chunk) < 2 {
		return nil
	}
	if srcRate <= 0 {
		srcRate = DefaultWavSampleRate
	}
	dstRate := OpenAIRate

	// If source rate is already 24kHz, simply mix stereo down to mono and convert to PCM16
	if srcRate == dstRate {
		monoLen := len(chunk) / 2
		downsampled := make([]int16, monoLen)
		for i := 0; i < monoLen; i++ {
			avg := (chunk[i*2] + chunk[i*2+1]) / 2.0
			if avg > 1.0 {
				avg = 1.0
			} else if avg < -1.0 {
				avg = -1.0
			}
			downsampled[i] = int16(avg * 32767)
		}
		bytes := make([]byte, monoLen*2)
		for i, v := range downsampled {
			bytes[i*2] = byte(v & 0xff)
			bytes[i*2+1] = byte(v >> 8)
		}
		return bytes
	}

	// General downsampling using accumulators/ratios
	ratio := float64(srcRate) / float64(dstRate)
	srcFrames := len(chunk) / 2
	dstFrames := int(float64(srcFrames) / ratio)
	if dstFrames <= 0 {
		return nil
	}

	downsampled := make([]int16, dstFrames)
	for i := 0; i < dstFrames; i++ {
		startFrame := int(float64(i) * ratio)
		endFrame := int(float64(i+1) * ratio)
		if endFrame > srcFrames {
			endFrame = srcFrames
		}
		if endFrame <= startFrame {
			endFrame = startFrame + 1
		}

		var sum float32
		count := 0
		for f := startFrame; f < endFrame; f++ {
			sum += chunk[f*2] + chunk[f*2+1]
			count += 2
		}
		avg := sum / float32(count)
		if avg > 1.0 {
			avg = 1.0
		} else if avg < -1.0 {
			avg = -1.0
		}
		downsampled[i] = int16(avg * 32767)
	}

	bytes := make([]byte, len(downsampled)*2)
	for i, v := range downsampled {
		bytes[i*2] = byte(v & 0xff)
		bytes[i*2+1] = byte(v >> 8)
	}
	return bytes
}
