# Implementation Plan: Admin Authentication Hardening, Dynamic Session Secrets, Server-Side Revocation, and Login Redirects

**Branch**: `006-auth-session-hardening` | **Date**: 2026-10-10 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/006-auth-session-hardening/spec.md`

## Summary

This plan hardens the administrative authentication system in Abel Audio Utils:
1. **Dynamic Session Signing Secret**: Replaces hardcoded `"secret"` with an automatically generated 256-bit cryptographically secure key persisted to `~/.config/abel/session.key` (`0600` permissions), with optional `session_secret` override in `config.yaml`.
2. **Server-Side Session Revocation**: Implements `DELETE /api/auth/session` (and `POST /api/auth/logout`) and an in-memory `RevokedSessions` registry in `AppState` to immediately reject replayed or logged-out session cookies.
3. **Unauthenticated Admin Redirection**: Enhances frontend routing in SvelteKit so that unauthenticated access to `/admin` immediately redirects to `/login?redirect=/admin`, and successful login returns to the intended destination.

## Technical Context

**Language/Version**: Go 1.24+ (Backend), TypeScript / Svelte 5 / SvelteKit (Frontend)

**Primary Dependencies**: `github.com/gin-contrib/sessions`, `github.com/gin-contrib/sessions/cookie`, `github.com/gin-gonic/gin`, SvelteKit

**Storage**: Local secure file `session.key` (`0600` permissions), in-memory `sync.Map` in `AppState`

**Testing**: Go standard testing with `testify`, Vitest / Playwright

**Target Platform**: Linux / macOS server and modern web browsers

**Project Type**: Web service + Audio interface daemon

**Performance Goals**: Session authentication check latency < 0.1ms; instant (< 10ms) client redirection

**Constraints**: Must not require external databases (Redis/PostgreSQL); single-binary standalone deployment

**Scale/Scope**: Single church operator console with occasional concurrent admin tabs

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Principle I: Radical Simplicity & Code Minimization**: PASS. Reuses existing `gin-contrib/sessions` cookie infrastructure with a generated key file and an in-memory revocation map instead of adding heavy database or Redis dependencies.
- **Principle II: Streamlined & Maintainable Unit Tests**: PASS. Clean unit and HTTP handler tests covering key generation, persistence, restart validation, and logout revocation.
- **Principle III: Converged Architecture & Cohesion**: PASS. Session state logic consolidated within `state.AppState` and `web.handlers_auth`.
- **Principle IV: Hierarchical Domain Organization**: PASS. Auth handlers in `web`, state in `state`, config in `config`.
- **Principle V: Zero Duplication**: PASS. Single authoritative source for session secret loading and revocation checking.
- **Principle VI: Frequent Atomic Commits**: PASS. Discrete tasks committed atomically.

## Project Structure

### Documentation (this feature)

```text
specs/006-auth-session-hardening/
├── plan.md              # This plan
├── research.md          # Architectural research & decisions
├── data-model.md        # Entities and state layout
├── quickstart.md        # Run and validation guide
├── contracts/           # API contracts
│   └── api-contracts.md
└── checklists/
    └── requirements.md  # Requirements checklist
```

### Source Code Layout

```text
src/
├── backend/
│   ├── lib/
│   │   ├── config/
│   │   │   ├── config.go                # SessionSecret field and key resolution helper
│   │   │   └── config_test.go           # Key generation and persistence tests
│   │   ├── state/
│   │   │   └── app_state.go             # RevokedSessions map and helper methods
│   │   └── web/
│   │       ├── router.go                # Uses resolved session key
│   │       ├── handlers_auth.go         # LogoutHandler & revocation in SessionAuthMiddleware
│   │       └── handlers_auth_test.go    # Tests for revocation and cookie verification
└── frontend/
    └── src/
        ├── lib/
        │   ├── audioState.svelte.ts     # logout() calls DELETE /api/auth/session
        │   └── components/
        │       └── login/
        │           └── LoginView.svelte # Honors redirect query parameter
        └── routes/
            ├── admin/
            │   └── +page.svelte         # Redirects unauthenticated visits to /login?redirect=/admin
            └── login/
                └── +page.svelte         # Redirects authenticated users to redirect query param
```

**Structure Decision**: Standard Abel web architecture.

## Complexity Tracking

No constitution violations or unjustified abstractions introduced.
