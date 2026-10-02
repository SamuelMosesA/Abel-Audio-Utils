package audioengine

import (
	"abel/src/backend/lib/state"
	"abel/src/backend/lib/telemetry"
	"context"
	"encoding/binary"
	"io"
	"log/slog"
	"os"
	"time"
)

// StartStorageWorker starts a goroutine that processes audio chunks and writes them to disk.
//
// Data Flow:
// 1. Receives float32 audio chunks from recordChan (stereo interleaved: [L, R, L, R, ...])
// 2. Converts each float32 sample to int16 (clamped [-1.0, 1.0])
// 3. Batches converted int16 pairs into a byte buffer (Little Endian)
// 4. Writes batched bytes via appState.Engine().WriteWithFile to protect against concurrent file closures
// 5. Tracks total samples written safely under mutex
func StartStorageWorker(appState *state.AppState, recordChan <-chan []float32) {
	logger := slog.With("component", "storage")
	go func() {
		for chunk := range recordChan {
			if !appState.IsRecording() {
				continue
			}

			startTime := time.Now()
			n, err := appState.Engine().WriteWithFile(func(f *os.File) (int, error) {
				return WriteAudio(f, chunk)
			})
			if err != nil {
				logger.Error("Failed to write audio chunk", slog.Any("error", err))
			}
			if n > 0 && telemetry.RecordingLatency != nil {
				telemetry.RecordingLatency.Record(context.Background(), float64(time.Since(startTime).Nanoseconds())/1e6)
			}
		}
	}()
}

// WriteAudio converts float32 stereo chunks to int16 and writes them to the provided writer in a single batched buffer write.
// Returns the number of stereo samples (pairs) written.
func WriteAudio(w io.Writer, chunk []float32) (int, error) {
	numPairs := len(chunk) / 2
	if numPairs == 0 {
		return 0, nil
	}

	buf := make([]byte, numPairs*4)
	for i := 0; i < numPairs; i++ {
		sL, sR := chunk[i*2], chunk[i*2+1]
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

		iL := int16(sL * 32767)
		iR := int16(sR * 32767)

		binary.LittleEndian.PutUint16(buf[i*4:], uint16(iL))
		binary.LittleEndian.PutUint16(buf[i*4+2:], uint16(iR))
	}

	nBytes, err := w.Write(buf)
	return nBytes / 4, err
}

