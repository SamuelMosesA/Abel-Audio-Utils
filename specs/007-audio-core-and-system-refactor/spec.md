# Feature Specification: Audio Core Pipeline, OpenAI Managers, and Network Refactor

**Feature Branch**: `007-audio-core-and-system-refactor`

**Created**: 2026-10-10

**Status**: Draft

**Input**: User description: "Do multiple features: Audio engine storage clamping loop extracted into utility; wav.go and hls.go moved to audio_processing (renaming conversion to audio_processing); simplify translation.go and transcription.go with typed sessions instead of sync.Map, deduplicate broadcasting, simple restart auditing, and resilient reconnect buffer; Wi-Fi SSID via macOS ipconfig getsummary en0, externally accessible IP and port in QR code and text ip:port underneath; auto-update recording list on normal/trimmed processing and channel viewers, with a single master refresh button on top; code quality refactoring for large functions, state/interface clarity, field encapsulation, constitution updates, and simple manually verifiable engine-to-recording path without recording loop mutexes."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Streamlined & Lock-Free Audio Recording Pipeline (Priority: P1)

As an audio engineer or operator, I need the audio recording pipeline from the capture engine to disk to be direct, lock-free on the inner writing loop, and composed of clear, reusable audio conversion utilities so that audio captures are reliably written without thread contention or dropped samples.

**Why this priority**: The recording pipeline is the core mission of the application. Unnecessary mutex locks on per-chunk audio writes degrade performance, and unstructured conversion logic hinders manual auditability.

**Independent Test**: Record an audio session via PortAudio. Verify that audio chunks are clamped, converted to 16-bit PCM little-endian via dedicated audio processing utilities, and written to disk without per-chunk mutex synchronization, resulting in a bit-exact, valid WAV file.

**Acceptance Scenarios**:
1. **Given** an active audio capture stream producing float32 interleaved stereo samples, **When** chunks are processed for recording, **Then** samples are clamped to `[-1.0, 1.0]` and converted to Little-Endian 16-bit PCM via a dedicated `audio_processing` utility function.
2. **Given** the storage worker goroutine processing the audio stream, **When** recording is in progress, **Then** chunk writes execute without locking mutexes inside the chunk write loop.
3. **Given** the `audioengine` package structure, **When** inspected, **Then** WAV header generation/finalization and HLS audio segment processing reside in `src/backend/lib/audioengine/audio_processing/` (replacing the previous `conversion/` package).
4. **Given** an operator auditing the codebase, **When** tracing the execution from audio engine input to disk file output, **Then** the entire call path consists of small, explicitly named functional steps that are straightforward to verify manually.

---

### User Story 2 - Resilient, Typed, and Deduplicated OpenAI Managers (Priority: P1)

As a listener and broadcaster using AI live transcription and translation, I need the OpenAI session management to be robust against transient API errors, buffer audio seamlessly during disconnections, and use clean, typed session and subscriber registries so that translated audio and subtitles never stutter or drop unexpectedly.

**Why this priority**: Live AI translation and subtitles rely on OpenAI WebSocket connections which are subject to network drops and 60-minute session limits. The current manager code suffers from duplicate broadcasting code, untyped `sync.Map` containers, and complex connection lifecycle management.

**Independent Test**: Connect subtitle and audio subscribers to a translation session, simulate an OpenAI WebSocket disconnect, and verify that incoming audio continues buffering without loss, reconnects automatically with exponential backoff, and resumes broadcast without listener disconnection.

**Acceptance Scenarios**:
1. **Given** `TranslationManager` and `TranscriptionManager`, **When** managing sessions and subscribers, **Then** sessions and subscriber channels are stored in strictly typed structs rather than unstructured `sync.Map` instances.
2. **Given** subtitle deltas or completed transcripts emitted by OpenAI, **When** broadcast to active listeners, **Then** broadcasting uses a single shared broadcasting component rather than duplicated per-manager loops.
3. **Given** a network drop or OpenAI session expiration, **When** the connection is interrupted, **Then** incoming audio is held in a bounded FIFO buffer and replayed upon reconnection without dropping the client stream.
4. **Given** repeated API errors or dial failures, **When** retrying, **Then** backoff and error auditing follow a clear, consolidated lifecycle that does not overwhelm system logs.

---

### User Story 3 - Accurate Network Connection Discovery and QR Code Display (Priority: P2)

As a mobile listener attending an event, I want to scan a QR code or see the exact external IP and port displayed on the landing page so that I can easily connect my phone to the audio broadcast over the local network or Wi-Fi.

**Why this priority**: Mobile listeners cannot connect if the QR code defaults to `localhost` or an unreachable internal interface. Having the correct externally reachable `ip:port` and Wi-Fi SSID makes onboarding effortless.

**Independent Test**: Query `/api/system/connection` on macOS and Linux. Verify that the response contains the active Wi-Fi SSID (using `ipconfig getsummary en0` on macOS), the externally accessible host IP, and the active listening port, and verify that the landing page renders the QR code with `ip:port` text underneath.

**Acceptance Scenarios**:
1. **Given** a macOS host, **When** the system queries the active Wi-Fi network, **Then** it executes `ipconfig getsummary en0 | awk -F ' SSID : ' '/ SSID : / {print $2}'` (with suitable fallback) to return the correct SSID name.
2. **Given** a running backend instance, **When** the system determines its network address, **Then** it discovers the publicly reachable interface IP and server port rather than an unreachable loopback address.
3. **Given** a user viewing the landing page, **When** the page loads, **Then** the QR code encodes the reachable external backend URL, and the text underneath explicitly displays `ip:port`.

---

### User Story 4 - Live Recording Updates and Master Console Refresh (Priority: P2)

As an administrator managing recordings and broadcasts, I want the recording list to automatically refresh when files finish processing (normal or trimmed), channel viewer counts to update in real time, and a single master refresh button on the console header so that I can monitor and control the station without fragmented per-card reloads.

**Why this priority**: Administrators currently have to manually reload sections or click multiple separate refresh buttons across different cards to see newly trimmed or processed MP3s and viewer updates.

**Independent Test**: Complete a recording or trim operation; verify that an SSE state change notification is broadcast to the frontend, causing the recording library to update automatically. Click the top-level master refresh button and verify that all console sections (devices, levels, recordings, translations) update synchronously.

**Acceptance Scenarios**:
1. **Given** a recording that completes normal MP3 processing or trimming, **When** the background job finishes, **Then** the backend broadcasts a `recording` state change event over SSE.
2. **Given** an admin console connected to the SSE changelog stream, **When** a `recording` state change event is received, **Then** the recording library automatically fetches and displays the updated file list.
3. **Given** active channel viewers connecting or disconnecting, **When** subscriber counts change, **Then** the updated listener counts are reflected live in the UI.
4. **Given** the admin console UI, **When** viewed, **Then** individual per-section refresh buttons are removed, and a single Master Refresh button is present in the top header to refresh all console telemetry and assets in one action.

---

### User Story 5 - Code Quality Architecture, Encapsulation, and Constitution Update (Priority: P3)

As a maintainer, I want all large or ambiguous functions decomposed into explicit functional names, state and interface files clearly separated, struct fields protected from external mutation, and these quality learnings codified into the project Constitution so that the codebase remains clean, maintainable, and verifiable.

**Why this priority**: Prevents code entropy, ensures that internal state mutations cannot leak across packages, and provides binding architectural rules for ongoing development.

**Independent Test**: Review the implementation plan for the Code Quality Report, audit struct visibility and function sizing across modified packages, and verify that `.specify/memory/constitution.md` includes the new code quality principles.

**Acceptance Scenarios**:
1. **Given** functions with large bodies or unclear names, **When** refactored, **Then** they are broken down into small, composable functions with descriptive functional names.
2. **Given** package structs representing internal state, **When** defined, **Then** their internal fields are unexported and accessed only through explicit thread-safe methods or constructors.
3. **Given** state models and interface declarations, **When** organized in packages, **Then** state implementations and interface contracts reside in distinctly named, logically arranged files.
4. **Given** `.specify/memory/constitution.md`, **When** updated, **Then** it incorporates principles on explicit functional naming, strict encapsulation, manual auditability of audio pipelines, and single-responsibility modules.

---

### Edge Cases

- **macOS Wi-Fi Interface Not `en0`**: If `en0` is an Ethernet adapter or Wi-Fi is on `en1`, or if the host is running in a headless VM/container without Wi-Fi, the SSID detection must gracefully return an empty string or `"N/A"` without panicking or hanging.
- **Network Interface Determination**: When the server is bound to `0.0.0.0` or running behind a reverse proxy, the system must inspect configured hosts, client request headers (e.g., `X-Forwarded-Host`, `Host`), or primary outbound network interfaces to resolve a sensible, reachable external IP and port.
- **OpenAI Rapid Disconnects**: If OpenAI repeatedly closes the WebSocket in rapid succession, exponential backoff must prevent tight reconnect loops while the bounded FIFO buffer preserves recent audio without overflowing memory.
- **Concurrent Recording State Transitions**: When stopping a recording, any audio chunk currently being converted or written must complete cleanly without racing against file closure or file finalization.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST extract the float32-to-int16 clamping and Little-Endian byte encoding loop from `storage.go` into a shared, testable function in `audio_processing`.
- **FR-002**: System MUST consolidate `wav.go`, `hls.go`, and existing audio conversion logic inside `src/backend/lib/audioengine/audio_processing/`, renaming and deprecating the old `conversion/` package name.
- **FR-003**: System MUST remove per-chunk mutex synchronization from the recording loop in `storage.go`, ensuring that chunk writing by the single storage worker goroutine is lock-free.
- **FR-004**: System MUST ensure the path from audio engine capture to WAV file recording is composed of simple, direct, manually verifiable functions.
- **FR-005**: System MUST replace untyped `sync.Map` session and subscriber tracking in `translation.go` and `transcription.go` with strongly-typed registry structures.
- **FR-006**: System MUST deduplicate subtitle and audio broadcasting between translation and transcription into a shared broadcast utility.
- **FR-007**: System MUST provide a bounded audio buffer for OpenAI managers that preserves audio across socket reconnections and retries to prevent listener audio dropouts.
- **FR-008**: System MUST simplify OpenAI session dial, backoff, and error auditing into clean, maintainable lifecycles.
- **FR-009**: System MUST retrieve the Wi-Fi SSID on macOS using `ipconfig getsummary en0 | awk -F ' SSID : ' '/ SSID : / {print $2}'` with graceful fallbacks.
- **FR-010**: System MUST determine the externally accessible network IP address and port on which the backend is running.
- **FR-011**: System MUST return the external URL and Wi-Fi SSID via `/api/system/connection`.
- **FR-012**: Frontend MUST generate the mobile join QR code using the externally accessible URL and display `ip:port` text directly underneath the QR code.
- **FR-013**: System MUST broadcast a `recording` state change via SSE whenever normal MP3 processing or audio trimming completes.
- **FR-014**: Frontend MUST listen to SSE `recording` change events and automatically refresh the recording library.
- **FR-015**: Frontend MUST remove disparate section-level refresh buttons across admin console cards and provide a single Master Refresh button in the console header.
- **FR-016**: System MUST decompose large, ambiguous functions into small, explicitly named functions.
- **FR-017**: System MUST protect struct fields from unauthorized external modification by enforcing encapsulation and clear interface boundaries.
- **FR-018**: System MUST update `.specify/memory/constitution.md` with codified principles on code quality, functional naming, state encapsulation, and audio path verifiability.

### Key Entities

- **AudioProcessing**: Cohesive domain package in `src/backend/lib/audioengine/audio_processing/` containing WAV formatting, HLS segmentation, sample rate conversion, and sample clamping/encoding.
- **RecordingWriter**: Dedicated, lock-free audio capture file writer executing exclusively on the storage worker thread.
- **RealtimeRegistry**: Strongly-typed container managing active OpenAI sessions, channel buffers, and client subscribers for translation and transcription.
- **SystemConnectionInfo**: Entity conveying the externally accessible host IP, port, server URL, and Wi-Fi SSID to clients and UI.
- **MasterRefresh**: Frontend action coordinating synchronized refresh of audio devices, engine state, recordings, and translation sessions.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of audio chunks written during active recording bypass mutex locking in the inner chunk write loop.
- **SC-002**: Audio conversion and clamping functions in `audio_processing` achieve 100% unit test coverage with verified edge-case clamping at `[-1.0, 1.0]`.
- **SC-003**: Transient OpenAI socket drops lasting up to 15 seconds reconnect without dropping client subscribers or losing buffered audio.
- **SC-004**: Landing page displays the valid external `ip:port` and QR code renders properly on mobile and desktop viewports.
- **SC-005**: Recording library updates automatically within 500ms of background MP3 processing or trimming completion without requiring manual user refresh.
- **SC-006**: All Go tests (`go test -race ./src/backend/...`) and frontend unit tests (`bun run test:unit`) pass cleanly without data races.

## Assumptions

- The backend runs on macOS or Linux environments; platform-specific network queries handle absence of tools gracefully.
- The storage worker is the sole writer to the active recording file during capture, allowing lock-free writes while the file handle is open.
- The mobile join QR code is intended for devices on the same local network or reachable via the backend's external IP.
- Frontend master refresh triggers existing API endpoints concurrently using `Promise.all` for fast, unified updates.
