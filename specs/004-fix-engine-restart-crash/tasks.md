# Implementation Tasks: Fix Audio Engine Restart Crash

**Feature**: Safe and Crash-Resilient Audio Engine Restart
**Branch**: `004-fix-engine-restart-crash`
**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)

## Phase 1: Setup

**Purpose**: Baseline verification of current backend audio tests.

- [ ] T001 Verify baseline audio engine tests via `go test ./src/backend/lib/audioengine/...`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core synchronization primitives and channel lifecycle management.

- [ ] T002 Add engine lifecycle done coordination (`DoneAudio chan struct{}`) and thread-safe access in `src/backend/lib/state/app_state.go`
- [ ] T003 Introduce package-level engine mutex (`engineMu sync.Mutex`) in `src/backend/lib/audioengine/engine.go`

**Checkpoint**: Foundational primitives in place; user story implementation can proceed.

---

## Phase 3: User Story 1 - Resilient Audio Engine Restart & Hardware Discovery (Priority: P1) 🎯 MVP

**Goal**: Cleanly stop active stream, safely reinitialize PortAudio without SIGSEGV, scan devices, and reconnect previous device.

**Independent Test**: Trigger engine restart during active streaming; verify 200 OK, refreshed devices, and uninterrupted server operation.

- [ ] T004 [US1] Implement `StopAudioEngine(appState *state.AppState)` with deterministic goroutine wait and safe channel closure in `src/backend/lib/audioengine/engine.go`
- [ ] T005 [US1] Synchronize `refreshDevices` and `RestartEngine` under `engineMu` in `src/backend/lib/audioengine/engine.go`
- [ ] T006 [US1] Add unit tests for synchronized restart and device re-connection in `src/backend/lib/audioengine/engine_test.go`

**Checkpoint**: User Story 1 functional; restart works reliably without crashing the server.

---

## Phase 4: User Story 2 - Protection Against Concurrent & Rapid Restart Requests (Priority: P2)

**Goal**: Prevent overlapping restart executions and channel double-closures under concurrent requests.

**Independent Test**: Fire 5 concurrent restart requests; verify that requests are serialized or return 409 Conflict without crashing or panicking.

- [ ] T007 [US2] Guard `RestartAudioEngine` HTTP handler against concurrent executions with conflict status in `src/backend/lib/web/handlers_audio.go`
- [ ] T008 [US2] Add unit test for concurrent restart requests returning 409 Conflict in `src/backend/lib/web/handlers_audio_test.go`

**Checkpoint**: User Stories 1 and 2 functional; concurrent bursts handled safely.

---

## Phase 5: User Story 3 - Deterministic Stream Teardown Handshake (Priority: P3)

**Goal**: Ensure blocking `stream.Read()` or disconnected device loops terminate cleanly and signal completion before driver termination.

**Independent Test**: Simulate device read errors and verify stream goroutine exits cleanly within deadline.

- [ ] T009 [US3] Ensure capture loop handles read errors, breaks on teardown signal, and signals `DoneAudio` on exit in `src/backend/lib/audioengine/engine.go`
- [ ] T010 [US3] Add unit test for graceful teardown during active mock streaming in `src/backend/lib/audioengine/engine_test.go`

**Checkpoint**: All user stories functional.

---

## Phase 6: Polish & Verification

**Purpose**: System-wide regression checks and stress testing.

- [ ] T011 Run full backend test suite with race detector via `go test -race ./src/backend/...`
- [ ] T012 Run quickstart validation against live binary with burst restart curl script
- [ ] T013 Verify frontend build and checks via `bun run check`
