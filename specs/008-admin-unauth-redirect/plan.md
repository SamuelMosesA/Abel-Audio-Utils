# Implementation Plan: Admin UI Unauthenticated Session Redirect

**Branch**: `008-admin-unauth-redirect` | **Date**: 2026-10-10 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/008-admin-unauth-redirect/spec.md`

## Summary

Eliminate unauthorized access vulnerabilities and UI flickering on the administrative interface by replacing client-side `localStorage` authentication assumptions with authoritative backend session validation, layout-level route guards in SvelteKit, and active 401 interception. When unauthenticated or when sessions expire, stale storage is cleared and users are smoothly routed to `/login?redirect=/admin&reason=expired` with informative feedback.

## Technical Context

**Language/Version**: TypeScript 5.6 / SvelteKit 2 (Svelte 5 runes) for frontend; Go 1.23+ with Gin for backend.

**Primary Dependencies**: SvelteKit, Gin (`github.com/gin-gonic/gin`), `github.com/gin-contrib/sessions`.

**Storage**: Browser `localStorage` (for non-authoritative preference tracking only); server-side cookie sessions via Gin.

**Testing**: Vitest (`bun test`) for frontend store and component unit tests; `go test` for Gin HTTP handlers.

**Target Platform**: Modern web browsers (macOS, iOS Safari, Android Chrome, desktop browsers).

**Project Type**: Web Application (Go backend embedding SvelteKit static SPA).

**Performance Goals**: Redirection executed within <100ms of navigation; zero unauthorized API bursts.

**Constraints**: Compliant with Abel Constitution v1.2.0 (Radical Simplicity, State Encapsulation, Explicit Functional Decomposition).

**Scale/Scope**: Administrative access control across 1 main dashboard view (`/admin`) and login view (`/login`).

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Compliance Assessment | Status |
| :--- | :--- | :---: |
| **I. Radical Simplicity & Code Minimization** | Minimizes lines by leveraging SvelteKit's native layout hierarchy and Gin's existing session middleware instead of adding heavy third-party auth guards. | **PASS** |
| **II. Streamlined & Maintainable Unit Tests** | Dedicated unit tests for `SystemStore`, `redirect.ts`, and `handlers_auth.go` without mock bloat. | **PASS** |
| **III. Converged Architecture & Cohesion** | Centralizes route guarding at the layout level (`admin/+layout.svelte`) and 401 interception in `api.ts`. | **PASS** |
| **IV. Hierarchical Domain Organization** | Preserves domain isolation between frontend utilities, routes, and backend web handlers. | **PASS** |
| **V. Zero Duplication (DRY)** | Single shared `resolveRedirect` utility and single `clearSession` routine. | **PASS** |
| **VI. Frequent Atomic Commits** | Discrete tasks will be committed atomically. | **PASS** |
| **VII. Explicit Functional Decomposition (from 007)** | Functions decomposed into small, single-responsibility units (`validateSession`, `clearSession`, `handleUnauthorizedRedirect`). | **PASS** |
| **VIII. Strict State Encapsulation (from 007)** | `SystemStore` fields (`isAuthenticated`, `sessionId`) are not directly modified externally; mutations are gated by explicit methods. | **PASS** |
| **IX. Auditable & Lock-Free Audio Pipeline** | Auth changes do not touch the audio streaming or recording paths. | **PASS** |

## Project Structure

### Documentation (this feature)

```text
specs/008-admin-unauth-redirect/
├── spec.md                 # Feature specification
├── plan.md                 # This file (/speckit-plan output)
├── research.md             # Architecture decisions and research
├── data-model.md           # Client and server session data models
├── quickstart.md           # Runnable validation steps
├── contracts/
│   └── api-contracts.md    # Session verification & redirect contracts
└── checklists/
    └── requirements.md     # Quality checklist
```

### Source Code (repository root)

```text
src/
├── backend/
│   └── lib/
│       └── web/
│           ├── handlers_auth.go       # Add GetSessionHandler (GET /api/auth/session)
│           ├── handlers_auth_test.go  # Unit test for session verification
│           └── router.go              # Wire GET /api/auth/session route
└── frontend/
    └── src/
        ├── lib/
        │   ├── audioState.svelte.ts   # Encapsulate auth state & add validateSession()
        │   ├── audioState.test.ts     # Store unit tests
        │   └── utils/
        │       ├── api.ts             # 401 response interceptor & auto-redirect
        │       ├── redirect.ts        # Open redirect sanitization
        │       └── redirect.test.ts   # Sanitizer tests
        └── routes/
            ├── admin/
            │   ├── +layout.svelte     # SvelteKit layout route guard
            │   └── +page.svelte       # Clean admin page component
            └── login/
                └── +page.svelte       # Feedback banner for ?reason=expired
```

**Structure Decision**: Standard web application structure following existing SvelteKit and Go layout conventions.

## Complexity Tracking

*No constitutional violations. Radical simplicity maintained.*
