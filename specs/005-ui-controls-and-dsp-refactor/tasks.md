# Tasks: UI Control Refinements, Per-Language AI Killswitch, and DSP Boost Utility Extraction

**Branch**: `005-ui-controls-and-dsp-refactor` | **Date**: 2026-10-10 | **Spec**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md)

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Verify dependencies and test harness readiness across backend and frontend

- [x] T001 Verify backend Go environment and test runner with `go test ./src/backend/lib/config/...`
- [x] T002 [P] Verify frontend environment and Vitest runner with `bun run test:unit` in `src/frontend`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core state representations and interfaces required by all user stories

**⚠️ CRITICAL**: Foundational state methods must be in place before wiring endpoints and UI controls

- [x] T003 Implement `BlockedLanguages` map and thread-safe `IsBlocked` / `SetBlocked` methods on `AIConfig` in `src/backend/lib/state/config.go`
- [x] T004 [P] Add unit tests for `AIConfig.IsBlocked` and `AIConfig.SetBlocked` in `src/backend/lib/state/state_test.go`
- [x] T005 Add `GetListenerCount(language string) int` to `state.Translator` interface in `src/backend/lib/state/app_state.go`

**Checkpoint**: Foundational state and interface contracts compiled and verified.

---

## Phase 3: User Story 1 - On-Demand Recording Library Refresh (Priority: P1) 🎯 MVP

**Goal**: Remove background polling interval from recording library view and add a manual Refresh button.

**Independent Test**: Load the Admin Recording Library view, verify network tab shows no automated background requests to `/api/recordings/library`, click Refresh button, verify library updates on demand.

### Implementation for User Story 1

- [x] T006 [US1] Remove recurring `setInterval` polling for `fetchFiles()` in `src/frontend/src/lib/components/admin/RecordingList.svelte`
- [x] T007 [US1] Add manual Refresh button with icon and busy state in `src/frontend/src/lib/components/admin/RecordingList.svelte`
- [x] T008 [US1] Verify recording list refresh behavior and ensure no regressions in `src/frontend/src/lib/components/admin/RecordingList.svelte`

**Checkpoint**: User Story 1 functional; recording polling completely eliminated.

---

## Phase 4: User Story 2 - Per-Language AI Killswitch with Active Client Metrics (Priority: P1)

**Goal**: Provide granular per-language killswitch control, listener count tracking, and manual refresh in AI translation panel.

**Independent Test**: View the AI translation admin panel to confirm all configured languages are displayed with listener counts. Toggle a language killswitch, verify that active streaming stops, incoming client connections are blocked, and the blocked status remains across UI reloads.

### Backend Implementation for User Story 2

- [x] T009 [P] [US2] Implement listener count calculation (SSE subscribers + active streams) and `GetListenerCount` in `src/backend/lib/openai/translation.go` and `src/backend/lib/openai/realtime_base.go`
- [x] T010 [P] [US2] Update `MockTranslator` with `GetListenerCount` in `src/backend/lib/web/test_helpers.go`
- [x] T011 [US2] Update `UpdateAIStreams` handler to support `action == "toggle_language"` with language stop logic in `src/backend/lib/web/handlers_ai.go`
- [x] T012 [US2] Update `GetAIStreamsStatus` handler to return enriched `languages` list (code, name, blocked, active, listeners) in `src/backend/lib/web/handlers_ai.go`
- [x] T013 [US2] Add killswitch block guard to `SubtitlesHandler` (return HTTP 503 or error message if language is blocked) in `src/backend/lib/web/handlers_ai.go`
- [x] T014 [US2] Add killswitch block guard to HLS audio handler (reject request with HTTP 503 if language is blocked) in `src/backend/lib/web/handlers_audio.go`
- [x] T015 [US2] Add unit tests for `toggle_language`, blocked status in `GetAIStreamsStatus`, and subtitle rejection in `src/backend/lib/web/handlers_ai_test.go`
- [x] T016 [US2] Add unit tests for HLS blocked language rejection in `src/backend/lib/web/handlers_audio_test.go`

### Frontend Implementation for User Story 2

- [x] T017 [US2] Update AI state manager in `src/frontend/src/lib/audioState.svelte.ts` with `toggleLanguageKillswitch` action and `refreshAIStreams` method
- [x] T018 [US2] Update `src/frontend/src/lib/components/admin/TranslationAdmin.svelte` to display all configured languages with active listener counts and individual killswitch toggles
- [x] T019 [US2] Add manual Refresh button in `src/frontend/src/lib/components/admin/TranslationAdmin.svelte` toolbar to trigger on-demand metrics and status updates

**Checkpoint**: User Story 2 functional; per-language killswitches operate and active listener metrics display cleanly.

---

## Phase 5: User Story 3 - Modular Audio DSP Boost & Clamping Utility Extraction (Priority: P2)

**Goal**: Extract channel routing, digital gain multiplication, and sample clamping (`[-1.0, 1.0]`) into `ExtractStereoChunk` in the audio conversion package.

**Independent Test**: Run unit tests in `conversion` to verify samples are properly scaled, strictly bounded within `[-1.0, 1.0]`, and that non-positive boost defaults to 1.0. Run audio engine tests to confirm zero regressions.

### Implementation for User Story 3

- [x] T020 [P] [US3] Implement `ExtractStereoChunk(in []float32, bufferSize int, openedChannels int, chL int, chR int, boost float32) []float32` in `src/backend/lib/audioengine/conversion/pcm.go`
- [x] T021 [P] [US3] Add unit tests `TestExtractStereoChunk` covering gain scaling, clamping boundaries `[-1.0, 1.0]`, and `<= 0.0` boost fallback in `src/backend/lib/audioengine/conversion/conversion_test.go`
- [x] T022 [US3] Refactor audio capture loop in `src/backend/lib/audioengine/engine.go` (lines 283–311) to call `conversion.ExtractStereoChunk`
- [x] T023 [US3] Run audio engine unit tests to confirm capture loop operates without regressions with `go test -v ./src/backend/lib/audioengine/...`

**Checkpoint**: User Story 3 complete; DSP utility extracted and verified.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: End-to-end regression testing and validation against quickstart scenarios

- [x] T024 [P] Run full backend test suite with race detector with `go test -race ./src/backend/...`
- [x] T025 [P] Run frontend test suite with `bun run test:unit` in `src/frontend`
- [x] T026 Execute manual quickstart verification scenarios per `specs/005-ui-controls-and-dsp-refactor/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: Independent, starts immediately
- **Foundational (Phase 2)**: Depends on Phase 1 - BLOCKS User Story 2
- **User Story 1 (Phase 3)**: Frontend-only; can run independently after Phase 1
- **User Story 2 (Phase 4)**: Depends on Phase 2 (foundational state and interface changes)
- **User Story 3 (Phase 5)**: Audio engine DSP extraction; can run independently after Phase 1
- **Polish (Phase 6)**: Depends on completion of all stories (Phase 3, 4, 5)

### User Story Completion Order

```text
Foundational (Phase 2)
  ├── User Story 1: Recording Library Refresh (Phase 3) [MVP candidate]
  ├── User Story 2: Per-Language AI Killswitch (Phase 4)
  └── User Story 3: Audio DSP Utility Extraction (Phase 5)
```

### Parallel Opportunities

- T001 and T002 can execute in parallel
- T003 and T004 can execute in parallel
- T006, T009, and T020 are in distinct packages/files and can execute in parallel
- T024 and T025 can run in parallel

---

## Implementation Strategy

### MVP First (User Story 1)
1. Complete Phase 1 (Setup)
2. Complete Phase 3 (User Story 1)
3. Validate elimination of `/api/recordings/library` 2s polling

### Incremental Delivery
1. Phase 1 & 2: Setup & Foundational State
2. Phase 3: Deliver Recording Library Refresh
3. Phase 4: Deliver Per-Language AI Killswitch & Listener Metrics
4. Phase 5: Deliver Audio DSP Utility Extraction
5. Phase 6: Run full test suites and quickstart validation
