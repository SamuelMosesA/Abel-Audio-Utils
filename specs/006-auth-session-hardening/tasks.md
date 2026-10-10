# Tasks: Admin Authentication Hardening, Dynamic Session Secrets, Server-Side Revocation, and Login Redirects

**Branch**: `006-auth-session-hardening` | **Date**: 2026-10-10 | **Spec**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md)

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Verify dependencies and test harness readiness across backend and frontend

- [x] T001 Verify backend Go environment and test runner with `go test ./src/backend/lib/config/...`
- [x] T002 [P] Verify frontend environment and Vitest runner with `bun run test:unit` in `src/frontend`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core state representations and config fields required by all user stories

**⚠️ CRITICAL**: Foundational state methods and config fields must be in place before wiring endpoints and middleware

- [x] T003 Add `SessionSecret string` (yaml tag `session_secret`) field to `Config` struct in `src/backend/lib/config/config.go`
- [x] T004 [P] Implement `RevokeSession(sessionID string)` and `IsSessionRevoked(sessionID string) bool` on `AppState` using thread-safe `sync.Map` in `src/backend/lib/state/app_state.go`
- [x] T005 Add unit tests for `RevokeSession` and `IsSessionRevoked` covering concurrent access and lookup in `src/backend/lib/state/state_test.go`

**Checkpoint**: Foundational state and config structures compiled and verified.

---

## Phase 3: User Story 1 - Automatic Unauthenticated Admin Page Redirect (Priority: P1) 🎯 MVP

**Goal**: Automatically redirect unauthenticated visits to `/admin` directly to `/login?redirect=/admin`, and return to `/admin` upon successful login.

**Independent Test**: In a clean browser session without cookies, visit `/admin`. Verify immediate redirection to `/login?redirect=/admin`. Log in with valid credentials, verify immediate navigation back to `/admin`.

### Tests for User Story 1 ⚠️

- [x] T006 [P] [US1] Add unit tests for redirect query parameter persistence and navigation in `src/frontend/src/lib/audioState.test.ts`

### Implementation for User Story 1

- [x] T007 [US1] Update unauthenticated effect guard in `src/frontend/src/routes/admin/+page.svelte` to redirect via `goto('/login?redirect=/admin')`
- [x] T008 [US1] Update login redirection logic in `src/frontend/src/routes/login/+page.svelte` and `src/frontend/src/lib/components/login/LoginView.svelte` to parse and navigate to `redirect` query parameter after successful authentication

**Checkpoint**: User Story 1 complete; unauthenticated visits to `/admin` redirect cleanly to `/login` and return on login.

---

## Phase 4: User Story 2 - Dynamic Cryptographic Session Secret Generation & Persistence (Priority: P1)

**Goal**: Replace hardcoded `"secret"` string with a high-entropy 256-bit random key persisted in `~/.config/abel/session.key` (`0600` permissions) or loaded from `config.yaml` (`session_secret`).

**Independent Test**: Start server without existing key file, verify 256-bit key is generated and stored with `0600` permissions. Log in to acquire session cookie. Restart server, verify existing session cookie continues to authenticate. Verify cookies signed with hardcoded `"secret"` are rejected.

### Tests for User Story 2 ⚠️

- [x] T009 [P] [US2] Add unit tests for `ResolveSessionSecret` testing 32-byte generation, file persistence with `0600` mode, and config override in `src/backend/lib/config/config_test.go`
- [x] T010 [P] [US2] Add integration test in `src/backend/lib/web/handlers_auth_test.go` verifying cookies signed with invalid secret or hardcoded `"secret"` are rejected

### Implementation for User Story 2

- [x] T011 [US2] Implement `ResolveSessionSecret(cfg *Config) ([]byte, error)` in `src/backend/lib/config/config.go` to generate 32 random bytes or load from `session.key` with `0600` file permissions
- [x] T012 [US2] Update `NewRouter` in `src/backend/lib/web/router.go` to initialize `cookie.NewStore` with resolved session secret instead of hardcoded `"secret"`

**Checkpoint**: User Story 2 complete; dynamic session keys generated, persisted, and verified across server restarts.

---

## Phase 5: User Story 3 - Server-Side Session Revocation and Immediate Invalidation (Priority: P2)

**Goal**: Provide server-side session revocation on logout, immediately invalidating the session ID in `AppState` and preventing cookie replay.

**Independent Test**: Log in to obtain a valid session cookie, call `DELETE /api/auth/session` (or `POST /api/auth/logout`), then attempt an admin API request reusing the same cookie. Verify request returns HTTP 401 Unauthorized.

### Tests for User Story 3 ⚠️

- [x] T013 [P] [US3] Add unit tests for session revocation, cookie clearing, and revoked token rejection in `src/backend/lib/web/handlers_auth_test.go`

### Implementation for User Story 3

- [x] T014 [US3] Implement `LogoutHandler(appState *state.AppState)` in `src/backend/lib/web/handlers_auth.go` supporting `DELETE /api/auth/session` and `POST /api/auth/logout` with cookie clearing and server revocation
- [x] T015 [US3] Register `DELETE /api/auth/session` and `POST /api/auth/logout` routes in `src/backend/lib/web/router.go`
- [x] T016 [US3] Update `SessionAuthMiddleware` in `src/backend/lib/web/handlers_auth.go` to check `appState.IsSessionRevoked(sessionID)` and return 401 Unauthorized if revoked
- [x] T017 [US3] Update `SystemStore.logout()` in `src/frontend/src/lib/audioState.svelte.ts` to send `DELETE /api/auth/session` request before clearing client state

**Checkpoint**: User Story 3 complete; server-side revocation and logout integration fully operational.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: End-to-end regression testing and validation against quickstart scenarios

- [x] T018 [P] Run full backend test suite with race detector with `go test -race ./src/backend/...`
- [x] T019 [P] Run frontend test suite with `bun run test:unit` in `src/frontend`
- [x] T020 Execute manual quickstart verification scenarios per `specs/006-auth-session-hardening/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: Independent, starts immediately
- **Foundational (Phase 2)**: Depends on Phase 1 - BLOCKS User Stories 2 and 3
- **User Story 1 (Phase 3)**: Frontend-only; can run independently after Phase 1 [MVP candidate]
- **User Story 2 (Phase 4)**: Depends on Phase 2 (Foundational config field)
- **User Story 3 (Phase 5)**: Depends on Phase 2 (Foundational AppState revocation methods)
- **Polish (Phase 6)**: Depends on completion of all stories (Phase 3, 4, 5)

### User Story Completion Order

```text
Foundational (Phase 2)
  ├── User Story 1: Admin Page Redirection (Phase 3) [MVP candidate]
  ├── User Story 2: Dynamic Session Secrets & Persistence (Phase 4)
  └── User Story 3: Server-Side Session Revocation (Phase 5)
```

### Parallel Opportunities

- T001 and T002 can run in parallel
- T003 and T004 can run in parallel
- T006, T009, T010, and T013 are in distinct subsystems/files and can run in parallel
- T018 and T019 can run in parallel

---

## Parallel Execution Examples

### Parallel Example: User Story 1
```bash
# Frontend test and route guard updates in parallel:
Task: "T006 [P] [US1] Add unit tests for redirect query parameter persistence in src/frontend/src/lib/audioState.test.ts"
Task: "T007 [US1] Update unauthenticated effect guard in src/frontend/src/routes/admin/+page.svelte"
```

### Parallel Example: User Story 2
```bash
# Backend config test and handler contract tests in parallel:
Task: "T009 [P] [US2] Add unit tests for ResolveSessionSecret in src/backend/lib/config/config_test.go"
Task: "T010 [P] [US2] Add integration test in src/backend/lib/web/handlers_auth_test.go"
```

### Parallel Example: User Story 3
```bash
# Revocation test and handler implementation in parallel:
Task: "T013 [P] [US3] Add unit tests for session revocation in src/backend/lib/web/handlers_auth_test.go"
Task: "T014 [US3] Implement LogoutHandler in src/backend/lib/web/handlers_auth.go"
```

---

## Implementation Strategy

### MVP First (User Story 1)
1. Complete Phase 1 (Setup)
2. Complete Phase 3 (User Story 1: Admin Page Redirection)
3. Validate `/admin` redirect in browser

### Incremental Delivery
1. Phase 1 & 2: Setup & Foundational State
2. Phase 3: Deliver Admin Page Redirection (US1) [MVP]
3. Phase 4: Deliver Dynamic Session Secrets & Persistence (US2)
4. Phase 5: Deliver Server-Side Session Revocation (US3)
5. Phase 6: Run full test suites and quickstart validation

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story is independently completable and testable
- Verify tests fail before implementing
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
