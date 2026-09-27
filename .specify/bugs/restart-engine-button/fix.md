# Bug Fix: Add a button to restart the engine (detect new devices, reload config)

- **Slug**: restart-engine-button
- **Fixed**: 2026-09-27
- **Assessment**: ./assessment.md
- **Status**: applied

## Summary

Added an admin-only `POST /api/audio/restart` endpoint and a **Restart Engine** button above "Audio Engine Configuration". A restart:
1. stops the audio engine and waits for its stream to close,
2. re-initializes PortAudio and re-scans input devices, so hardware connected after startup appears,
3. reloads `config.yaml` and the credentials file,
4. reconnects to the previously used device by name.

It is refused while recording, and the button is disabled while recording. Also fixes the negative `deviceID` panic found during assessment.

This deliberately compact version (5 code files) replaces an earlier 27-file version. The earlier version is kept on the local branch `fix/restart-engine-button-full`.

## Changes

| File | Change | Notes |
|------|--------|-------|
| `src/backend/lib/audioengine/engine.go` | modified | `RestartEngine` + `RestartResult`; `refreshDevices` (stop → wait → PortAudio re-init → re-scan); `StartAudioEngine` rejects `deviceID < 0` and validates before touching the running engine |
| `src/backend/lib/config/config.go` | modified | `Config.Path`, set by `LoadConfig`, so the restart can reload the same file |
| `src/backend/lib/web/handlers_audio.go` | modified | `RestartAudioEngine` handler (409 while recording, 500 on failure, 200 with result) |
| `src/backend/lib/web/router.go` | modified | `POST /api/audio/restart` in `RegisterAdminRoutes` (session-protected) |
| `src/frontend/src/lib/components/admin/AudioAdminView.svelte` | modified | button, `disabled={audio.isRecording \|\| restarting}`, confirm prompt, result notification |
| `src/backend/lib/audioengine/engine_test.go` | added tests | see below |
| `src/backend/lib/web/handlers_audio_test.go` | added tests | see below |

## Diff Highlights

- **Waiting for the stream to close needs no new state.** In the engine goroutine, `SetRunning(false)` is deferred before `stream.Stop()`/`Close()`, so it runs after them (LIFO). `refreshDevices` polls `IsRunning()` (2s timeout) before calling `pa.Terminate()`.
- **Reconnect by name.** Device IDs are list positions that shift when hardware changes. If the device isn't found, the engine stays stopped rather than falling back to another input.
- **Config reload applies:** credentials, `default_ch_l`, `default_ch_r`, `default_boost`. Live routing/gain is overwritten only when those defaults changed in the file, so values tuned in the UI survive a plain re-scan. A broken or missing file is reported (`configError`) and the previous config is kept.

## Tests Added or Updated

- `audioengine/engine_test.go::TestStartAudioEngineRejectsInvalidDevice`: `-1`, `1`, `99` return an error; no panic; the engine is not marked running.
- `audioengine/engine_test.go::TestRestartEngineReconnectsByNameAndReloadsConfig`: the mixer moves from #1 to #0 and is reconnected by name; output-only devices are skipped; the stream is closed **before** PortAudio re-initializes; the new password and changed default boost are applied.
- `audioengine/engine_test.go::TestRestartEngineStaysStoppedWhenDeviceMissing`: no fallback device; a missing config file is reported and the previous config is kept; the UI-tuned channel is kept.
- `audioengine/engine_test.go::TestRestartEngineFailsWhenAudioReinitFails`
- `web/handlers_audio_test.go::TestRestartAudioEngineRequiresLogin`: 401.
- `web/handlers_audio_test.go::TestRestartAudioEngineRefusedWhileRecording`: 409; the recording is untouched.
- `web/handlers_audio_test.go::TestUpdateAudioConfigInvalidDeviceDoesNotPanic`

## Local Verification

- `go build ./...`, `go vet ./src/backend/...` → clean.
- `go test ./src/backend/...` → all ok.
- `go test -race -skip TestSubtitlesHandler ./src/backend/lib/...` → all ok, no races. (`TestSubtitlesHandler` has a pre-existing race on `main`, unrelated.)
- Frontend: `npm run check` → 0 errors; `npx vitest run` → 8/8. No frontend test was added; the disabled-while-recording rule is backed by the backend 409 test and checked manually.
- Manual: the feature was run on the church PC with built-in inputs. **Not yet tested with the Behringer mixer.**

## Deviations from Assessment

- **Only live-safe settings are reloaded** (credentials, default channels/boost). Port, storage paths, sample rate/buffer size, OpenAI and AI-language settings still need a full restart of Abel.
- **Kept minimal on concurrency.** There is no mutex around `cfg.Credentials` or `appState.Devices`, and no guard against starting a recording in the split second between the recording check and the stop. All are very unlikely with a single admin; the fuller version on `fix/restart-engine-button-full` has them if the team wants them.

## Follow-ups

- Hands-on test with the Behringer mixer.
- Regenerate swagger docs (`swag init`).
- Pre-existing `TestSubtitlesHandler` data race.
- README: document that `ffmpeg` must be on PATH on Windows since #44.
