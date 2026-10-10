# Tasks: Admin UI Unauthenticated Session Redirect

**Branch**: `008-admin-unauth-redirect` | **Date**: 2026-10-10 | **Spec**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Verify baseline tests and establish test environment for auth refactoring

- [X] T001 Verify existing frontend and backend test suites run cleanly via `go test ./src/backend/lib/web` and `bun test` in `src/frontend`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Authoritative server-side session check endpoint required by frontend guards

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T002 Implement `GetSessionHandler` returning session status `200 OK` or `401 Unauthorized` in `src/backend/lib/web/handlers_auth.go`
- [X] T003 Register `GET /api/auth/session` route with `SessionAuthMiddleware` in `src/backend/lib/web/router.go`
- [X] T004 [P] Add unit test suite for `GET /api/auth/session` in `src/backend/lib/web/handlers_auth_test.go`

**Checkpoint**: Foundation ready - backend session verification is live and testable.

---

## Phase 3: User Story 1 - Direct Navigation to Admin When Unauthenticated (Priority: P1) 🎯 MVP

**Goal**: Prevent unauthorized access and UI flickering by intercepting `/admin` at the layout level and redirecting unauthenticated visits.

**Independent Test**: Navigate directly to `/admin` in a clean browser session with no credentials. Verify instant redirection to `/login?redirect=/admin` with zero flash of admin components.

### Tests for User Story 1

- [X] T005 [P] [US1] Add unit and component tests for layout guard redirect behavior in `src/frontend/src/routes/admin/admin.test.ts`

### Implementation for User Story 1

- [X] T006 [P] [US1] Create SvelteKit layout route guard in `src/frontend/src/routes/admin/+layout.svelte` guarding child components during validation
- [X] T007 [US1] Refactor `src/frontend/src/routes/admin/+page.svelte` to remove inline auth redirect logic in favor of the layout guard

**Checkpoint**: User Story 1 fully functional. Direct visits to `/admin` are cleanly guarded.

---

## Phase 4: User Story 2 - Stale LocalStorage Session Eviction and Redirect (Priority: P1)

**Goal**: Validate sessions against backend server truth, purge stale `localStorage` tokens upon 401s, and provide clear user feedback on login view.

**Independent Test**: Inject a bogus `session_id` into `localStorage`, attempt an admin API call or navigate to `/admin`, and verify that `localStorage` is purged, user is routed to `/login?redirect=/admin&reason=expired`, and an expiry alert banner is displayed.

### Tests for User Story 2

- [X] T008 [P] [US2] Add unit tests for `SystemStore.validateSession()`, `clearSession()`, and 401 eviction in `src/frontend/src/lib/audioState.test.ts`

### Implementation for User Story 2

- [X] T009 [P] [US2] Refactor `SystemStore` in `src/frontend/src/lib/audioState.svelte.ts` to encapsulate auth state with `validateSession()`, `clearSession()`, and `setAuthenticated()`, eliminating unchecked `localStorage` authentication assumptions
- [X] T010 [P] [US2] Enhance `fetchWithSync` in `src/frontend/src/lib/utils/api.ts` to intercept `401 Unauthorized` responses, trigger `system.clearSession()`, and navigate to `/login?redirect=/admin&reason=expired`
- [X] T011 [US2] Update `src/frontend/src/routes/login/+page.svelte` to detect `reason=expired` query parameter and display an expiration warning alert banner

**Checkpoint**: User Story 2 fully functional. Stale tokens are evicted and feedback is presented.

---

## Phase 5: User Story 3 - Post-Authentication Return to Target View (Priority: P2)

**Goal**: Safely preserve and restore user destination post-login without exposing open redirect vulnerabilities.

**Independent Test**: Navigate to `/login?redirect=/admin`, authenticate with valid credentials, and verify immediate forwarding to `/admin`. Verify malicious targets (`//evil.com`, `https://...`) fall back safely to `/admin`.

### Tests for User Story 3

- [X] T012 [P] [US3] Add unit tests for `resolveRedirect` sanitization edge cases in `src/frontend/src/lib/utils/redirect.test.ts`

### Implementation for User Story 3

- [X] T013 [P] [US3] Harden `resolveRedirect` in `src/frontend/src/lib/utils/redirect.ts` against protocol-relative URLs (`//`), backslashes, and schema injection
- [X] T014 [US3] Ensure post-login redirection in `src/frontend/src/routes/login/+page.svelte` consumes `resolveRedirect` with sanitized target

**Checkpoint**: All user stories complete and functional.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Validation, linting, regression testing, and verification

- [X] T015 [P] Run full backend Go test suite `go test -race ./src/backend/...` to verify race-free execution
- [X] T016 [P] Run full frontend unit test suite `bun test` in `src/frontend`
- [X] T017 Execute manual validation scenarios in `specs/008-admin-unauth-redirect/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - executes immediately.
- **Foundational (Phase 2)**: Depends on Setup - BLOCKS all frontend user stories (provides `GET /api/auth/session`).
- **User Story 1 (Phase 3)**: Depends on Foundational phase.
- **User Story 2 (Phase 4)**: Depends on Foundational phase; integrates with `SystemStore` and API utility.
- **User Story 3 (Phase 5)**: Depends on Foundational phase; integrates with login view.
- **Polish (Phase 6)**: Runs after all user stories are complete.

### Parallel Opportunities

- T004 (backend auth tests) can run in parallel with frontend test preparation.
- T005, T008, T012 (test tasks across US1, US2, US3) can be authored in parallel once Phase 2 completes.
- T009 (`audioState.svelte.ts`) and T013 (`redirect.ts`) can be implemented in parallel as they touch independent files.
- T015 and T016 (test runs) can execute concurrently.

---

## Implementation Strategy

### MVP First (User Stories 1 & 2)
1. Complete Phase 1 (Setup) and Phase 2 (Foundational backend endpoint).
2. Complete Phase 3 (US1 Layout Route Guard) → Immediate protection against unauthenticated visitors.
3. Complete Phase 4 (US2 Eviction & Expiry banner) → Complete resilience against stale credentials and mid-session timeouts.
4. Complete Phase 5 (US3 Redirect Sanitization) → Hardened user workflow and open-redirect protection.
5. Finalize with Phase 6 (Polish & Quickstart validation).
