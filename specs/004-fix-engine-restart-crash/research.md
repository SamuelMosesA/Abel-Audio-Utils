# Phase 0 Research: Safe and Crash-Resilient Audio Engine Restart

## Context & Root Cause Investigation

When an administrator requests an audio engine restart (`POST /api/audio/restart`), the backend server crashed with:
```text
SIGSEGV: segmentation violation
signal arrived during cgo execution
runtime.cgocall(...)
github.com/gordonklaus/portaudio._Cfunc_Pa_Initialize()
github.com/gordonklaus/portaudio.Initialize()
...
abel/src/backend/lib/audioengine.refreshDevices(...)
abel/src/backend/lib/audioengine.RestartEngine(...)
```
And during rapid or concurrent restart requests:
```text
{"error":"reinitialize audio: PortAudio not initialized"}
{"error":"list audio devices: PortAudio not initialized"}
```

### Analysis of Failure Vectors

1. **PortAudio C Library Thread-Safety**:
   - Gordon Klaus's Go wrapper `portaudio` invokes C PortAudio (`Pa_Initialize()`, `Pa_Terminate()`, `Pa_Devices()`, `Pa_OpenStream()`).
   - C PortAudio is fundamentally **not thread-safe** across initialization and termination. Concurrent calls to `Pa_Initialize` and `Pa_Terminate` corrupt internal C state tables and ALSA/Jack client pointers, causing hard segmentation faults (`SIGSEGV`).
2. **Missing Synchronization in Engine Lifecycle**:
   - `RestartEngine`, `StartAudioEngine`, and `refreshDevices` contained zero mutex protection.
   - If two restart requests arrive concurrently (e.g. rapid user clicks or multiple tabs), both execute `refreshDevices` simultaneously.
3. **Non-Deterministic Audio Stream Teardown**:
   - In `StartAudioEngine`, the capture loop calls `stream.Read(in)`, which is a blocking call waiting on the audio hardware buffer.
   - When `refreshDevices` closes `appState.QuitAudio`, it sleeps in 10ms intervals checking `appState.Engine().IsRunning()`.
   - However, in `StartAudioEngine`, there was no synchronization mechanism: calling `StartAudioEngine` directly simply closed `QuitAudio`, slept 100ms arbitrarily without waiting for the old engine goroutine to stop, and launched a second goroutine reading from PortAudio!
4. **Channel Panics & Double Close**:
   - `appState.QuitAudio` had no mutex protection. Simultaneous calls could attempt to close an already-closed channel or overwrite it while an active goroutine is listening.

---

## Technical Decisions

### Decision 1: Dedicated Concurrency Mutex for Engine Operations
- **Decision**: Add a package-level mutex (`engineMu sync.Mutex`) in `audioengine` protecting `RestartEngine`, `StartAudioEngine`, `StopAudioEngine`, and driver re-initialization.
- **Rationale**: Completely serializes hardware driver operations, guaranteeing that only one driver initialization/termination or stream creation can happen at any time.
- **Alternative considered**: Mutex inside HTTP handler only. Rejected because `StartAudioEngine` is also called from `UpdateAudioConfig` and background routines; protection must be enforced at the engine domain level.

### Decision 2: Deterministic Stop Function with WaitGroup / Done Channel
- **Decision**: Introduce `StopAudioEngine(appState *state.AppState)` with a dedicated `done` channel or `sync.WaitGroup` stored on `appState` (or internal engine state) so callers can cleanly await goroutine exit before terminating PortAudio.
- **Rationale**: Eliminates arbitrary `time.Sleep(100ms)` and guarantees that `stream.Stop()` and `stream.Close()` have completed before `pa.Terminate()` is called.
- **Alternative considered**: Terminate PortAudio immediately while stream is active. Rejected because PortAudio explicitly prohibits terminating while streams are open (undefined behavior/SIGSEGV).

### Decision 3: HTTP Handler Conflict Handling
- **Decision**: In `RestartAudioEngine` handler, reject concurrent restart requests or wait under mutex with a non-blocking `TryLock()` returning HTTP 409 Conflict ("Engine restart already in progress") or orderly serialization.
- **Rationale**: Immediate feedback to UI and prevents accumulation of queued restarts when hardware re-scan takes 100-200ms.
- **Alternative considered**: Unbounded queueing. Rejected because running multiple successive hardware rescans back-to-back causes unnecessary audio dropouts.

### Decision 4: Safe Reinit & Error Recovery
- **Decision**: Wrap `reinitPortAudio` so that if `pa.Terminate()` fails or `pa.Initialize()` fails, PortAudio is re-initialized safely, and any error is returned cleanly without panicking.
- **Rationale**: Ensures the web server process survives transient hardware disconnection or driver errors.
