# Bug Assessment: Audio streams blocked on iOS and Safari

- **Slug**: ios-safari-audio-blocked
- **Created**: 2026-09-27
- **Source**: pasted text
- **Verdict**: likely valid, needs reproduction
- **Severity**: high

## Report (verbatim or summarized)

> The audio stream players are being blocked on IOS and Safari.

No device model, iOS/macOS version, Safari version, player state, console error, or network response was supplied.

## Symptom

The original and translated live-audio players reportedly fail to play in Safari and on iOS, while they are expected to begin or allow continuous playback of the selected live stream. The report is plausible from the current implementation, but the exact Safari failure mode has not yet been reproduced.

## Reproduction

1. Start the application and its audio engine so that `/api/audio/stream/default` is receiving live PCM samples.
2. Open the landing page in Safari on macOS or iOS and use the Original Source Audio control.
3. Select a translation and open `/ai_live_audio/<language>` to exercise the translated player, which also requests autoplay.
4. Observe whether Safari rejects playback immediately, remains loading, or disconnects/retries the stream.

[NEEDS CLARIFICATION: Which device, OS/Safari versions, and player(s) fail? Does a direct tap on Play also fail, and what media error or Network-tab status is reported?]

## Suspected Code Paths

- `src/backend/lib/web/handlers_audio.go:128` — every live player receives `audio/wav` from this handler as one unbounded, progressively flushed HTTP response.
- `src/backend/lib/web/handlers_audio.go:133` — constructs a nominal RIFF/WAVE file with both RIFF and data sizes set to `0xFFFFFFFF`, but without an RF64 `ds64` chunk or a finite resource length; this is a nonstandard “forever WAV” convention whose demuxing behavior is browser-dependent.
- `src/backend/lib/web/handlers_audio.go:154` — flushes the WAV header before any samples and then keeps the response open indefinitely, with no finite `Content-Length`, seek model, or range handling.
- `src/frontend/src/lib/components/views/LandingView.svelte:159` — the original-source player uses the same live WAV endpoint, so autoplay alone cannot explain the reported failure of all players.
- `src/frontend/src/lib/components/views/StreamingView.svelte:128` — the translated player adds unmuted `autoplay`; WebKit documents that audio playback should be assumed to require a user gesture and that blocked playback must be handled.
- `src/backend/lib/web/handlers_audio_test.go:34` — the current stream test only verifies channel registration; it does not assert response headers/body format, decode the media, or cover browser playback.

## Root Cause Hypothesis

**Confidence: medium.** The primary suspected cause is the transport format: the server presents an endless raw PCM stream as a conventional RIFF/WAVE resource with sentinel `0xFFFFFFFF` sizes. That header does not describe a valid finite WAV and is not a standards-based live-stream container, so Safari may keep the media in a loading/blocked state rather than decode bytes as they arrive. This hypothesis explains both the original and translated players because both use the same endpoint. A second, independently confirmed incompatibility affects the translated view: it attempts unmuted autoplay without a user gesture or rejection handling, which Safari/WebKit intentionally blocks by default. WebKit's guidance is to assume `<audio>` needs a user gesture and handle a rejected `play()` promise: <https://webkit.org/blog/7734/auto-play-policy-changes-for-macos/> and <https://webkit.org/blog/6784/new-video-policies-for-ios/>.

## Proposed Remediation

**Preferred**: replace the unbounded RIFF/WAV response with a real live-media delivery format supported natively by Safari. HLS with AAC audio is the most interoperable Safari target; use short segments (or Low-Latency HLS if latency requirements justify the additional complexity), generate one playlist per original/translated stream, and point the existing `<audio>` elements at those playlists. Apple documents HLS as its HTTP technology for live audio/video delivery: <https://developer.apple.com/streaming/>. Preserve the existing channel fan-out behind the packaging layer so original and translated sources share the same transport implementation.

In the frontend, remove the assumption that translated audio can start automatically. Require an explicit Start Listening action (or call `play()` from the native control/user gesture), catch `HTMLMediaElement.play()` rejection, and show a useful retry/error state. Keep the same media element when changing sources where practical.

**Alternatives**:

- Encode and serve a continuous AAC or MP3 stream over chunked HTTP. This is smaller operationally than HLS and may reduce latency, but browser/proxy behavior for never-ending progressive responses still needs explicit Safari device validation and reconnect handling.
- Deliver PCM frames through a public WebSocket and play them through an `AudioWorklet`. This gives tight latency and buffering control, but requires a custom player, resampling/buffer management, an explicit user gesture to resume the Web Audio context, and substantially more client-side compatibility testing.
- Merely removing `autoplay` would fix the translated view's policy violation but would not explain or fix the reported failure of the manually controlled original player.

**Files likely to change**:

- `src/backend/lib/web/handlers_audio.go`
- `src/backend/lib/web/router.go`
- `src/backend/lib/audioengine/broadcaster.go`
- `src/frontend/src/lib/components/views/LandingView.svelte`
- `src/frontend/src/lib/components/views/StreamingView.svelte`
- `src/backend/lib/web/handlers_audio_test.go`
- `src/frontend/tests/audio-stream.spec.ts` (new)

**Tests to add or update**:

- Backend contract test for the selected live format, including MIME type, playlist/segment validity or continuous-stream framing, cancellation, and original/translated routing.
- Backend test that verifies a disconnected listener is removed and does not stall broadcaster fan-out.
- WebKit Playwright test that loads each player, starts playback through an explicit user gesture, and asserts the media element reaches a playable/playing state using a deterministic audio fixture.
- Frontend test for rejected `play()` promises and the visible Start Listening/retry state.
- Manual device matrix covering current iPhone/iPad Safari and macOS Safari, including initial play, background/foreground, reconnect, and language changes.

## Risks & Considerations

- HLS/AAC introduces encoding, segment retention, and cleanup work; segment duration directly trades latency against stability and request overhead.
- AAC encoding may add a native dependency and licensing/deployment considerations; the chosen encoder must work in the Homebrew service package.
- Live listeners should join near the current live edge rather than replaying stale buffered segments.
- Original and translated streams may have different production timing; each needs bounded buffering and discontinuity/restart handling.
- Safari autoplay restrictions remain relevant regardless of transport, so playback must begin from a user gesture and failures must be surfaced.
- The current endpoint has no observable client media errors; add structured stream-start/disconnect reasons and client-side media error telemetry without logging sensitive data.
- Preserve public access semantics for listener routes and do not accidentally expose admin-only WebSocket/control traffic.

## Open Questions

- [NEEDS CLARIFICATION: Which exact iOS/macOS and Safari versions are affected?]
- [NEEDS CLARIFICATION: Does the failure affect the original player, translated player, or both?]
- [NEEDS CLARIFICATION: After tapping Play, does the control remain paused, show an error, or stay indefinitely loading?]
- [NEEDS CLARIFICATION: What end-to-end latency is acceptable? This determines whether standard HLS, Low-Latency HLS, or a lower-level streaming transport is appropriate.]
- [NEEDS CLARIFICATION: Are HTTPS, a reverse proxy, or cross-origin embedding involved in affected deployments?]
