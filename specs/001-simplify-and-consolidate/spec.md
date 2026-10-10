# Feature Specification: Simplify & Consolidate Audio Codebase

**Feature Branch**: `001-simplify-and-consolidate`

**Created**: 2026-10-10

**Status**: Ready for Planning

**Input**: User description: "I see many areas of improvement. I want all the audio conversions especially between bitrates and formats being scattered all around. I want a single folder for it. The frontend is too complex as well. Make as less classes and code as possible. Same for the logic that handles disconnections in the AI transaction and transcipriton. it is duplicated. I don' even read all the unit tests they are a lot. They might need simplifictaion"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Centralized & Cohesive Audio Conversions (Priority: P1)

As a backend engineer maintaining Abel Audio Utils,
I need all audio format transformations, sample rate conversions, and bit depth conversions in a dedicated, unified package,
So that encoding/decoding behaviors are consistent across storage, HLS, live broadcasting, and AI streaming without scattered ad-hoc conversion math.

**Why this priority**: Directly satisfies Constitution Principle IV (Hierarchical Domain Organization) and Principle V (Zero Duplication), eliminating bug surface in audio pipeline conversions.

**Independent Test**: Can be validated by converting test audio buffers (Float32, PCM16, sample rate transitions like 48kHz to 24kHz) through the centralized conversion module and verifying output parity across all consuming pipelines.

**Acceptance Scenarios**:

1. **Given** raw stereo float32 audio buffers, **When** converting to 16-bit PCM (for WAV or HLS streaming), **Then** conversion is performed exclusively by the centralized audio conversion package with verified clamping and endianness.
2. **Given** arbitrary sample rates from PortAudio capture, **When** feeding transcription/translation managers requiring 24kHz downsampling, **Then** downsampling logic is delegated to the shared conversion package rather than custom per-file loops.
3. **Given** scattered conversion routines across `hls.go`, `wav.go`, `transcription.go`, and `translation.go`, **When** the refactoring is applied, **Then** all callers import and use the converged conversion package.

---

### User Story 2 - Unified AI Streaming Connection & Disconnection Lifecycle (Priority: P2)

As a system operator running AI real-time audio translation and transcription,
I need a single, robust connection lifecycle and error/disconnection handler,
So that transient network drops, reconnect backoffs, and WebSocket shutdowns behave deterministically without duplicated management logic.

**Why this priority**: Eliminates redundant state machines and connection management code in `transcription.go` and `translation.go`, enforcing Constitution Principle I (Radical Simplicity) and Principle V (Zero Duplication).

**Independent Test**: Can be independently tested by simulating WebSocket disconnects and network terminations against a single mock connection lifecycle manager and observing identical reconnect/cleanup behavior for both transcription and translation streams.

**Acceptance Scenarios**:

1. **Given** an active OpenAI Realtime WebSocket connection, **When** an unexpected disconnect occurs, **Then** a unified connection manager handles error logging, socket cleanup, and reconnection attempts using shared logic.
2. **Given** user-initiated stop or configuration update, **When** terminating sessions, **Then** the unified lifecycle manager closes resources cleanly without orphaned goroutines or dangling channels.

---

### User Story 3 - Lean Frontend UI Architecture (Priority: P3)

As a user and maintainer interacting with the Abel web interface,
I want the frontend components and styles to be minimal and straightforward,
So that the web UI loads fast, contains minimal boilerplate CSS/classes, and avoids overly abstract component hierarchies.

**Why this priority**: Satisfies Constitution Principle I (Radical Simplicity & Code Minimization) on the client side, trimming unnecessary component nesting and styling complexity.

**Independent Test**: Can be verified by running the frontend application, checking that all user controls (metering, audio playback, admin configurations, translation views) function seamlessly while total frontend code volume and class complexity are noticeably reduced.

**Acceptance Scenarios**:

1. **Given** UI components with excessive styling classes or superfluous wrapper elements, **When** inspected after refactoring, **Then** markup is streamlined to minimal idiomatic Tailwind/CSS rules without visual regression.
2. **Given** client-side state handling and utility wrappers, **When** evaluated, **Then** duplicate helper functions and overly complex reactive patterns are consolidated into concise shared modules.

---

### User Story 4 - Streamlined, Readable Unit Test Suites (Priority: P4)

As a developer running local tests and CI pipelines,
I want concise, readable table-driven tests that focus strictly on core invariants,
So that tests are fast to read, simple to maintain, and do not discourage frequent refactoring.

**Why this priority**: Directly enforces Constitution Principle II (Streamlined & Maintainable Unit Tests), eliminating test bloat and brittle, repetitive test code.

**Independent Test**: Can be verified by executing the entire test suite (`go test ./...` and `npm run test`), confirming 100% passing status while reducing boilerplate test assertions and verbose setup code.

**Acceptance Scenarios**:

1. **Given** repetitive or overly verbose test suites in backend packages, **When** refactored, **Then** tests are restructured into idiomatic, concise table-driven test cases using standard assertions.
2. **Given** frontend test files with redundant mock fixtures, **When** simplified, **Then** tests focus strictly on primary user interactions and observable state changes.

---

### Edge Cases

- **Variable Hardware Sample Rates**: Handling source inputs from PortAudio that do not divide evenly into 24kHz downsampling targets in the centralized conversion package.
- **Abrupt AI WebSocket Interruptions**: WebSocket frame read errors while an audio packet is in transit during connection teardown.
- **Zero-length Audio Chunks**: Edge case handling in the shared conversion package when an empty or silence frame is passed.
- **Client Offline / Reconnect**: Frontend UI displaying clean connection state when backend WebSocket temporarily drops.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST consolidate all audio conversions (Float32 to PCM16, sample rate resampling/downsampling, byte buffer packing) into a dedicated package under `src/backend/lib/audioengine/conversion/`.
- **FR-002**: Audio conversion modules MUST expose minimal, stateless, reusable functions with comprehensive unit tests for mathematical accuracy and bounds clamping.
- **FR-003**: System MUST eliminate duplicate WebSocket connection, heartbeat, error handling, and reconnection loops between `transcription.go` and `translation.go` by converging them into `realtime_base.go`.
- **FR-004**: System MUST simplify frontend presentation components, removing redundant classes, excessive container wrappers, and dead utility abstractions across `src/frontend/src/lib/`.
- **FR-005**: System MUST streamline unit test suites across backend and frontend, replacing duplicated test setups with compact table-driven tests.
- **FR-006**: Existing audio playback, WAV recording, HLS streaming, live translation, and session authentication capabilities MUST remain fully functional without behavioral regression.

### Key Entities

- **AudioConversionPackage**: The centralized package responsible for format conversion (Float32 <-> Int16/PCM) and resampling.
- **RealtimeConnectionManager**: The unified connection and lifecycle controller for OpenAI Realtime WebSocket streaming sessions.
- **StreamlinedUI**: The lean frontend application delivering the metering, listening, and administration user experience with minimized complexity.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of audio format and sample rate conversions across the backend are invoked through the single centralized conversion directory.
- **SC-002**: AI connection and disconnection lifecycle code in `transcription.go` and `translation.go` is unified, reducing redundant connection handling logic across the two modules by at least 60%.
- **SC-003**: All existing backend and frontend test suites pass with zero regressions (`go test ./...` and frontend test runners).
- **SC-004**: Unit test code complexity and boilerplate lines are reduced across refactored packages while preserving critical test coverage for system invariants.
- **SC-005**: Overall line count and cognitive overhead in refactored domains decrease, adhering strictly to Constitution Principle I (Radical Simplicity).

## Assumptions

- Source audio format from PortAudio remains standard stereo/mono float32 buffers.
- OpenAI Realtime API continues to utilize 24kHz 16-bit PCM mono over WebSocket.
- Frontend remains built with SvelteKit and Vite, retaining existing routing endpoints and API contracts.
