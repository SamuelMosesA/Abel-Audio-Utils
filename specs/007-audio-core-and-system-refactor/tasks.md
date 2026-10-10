# Tasks: Audio Core Pipeline, OpenAI Managers, and Network Refactor

**Feature Branch**: `007-audio-core-and-system-refactor`
**Date**: 2026-10-10
**Spec**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md)

## Phase 1: Setup (Dependencies & Infrastructure)

**Purpose**: Verify dependencies and prepare package directories for consolidation.

- [ ] T001 Verify `github.com/zolstein/sync-map` dependency in [go.mod](file:///home/samuelmoses/Workspace/Church/Abel-Audio-Utils/go.mod) and [go.sum](file:///home/samuelmoses/Workspace/Church/Abel-Audio-Utils/go.sum)
- [ ] T002 Create destination directory `src/backend/lib/audioengine/audio_processing/` for consolidated audio domain logic

---

## Phase 2: Foundational (Audio Processing Domain Foundation)

**Purpose**: Establish core audio conversion and header logic in `audio_processing` before refactoring callers.

- [ ] T003 [P] Implement `ConvertStereoFloat32ToPCM16LE` with strict `[-1.0, 1.0]` clamping and little-endian encoding in `src/backend/lib/audioengine/audio_processing/pcm.go`
- [ ] T004 [P] Move and consolidate WAV header utilities (`GenerateWavHeader`, `WritePlaceholderWavHeader`, `FinalizeWavHeader`) into `src/backend/lib/audioengine/audio_processing/wav.go`
- [ ] T005 [P] Move and consolidate downsampling and audio delta decoding (`DownsampleStereoToMonoPCM24k`, `DecodeAudioDelta`) into `src/backend/lib/audioengine/audio_processing/resample.go`
- [ ] T006 [P] Move HLS playlist generation and audio segment slicing into `src/backend/lib/audioengine/audio_processing/hls.go`
- [ ] T007 Add comprehensive unit tests covering clamping, WAV headers, downsampling, and HLS in `src/backend/lib/audioengine/audio_processing/audio_processing_test.go`

**Checkpoint**: Foundation ready — `audio_processing` domain package is fully tested and ready for engine and storage integration.

---

## Phase 3: User Story 1 - Streamlined & Lock-Free Audio Recording Pipeline (Priority: P1) 🎯 MVP

**Goal**: Make the recording pipeline from PortAudio capture to disk completely lock-free on the inner chunk write loop, using `audio_processing` utilities for a direct, manually verifiable path.

**Independent Test**: Record an audio session via PortAudio. Verify audio chunks are clamped and written to disk without per-chunk mutex synchronization, producing a valid, bit-exact WAV file with zero data races.

### Implementation for User Story 1

- [ ] T008 [US1] Refactor `storage.go` to use `audio_processing.ConvertStereoFloat32ToPCM16LE` and remove the inline clamping loop in `src/backend/lib/audioengine/storage.go`
- [ ] T009 [US1] Remove inner-loop mutex locking (`WriteWithFile` file mutex acquisition) on per-chunk audio writes in `src/backend/lib/audioengine/storage.go`
- [ ] T010 [US1] Refactor `EngineState` and `Engine` in `src/backend/lib/state/engine.go` and `src/backend/lib/audioengine/engine.go` to allow lock-free chunk writes by the single storage worker while safely coordinating start/stop recording transitions
- [ ] T011 [US1] Simplify and clarify the engine-to-recording call graph so PortAudio capture to file recording is a transparent, 3-step functional flow in `src/backend/lib/audioengine/engine.go` and `src/backend/lib/audioengine/storage.go`
- [ ] T012 [US1] Remove deprecated delegates from `src/backend/lib/audioengine/wav.go` and clean up `src/backend/lib/audioengine/hls.go` to import from `audio_processing`
- [ ] T013 [US1] Delete obsolete `src/backend/lib/audioengine/conversion/` package and update all backend import paths across the repository
- [ ] T014 [US1] Run unit and race tests on audio engine and storage in `src/backend/lib/audioengine/...` and `src/backend/lib/state/...`

**Checkpoint**: User Story 1 complete — recording pipeline is lock-free, clean, and directly auditable.

---

## Phase 4: User Story 2 - Resilient, Typed, and Deduplicated OpenAI Managers (Priority: P1)

**Goal**: Replace untyped `sync.Map` in `TranslationManager` and `TranscriptionManager` with `github.com/zolstein/sync-map`, deduplicate broadcasting logic, route PCM conversions through a clean shared utility function, simplify connection lifecycle/auditing, and standardize bounded FIFO audio buffering across reconnects.

**Independent Test**: Connect subtitle and audio subscribers, simulate an OpenAI WebSocket disconnect, and verify incoming audio buffers without loss, reconnects with exponential backoff, and resumes broadcasting without client dropouts or type assertion panics.

### Implementation for User Story 2

- [ ] T015 [P] [US2] Implement shared `PendingAudioBuffer` (bounded 15-second FIFO ring buffer of 24 kHz mono PCM16) in `src/backend/lib/openai/buffer.go`
- [ ] T016 [P] [US2] Implement shared `SubtitleBroadcaster` helper for JSON subtitle encoding and non-blocking delivery in `src/backend/lib/openai/broadcast.go`
- [ ] T017 [P] [US2] Implement clean PCM conversion and audio delta decoding utility functions (`DownsampleChunkForAI`, `DecodeAIDelta`) in `src/backend/lib/openai/audio_util.go`
- [ ] T018 [US2] Refactor `TranslationManager` in `src/backend/lib/openai/translation.go` to use `sync_map.Map[string, *RealtimeSession]`, `sync_map.Map[string, []chan string]`, and `sync_map.Map[string, time.Time]` from `github.com/zolstein/sync-map`, route downsampling through `audio_util.go`, and delegate broadcasting to `broadcast.go`
- [ ] T019 [US2] Refactor `TranscriptionManager` in `src/backend/lib/openai/transcription.go` to use `sync_map.Map` from `github.com/zolstein/sync-map`, route downsampling through `audio_util.go`, integrate `PendingAudioBuffer` for resilient reconnects, and delegate broadcasting to `broadcast.go`
- [ ] T020 [US2] Simplify connection dialing, backoff, and error auditing in `src/backend/lib/openai/translation.go` and `src/backend/lib/openai/transcription.go` into concise, explicitly named functions
- [ ] T021 [US2] Update `audio_processing` imports in `src/backend/lib/openai/translation.go` and `src/backend/lib/openai/transcription.go`
- [ ] T022 [US2] Add unit tests verifying typed session registration, PCM conversion utility, subtitle broadcasting, and reconnect buffering in `src/backend/lib/openai/openai_test.go`

**Checkpoint**: User Story 2 complete — OpenAI managers are typed, deduplicated, and resilient with modular PCM conversion utilities.

---

## Phase 5: User Story 3 - Accurate Network Connection Discovery and QR Code Display (Priority: P2)

**Goal**: Detect macOS Wi-Fi SSID using `ipconfig getsummary en0`, determine the public/LAN-exposed IP and server port, and display `ip:port` on the landing page underneath the QR code.

**Independent Test**: Query `GET /api/system/connection` on macOS. Verify SSID is retrieved via `ipconfig getsummary en0`, server URL contains the outward-facing IP and port, and the landing page displays `ip:port` under the QR code.

### Implementation for User Story 3

- [ ] T023 [US3] Update `GetWiFiSSID()` in `src/backend/lib/web/handlers_system.go` to run `ipconfig getsummary en0 | awk -F ' SSID : ' '/ SSID : / {print $2}'` on macOS (with graceful fallbacks)
- [ ] T024 [US3] Enhance `GetLocalIP()` and `GetSystemConnection()` in `src/backend/lib/web/handlers_system.go` to resolve the externally reachable host IP and return `displayEndpoint` (`ip:port`), `host`, `port`, `serverUrl`, and `ssid`
- [ ] T025 [US3] Update `LandingView.svelte` in `src/frontend/src/lib/components/views/LandingView.svelte` to use the external `serverUrl` for QR code generation and display the `displayEndpoint` text (`ip:port`) directly underneath the QR code box
- [ ] T026 [US3] Add unit tests for network connection resolution and macOS Wi-Fi parsing in `src/backend/lib/web/handlers_system_test.go`

**Checkpoint**: User Story 3 complete — QR code and network details show the accessible external IP and port.

---

## Phase 6: User Story 4 - Live Recording Updates and Master Console Refresh (Priority: P2)

**Goal**: Emit real-time SSE notifications upon completion of normal and trimmed recordings, enable automatic recording list updates, and replace fragmented card refresh buttons with a single Master Refresh button on the console header.

**Independent Test**: Complete a recording or trim operation; verify that an SSE state change event (`recording`) triggers an automatic recording library refresh in the UI, and verify that the Master Refresh button updates all console sections simultaneously.

### Implementation for User Story 4

- [ ] T027 [US4] Broadcast `state.SectionRecording` via `appState.Broadcast(state.SectionRecording)` upon normal MP3 processing completion and audio trim job completion in `src/backend/lib/web/file_watcher.go` and `src/backend/lib/web/handlers_audio.go`
- [ ] T028 [US4] Wire frontend SSE changelog listener in `src/frontend/src/lib/audioState.svelte.ts` to trigger `files.fetchFiles()` when receiving `section === "recording"` events
- [ ] T029 [US4] Remove section-specific refresh button from `RecordingList.svelte` in `src/frontend/src/lib/components/admin/RecordingList.svelte`
- [ ] T030 [US4] Add a unified Master Refresh button in the console header of `AudioAdminView.svelte` in `src/frontend/src/lib/components/admin/AudioAdminView.svelte` that coordinates refreshing devices, engine state, recordings, and network connection info
- [ ] T031 [US4] Verify live updates and Master Refresh functionality with frontend unit tests in `src/frontend/src/lib/audioState.test.ts` and `src/frontend/src/lib/components/admin/AudioPlayer.test.ts`

**Checkpoint**: User Story 4 complete — live auto-updates for recordings and unified master refresh in place.

---

## Phase 7: User Story 5 - Code Quality Architecture, Encapsulation, and Verification (Priority: P3)

**Goal**: Audit large functions across touched packages, enforce struct field encapsulation, separate state from interface definitions, and ensure total compliance with Constitution Principles I-IX.

**Independent Test**: Audit struct visibility, functional naming, and separation of concerns across `audioengine`, `openai`, `state`, and `web`. Verify no external access to private struct fields and confirm all functions follow explicit verb-noun naming.

### Implementation for User Story 5

- [ ] T032 [US5] Audit and decompose any remaining large functions in `audioengine`, `openai`, and `web` into small, explicitly named functions (<40 lines each)
- [ ] T033 [US5] Enforce struct field encapsulation by making mutable internal fields unexported and providing clean accessor methods across `state` and `openai`
- [ ] T034 [US5] Ensure state models and interface declarations reside in distinctly named, logically separated files in `src/backend/lib/state/`
- [ ] T035 [US5] Verify that the path from PortAudio engine to recording storage is transparent, modular, and manually verifiable in under 3 minutes

**Checkpoint**: User Story 5 complete — code quality standards and encapsulation strictly satisfied.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: End-to-end verification, cleanup, and documentation confirmation.

- [ ] T036 [P] Run full backend race test suite `go test -v -race ./src/backend/...` and verify 0 failures and 0 race conditions
- [ ] T037 [P] Run frontend test suite `bun run test:unit` and build check `bun run build`
- [ ] T038 Validate all scenarios in [quickstart.md](file:///home/samuelmoses/Workspace/Church/Abel-Audio-Utils/specs/007-audio-core-and-system-refactor/quickstart.md)
- [ ] T039 Clean up any unused files, comments, or temporary artifacts

---

## Dependencies & Execution Order

### Phase Dependencies

```text
Phase 1: Setup
  ↓
Phase 2: Foundational (audio_processing domain)
  ↓
Phase 3: User Story 1 (Lock-Free Storage & Engine Pipeline) [P1 MVP]
  ↓
Phase 4: User Story 2 (Typed OpenAI Managers & PCM Util) [P1]
  ↓
Phase 5: User Story 3 (Network Discovery & QR Code Endpoint) [P2]
  ↓
Phase 6: User Story 4 (Real-Time SSE Updates & Master Refresh) [P2]
  ↓
Phase 7: User Story 5 (Code Quality, Encapsulation & Decomposition) [P3]
  ↓
Phase 8: Polish & Verification
```

### Parallel Execution Opportunities
- **Within Phase 2**: T003, T004, T005, T006 can be developed concurrently across separate files in `audio_processing`.
- **Within Phase 4**: T015 (`buffer.go`), T016 (`broadcast.go`), and T017 (`audio_util.go`) can be written in parallel.
- **Within Phase 8**: Backend test run (T036) and frontend build check (T037) can run concurrently.

---

## Implementation Strategy: MVP First

1. **Step 1**: Establish `audio_processing` (Phase 1 & Phase 2).
2. **Step 2**: Implement User Story 1 (Phase 3) to achieve a lock-free, clean recording pipeline (MVP). Validate with `go test -race ./src/backend/lib/audioengine/...`.
3. **Step 3**: Implement User Story 2 (Phase 4) with `github.com/zolstein/sync-map`, `audio_util.go` PCM helper, and resilient buffering. Validate with `go test -race ./src/backend/lib/openai/...`.
4. **Step 4**: Implement User Story 3 (Phase 5) for macOS Wi-Fi SSID and external QR code.
5. **Step 5**: Implement User Story 4 (Phase 6) for SSE recording updates and master console refresh.
6. **Step 6**: Complete code quality encapsulation and full test verification (Phases 7 & 8).
