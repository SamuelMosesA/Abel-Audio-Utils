# Feature Specification: UI Control Refinements, Persistent Per-Language AI Killswitch, and DSP Boost Utility Extraction

**Feature Branch**: `005-ui-controls-and-dsp-refactor`

**Created**: 2026-10-10

**Status**: Draft

**Input**: User description: "Remove the auto update loop for recordings. Put a refresh button there instead. For Ai translation controls, by default show a killswitch for each configured language. Put a refresh button there as well. It should tell me the number of clients on each translation thing. And the killswitch should persist and block for a specific language. THi sis n additoin to the master AI switch. How to make Embedded resource (file:///home/samuelmoses/Workspace/Church/Abel-Audio-Utils/src/backend/lib/audioengine/engine.go#L294:310): make the code like this into the audio uitilies function"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - On-Demand Recording Library Refresh (Priority: P1)

An administrator managing church sermon recordings in the web console needs the recording list to remain stable without background polling loops triggering repeated network requests (`/api/recordings/library`) every 2 seconds. The administrator triggers a refresh on demand using a dedicated Refresh button, which re-queries the recording library and updates the displayed recordings.

**Why this priority**: Continuous 2-second background polling creates unnecessary CPU and network load, disrupts UI focus during editing/trimming operations, and floods web server access logs.

**Independent Test**: Mount the admin recording view, verify that no automated recurring HTTP requests are dispatched in the background, click the Refresh button, and verify that the recording library fetches and re-renders successfully.

**Acceptance Scenarios**:

1. **Given** the admin console is open on the Master Recording Library, **When** left idle for several minutes, **Then** zero automatic recurring background requests to `/api/recordings/library` are dispatched.
2. **Given** new audio recordings or completed exports exist on the server, **When** the administrator clicks the Refresh button on the recording panel, **Then** the UI fetches the updated list of recordings and displays the current state.

---

### User Story 2 - Per-Language AI Killswitch with Active Client Metrics & Persistence (Priority: P1)

An administrator monitoring live translations needs granular control over each configured translation language. The AI Translation panel must display every configured language with:
1. Its active client/listener count (real-time or on-demand).
2. A dedicated killswitch toggle (Blocked vs Allowed/Active) for that specific language.
3. A manual Refresh button to update translation status and client metrics.
When a language is blocked via its killswitch, any ongoing translation session for that language is terminated, incoming translation requests for that language are rejected, and the blocked status persists across reloads and runs in addition to the master AI enable switch.

**Why this priority**: Church administrators need the ability to disable specific language models (e.g. if an AI output glitches, offends, or incurs runaway token costs for a specific dialect) while leaving other language streams operational, and to monitor how many congregation members are listening to each stream.

**Independent Test**: Navigate to the AI Translation panel, verify that all configured languages are displayed by default with their respective client counts. Toggle the killswitch for a language, verify that its translation session stops and cannot be accessed by clients, and confirm that the blocked status remains active upon page reload and server query.

**Acceptance Scenarios**:

1. **Given** multiple languages configured in Abel (e.g., Spanish, French, German), **When** viewing the AI Translation panel, **Then** all configured languages are displayed with their individual killswitches and active client counts.
2. **Given** an active translation session for a language, **When** an administrator toggles the killswitch to block that language, **Then** the active session for that language is stopped immediately, client access to that language's stream/subtitles is blocked, and the blocked state persists.
3. **Given** a language blocked via killswitch, **When** a listener requests subtitles or audio stream for that language, **Then** the system refuses the request with an unavailable notice.
4. **Given** the master AI switch is enabled, **When** an administrator unblocks a previously blocked language, **Then** translation for that language is allowed to resume when requested.

---

### User Story 3 - Modular Audio DSP Boost & Clamping Utility Extraction (Priority: P2)

A developer or audio engineer working on the backend audio pipeline needs common stereo digital gain boosting and hard-limiting clamping logic (multiplying channel samples by digital boost factor and clamping values to the `[-1.0, 1.0]` range) extracted into a reusable, well-tested function in the audio conversion/DSP utility package (`src/backend/lib/audioengine/conversion`).

**Why this priority**: Adheres to Constitution Principle I (Radical Simplicity), Principle III (Converged Architecture), and Principle V (Zero Duplication), eliminating duplicated inline sample math and ensuring audio clipping bounds are enforced in one authoritative location.

**Independent Test**: Can be tested via unit tests in `conversion` verifying that stereo chunks are correctly scaled by the boost factor, clamped between `-1.0` and `1.0`, and that the audio engine streaming loop calls this utility function.

**Acceptance Scenarios**:

1. **Given** input audio channel samples and a boost factor, **When** passed to the DSP boost clamping utility function, **Then** each sample is multiplied by the boost factor and strictly clamped within `[-1.0, 1.0]`.
2. **Given** the audio engine capture loop in `StartAudioEngine`, **When** processing incoming PortAudio buffers, **Then** it delegates channel routing, gain boost, and clamping to the shared conversion utility function.

---

### Edge Cases

- **Recording Library refresh clicked while a recording upload or trim is active**: The refresh operation updates the library without interrupting or canceling the in-flight file upload or encoding job.
- **Master AI switch disabled vs Per-language killswitch**: If Master AI is disabled, all translation sessions are paused/blocked regardless of individual language killswitch states. If Master AI is enabled, only languages not blocked by their individual killswitch are permitted to run.
- **Language killswitch persistence across restarts**: The blocked languages set must be stored in runtime state and persisted (e.g., config / persistent state store) so server restarts preserve administrator restrictions.
- **Zero clients listening**: Languages with 0 active listeners should display `0 listeners` without errors or empty state crashes.
- **Digital gain boost factor of 0.0 or negative**: The DSP utility function treats a boost factor of `<= 0.0` as `1.0` (unity gain) to prevent accidental total signal mute.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The Master Recording Library UI MUST NOT run an automated polling interval (`setInterval`) to fetch files; it MUST provide a manual Refresh button that queries `/api/recordings/library` when clicked.
- **FR-002**: The AI Translation Admin panel MUST display all configured languages by default, showing each language's display name, active listener count, and individual killswitch state.
- **FR-003**: The AI Translation Admin panel MUST include a manual Refresh button to re-fetch translation sessions, listener counts, and stream status on demand.
- **FR-004**: The system MUST support individual per-language killswitches that can be toggled via the REST API (`POST /api/ai/streams`).
- **FR-005**: When a language is killed/blocked, any active translation and subtitle session for that language MUST be immediately terminated, and subsequent client connection attempts for that language MUST be rejected with HTTP 503 or error response.
- **FR-006**: The per-language blocked/killswitch status MUST persist across browser reloads and server restarts.
- **FR-007**: The system MUST report the number of active listeners/clients (both HLS audio listeners and subtitle subscribers) connected to each configured language.
- **FR-008**: The backend MUST provide an exported function in `src/backend/lib/audioengine/conversion` (e.g. `ApplyGainAndClampStereo` or `ExtractStereoChunk`) that handles channel selection, digital boost amplification, and hard clamping to `[-1.0, 1.0]`.
- **FR-009**: The capture loop in `src/backend/lib/audioengine/engine.go` MUST use the extracted DSP utility function instead of inline loop math.
- **FR-010**: All unit tests in `src/backend/lib/audioengine/...`, `src/backend/lib/web/...`, and frontend tests (`bun run test:unit`) MUST pass with zero regressions.

### Key Entities *(include if feature involves data)*

- **LanguageKillswitch**: Tracks the blocked/allowed state for each configured ISO/RFC language code, persisted in configuration/state.
- **LanguageSessionMetrics**: Reports the active translation session state, listener count (HLS audio clients and SSE subtitle subscribers), and killswitch status for each configured language.
- **StereoChunk**: The interleaved `[]float32` stereo audio buffer with digital gain and hard-clamping applied.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Zero automated background HTTP polling requests to `/api/recordings/library` when the recording tab is left open.
- **SC-002**: 100% of configured languages appear in the AI Translation console with individual killswitch controls and client counts.
- **SC-003**: Toggling a language killswitch immediately terminates that language's session and blocks new connections within 500ms.
- **SC-004**: The blocked language state survives a full page refresh and server reboot without resetting.
- **SC-005**: 100% of audio samples produced by the extracted DSP utility are bounded within `[-1.0, 1.0]`.
- **SC-006**: 100% test pass rate across backend Go tests (`go test -race ./src/backend/...`) and frontend Vitest suite.

## Assumptions

- Configured languages are defined in `config.yaml` (`ai_languages`).
- Client count for a language is measured by active SSE subtitle subscribers plus active HLS streaming requests for that language.
- Persisting blocked languages can be saved into runtime state and serialized to the configuration file or persistent state file.
