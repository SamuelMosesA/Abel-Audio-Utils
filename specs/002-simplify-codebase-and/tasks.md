# Implementation Tasks: Codebase Simplification, Documentation & Reduction

**Feature Name**: `simplify-codebase-and-reduce-size`  
**Feature Branch**: `002-simplify-codebase-and`  
**Spec**: [spec.md](file:///home/samuelmoses/Workspace/Church/Abel-Audio-Utils/specs/002-simplify-codebase-and/spec.md) | **Plan**: [plan.md](file:///home/samuelmoses/Workspace/Church/Abel-Audio-Utils/specs/002-simplify-codebase-and/plan.md)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Baseline measurement, automated screenshot capture scripts, and commit checkpoints

- [ ] T001 Record baseline LOC metrics across Go and Svelte/TS files in `specs/002-simplify-codebase-and/quickstart.md`
- [ ] T002 [P] Capture baseline UI screenshots (`landing_baseline.png`, `admin_baseline.png`) using headless chromium / playwright
- [ ] T003 Autocommit setup artifacts per Constitution VI

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core Go docstrings and domain boundaries

- [ ] T004 Add comprehensive godoc docstrings to all exported types and functions in `src/backend/lib/audioengine/conversion/`
- [ ] T005 [P] Add comprehensive godoc docstrings to all exported types and functions in `src/backend/lib/audioengine/`
- [ ] T006 [P] Add comprehensive godoc docstrings to all exported types and functions in `src/backend/lib/state/` and `src/backend/lib/config/`
- [ ] T007 Add comprehensive godoc docstrings to all exported types and functions in `src/backend/lib/openai/` and `src/backend/lib/web/`
- [ ] T008 Autocommit foundational docstrings per Constitution VI

---

## Phase 3: User Story 1 - Human-Readable & Hierarchical Go Backend (Priority: P1) 🎯 MVP

**Goal**: Break up monolithic functions, reduce indentation depth, and improve backend readability

**Independent Test**: Code inspection confirming zero monolithic functions >60 lines and clear godoc documentation.

- [ ] T009 [US1] Decompose `recording_processor.go` by extracting helper execution and status reporting routines into modular helpers
- [ ] T010 [P] [US1] Simplify `handlers_recordings.go` by extracting repeated file response and error serialization into shared handlers
- [ ] T011 [US1] Autocommit User Story 1 refactoring per Constitution VI

---

## Phase 4: User Story 2 - Codebase Volume Reduction (~25% Deletion) (Priority: P2)

**Goal**: Prune dead code, redundant wrappers, and over-abstracted utilities across backend and frontend

**Independent Test**: Measure LOC and verify reduction towards 75% baseline with zero regressions in API endpoints.

- [ ] T012 [US2] Prune redundant validation boilerplate and dead struct methods in `src/backend/lib/web/`
- [ ] T013 [P] [US2] De-duplicate overlapping OpenAI reconnect logic across `translation.go` and `transcription.go` into `realtime_base.go`
- [ ] T014 [US2] Autocommit code pruning changes per Constitution VI

---

## Phase 5: User Story 3 - Centralized CSS & Streamlined Frontend Styling (Priority: P3)

**Goal**: Centralize CSS customization in `app.css` using semantic design tokens, prune inline class bloat, and validate visually with screenshots

**Independent Test**: Run `cd src/frontend && bun run test:unit && bun run check && bun run build` with 100% pass rate, then compare visual screenshots.

- [ ] T015 [US3] Centralize semantic design tokens in `src/frontend/src/app.css` and delete dead classes (`.glow-primary`, `.card-premium`)
- [ ] T016 [P] [US3] Streamline `src/frontend/src/lib/components/admin/RecordingList.svelte` to replace multi-line inline Tailwind chains with centralized design classes
- [ ] T017 [P] [US3] Streamline `src/frontend/src/lib/components/admin/AudioAdminView.svelte` to prune excess wrapper containers and simplify classes
- [ ] T018 [US3] Simplify reactive state management in `src/frontend/src/lib/audioState.svelte.ts`
- [ ] T019 [US3] Capture post-refactoring screenshots (`landing_after.png`, `admin_after.png`) and perform visual comparison against baseline
- [ ] T020 [US3] Run frontend verification suite (`bun run test:unit`, `bun run check`, `bun run build`) and autocommit per Constitution VI

---

## Phase 6: User Story 4 - Lean & Readable Test Suites (Priority: P4)

**Goal**: Simplify repetitive unit test fixtures into concise table-driven tests

**Independent Test**: Verify full passing status for all unit tests with reduced test code line counts.

- [ ] T021 [US4] Convert repetitive assertions in `recording_processor_test.go` to compact table-driven tests
- [ ] T022 [P] [US4] Simplify frontend test files (`audioState.test.ts`, `audioVisuals.test.ts`)
- [ ] T023 [US4] Autocommit test suite simplifications per Constitution VI

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Final audit, LOC verification, and system documentation

- [ ] T024 Run final LOC measurement confirming progress toward the 75% target size
- [ ] T025 Final frontend verification (`bun run test:unit && bun run check && bun run build`)
- [ ] T026 Update `ARCHITECTURE.md` and commit final polish increment per Constitution VI
