# Bug Fix: Add a button to restart the engine (detect new devices, reload config)

- **Slug**: restart-engine-button
- **Fixed**: 2026-09-27
- **Assessment**: ./assessment.md
- **Status**: applied

## Summary

Added an admin-only `POST /api/system/restart` endpoint and a **Restart Engine** button in the admin Engine Control card. A restart does four things: it stops the audio engine and waits for it to close its stream, re-initializes PortAudio so newly connected devices are detected, reloads `config.yaml` and the credentials file (applying only the settings that are safe to change live), and reconnects to the previously selected device **by name**. The button is disabled while recording, and the backend independently refuses a restart while recording. The out-of-range / negative `deviceID` panic found during assessment is also fixed.

Open questions from the assessment were resolved with the recommended defaults, which the user approved on 2026-09-27:
1. The credentials file **is** reloaded; existing sessions stay valid.
2. If the previous device is missing after the re-scan, the engine **stays stopped** and does not fall back to another device.
3. Interrupting listeners is allowed **after a confirmation prompt**.

## Changes

| File | Change | Notes |
|------|--------|-------|
| `src/backend/lib/audioengine/restart.go` | added | `Restarter`, `RestartResult`, `ErrRecordingInProgress`, `ErrRestartInProgress`; PortAudio/config hooks are injectable for tests |
| `src/backend/lib/audioengine/engine.go` | modified | `StopAudioEngine` waits on a done channel (replaces `time.Sleep(100ms)`); device ID validated (`< 0` and `>= len`) **before** touching the running engine; start/stop serialized by `AppState.EngineLifecycle` |
| `src/backend/lib/audioengine/detect.go` | modified | new `InputDevices` helper (shared by startup and restart); `GetDevices` returns `[]` instead of `null` when empty |
| `src/backend/lib/state/app_state.go` | modified | device list moved behind `Devices()` / `SetDevices()` with its own RWMutex; added `EngineDone` and `EngineLifecycle` |
| `src/backend/lib/state/engine.go` | modified | `BeginRestart` / `EndRestart` / `IsRestarting` atomic flag |
| `src/backend/lib/config/config.go` | modified | `CheckCredentials` (locked read); `ApplyReload` applies live-safe fields and lists those that need a full restart |
| `src/backend/lib/web/handlers_system.go` | modified | `RestartEngineHandler` (200 / 409 / 500 / 503) + swagger annotations; `EngineRestarter` interface |
| `src/backend/lib/web/router.go` | modified | registers `POST /api/system/restart` inside `RegisterAdminRoutes` (session-protected) |
| `src/backend/lib/web/handlers_audio.go` | modified | `UpdateAudioConfig` starts the engine **outside** `state.Update` (see Deviations); `DevicesHandler` uses `Devices()` |
| `src/backend/lib/web/handlers_recordings.go` | modified | recording start refused while a restart is in progress |
| `src/backend/lib/web/handlers_auth.go` | modified | login uses `cfg.CheckCredentials` |
| `src/backend/main.go` | modified | uses `InputDevices`/`SetDevices`; builds `NewRestarter` with the config path |
| `src/backend/lib/web/test_helpers.go` | modified | `setupTestRouterWithRestarter` |
| `src/frontend/src/lib/components/admin/RestartEngineButton.svelte` | added | button + hint; disabled when `isRecording \|\| isRestarting` |
| `src/frontend/src/lib/components/admin/AudioAdminView.svelte` | modified | renders the button directly above the "Audio Engine Configuration" card (moved there at the user's request, 2026-09-27), confirm prompt, result notification; START also disabled while restarting |
| `src/frontend/src/lib/audioState.svelte.ts` | modified | `AudioStore.isRestarting`, `restartEngine()`, `describeRestart()` |
| tests | added / updated | see below |

## Diff Highlights

Stopping now waits for the engine instead of sleeping:

```go
close(q)
...
select {
case <-done:
	return nil
case <-time.After(engineStopTimeout): // 2s
	return ErrEngineStopTimeout
}
```

Restart order (`restart.go`): `BeginRestart` → refuse if recording → stop engine (wait) → `pa.Terminate` + `pa.Initialize` → re-scan → reload config → reconnect by name.

The button (`RestartEngineButton.svelte`):

```svelte
<Button ... onclick={onrestart} disabled={isRecording || isRestarting}>
```

## Tests Added or Updated

Backend (Go):
- `audioengine/restart_test.go::TestRestartRefusedWhileRecording`: restart refused while recording; PortAudio untouched.
- `audioengine/restart_test.go::TestRestartRefusedWhenAlreadyRestarting`: a concurrent restart gets `ErrRestartInProgress`.
- `audioengine/restart_test.go::TestRestartDetectsNewDevicesAndReconnectsByName`: after a re-scan the mixer moves from #1 to #0 and is reconnected by name; output-only devices are filtered; the old stream is closed **before** PortAudio is re-initialized.
- `audioengine/restart_test.go::TestRestartLeavesEngineStoppedWhenDeviceMissing`: no fallback to another device.
- `audioengine/restart_test.go::TestRestartWithEngineStoppedOnlyRescans`
- `audioengine/restart_test.go::TestRestartReloadsConfig`: new password works and the old one doesn't; a changed `default_boost` is applied; a UI-tuned channel survives; `port` is reported as needing a full restart.
- `audioengine/restart_test.go::TestRestartKeepsConfigWhenReloadFails`: a broken YAML is reported, the old config is kept, and the device re-scan still succeeds.
- `audioengine/restart_test.go::TestRestartFailsWhenAudioReinitFails`
- `audioengine/engine_test.go::TestStartAudioEngineRejectsInvalidDevice`: `-1`, `1`, `99` return errors, no panic, engine not marked running.
- `audioengine/engine_test.go::TestStopAudioEngineWaitsForStreamClose`
- `config/config_test.go`: `CheckCredentials`, `ApplyReload` (live vs. full-restart fields, no-change case), `LoadConfig` reading the credentials file.
- `web/handlers_system_test.go`: restart requires login (401), returns the result (200), conflicts (409), failure (500), unavailable (503), and a real `Restarter` over HTTP refuses while recording.
- `web/handlers_recordings_test.go::TestCreateRecordingRefusedWhileEngineRestarting`
- `web/handlers_audio_test.go::TestUpdateAudioConfigInvalidDeviceDoesNotPanic`

Frontend (Vitest):
- `components/admin/RestartEngineButton.test.ts`: **disabled while recording** (with hint), enabled and clickable when idle, disabled while restarting.
- `audioState.test.ts`: `restartEngine()` posts to `/api/system/restart`, refreshes devices and config, and clears `isRestarting`; a 409 is surfaced; `describeRestart` messages.

## Local Verification

Go used: `C:\msys64\ucrt64\bin` with `GOROOT=C:\msys64\ucrt64\lib\go`, go1.27.1.

- Baseline before changes: `go test ./src/backend/...` → all packages ok. `npx vitest run` → 8/8 passed.
- `go build ./...` → ok. `go vet ./src/backend/...` → clean.
- `go test ./src/backend/...` → **all packages ok**.
- `go test -race -count=1 -skip TestSubtitlesHandler ./src/backend/lib/...` → **all packages ok, no data races**.
- `go test -race ./src/backend/lib/...` without the skip → `TestSubtitlesHandler` reports a data race in the `MockTranslator` subtitle channel (`test_helpers.go` / `handlers_ai_test.go:55`). It was **confirmed pre-existing** by running the same test with `-race` on an unmodified checkout of HEAD (`9c7d348`) in a temporary worktree: it fails there too. It is unrelated to this change.
- `npx vitest run` → **14/14 passed** (8 existing + 6 new).
- `npm run check` (svelte-check + tsc) → 0 errors, 0 warnings.
- Manual checks: **none yet.** Nothing was run against real audio hardware, and the app binary was not rebuilt or launched. The real PortAudio `Terminate`/`Initialize` cycle is only exercised through a mocked hook in the tests.

## Deviations from Assessment

- **`UpdateAudioConfig` restructured (scope expansion, needed for safety).** It used to call `StartAudioEngine` inside `state.Update`, which holds the AppState write lock. Now that `StartAudioEngine` waits for the previous engine goroutine to exit, and that goroutine takes the AppState read lock on every loop iteration, calling it under the lock would deadlock until the 2s timeout. The engine is now started first and the interface state updated afterwards. Response codes are unchanged.
- **Invalid device IDs no longer stop the running engine.** Previously an invalid ID stopped the current engine and then failed, leaving state saying "running". Now it's validated first, so a bad request leaves the current engine untouched.
- **Recording start is refused during a restart** (`handlers_recordings.go`). This wasn't listed in the assessment. It closes the window between the restart's "is recording?" check and the engine stop, which enforces the user's requirement that a restart can never interfere with a recording.
- **Button in its own component** (`RestartEngineButton.svelte`) rather than inline in `AudioAdminView.svelte`, so the disabled-while-recording rule can be unit-tested without mounting the whole admin view.
- **Live-reloaded settings are limited** to credentials, default channels/boost, sample rate and buffer size. OpenAI settings, AI languages and the OTLP endpoint are captured by long-lived objects (translator, telemetry) at startup, so they're reported under `needsFullRestart` together with port and storage paths rather than applied.
- **Swagger docs (`docs/`) not regenerated**: `swag` is not installed locally, and installing it requires a network download (see Follow-ups). The handler carries the annotations.

## Follow-ups

- Run `/speckit-bug-test slug=restart-engine-button`.
- **Manual hardware check (needed before the PR):** start Abel with the mixer unplugged → plug it in → Restart Engine → the mixer is listed and meters move; start recording → the button is greyed out; stop → the button is enabled again; also try from a phone.
- Rebuild the frontend static bundle and `abel.exe` before the manual check, since the backend embeds `static/`.
- Regenerate swagger docs (`swag init`) once `swag` is available.
- Fix the pre-existing `TestSubtitlesHandler` data race in the `MockTranslator` test helper (separate issue).
- Known residual: if a device hangs so badly that its stream never returns from `Read()`, the restart gives up after 2s with an error rather than terminating PortAudio under an open stream. The admin would then need a full restart.
