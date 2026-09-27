# Bug Assessment: Add a button to restart the engine (detect new devices, reload config)

- **Slug**: restart-engine-button
- **Created**: 2026-09-27
- **Source**: https://github.com/SamuelMosesA/Abel-Audio-Utils/issues/28 (host: github.com, policy: allowlisted)
- **Verdict**: valid
- **Severity**: medium

## Report (verbatim or summarized)

> **Add button to restart engine** (#28, open, author: SamuelMosesA, created 2026-08-09, no labels/comments)
>
> The engine needs to be restarted to detect new devices. Also try to make the config also refreshed with this button.

The issue is phrased as a feature request, but it describes a real defect. An audio interface plugged in (or powered on) after Abel starts never shows up in the device list. Today the only fix is to kill and relaunch `abel.exe`.

## Symptom

**Observed:** If an audio interface (e.g. the Behringer) is connected after Abel starts, it does not appear in the admin "Interface Device" list, and it can't be selected until the whole process is restarted. Edits to `~/.config/abel/config.yaml` or `admin_user_credentials.json` are also ignored until a full process restart.

**Expected:** An admin can press a "Restart engine" button in the UI. The button re-scans audio devices and reloads the configuration without killing the server.

## Reproduction

1. Start Abel (`start-abel.bat`) with the audio interface unplugged.
2. Log in at `/admin`. The interface is not in the "Interface Device" dropdown.
3. Plug in the interface and refresh the page. It is still not listed: `GET /api/audio/devices` returns the same list as at startup.
4. Edit `~/.config/abel/config.yaml` (e.g. change `default_boost`) or the credentials file. The running app does not pick up the change.

[NEEDS CLARIFICATION: confirm step 3 on this machine with the real interface. The code analysis below makes it near-certain, but it has not been reproduced by hand in this assessment.]

## Suspected Code Paths

- `src/backend/main.go:76-77`: `pa.Initialize()` is called exactly once at startup (`defer pa.Terminate()` at exit). PortAudio builds its device table inside `Pa_Initialize` and does **not** support hot-plug, so new devices are only seen after `Terminate()` + `Initialize()`.
- `src/backend/main.go:108-113`: `appState.Devices` is filled once from `pa.Devices()` and never refreshed.
- `src/backend/lib/state/app_state.go:91`: `Devices []*pa.DeviceInfo` is a plain exported slice with no locking. Every reader (handlers, engine) assumes it never changes.
- `src/backend/lib/web/handlers_audio.go:15-23` (`DevicesHandler`): returns the cached `state.Devices`, so it can never report new hardware.
- `src/backend/lib/audioengine/engine.go:15-34` (`StartAudioEngine`): picks the device by **index** into `appState.Devices`. After a re-scan, indexes can shift, so a saved `deviceID` may point at a different device.
- `src/backend/lib/audioengine/engine.go:20-23`: stops the previous engine goroutine by closing `QuitAudio` and then `time.Sleep(100ms)`. There is no real "stopped" signal, which is unsafe if we `pa.Terminate()` right after.
- `src/backend/main.go:42-53` and `src/backend/lib/config/config.go:63-100` (`LoadConfig`): config and credentials are read once. The `*config.Config` pointer is shared by the router, handlers, engine, broadcaster and OpenAI manager (`main.go:94,116,119`), and nothing reloads it.
- `src/backend/lib/web/router.go:149-159` (`RegisterAdminRoutes`): no endpoint to stop, restart or reload anything.
- `src/frontend/src/lib/audioState.svelte.ts:66-73,378`: `fetchDevices()` is only called once, when the store starts.
- `src/frontend/src/lib/components/admin/AudioAdminView.svelte:80-108`: the "Engine Control" card has only START/STOP recording buttons, no restart.

## Root Cause Hypothesis

PortAudio snapshots the list of audio devices when `Pa_Initialize` is called, and Abel calls it only once at process start (`main.go:76`). It then copies the input-capable devices into `appState.Devices` (`main.go:108-113`) and serves that frozen list forever. There is no code path, and no API or UI control, that terminates and re-initializes PortAudio, re-enumerates devices, or re-reads `config.yaml`. That is why a process restart is currently the only way to see new hardware or config. **Confidence: high.**

## Proposed Remediation

**Preferred**: add an admin-only `POST /api/system/restart` endpoint plus a "Restart Engine" button in the admin Engine Control card.

> **Requirement (confirmed by user, 2026-09-27): the Restart Engine button MUST be disabled while a recording is in progress.** It becomes clickable again as soon as recording stops. The backend must *also* refuse a restart while recording (step 1 below), so a stale page, a second admin tab, or a direct API call can't interrupt a recording either.

Backend (`lib/audioengine`, `lib/state`, `lib/web`, `main.go`):

1. **Reject while recording.** Return `409`/`400` if `appState.IsRecording()`, the same guard `UpdateAudioConfig` uses.
2. **Stop the engine cleanly.** Add a `StopAudioEngine(appState)` that closes `QuitAudio` and **waits** for the engine goroutine to exit, e.g. through a `done` channel the goroutine closes after `stream.Close()`. This replaces the `time.Sleep(100ms)` pattern. The stream must be closed before step 3.
3. **Re-initialize PortAudio.** Call `pa.Terminate()` then `pa.Initialize()` to rebuild the device table.
4. **Re-enumerate devices.** Move the `main.go:108-113` loop into a shared `audioengine.RefreshDevices(appState)`. Guard `Devices` with the `AppState` mutex (a `Devices()` getter / `SetDevices()` setter) so handlers never read a half-replaced slice.
5. **Reload config.** Call `config.LoadConfig(<same path>)` and apply only the fields that are safe to change live: credentials, default channels, boost, sample rate, buffer size, AI language list and OpenAI settings. Log which fields need a full restart (`port`, `storage_location`, `cloud_drive_location`) and don't apply them. Because `*config.Config` is shared, replace its contents under a lock, or pass a getter, rather than reassigning the pointer.
6. **Restore the previous device if possible.** If the engine was running before, find the previously selected device **by name** in the new list and restart the engine on it. Otherwise set `deviceID = -1` and `isRunning = false`.
7. **Tell the UI.** Broadcast `SectionInterface` so connected admin pages re-sync, and return `{devices, config, restored}`.

Frontend:

- Add a `restartEngine()` method to the audio store in `audioState.svelte.ts` that calls the new endpoint, then `fetchDevices()` and `sync()`.
- Add a "Restart Engine" button in the Engine Control card of `AudioAdminView.svelte`. It must be `disabled={audio.isRecording}`, the same way the STOP/START buttons and config inputs already react to recording state, with a hint such as "Stop recording to restart". Show a spinner while the request is in flight, and show a success or error banner using the existing `NotificationBanner`.

**Alternatives**:
- *Only re-enumerate devices (no config reload).* Smaller and lower risk, but it leaves the second half of the issue unfixed.
- *Restart the whole process* (exit and let `start-abel.bat` or a service manager relaunch it). Simplest and fully refreshes everything, including port and paths. However, `start-abel.bat` doesn't loop, so the app would just stop, and every WebSocket and listener gets dropped. Not recommended for a live service.
- *Automatic hot-plug polling.* Periodically re-initialize PortAudio. This can't safely run while a stream is open, so a manual button is better.

**Files likely to change**:
- `src/backend/lib/audioengine/engine.go` (add `StopAudioEngine`, done-signal, `RefreshDevices`)
- `src/backend/lib/audioengine/detect.go` (possibly host the refresh helper)
- `src/backend/lib/state/app_state.go` (lock-guarded device list accessors, engine done channel)
- `src/backend/lib/config/config.go` (reload helper / safe-field apply)
- `src/backend/lib/web/handlers_system.go` (new restart handler + swagger annotations)
- `src/backend/lib/web/handlers_audio.go` (use device getter)
- `src/backend/lib/web/router.go` (register the admin route)
- `src/backend/main.go` (use shared refresh helper; keep config path for reload)
- `src/frontend/src/lib/audioState.svelte.ts`
- `src/frontend/src/lib/components/admin/AudioAdminView.svelte`
- `docs/` swagger artifacts (regenerate)

**Tests to add or update**:
- `lib/web/handlers_system_test.go`: restart returns 401 without a session, is rejected while recording, and on success returns the refreshed device list and applies reloaded config (use a temp config file).
- `lib/audioengine/engine_test.go`: `StopAudioEngine` waits for the goroutine to close the (mock) stream; `StartAudioEngine` after a device refresh chooses the right device; negative or out-of-range `deviceID` returns an error instead of panicking.
- `lib/config`: reload applies live-safe fields and ignores `port`/storage paths.
- `src/frontend/src/lib/audioState.test.ts`: `restartEngine()` calls the endpoint, then refreshes devices and config.
- Frontend component test: the Restart Engine button is disabled when `audio.isRecording` is true and enabled when it is false.
- Manual check: plug the interface in after startup, press Restart Engine, and confirm it appears and records.

## Risks & Considerations

- **Terminating PortAudio with an open stream** can crash or hang the native layer. The engine must be fully stopped (a real done signal, not a sleep) before `pa.Terminate()`.
- **Live listeners and AI streams:** stopping the engine currently calls `Translator.CloseAll()` (`engine.go:42-46`), so a restart interrupts listeners and translation sessions. The UI should warn about this.
- **Device index drift:** IDs are positions in the list, so after a re-scan the old `deviceID` may be a different device. Match by name.
- **Data races:** `appState.Devices` and `*config.Config` are read without locks today. Swapping them at runtime needs synchronization; verify with `go test -race`.
- **Partial config reload:** port and storage paths can't change live; make that visible (log + response) so admins aren't confused.
- **Security:** the new endpoint must sit behind `SessionAuthMiddleware` (inside `RegisterAdminRoutes`).
- **Related latent bug found during assessment:** `StartAudioEngine` doesn't reject `deviceID < 0` (`engine.go:31` only checks the upper bound), so `PATCH /api/audio/config` with `deviceID: -1` would panic on `devices[-1]` (recovered by `gin.Recovery`, but still a 500 plus a leaked "running" flag, since `SetRunning(true)` at `engine.go:27` runs before validation). It's worth fixing in the same change since the restart flow may set `deviceID = -1`.

## Open Questions

- [NEEDS CLARIFICATION: Should the restart also reload `admin_user_credentials.json`? Recommended yes. Existing sessions stay valid because sessions are cookie-based.]
- [NEEDS CLARIFICATION: If the previously selected device is gone after re-scan, should the engine stay stopped (recommended) or fall back to the first available input?]
- [NEEDS CLARIFICATION: Is interrupting active listeners and AI translation during a restart acceptable, or should the button be disabled while listeners are connected?]
