package audioengine

import (
	"abel/src/backend/lib/audioengine/audio_processing"
	"abel/src/backend/lib/state"
	"abel/src/backend/lib/telemetry"
	"context"
	"io"
	"log/slog"
	"os"
	"time"
)

// StartStorageWorker starts a dedicated goroutine that consumes audio chunks and writes them to disk.
//
// Lock-Free Data Flow:
// 1. Receives float32 interleaved stereo chunks from recordChan ([L0, R0, L1, R1, ...]).
// 2. Converts and clamps samples to 16-bit Little-Endian PCM via audio_processing.ConvertStereoFloat32ToPCM16LE.
// 3. Writes converted bytes directly to disk via appState.Engine().WriteWithFile without mutex lock contention.
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

// WriteAudio converts float32 stereo chunks using audio_processing.ConvertStereoFloat32ToPCM16LE
// and writes them to the provided writer in a single batched buffer write.
// Returns the number of stereo samples (pairs) written.
func WriteAudio(w io.Writer, chunk []float32) (int, error) {
	pcm := audio_processing.ConvertStereoFloat32ToPCM16LE(chunk)
	if len(pcm) == 0 {
		return 0, nil
	}
	nBytes, err := w.Write(pcm)
	return nBytes / 4, err
}
