# Bug Fix: Improve stability of recording path

- **Slug**: recording-stability
- **Fixed**: 2026-09-27
- **Assessment**: ./assessment.md
- **Status**: applied

## Summary

Optimized audio sample serialization with batched buffer writes, resolved concurrency data races during recording start/stop lifecycle using `EngineState` mutex protection with atomic `TakeFile` / `WriteWithFile`, and added telemetry tracking for dropped recording audio chunks.

## Changes

| File | Change | Notes |
|------|--------|-------|
| `src/backend/lib/audioengine/storage.go` | modified | Batched float32-to-int16 sample conversion into a single buffer write; switched worker to safe `WriteWithFile`. |
| `src/backend/lib/state/engine.go` | modified | Added mutex synchronization to `EngineState` and implemented `WriteWithFile` and `TakeFile` to prevent race conditions on file closure. |
| `src/backend/lib/web/handlers_recordings.go` | modified | Used `TakeFile()` in stop recording handler to ensure clean file detachment and accurate sample count finalization. |
| `src/backend/lib/audioengine/engine.go` | modified | Added warning logging and telemetry tracking when `recordChan` is saturated and drops frames. |
| `src/backend/lib/telemetry/metrics.go` | modified | Registered `dropped_audio_chunks_total` counter metric. |
| `src/backend/lib/audioengine/storage_test.go` | modified | Added tests for clamping, edge cases (empty, odd slices), and concurrent stop/take operations under active load. |
| `src/backend/lib/state/state_test.go` | modified | Added unit tests for `EngineState` safe operations and sample tracking. |
| `src/backend/lib/web/handlers_recordings_test.go` | modified | Added integration test `TestCreateRecordingWithActiveStorageWorker` testing full capture, stop, and WAV header verification. |

## Diff Highlights

```go
// src/backend/lib/audioengine/storage.go
func WriteAudio(w io.Writer, chunk []float32) (int, error) {
    numPairs := len(chunk) / 2
    if numPairs == 0 {
        return 0, nil
    }

    buf := make([]byte, numPairs*4)
    for i := 0; i < numPairs; i++ {
        sL, sR := chunk[i*2], chunk[i*2+1]
        if sL > 1.0 { sL = 1.0 } else if sL < -1.0 { sL = -1.0 }
        if sR > 1.0 { sR = 1.0 } else if sR < -1.0 { sR = -1.0 }

        iL := int16(sL * 32767)
        iR := int16(sR * 32767)

        binary.LittleEndian.PutUint16(buf[i*4:], uint16(iL))
        binary.LittleEndian.PutUint16(buf[i*4+2:], uint16(iR))
    }

    nBytes, err := w.Write(buf)
    return nBytes / 4, err
}
```

```go
// src/backend/lib/state/engine.go
func (e *EngineState) TakeFile() (*os.File, int64) {
    e.fileMu.Lock()
    defer e.fileMu.Unlock()
    f := e.file
    e.file = nil
    samples := e.samplesWrote.Load()
    return f, samples
}
```

## Tests Added or Updated

- `src/backend/lib/audioengine/storage_test.go::TestWriteAudioEdgeCasesAndClamping` — Validates empty slice handling, odd length slices, and clamping out-of-bounds float values.
- `src/backend/lib/audioengine/storage_test.go::TestStartStorageWorkerConcurrentStop` — Stresses concurrent high-frequency chunk writes while stopping recording and finalizing WAV headers without data races.
- `src/backend/lib/state/state_test.go::TestEngineStateOperations` — Tests thread-safe state operations (`WriteWithFile`, `TakeFile`, `ResetSamples`, `AddSamples`).
- `src/backend/lib/web/handlers_recordings_test.go::TestCreateRecordingWithActiveStorageWorker` — Tests full end-to-end recording workflow through HTTP API with live storage worker.

## Local Verification

- Commands run:
  - `go test ./...` in `src/backend` → PASS (all package tests passed)
  - `go test -race ./lib/audioengine ./lib/state` in `src/backend` → PASS (0 data races detected)
- Targeted test run:
  - `go test -v ./lib/audioengine ./lib/state ./lib/web -run "Test.*Recording.*|TestWrite.*|TestStartStorage.*|TestEngineState.*"` → PASS

## Deviations from Assessment

None. The implementation followed the preferred remediation exactly.

## Follow-ups

- Monitor `dropped_audio_chunks_total` metric in Prometheus/Grafana dashboard during long recording sessions.
