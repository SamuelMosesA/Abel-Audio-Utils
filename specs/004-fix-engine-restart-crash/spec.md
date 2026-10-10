# Feature Specification: Safe and Crash-Resilient Audio Engine Restart

**Feature Branch**: `004-fix-engine-restart-crash`

**Created**: 2026-10-10

**Status**: Draft

**Input**: User description: "The backend crashes when the engine is restarted"

## Clarifications

### Session 2026-10-10

- Q: How should the backend handle concurrent or rapid successive requests to restart the audio engine? → A: Reject overlapping requests immediately with HTTP 409 Conflict ("Audio engine restart already in progress").
- Q: How should the system behave if audio driver re-initialization fails during a restart? → A: Return HTTP 500 with driver error details, keep web server and UI alive, and leave engine in stopped state (deviceID: -1).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Resilient Audio Engine Restart & Hardware Discovery (Priority: P1)

An administrator managing church audio needs to restart the audio engine (e.g. after connecting a new USB audio interface, updating configuration, or clearing an audio glitch) via the administration UI or REST API. The system must cleanly shut down any active audio stream, reinitialize driver and hardware device detection, reload configuration, and re-establish connection to the previous device without crashing or terminating the server process.

**Why this priority**: When the audio engine restart causes the backend server to crash, the entire system (including active WebSockets, listener streams, and admin controls) is taken offline, requiring manual server restarts and disrupting live services.

**Independent Test**: Can be fully tested by triggering `POST /api/audio/restart` while an audio stream is actively capturing, verifying that the server responds with HTTP 200, returns the updated device catalog, automatically reconnects to the configured device, and continues operating without crashing or dropping API connectivity.

**Acceptance Scenarios**:

1. **Given** an active audio stream capturing from a sound card, **When** an administrator requests an engine restart, **Then** the active stream stops cleanly, the audio subsystem re-scans hardware, the previous device is reconnected, and the server process remains fully operational.
2. **Given** an audio engine in an idle/stopped state, **When** an administrator requests an engine restart, **Then** the system refreshes available hardware devices, updates internal configuration, and returns the updated device list without error.

---

### User Story 2 - Protection Against Concurrent & Rapid Restart Requests (Priority: P2)

An administrator or automated client might click the restart button multiple times in rapid succession or have multiple admin tabs open. The system must prevent concurrent driver initialization calls and eliminate race conditions, preventing segmentation violations or panics caused by simultaneous access to non-thread-safe audio subsystem resources.

**Why this priority**: Hardware driver interfaces (such as PortAudio / ALSA) are strictly non-thread-safe during initialization and teardown. Unsynchronized concurrent calls trigger memory corruption (`SIGSEGV`) and bring down the entire application.

**Independent Test**: Can be fully tested by firing multiple simultaneous restart requests to `/api/audio/restart` (e.g., 5 concurrent requests) and verifying that the in-flight request proceeds while overlapping requests are rejected with HTTP 409 Conflict without any process crash or panic.

**Acceptance Scenarios**:

1. **Given** an engine restart operation already in progress, **When** a concurrent restart request arrives, **Then** the system rejects the request immediately with HTTP 409 Conflict ("Audio engine restart already in progress"), preventing concurrent calls to the audio driver layer.
2. **Given** multiple concurrent restart triggers, **Then** zero memory corruption signals or channel closure panics occur.

---

### User Story 3 - Deterministic Stream Teardown Handshake (Priority: P3)

When an audio engine restart is requested, the system must deterministically coordinate the shutdown of the stream capture loop, ensuring all pending read operations and background buffers are finalized before the driver subsystem is reinitialized.

**Why this priority**: Attempting to terminate audio drivers while stream capture loops are blocked on buffer reads leads to undefined behavior, frozen driver handles, or hard segmentation faults in native driver code.

**Independent Test**: Can be tested by invoking restart during heavy audio buffer streaming and verifying that the streaming loop terminates cleanly before device re-initialization begins.

**Acceptance Scenarios**:

1. **Given** an audio capture loop blocked or active on reading buffer chunks, **When** shutdown is signaled, **Then** the system signals the loop and synchronously waits for clean termination before reinitializing driver handles.
2. **Given** a stream that does not terminate within a safety timeout, **Then** the system logs a timeout error gracefully rather than abruptly terminating native driver handles in an unsafe state.

---

### Edge Cases

- **Restart when audio device is unplugged or missing**: The system must detect that the previous device is no longer enumerated, log an appropriate warning, leave the device unselected (`deviceID: -1`), and remain healthy without crashing.
- **Rapid double-clicks on restart UI**: The backend rejects in-flight duplicates with HTTP 409 Conflict, avoiding channel panic or driver memory corruption.
- **Restart requested while recording is active**: The existing protection (rejecting restart with HTTP 409 Conflict when recording is underway) must remain intact.
- **PortAudio driver failure during reinitialization**: If native initialization fails, the system captures the error, returns HTTP 500 JSON response with error details, leaves the engine stopped (`deviceID: -1`), and keeps the web server process running.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST serialize audio engine restart and device re-initialization operations using strict concurrency controls to prevent simultaneous driver calls.
- **FR-002**: The system MUST deterministically wait for active audio stream goroutines to stop and close their streams before reinitializing native audio drivers.
- **FR-003**: The system MUST safely manage lifecycle quit channels and flags to prevent duplicate channel closure panics under concurrent access.
- **FR-004**: When an audio engine restart is already in progress, the system MUST reject concurrent restart requests immediately with HTTP 409 Conflict ("Audio engine restart already in progress").
- **FR-005**: If audio driver re-initialization fails during a restart, the system MUST return HTTP 500 with driver error details, keep the web server and UI active, and transition the engine state to stopped (`deviceID: -1`).
- **FR-006**: The system MUST preserve user configuration (channels, gain/boost) across restarts unless the underlying configuration file specifies changed defaults.
- **FR-007**: The system MUST reject restart attempts with HTTP 409 Conflict when recording is active.
- **FR-008**: All unit and integration tests across the audio engine and web handlers MUST pass with the Go race detector enabled.

### Key Entities *(include if feature involves data)*

- **RestartResult**: The structured result returned to clients detailing the newly detected audio devices, reconnected device identifier and name, and any non-fatal configuration reload errors.
- **AudioEngine Lifecycle**: The thread-safe state machine governing running, stopping, and re-initializing the audio capture loop and device drivers.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of single and burst engine restart requests complete without crashing or terminating the server process.
- **SC-002**: Concurrent burst restart requests (e.g. 5 parallel requests) execute safely: exactly one proceeds while overlapping requests receive HTTP 409 Conflict, with zero process crashes and zero segmentation faults.
- **SC-003**: After a successful engine restart, audio device discovery succeeds within 2 seconds.
- **SC-004**: If the previously selected audio device is still connected, the audio engine automatically resumes capture without requiring manual administrator intervention.
- **SC-005**: 100% of unit tests pass with zero race condition warnings under `go test -race`.

## Assumptions

- The underlying operating system audio stack (ALSA/PulseAudio/PipeWire) supports re-enumeration via standard PortAudio initialization calls when invoked sequentially.
- Normal audio engine stop operations complete within 2 seconds under regular buffer latency.
- Audio recording safety takes precedence over engine restart; restart is forbidden while active recording is in progress.
