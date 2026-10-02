# Bug Assessment: Improve stability of recording path

- **Slug**: recording-stability
- **Created**: 2026-09-27
- **Source**: https://github.com/SamuelMosesA/Abel-Audio-Utils/issues/14 (user-provided issue reference & codebase triage)
- **Verdict**: valid
- **Severity**: high

## Report (verbatim or summarized)

> Improve stability of recording path
> https://github.com/SamuelMosesA/Abel-Audio-Utils/issues/14
>
> The audio recording pipeline exhibits instability under continuous capture and stop/start cycles, resulting in dropped frames, race conditions during file finalization, and high I/O overhead.

## Symptom

Under active recording or during stop/start operations, the recording pipeline suffers from dropped audio chunks (silently dropped frames causing stutter/gaps), concurrency races when closing/finalizing WAV files, and excessive syscall overhead from unbuffered per-sample binary serialization. Expected behavior is reliable, stutter-free recording with atomic and thread-safe start/stop lifecycle management and valid WAV headers.

## Reproduction

1. Configure an audio input device and start capture in Abel.
2. Initiate recording via `POST /api/recordings` (`action: "start"`).
3. Introduce disk I/O latency or record high-sample-rate audio continuously. Observe silent chunk drops when `recordChan` buffer (capacity 100) fills up.
4. Concurrently issue `POST /api/recordings` (`action: "stop"`) while audio chunks are actively arriving on `recordChan`.
5. Observe race conditions between `StartStorageWorker` (`WriteAudio`) and `CreateRecording` (`FinalizeWavHeader` / `file.Close()`), potentially producing corrupted WAV headers, writing to closed file descriptors, or incorrect sample counts.

## Suspected Code Paths

- `src/backend/lib/audioengine/storage.go:36-53` (`StartStorageWorker`) — Processes audio chunks without mutex/synchronization against stop/close operations; reads `appState.Engine().File()` unsafely across goroutines.
- `src/backend/lib/audioengine/storage.go:57-76` (`WriteAudio`) — Converts and writes audio sample-by-sample using individual `binary.Write` calls (2 syscalls + reflection per stereo frame = ~88,200 writes/sec for 44.1kHz), causing severe I/O bottleneck and latency spikes.
- `src/backend/lib/audioengine/engine.go:167-170` (`StartAudioEngine`) — Uses non-blocking `select` with empty `default` to push to `recordChan`, silently discarding audio frames without telemetry, logging, or drop counters when storage lags.
- `src/backend/lib/web/handlers_recordings.go:98-118` (`CreateRecording` stop handler) — Closes file and finalizes header while storage worker may still have unwritten chunks in `recordChan` or be mid-write on the file handle.
- `src/backend/lib/state/engine.go:10,26-32` (`EngineState`) — Stores `file *os.File` without mutex protection, despite concurrent access from HTTP handler and storage worker goroutines.

## Root Cause Hypothesis

Confidence: high. The recording instability stems from a combination of three root causes:
1. **Inefficient I/O Serialization**: `WriteAudio` writes 2 bytes per sample directly via `binary.Write` without batching into a byte buffer or `bufio.Writer`. This generates tens of thousands of syscalls per second, causing storage worker latency to spike and `recordChan` (size 100) to fill up.
2. **Silent Drop on Channel Congestion**: When `recordChan` is saturated, `engine.go` silently drops chunks in the `default` select branch without warning or tracking.
3. **Goroutine Data Race on File Lifecycle**: `CreateRecording` stops recording by reading `SamplesWrote()`, finalizing the header, and closing `file`, while `StartStorageWorker` is concurrently reading from `recordChan` and writing to that same `*os.File`. There is no synchronization/draining mechanism to ensure in-flight chunks are flushed before finalization and close.

## Proposed Remediation

**Preferred**:
1. **Batch Sample Conversion & Buffered Writes**:
   - Refactor `WriteAudio` to encode the entire chunk into a pre-allocated byte slice (using `binary.LittleEndian.PutUint16`) and execute a single bulk `Write` per chunk, eliminating per-sample syscall and reflection overhead.
2. **Synchronized File & Recording Lifecycle**:
   - Encapsulate file handle access and sample tracking within `EngineState` or dedicated recording session controller with mutex protection.
   - On stop, coordinate with `StartStorageWorker` (e.g., using a session stop/flush signal or channel draining) to ensure all pending chunks for the active file are written before `FinalizeWavHeader` and `file.Close()` are called.
3. **Drop Tracking & Observability**:
   - Add telemetry counters and warning logs when `recordChan` is full and chunks are dropped in `engine.go`.

**Alternatives**:
- Dedicated recording worker per session that owns the `os.File` directly (passed via channel), avoiding shared pointer access in `AppState`. When recording stops, a sentinel control message is sent through the channel to trigger flush, header finalization, and close in the worker itself.

**Files likely to change**:
- `src/backend/lib/audioengine/storage.go`
- `src/backend/lib/audioengine/engine.go`
- `src/backend/lib/state/engine.go`
- `src/backend/lib/web/handlers_recordings.go`
- `src/backend/lib/audioengine/storage_test.go`
- `src/backend/lib/web/handlers_recordings_test.go`

**Tests to add or update**:
- Unit test for batched `WriteAudio` verifying exact byte layout and throughput.
- Concurrency/race test stopping recording while high-frequency audio chunks are streamed through `recordChan`.
- End-to-end recording test verifying valid WAV header and correct file size matching total samples written without corrupt headers or file descriptor leaks.

## Risks & Considerations

- Buffer batching must preserve little-endian 16-bit PCM format compatibility with standard WAV parsers.
- Draining `recordChan` on stop must not block HTTP handler response indefinitely if engine stalls (use timeout or asynchronous finalization with state notification).
- Preserves backward compatibility with existing telemetry metrics (`RecordingLatency`).

## Open Questions

- None blocking. Implementation details can proceed directly via `/speckit-bug-fix`.
