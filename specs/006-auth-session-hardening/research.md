# Research: Admin Authentication Hardening, Dynamic Session Secrets, Server-Side Revocation, and Login Redirects

**Branch**: `006-auth-session-hardening`
**Date**: 2026-10-10

## Decision 1: Dynamic Session Secret Generation and Storage Location

- **Decision**: Store the dynamically generated 32-byte (256-bit) session signing secret in a dedicated file `session.key` located in the same directory as `config.yaml` (defaulting to `~/.config/abel/session.key`), with file mode `0600` (read/write only by the user). Allow overriding via `session_secret` in `config.yaml`.
- **Rationale**:
  - The secret is unique to the installation, eliminating the security vulnerability of the hardcoded `"secret"` string.
  - By persisting the key to disk with strict `0600` permissions, administrator sessions survive server restarts and audio engine re-initializations without requiring the operator to re-authenticate.
  - Placing `session.key` adjacent to `config.yaml` adheres to Principle III (Converged Architecture) and ensures it resides in a protected user configuration path.
- **Alternatives Considered**:
  - Ephemeral in-memory key generated on each startup: Rejected because it forces a logout on every server restart, conflicting with the user requirement that sessions survive daemon reboots.
  - Storing the secret directly in `config.yaml`: Rejected because modifying user-edited YAML programmatically risks destroying user formatting, comments, and structure.

## Decision 2: Server-Side Session Revocation Mechanism

- **Decision**: Track revoked session IDs in thread-safe runtime state (`sync.Map`) on `AppState`.
- **Rationale**:
  - Abel is a single-instance audio utility server.
  - Tracking revoked session IDs in `sync.Map` provides O(1) concurrent lookups in `SessionAuthMiddleware` with zero disk latency or external database dependencies.
  - When a user logs out (`DELETE /api/auth/session` or `POST /api/auth/logout`), the session ID is added to the revoked map and the cookie is cleared.
- **Alternatives Considered**:
  - Database table (SQLite) for sessions: Rejected as unnecessary complexity (Constitution Principle I: Radical Simplicity) given that Abel's cookie store is already signed and client-side, and active revoked token volumes are minimal.

## Decision 3: Client-Side and SvelteKit Routing Guards

- **Decision**: Implement both URL-level route guarding and return-path redirection (`/login?redirect=/admin`).
  - In `src/frontend/src/routes/admin/+page.svelte`: If `!system.isAuthenticated`, call `goto('/login?redirect=/admin')`.
  - In `src/frontend/src/routes/login/+page.svelte` and `LoginView.svelte`: Upon successful login, parse `page.url.searchParams.get("redirect") || "/admin"` and navigate there.
  - In `SystemStore.logout()`: Send a `DELETE /api/auth/session` request to the server, then clear `localStorage` and navigate to `/login`.
- **Rationale**:
  - Provides instant feedback to users without exposing unrendered admin components.
  - Preserves intended deep-link navigation across login flows.
