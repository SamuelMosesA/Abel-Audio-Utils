package portaudio

import (
	"behringerRecorder/lib/types"
	"encoding/binary"
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
func StartStorageWorker(state *types.AppState, recordChan <-chan []float32) {
	go func() {
		for chunk := range recordChan {
			state.Mu.Lock()
			if state.IsRecording {
				// Process pairs of float32 samples (stereo)
				for i := 0; i < len(chunk); i += 2 {
					sL, sR := chunk[i], chunk[i+1]
					// Convert float32 [-1.0, 1.0] to int16 [-32768, 32767]
					iL, iR := int16(sL*32767), int16(sR*32767)
					// Write as little-endian int16 values
					binary.Write(state.File, binary.LittleEndian, iL)
					binary.Write(state.File, binary.LittleEndian, iR)
				}
				// Track number of stereo sample pairs written
				state.SamplesWrote += int64(len(chunk) / 2)
			}
			state.Mu.Unlock()
		}
	}()
}
