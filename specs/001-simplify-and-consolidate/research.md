# Research: Simplify & Consolidate Audio Codebase

## Decision 1: Centralized Audio Conversion Domain

- **Decision**: Create `src/backend/lib/audioengine/conversion/` containing cohesive, single-responsibility files:
  - `pcm.go`: Float32 to Int16/PCM byte slicing and vice versa with clamping.
  - `resample.go`: Downsampling (e.g., 48kHz/44.1kHz -> 24kHz mono/stereo) and upsampling.
  - `wav.go`: WAV header generation and finalization (moved/refactored from `audioengine/wav.go`).
- **Rationale**: Currently, `float32ToPCM16` is duplicated in `hls.go` and `storage.go`, while `downsample` is copied line-for-line between `transcription.go` and `translation.go`. Unifying these satisfies Constitution Principles I, IV, and V.
- **Alternatives considered**:
  - Keep conversions inline: Rejected (causes drift and testing redundancy).
  - External C-binding / libsamplerate: Rejected (adds unnecessary heavy CGo dependencies; the existing pure Go accumulator algorithms are fast and robust enough for Abel's needs).

## Decision 2: Unified Realtime AI Session Lifecycle & Disconnection

- **Decision**: In `src/backend/lib/openai/realtime_base.go`, introduce a shared `RealtimeClient` / session runner:
  - Consolidates WebSocket dial, token authentication header setup, and disconnect/error teardown.
  - Consolidates token usage tracking (`response.done` handling) and telemetry recording.
  - Delegates payload-specific event handling (transcription vs translation deltas) via a lightweight event consumer callback or interface.
- **Rationale**: `runSession` in `transcription.go` and `translation.go` share ~70% identical lines for connection loop, error handling, token accounting, and channel selects.
- **Alternatives considered**:
  - Full rewrite with official OpenAI Realtime SDK: Rejected (violates minimal dependencies, adds unnecessary abstraction).
  - Keep separate managers with helper functions: Accepted and enhanced by extracting the connection and teardown loop into `realtime_base.go`.

## Decision 3: Frontend Simplification & Style De-duplication

- **Decision**:
  - Prune redundant nested wrapper `div`s and repeated class strings across `AudioAdminView.svelte`, `TranslationAdmin.svelte`, and `LiveAudioPlayer.svelte`.
  - Simplify reactive stores in `audioState.svelte.ts` and `audioVisuals.svelte.ts` to keep only direct, essential state.
- **Rationale**: Decreases bundle size, eliminates visual cognitive load, and respects the user constraint to have "as less classes and code as possible".

## Decision 4: Streamlined Test Suites

- **Decision**:
  - Backend: Refactor repetitive handler tests (`handlers_audio_test.go`, `handlers_recordings_test.go`, `handlers_system_test.go`) into clean table-driven tests (`tests := []struct{...}`).
  - Frontend: Combine redundant unit assertions in `audioState.test.ts` and `audioVisuals.test.ts`.
- **Rationale**: Shorter, declarative test tables are easier to read and maintain while preserving 100% invariant coverage.
