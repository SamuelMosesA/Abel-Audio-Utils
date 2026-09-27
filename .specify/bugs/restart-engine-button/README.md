# Restart Engine button: find newly connected audio equipment

Issue: [#28 Add button to restart engine](https://github.com/SamuelMosesA/Abel-Audio-Utils/issues/28)

This page explains the change for everyone: the problem, the solution, how to check it, and the risks. The detailed engineering notes are in [assessment.md](./assessment.md) and [fix.md](./fix.md).

> **Status:** Automated tests pass. Still to do: the hands-on test with the Behringer mixer (checklist below).

## 📖 In plain words (for everyone)

### The problem
When Abel starts, it checks **once** which audio equipment is plugged into the computer, and it never checks again.

So if the mixer is switched on or plugged in **after** Abel has started, it doesn't show up in Abel's list. The same goes for settings: if someone changes the password or the default volume, Abel ignores the change.

The only fix was to **close Abel completely and open it again**. That meant:
- someone had to walk to the church computer,
- everyone listening on their phones got disconnected,
- the admin had to log in again.

That's stressful just before or during a service.

### The solution
There's a new **"Restart Engine"** button on the admin page, just above "Audio Engine Configuration".

When you press it, Abel:
1. **Checks nobody is recording**, so a recording can never be ruined.
2. **Stops the sound** briefly and safely.
3. **Looks again** for connected equipment, so the mixer appears.
4. **Re-reads the settings**, such as a new password or default volume.
5. **Reconnects to the mixer** if it was in use before.
6. **Shows a message** saying what happened, e.g. *"Engine restarted. 13 devices found. Reconnected to Behringer UMC404HD."*

You can press it **from your phone** anywhere in the church, and the website stays up the whole time.

### What changes for volunteers
| Before | After |
|---|---|
| Mixer plugged in late → close and reopen Abel on the church PC | Press **Restart Engine** from your phone |
| Everyone listening gets disconnected for a long time | A short gap of a few seconds, with a warning first |
| Log in again | Stay logged in |
| Changed a setting → full restart needed | Press **Restart Engine** (a few settings still need a full restart, and Abel says which) |

### Safety built in
- 🔒 The button is **greyed out while recording**.
- 🔒 Even if someone tries anyway, Abel **refuses** to restart during a recording.
- 🔒 Only **logged-in admins** can use it.
- ⚠️ It **asks for confirmation** before interrupting listeners.
- 🎛️ If the mixer is unplugged, Abel **waits for you to pick a device** instead of recording from the wrong microphone.

## 🔧 Technical details (for developers)

### Root cause
PortAudio builds its device list in `Pa_Initialize`, which was only called once in `main.go`. `appState.Devices` was filled once and never refreshed. Config was also loaded once, and nothing could reload it.

### What changed
**New endpoint `POST /api/system/restart`** (session-protected). In order, it:
1. Returns 409 if recording, or if a restart is already running.
2. Stops the engine and **waits** for the stream to close. This replaces `time.Sleep(100ms)` with a done channel and a 2s timeout.
3. Runs `pa.Terminate()` + `pa.Initialize()` and re-scans devices.
4. Reloads `config.yaml` and the credentials file. Live-safe fields are applied: credentials, default channels/boost, sample rate, buffer size. The rest are reported as `needsFullRestart`.
5. Reconnects to the previous device **by name**, since IDs are list positions that shift.

**Frontend:** a new `RestartEngineButton.svelte` component, disabled when `isRecording || isRestarting`, placed above the Audio Engine Configuration card. START is also disabled while a restart is running.

**Also fixed:** `PATCH /api/audio/config` with `deviceID: -1` or out of range panicked on `devices[-1]` and left the engine flagged as running. It's now rejected cleanly.

### Where to look when reviewing
- `lib/audioengine/restart.go`: the restart sequence and lock ordering.
- `lib/audioengine/engine.go`: `StopAudioEngine` waits on a done channel.
- **`lib/web/handlers_audio.go`:** `UpdateAudioConfig` now starts the engine **outside** `state.Update`. This is required: the engine loop takes the state read lock every iteration, so waiting for it under the write lock would deadlock.
- `lib/config/config.go`: `ApplyReload` / `CheckCredentials` (runtime-changed fields are behind a mutex).
- `lib/state/app_state.go`: `Devices` is accessed through locked `Devices()` / `SetDevices()`.
- `lib/web/handlers_recordings.go`: recording start is refused during a restart, which closes the gap between the two checks.

## ✅ How to check it works (anyone can do this)
- [ ] Start Abel with the mixer **unplugged** → it's not in the "Interface Device" list.
- [ ] Plug in the mixer → press **Restart Engine** → the mixer appears.
- [ ] Select it → **Commit Configuration** → the sound meters move.
- [ ] With the mixer in use, press **Restart Engine** → it reconnects to the mixer by itself.
- [ ] Press **START** → Restart Engine turns grey. Press **STOP** → it works again.
- [ ] Unplug the mixer → **Restart Engine** → the message says it couldn't reconnect, and Abel waits.
- [ ] Change the admin password in the settings file → **Restart Engine** → the new password works.
- [ ] Try it from a **phone** on the church Wi-Fi.

## ⚠️ Risks and consequences
| Risk | What it means | How it's handled |
|---|---|---|
| Listeners hear a short gap | Pressing it pauses live sound and translation for a few seconds | A warning asks for confirmation first |
| Crash if restarted carelessly | Restarting the audio system while sound flows can freeze Abel | Abel waits for the sound to fully stop; if it doesn't stop within 2 seconds, the restart is cancelled with an error |
| Wrong microphone after the re-scan | Device numbers can change | Abel finds the mixer by **name**; if it's gone, Abel waits for you to choose |
| Some settings don't update | The website port, recording folders and translation settings need a full restart | Abel's message lists exactly which ones |
| Broken settings file | A typo in `config.yaml` | Abel keeps the old settings, shows the error, and still finds new devices |
| Behavior change (technical) | An invalid device number no longer stops the running engine | Safer: the current engine is left untouched |

## 🧪 Test results
- Backend: all Go tests pass (**25 new**). `go vet` clean. Race detector clean for all changed code.
- Frontend: **14/14** tests pass (**6 new**, including *"button is greyed out while recording"*). Type check clean.
- Tried on the church PC: pressing the button shows *"Engine restarted. 12 devices found."*
- ⏳ **Not yet tested with the Behringer mixer.** See the checklist above.

## 💡 Recommendations / follow-ups
1. **Install ffmpeg on the church PC.** Since the iPhone/Safari audio update (#44), Abel won't start without it. We hit this today and it's now installed here, but it should be added to the README's Windows setup steps and to `start-abel.bat`.
2. **Regenerate the API docs** (`swag init`). They were skipped because the tool wasn't installed.
3. **Fix an old test problem:** `TestSubtitlesHandler` has a pre-existing data race, unrelated to this change and confirmed on `main`. It's worth its own issue.

