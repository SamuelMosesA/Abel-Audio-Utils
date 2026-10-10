# Research: Admin UI Unauthenticated Session Redirect

**Branch**: `008-admin-unauth-redirect` | **Date**: 2026-10-10 | **Spec**: [spec.md](./spec.md)

## Research Topics & Decisions

### 1. Route Guarding Strategy in SvelteKit

#### Context
Visitors navigating to `/admin` without an active session currently encounter a brief flash of admin UI or broken cards because `+page.svelte` only checked `system.isAuthenticated` reactively inside an `$effect` after the component began rendering, and `SystemStore` erroneously treated `localStorage.getItem("session_id")` as proof of authentication.

#### Decision
Introduce a dedicated SvelteKit layout guard at `src/frontend/src/routes/admin/+layout.svelte` (or `+layout.ts`).
- When entering `/admin`, the layout checks `system.validateSession()`.
- While validating, render a clean, minimal loading spinner (`<AdminGuard>`).
- If unauthenticated, immediately perform client-side redirection via `goto('/login?redirect=/admin')` without ever mounting `AudioAdminView`.
- If authenticated, render the slot/children.

#### Rationale
- **Prevention of FOUC**: Placing the gatekeeper in the layout prevents admin cards from initializing WebSockets, fetching sensitive audio configs, or flashing in the DOM.
- **Hierarchical Domain Security**: Any future admin child routes (e.g. `/admin/recordings`, `/admin/settings`) inherit the guard automatically without duplicating auth checks.

#### Alternatives Considered
- *In-page `$effect` only*: Current approach; causes race conditions, visual flashes, and fails when `localStorage` has a stale key.
- *Server-side SvelteKit hooks (`hooks.server.ts`)*: SvelteKit in this project is built as a static/client SPA embedded in the Go binary (`adapter-static`), so SvelteKit server hooks do not run on navigation.

---

### 2. Session Validation Endpoint (`GET /api/auth/session`)

#### Context
The backend has `POST /api/auth/session` (login) and `DELETE /api/auth/session` (logout), but lacks a lightweight, non-mutating `GET /api/auth/session` endpoint to verify the active cookie against the backend session store and revocation list.

#### Decision
Add `GET /api/auth/session` to the backend router protected by `SessionAuthMiddleware(appState)`.
- If the session cookie is valid and not revoked: Returns `200 OK` with JSON:
  ```json
  {
    "status": "authenticated",
    "username": "admin",
    "session": "a1b2c3d4..."
  }
  ```
- If absent, invalid, or revoked: `SessionAuthMiddleware` halts execution and returns `401 Unauthorized` with JSON:
  ```json
  {
    "error": "Unauthorized session"
  }
  ```

#### Rationale
- Radical Simplicity: Reuses the existing `SessionAuthMiddleware` without writing custom session inspection logic.
- Definitive Server Truth: Completely decouples authentication state from unverified browser `localStorage`.

---

### 3. API Interceptor & Stale Credential Eviction

#### Context
When an admin session expires or is revoked on the backend while the user is actively viewing the console, subsequent background API calls (`/api/audio/config`, `/api/recordings`, etc.) return `401 Unauthorized`. Currently, `fetchWithSync` in `src/frontend/src/lib/utils/api.ts` only logs a warning.

#### Decision
In `fetchWithSync`:
- When `response.status === 401`, trigger `system.clearSession()`.
- If currently on an administrative path (e.g., `/admin`), trigger redirection to `/login?redirect=${encodeURIComponent(currentPath)}&reason=expired`.
- Suppress duplicate redirects if multiple concurrent requests return 401 using a simple redirect debounce/mutex flag.

#### Rationale
- Adheres to Constitution Principle VII (Explicit Functional Decomposition) and Principle VIII (Strict State Encapsulation).
- Guarantees immediate session cleanup across both state stores and browser storage.

---

### 4. Open Redirect Prevention

#### Context
Query parameter `redirect` can be abused if not strictly sanitized (e.g. `?redirect=//malicious-site.com` or `?redirect=javascript:...`).

#### Decision
Enhance `src/frontend/src/lib/utils/redirect.ts` with strict sanitization:
- Must start with a single `/`.
- Must NOT start with `//` or `/\\`.
- Rejects colon `:` characters before any slash to prevent protocol injection.
- Falls back safely to `/admin`.

#### Rationale
Eliminates open redirect vulnerabilities per OWASP best practices.
