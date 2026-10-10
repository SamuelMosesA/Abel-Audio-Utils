# Phase 1 Data Model: Audio Engine Lifecycle

## Entities & Concurrency Primitives

### 1. `EngineLifecycle` (Concurrency & State Control)
Coordinates safe transitions for the audio stream capture loop.

| Field | Type | Description |
|-------|------|-------------|
| `mu` | `sync.Mutex` | Package-level mutex serializing `RestartEngine`, `StartAudioEngine`, and `StopAudioEngine`. |
| `quit` | `chan struct{}` | Signals active streaming goroutine to terminate. |
| `done` | `chan struct{}` | Signaled by streaming goroutine upon exiting and closing PortAudio stream. |

#### Lifecycle State Transitions:
1. `IDLE` -> `STARTING`: Acquire `mu`, initialize stream via PortAudio, allocate `quit` and `done` channels, launch capture goroutine, set `isRunning = true`.
2. `RUNNING` -> `STOPPING`: Acquire `mu` (or called within `RestartEngine`), close `quit` channel, await `<-done` (or 2s timeout).
3. `STOPPING` -> `IDLE`: Capture goroutine defers `stream.Stop()`, `stream.Close()`, `appState.Engine().SetRunning(false)`, and `close(done)`.
4. `RESTARTING`: `StopAudioEngine` -> `reinitPortAudio()` under `mu` -> `pa.Devices()` -> re-read configuration -> `StartAudioEngine` with matching device ID.

---

### 2. `RestartResult` (API Output Contract)
Returned by `POST /api/audio/restart`.

| Field | Type | Description |
|-------|------|-------------|
| `devices` | `[]AudioDevice` | List of newly enumerated audio devices with inputs > 0. |
| `deviceID` | `int` | ID of reconnected device, or `-1` if no device reconnected. |
| `reconnected` | `string` (optional) | Name of device reconnected. |
| `configError` | `string` (optional) | Error message if configuration reload encountered non-fatal issues. |
