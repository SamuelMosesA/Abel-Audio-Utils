package audioengine

import (
	"behringerRecorder/lib/state"
	"encoding/binary"
	"io"
)

// StartStorageWorker starts a goroutine that processes audio chunks and writes them to disk.
//
// Data Flow:
// 1. Receives float32 audio chunks from recordChan (stereo interleaved: [L, R, L, R, ...])
// 2. Converts each float32 sample to int16:
//   - float32 range: -1.0 to +1.0
//   - int16 range: -32768 to +32767
//   - Conversion: float32 * 32767 ≈ int16
//
// 3. Writes int16 pairs (stereo samples) as little-endian bytes to state.File
// 4. Tracks total samples written in state.SamplesWrote
//
// Data Format:
//
//	Input (float32): 32-bit IEEE 754 floating point [-1.0 to 1.0]
//	Output (int16): 16-bit signed integer [-32768 to 32767]
//	Encoding: Little Endian (LSB first, native for x86/ARM)
//	Layout: Stereo interleaved [Left, Right, Left, Right, ...]
//
// Example:
//
//	Input chunk: [0.5, -0.3, 0.1, 0.2]
//	Converted: [16384, -9831, 3277, 6554] (approx)
//	On disk (hex): 00 40 59 D8 0C 0C 4A 19
func StartStorageWorker(appState *state.AppState, recordChan <-chan []float32) {
	go func() {
		for chunk := range recordChan {
			if !appState.IsRecording() || appState.Engine().File() == nil {
				continue
			}

			n, err := WriteAudio(appState.Engine().File(), chunk)
			if err == nil {
				appState.Engine().AddSamples(int64(n))
			}
		}
	}()
}

// WriteAudio converts float32 stereo chunks to int16 and writes them to the provided writer.
// Returns the number of stereo samples (pairs) written.
func WriteAudio(w io.Writer, chunk []float32) (int, error) {
	if len(chunk) == 0 {
		return 0, nil
	}

	// Process pairs of float32 samples (stereo)
	for i := 0; i < len(chunk)-1; i += 2 {
		sL, sR := chunk[i], chunk[i+1]
		// Convert float32 [-1.0, 1.0] to int16 [-32768, 32767]
		iL, iR := int16(sL*32767), int16(sR*32767)
		// Write as little-endian int16 values
		if err := binary.Write(w, binary.LittleEndian, iL); err != nil {
			return i / 2, err
		}
		if err := binary.Write(w, binary.LittleEndian, iR); err != nil {
			return i / 2, err
		}
	}
	return len(chunk) / 2, nil
}

