# Implementation Tasks: Simplify & Consolidate Audio Codebase

**Feature Name**: `simplify-and-consolidate-audio-codebase`  
**Feature Branch**: `001-simplify-and-consolidate`  
**Spec**: [spec.md](file:///home/samuelmoses/Workspace/Church/Abel-Audio-Utils/specs/001-simplify-and-consolidate/spec.md) | **Plan**: [plan.md](file:///home/samuelmoses/Workspace/Church/Abel-Audio-Utils/specs/001-simplify-and-consolidate/plan.md)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create centralized audio conversion package layout and infrastructure

- [X] T001 Create conversion directory structure at `src/backend/lib/audioengine/conversion/`
- [X] T002 [P] Establish package declaration and conversion constants in `src/backend/lib/audioengine/conversion/pcm.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core conversions required by audio engine and AI managers

- [X] T003 Implement `Float32ToPCM16` and `PCM16ToFloat32` with boundary clamping in `src/backend/lib/audioengine/conversion/pcm.go`
- [X] T004 [P] Implement `DownsampleStereoToMonoPCM24k` in `src/backend/lib/audioengine/conversion/resample.go`
- [X] T005 [P] Implement WAV header generation utilities `WritePlaceholderWavHeader` and `FinalizeWavHeader` in `src/backend/lib/audioengine/conversion/wav.go`
- [X] T006 Write table-driven unit tests for conversions in `src/backend/lib/audioengine/conversion/conversion_test.go`

---

## Phase 3: User Story 1 - Centralized Audio Conversions (Priority: P1) 🎯 MVP

**Goal**: Migrate all scattered audio conversions to use `audioengine/conversion`

**Independent Test**: Build and run audio tests verifying `hls.go`, `storage.go`, and `wav.go` produce identical PCM/WAV output.

- [X] T007 [P] [US1] Refactor `src/backend/lib/audioengine/hls.go` to replace local `float32ToPCM16` with `conversion.Float32ToPCM16`
- [X] T008 [P] [US1] Refactor `src/backend/lib/audioengine/wav.go` to delegate to `conversion.WritePlaceholderWavHeader` and `conversion.FinalizeWavHeader`
- [X] T009 [US1] Refactor `src/backend/lib/audioengine/storage.go` and `src/backend/lib/web/handlers_recordings.go` to use the centralized conversion package
- [X] T010 [US1] Update `src/backend/lib/audioengine/hls_test.go` and `storage_test.go` to verify streamlined conversion calls

---

## Phase 4: User Story 2 - Unified AI Realtime Lifecycle & Disconnection (Priority: P2)

**Goal**: Deduplicate WebSocket connection and disconnection error handling between transcription and translation

**Independent Test**: Connect mock Realtime sessions and verify clean disconnect/reconnect and token telemetry.

- [X] T011 [US2] Implement shared `RunRealtimeSession` loop with unified disconnect/error teardown in `src/backend/lib/openai/realtime_base.go`
- [X] T012 [US2] Refactor `src/backend/lib/openai/transcription.go` to use `conversion.DownsampleStereoToMonoPCM24k` and the unified `RunRealtimeSession`
- [X] T013 [US2] Refactor `src/backend/lib/openai/translation.go` to use `conversion.DownsampleStereoToMonoPCM24k` and the unified `RunRealtimeSession`
- [X] T014 [US2] Simplify unit tests in `src/backend/lib/web/handlers_ai_test.go`

---

## Phase 5: User Story 3 - Lean Frontend UI Architecture (Priority: P3)

**Goal**: Simplify frontend markup, eliminate redundant wrappers and classes, and minimize code volume

**Independent Test**: Run `npm run check` and `npm run test` in `src/frontend` to ensure zero regressions in UI views.

- [X] T015 [P] [US3] Streamline and prune redundant classes/wrappers in `src/frontend/src/lib/components/admin/AudioAdminView.svelte`
- [X] T016 [P] [US3] Streamline and prune redundant classes/wrappers in `src/frontend/src/lib/components/admin/TranslationAdmin.svelte`
- [X] T017 [US3] Simplify reactive store helpers in `src/frontend/src/lib/audioState.svelte.ts` and `src/frontend/src/lib/audioVisuals.svelte.ts`

---

## Phase 6: User Story 4 - Streamlined Unit Test Suites (Priority: P4)

**Goal**: Simplify long and repetitive unit tests into concise table-driven tests

**Independent Test**: Run full test suites (`go test ./...` and `npm run test`) and observe reduced test code size.

- [X] T018 [P] [US4] Convert verbose test blocks in `src/backend/lib/web/handlers_audio_test.go` to concise table-driven tests
- [X] T019 [P] [US4] Convert verbose test blocks in `src/backend/lib/web/handlers_recordings_test.go` to concise table-driven tests
- [X] T020 [US4] Simplify frontend test fixtures in `src/frontend/src/lib/audioState.test.ts`

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Global validation, compilation, and compliance check

- [X] T021 Execute full backend test pass: `go test -v ./src/backend/...`
- [X] T022 Execute full frontend test and build pass: `cd src/frontend && npm run build`
- [X] T023 Update `ARCHITECTURE.md` to reflect the simplified architecture and new `conversion/` hierarchy
