# Bug Fix: Safari-compatible live audio streaming

- **Slug**: ios-safari-audio-blocked
- **Fixed**: 2026-09-27
- **Assessment**: ./assessment.md
- **Status**: applied

## Summary

Replaced the browser-facing endless WAV transport with shared AAC/HLS streams and changed both listener views to require an explicit playback gesture with visible retry/error handling. The original and translated feeds now use bounded live playlists and MPEG-TS segments that Safari/WebKit can consume natively.

## Changes

| File | Change | Notes |
|------|--------|-------|
| `src/backend/lib/audioengine/hls.go` | added | Runs one FFmpeg AAC/HLS encoder per language, bounds the live playlist, validates paths, handles sample-rate restarts, and cleans up processes/files. |
| `src/backend/lib/audioengine/hls_test.go` | added tests | Verifies playlist generation, MPEG-TS framing, AAC decodability with `ffprobe`, sample clamping, and traversal rejection. |
| `src/backend/lib/audioengine/broadcaster.go` | modified | Publishes original-source PCM to the shared HLS encoder without blocking audio capture. |
| `src/backend/lib/audioengine/broadcaster_test.go` | added test | Verifies original audio and the effective sample rate reach the publisher. |
| `src/backend/lib/web/handlers_audio.go` | modified | Replaces endless WAV responses with legacy redirects plus HLS playlist and segment handlers. |
| `src/backend/lib/web/router.go` | modified | Registers public HLS playlist and segment routes. |
| `src/backend/lib/web/handlers_audio_test.go` | updated tests | Pins legacy redirects, HLS MIME types, cache policy, playlist body, and segment delivery. |
| `src/backend/lib/web/test_helpers.go` | modified | Updates the test route for the new legacy redirect handler signature. |
| `src/backend/main.go` | modified | Creates, injects, and closes the shared HLS publisher. |
| `src/frontend/src/lib/components/audio/LiveAudioPlayer.svelte` | added | Provides user-initiated playback and visible retry/error states for autoplay denials or unavailable media. |
| `src/frontend/src/lib/components/audio/LiveAudioPlayer.test.ts` | added tests | Verifies explicit play and rejected-playback recovery. |
| `src/frontend/src/lib/components/views/LandingView.svelte` | modified | Uses the original HLS feed through the shared player. |
| `src/frontend/src/lib/components/views/StreamingView.svelte` | modified | Uses translated HLS feeds and removes unmuted autoplay. |
| `src/frontend/tests/audio-stream.spec.ts` | added tests | Exercises HLS source selection, absence of autoplay, explicit start, and retry behavior in Playwright WebKit. |
| `src/frontend/playwright.config.ts` | modified | Adds the Desktop Safari/WebKit project. |
| `abel.rb` | modified | Declares FFmpeg as a runtime dependency. |
| `README.md` | modified | Documents the FFmpeg prerequisite for local installations. |

## Diff Highlights

The old `/api/audio/stream[/<language>]` URLs now issue temporary redirects to `/api/audio/hls/<language>/index.m3u8`. The publisher converts stereo float PCM to AAC, keeps a six-segment live window, removes expired segments, and shares each encoder across all listeners of that language.

Both pages use `LiveAudioPlayer`, which calls `HTMLMediaElement.play()` only from a Start Listening/Retry user action and surfaces promise rejection instead of silently remaining blocked.

## Tests Added or Updated

- `src/backend/lib/audioengine/hls_test.go::TestHLSPublisherProducesPlaylistAndMPEGTransportStream` — generates a live playlist, reads a segment, and verifies AAC decodability.
- `src/backend/lib/audioengine/hls_test.go::TestHLSPublisherRejectsPathTraversal` — prevents language/segment paths from escaping the temporary HLS workspace.
- `src/backend/lib/audioengine/hls_test.go::TestFloat32ToPCM16ClampsSamples` — pins safe PCM conversion.
- `src/backend/lib/audioengine/broadcaster_test.go::TestStartAudioBroadcasterPublishesOriginalAudio` — pins original-feed fan-out to HLS.
- `src/backend/lib/web/handlers_audio_test.go::TestStreamHandlerRedirectsToHLS` — preserves compatibility for the former WAV URL.
- `src/backend/lib/web/handlers_audio_test.go::TestHLSHandlersServeSafariCompatibleContract` — pins playlist/segment response contracts.
- `src/frontend/src/lib/components/audio/LiveAudioPlayer.test.ts` — covers successful and rejected `play()` calls.
- `src/frontend/tests/audio-stream.spec.ts` — covers playback controls in the Playwright WebKit engine.

## Local Verification

- Commands run: `go test ./src/backend/...` → passed.
- Commands run: `go test -race ./src/backend/lib/audioengine` → passed for the changed audio/HLS package.
- Commands run: `go vet ./src/backend/...` → passed.
- Commands run: `npm run test:unit` → passed, 3 files and 8 tests.
- Commands run: `npm run check` → passed with 0 errors and 0 warnings.
- Commands run: `npm run build` → passed and generated the static frontend bundle.
- Commands run: `npx playwright test --project=webkit` → passed, 4 tests including both live-audio cases.
- Manual checks: FFmpeg generated the rolling HLS playlist and segments locally; `ffprobe` recognized AAC audio in the served segment fixture. Physical iOS Safari playback was not available in this environment.
- Additional check: the wider `go test -race ./src/backend/lib/audioengine ./src/backend/lib/web` run found an existing race in `MockTranslator.subtitleChan` used by `TestSubtitlesHandler`; the changed `audioengine` package passed and the race is unrelated to HLS.

## Deviations from Assessment

- The implementation adds `src/backend/lib/audioengine/hls.go`, a reusable frontend player component, the FFmpeg formula dependency, and prerequisite documentation. These files were not individually listed in the assessment, but were required to implement the assessed HLS/AAC remediation without duplicating encoder or playback logic.
- Standard one-second HLS segments were selected based on the user's approval to proceed without a stated latency target. Low-Latency HLS was not implemented.
- WebKit automation validates user-gesture and rejection behavior, while codec/container validity is tested with FFmpeg/ffprobe. A physical iOS device remains necessary for final platform validation.

## Follow-ups

- Run the manual matrix on current iPhone/iPad Safari and macOS Safari, including background/foreground, reconnect, device/sample-rate changes, and translated feeds.
- Measure end-to-end latency in the deployment environment and consider Low-Latency HLS only if one-second segments are insufficient.
- Add structured client media-error telemetry and encoder restart metrics.
- Fix the pre-existing `MockTranslator.subtitleChan` test race so the entire web test package can pass under `-race`.
- Review the existing npm audit findings separately; dependency installation reported 16 vulnerabilities unrelated to this change.
