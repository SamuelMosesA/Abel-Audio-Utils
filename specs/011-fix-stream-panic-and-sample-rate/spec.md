# Feature Specification: Safe Audio Stream Teardown and Graceful Sample Rate Negotiation

**Feature Branch**: `011-fix-stream-panic-and-sample-rate`

**Created**: 2026-10-10

**Status**: Ready for Planning

**Input**: User description: "fix this ce default\" component=audio audio.requested_rate=48000 audio.default_rate=32000 audio.error=\"Invalid sample rate\" ... 2026/10/10 22:51:09 [Recovery] 2026/10/10 - 22:51:09 panic recovered: close of closed channel mp3_stream.go:221 from handlers_audio.go:231"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Deterministic and Panic-Free Audio Stream Unsubscription (Priority: P1)

As a listener streaming live translation audio via the web interface, when an administrator toggles the AI master switch or kills a specific language stream, the streaming HTTP connection should close gracefully without causing an internal server panic or triggering Gin recovery middleware.

**Why this priority**: Panics in HTTP handlers can crash worker goroutines or destabilize web server connections. Closing a closed Go channel is a fatal runtime panic.

**Independent Test**: Can be tested by initiating an MP3 streaming connection, shutting down the stream pipeline via `StopStream()` or context cancellation, and asserting that the handler teardown exits with HTTP 200/EOF and zero panics.

**Acceptance Scenarios**:

1. **Given** one or more active HTTP clients subscribed to an MP3 language stream, **When** the stream pipeline is terminated from the server side, **Then** all listener channels are closed exactly once, deferred unsubscribe callbacks execute safely without panicking, and connections terminate cleanly.
2. **Given** a client abruptly disconnects from an MP3 stream while the stream is actively running, **When** `unsubscribe()` is invoked, **Then** the listener channel is deleted from the stream registry and closed safely without racing against background stream teardown.

---

### User Story 2 - Device-Native Sample Rate Fallback & ALSA Compatibility (Priority: P2)

As a system operator using an audio interface with non-standard hardware sample rates (such as 32000 Hz, 44100 Hz, or 48000 Hz), the audio engine should negotiate the appropriate sample rate with the audio driver smoothly, falling back to the device's native hardware default rate without crashing or producing conflicting state.

**Why this priority**: Hardware devices (such as USB mics, digital mixers, and ALSA capture interfaces) often only support specific fixed sample rates. If the configured rate is unsupported, the system must adapt seamlessly.

**Independent Test**: Can be tested by configuring an unsupported sample rate (e.g. 48000 Hz on a 32000 Hz device mock); verify the engine detects the capability mismatch, opens the device at its native default sample rate, and updates internal state accurately.

**Acceptance Scenarios**:

1. **Given** a device with default sample rate 32000 Hz and configured rate 48000 Hz, **When** the audio engine initializes, **Then** it cleanly detects the native capability and opens the audio stream at the device's supported rate.
2. **Given** the stream opens at the device's default sample rate, **When** downstream processing (recording, MP3 streaming, AI transcription) runs, **Then** all components receive and utilize the actual negotiated sample rate rather than the mismatched initial request.

---

### Edge Cases

- **Concurrent Unsubscribe and Stream Shutdown**: A client disconnecting at the exact moment an administrator stops the stream must not produce a race condition or double-channel close.
- **Nil or Inactive Stream Unsubscribe**: Calling an unsubscribe function when a stream has already been cleaned up and removed from the active map must be a safe no-op.
- **Zero or Negative Device Default Rate**: If `dev.DefaultSampleRate` reports 0 or negative from PortAudio, fallback defaults to 44100 Hz safely.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: `LiveAudioBroadcaster` MUST ensure that listener channels are closed at most once across both background stream shutdown and client-initiated unsubscription.
- **FR-002**: The `unsubscribe` callback returned by `Subscribe()` MUST safely check under synchronization whether the listener channel is still registered and open before closing it.
- **FR-003**: The audio engine MUST query and validate device sample rate compatibility before opening PortAudio streams, preferring `dev.DefaultSampleRate` when the configured sample rate is unsupported.
- **FR-004**: Downstream audio processing components (broadcaster, recording processor, OpenAI translation/transcription) MUST be informed of the actual negotiated sample rate.
- **FR-005**: All stream cleanup routines MUST be idempotent and thread-safe.

### Key Entities

- **MP3 Stream (`mp3Stream`)**: In-memory transcoding pipeline managing FFmpeg subprocess and listener channel registry.
- **Listener Channel (`chan []byte`)**: Buffered channel delivering progressive MP3 chunks to individual HTTP client response writers.
- **Audio Stream Configuration**: Parameters negotiating input device, channel count, buffer size, and hardware sample rate.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 0 runtime panics when stopping active audio streams with multiple connected listeners under race conditions.
- **SC-002**: Audio stream opens and captures audio without failure on devices whose default rate differs from `cfg.SampleRate`.
- **SC-003**: 100% of unit tests pass with Go race detector enabled (`go test -race ./src/backend/...`).

## Assumptions

- Operating system ALSA / CoreAudio drivers accurately report `DefaultSampleRate` in PortAudio device descriptors.
- Listener channels are owned by `LiveAudioBroadcaster` and consumers consume until channel close or context cancellation.
