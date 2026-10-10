# Feature Specification: Replace HLS with Cross-Platform Progressive MP3 Streaming

**Feature Branch**: `009-replace-hls-with-mp3`

**Created**: 2026-10-10

**Status**: Draft

**Input**: User description: "Replace complex and unstable HLS segmenting and playlist broadcasting with reliable cross-platform progressive chunked MP3 streaming to iOS and Android browsers and reduce code. Address warning: HLS playlist not ready, HLS stream is not ready."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Instant Progressive MP3 Audio Streaming for Web & Mobile (Priority: P1)

As an event listener using iOS Safari, Android Chrome, or a desktop browser, I want to listen to live audio via an immediate HTTP MP3 stream so that audio starts playing immediately without segment-loading delays, playlist-polling errors ("HLS stream is not ready"), or dropped chunks.

**Why this priority**: Live audio broadcast is the primary user-facing capability. HLS segment generation causes 4–8 second startup latency, disk I/O overhead, and race conditions when listeners connect before initial segments are cut. Progressive chunked MP3 streams immediately to native browser `<audio>` elements across all mobile and desktop platforms.

**Independent Test**: Connect to `/api/audio/stream` or `/api/audio/stream/es` from iOS Safari and Android Chrome. Verify that audio plays immediately within 1 second, continues smoothly, and requires zero client-side HLS JavaScript libraries.

**Acceptance Scenarios**:
1. **Given** an active audio broadcast, **When** a user clicks play in the web UI on iOS Safari or Android Chrome, **Then** audio starts streaming immediately via standard `<audio src="/api/audio/stream">` without segment buffering delays.
2. **Given** a listener accessing a translated language stream (e.g., Spanish `/es`), **When** the stream is requested, **Then** the server delivers progressive chunked MP3 audio generated in-memory in real time.
3. **Given** multiple listeners connecting at varying times, **When** they connect, **Then** each listener receives real-time MP3 frames from that moment forward without waiting for segment file rotations or playlist generation.

---

### User Story 2 - Elimination of HLS Complexity & Disk I/O (Priority: P1)

As a system maintainer and operator, I want all HLS segment generation, playlist creation (`.m3u8`), and temporary file watchers deleted from the backend so that the audio pipeline is purely in-memory, eliminates disk thrashing, and substantially reduces code complexity in accordance with the project Constitution (Radical Simplicity).

**Why this priority**: HLS segmentation requires writing temporary `.ts` and `.m3u8` files to disk, managing temp directories, cleaning expired chunks, and polling files. Replacing it with in-memory MP3 streaming eliminates disk I/O, prevents race conditions, and removes hundreds of lines of fragile code.

**Independent Test**: Audit the backend codebase and file system while streaming audio. Verify zero temporary audio files are written to disk for live streaming and all HLS generation packages/handlers are deleted.

**Acceptance Scenarios**:
1. **Given** the audio processing package, **When** inspected, **Then** all HLS segmenters, chunk rotators, and playlist writers are removed.
2. **Given** active live streams running, **When** monitoring the server, **Then** zero disk operations occur for live audio delivery.
3. **Given** the HTTP router, **When** legacy `/api/audio/hls/...` endpoints are requested, **Then** clients are smoothly redirected to `/api/audio/stream/...` or cleanly notified.

---

### User Story 3 - Seamless Mobile Background Playback & Lock-Screen Audio (Priority: P2)

As a mobile listener attending an event, I want audio to continue playing uninterrupted when my phone screen locks or when I switch to other applications.

**Why this priority**: Event listeners frequently lock their mobile screens or switch to other apps while listening. Native progressive MP3 playback in `<audio playsinline>` is first-class on iOS and Android without requiring complex audio worklet keep-alive hacks.

**Independent Test**: Open the broadcast on iOS Safari and Android Chrome, start playback, lock the device screen, and verify audio continues playing continuously with native lock-screen media controls.

**Acceptance Scenarios**:
1. **Given** active audio streaming on iOS Safari or Android Chrome, **When** the device screen locks, **Then** playback continues without interruption.
2. **Given** a locked mobile device, **When** the lock screen is activated, **Then** system audio controls (play/pause, volume) function as expected.

---

### Edge Cases

- **Slow Network / Client Backpressure**: If a client's network connection throttles or stalls, the broadcaster buffer must drop old frames for that client rather than blocking other listeners or causing unbounded memory growth.
- **Client Disconnect**: When a listener closes their browser tab or disconnects, the server response writer must detect context cancellation and immediately terminate the client stream to prevent resource leaks.
- **No Active Audio Source / Silent Input**: When silence or no audio is actively being captured, the MP3 streamer must continue sending valid silent MP3 frames or hold the chunked connection alive without crashing.

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST stream real-time main broadcast audio over HTTP using `Content-Type: audio/mpeg` and `Transfer-Encoding: chunked` at `GET /api/audio/stream`.
- **FR-002**: System MUST stream real-time translated audio over HTTP using `Content-Type: audio/mpeg` and `Transfer-Encoding: chunked` at `GET /api/audio/stream/:lang`.
- **FR-003**: System MUST encode audio from the capture ring buffer or translation channels directly into MP3 frames in memory without writing files to disk.
- **FR-004**: System MUST remove all HLS segment generators, Transport Stream (`.ts`) encoders, playlist (`.m3u8`) writers, and temporary directory cleanup routines.
- **FR-005**: Frontend MUST use standard HTML5 `<audio>` elements for live audio playback without requiring HLS.js or custom segment-polling logic.
- **FR-006**: System MUST isolate each client connection, ensuring that slow consumers or network disconnects do not block the central audio broadcast.

### Key Entities

- **MP3BroadcastStreamer**: In-memory encoder and multi-subscriber distributor broadcasting continuous MP3 frames to active HTTP response writers.
- **LiveAudioPlayer**: Clean frontend audio component playing progressive chunked audio directly from the audio stream URL.

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Audio playback begins within 1 second of user tap/click on both desktop and mobile browsers.
- **SC-002**: 0 temporary files created on disk for live audio streaming.
- **SC-003**: 0 "HLS playlist not ready" or "HLS stream is not ready" warnings in server logs.
- **SC-004**: Over 300 lines of complex HLS code deleted from the repository.
- **SC-005**: 100% of tested iOS Safari and Android Chrome sessions maintain uninterrupted playback when the device screen locks.

---

## Assumptions

- MP3 (`audio/mpeg`) is universally supported natively by all modern web browsers (iOS Safari, Android Chrome, Edge, Firefox, Chrome) via HTML5 `<audio>`.
- Encoding to MP3 in memory at 128kbps or 192kbps introduces negligible CPU overhead while reducing network bandwidth compared to raw PCM or WAV.
- Existing translation channels produce 24kHz or 16kHz mono audio that can be encoded directly into standard MP3 frames.
