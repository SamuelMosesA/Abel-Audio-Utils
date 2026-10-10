# Feature Specification: Admin Authentication Hardening, Dynamic Session Secrets, Server-Side Revocation, and Login Redirects

**Feature Branch**: `006-auth-session-hardening`

**Created**: 2026-10-10

**Status**: Draft

**Input**: User description: "around authentication (for example: dynamic secret key generation, server-side session revocation. And also if the admin page is accessed unauthenticated, it should rediredt to the login page"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Automatic Unauthenticated Admin Page Redirect (Priority: P1)

An administrator or unauthenticated visitor navigates directly to the `/admin` URL in their browser (or their session has expired or been revoked). Rather than displaying a partially rendered admin console, broken widgets, or relying solely on API call failures, the web application immediately detects the unauthenticated state and redirects the browser to `/login` with an optional return path. Once the user enters valid credentials on `/login`, they are redirected back to `/admin`.

**Why this priority**: Unauthenticated users visiting `/admin` should have a clean, intuitive user journey and clear login prompt rather than seeing empty controls or raw authentication failure alerts.

**Independent Test**: Open a clean browser or private browsing window without active session cookies, navigate to `/admin`, and verify that the browser is immediately redirected to `/login`. Log in, and verify that the user lands on `/admin`.

**Acceptance Scenarios**:

1. **Given** a user without an active or authenticated session, **When** they navigate to `/admin`, **Then** the application immediately redirects them to `/login?redirect=/admin`.
2. **Given** a user on `/login` after an automated redirect, **When** they submit valid credentials, **Then** they are logged in and redirected back to `/admin`.
3. **Given** an authenticated administrator actively on `/admin`, **When** their session is revoked or cleared, **Then** subsequent navigation or actions redirect them to `/login`.

---

### User Story 2 - Dynamic Cryptographic Session Secret Generation & Persistence (Priority: P1)

A systems administrator running Abel Audio Utils needs the session cookies to be signed with a strong, installation-specific cryptographic key rather than a static, publicly known fallback string (`"secret"`). When the backend starts up, it generates a high-entropy 256-bit session key if none exists, and saves it to a protected local key file (`~/.config/abel/session.key` with `0600` permissions) or loads an administrator-specified key from configuration. When the server restarts, it loads the persisted installation key, ensuring existing valid sessions survive server reboots without exposing a default key to forgery attacks.

**Why this priority**: Hardcoded session signing keys allow attackers to forge valid admin session cookies without credentials. Dynamic keys with local persistence prevent forgery while maintaining session continuity across daemon restarts.

**Independent Test**: Start the server without an existing key file, verify that a secure 256-bit key is generated and stored with `0600` permissions. Log in to acquire a session cookie. Restart the server, issue an authenticated request using the same cookie, and verify that authentication succeeds without error. Verify that cookies signed with `"secret"` are rejected.

**Acceptance Scenarios**:

1. **Given** Abel starts for the first time without an existing session key file, **When** the server initializes, **Then** it generates a 256-bit random cryptographic key and stores it in `~/.config/abel/session.key` with file mode `0600`.
2. **Given** an administrator who is logged in and holds an active session cookie, **When** the backend server restarts, **Then** the server reloads the persisted session key and continues to accept the valid session cookie.
3. **Given** an attacker who crafts an HTTP request with an `abel_session` cookie signed using the default `"secret"` string, **When** sent to protected admin endpoints, **Then** the server rejects the request with HTTP 401 Unauthorized.

---

### User Story 3 - Server-Side Session Revocation and Immediate Invalidation (Priority: P2)

An administrator logging out of the console needs the session to be invalidated server-side, not just deleted from client-side browser storage. When the user logs out or an operator revokes an active session ID, the server adds the session ID to a revoked sessions blacklist in runtime state, clears the cookie header, and rejects all subsequent requests presenting that session token, preventing replayed or copied cookies from continuing to access admin endpoints.

**Why this priority**: Client-side cookie deletion alone does not prevent cookie replay if the cookie was copied, intercepted, or retained in another client tab. Server-side revocation provides true session termination.

**Independent Test**: Log in to obtain a valid session cookie, execute a successful admin request, call the logout endpoint, and then attempt another admin request reusing the logged-out session cookie. Verify that the subsequent request returns HTTP 401 Unauthorized.

**Acceptance Scenarios**:

1. **Given** an authenticated administrator, **When** they click "Logout" in the web console, **Then** the client sends a session termination request to `DELETE /api/auth/session` (or `POST /api/auth/logout`), the server marks that session ID as revoked, and clears the session cookie.
2. **Given** a revoked session ID, **When** a client sends an HTTP request presenting that session cookie to any protected `/api/...` route, **Then** the server rejects the request with HTTP 401 Unauthorized.
3. **Given** a user who has logged out, **When** viewing the UI, **Then** the frontend clears local authentication state and routes the user back to `/login`.

---

### Edge Cases

- **Corrupted or unreadable session key file**: If `session.key` exists but is empty or corrupted, log a warning, generate a new valid 256-bit key, and overwrite the file securely, gracefully invalidating previous sessions.
- **Configured session secret override**: If an explicit `session_secret` is defined in `config.yaml`, the server uses that secret directly instead of generating or loading `session.key`.
- **Concurrent requests during logout**: If requests arrive in-flight during logout, once the session ID is added to the revoked list, all concurrent and subsequent requests fail with 401.
- **Client storage cleared while server session remains**: If a client clears `localStorage` without hitting the logout endpoint, navigating to `/admin` redirects them to `/login`; logging in again issues a fresh session ID.
- **Server restart with revoked sessions in memory**: Revoked session IDs held in memory are cleared on restart; however, sessions have an absolute max age (7 days) and installation key protection.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The frontend router and layout guard MUST automatically redirect unauthenticated visits to `/admin` to `/login` with a `redirect` query parameter.
- **FR-002**: Upon successful login, the frontend MUST navigate to the URL specified by the `redirect` query parameter, defaulting to `/admin`.
- **FR-003**: The backend server MUST generate a cryptographically secure 256-bit (32-byte) random session signing key if no key is configured or persisted.
- **FR-004**: The generated session key MUST be saved to the local configuration directory (`~/.config/abel/session.key`) with restricted file permissions (`0600`) to persist across server restarts.
- **FR-005**: If `session_secret` is explicitly configured in `config.yaml`, the server MUST use the configured value as the signing key.
- **FR-006**: The backend MUST provide a logout / session termination endpoint (`DELETE /api/auth/session` or `POST /api/auth/logout`) that clears the session cookie and registers the session ID as revoked.
- **FR-007**: The backend `SessionAuthMiddleware` MUST check whether the session ID extracted from the cookie has been revoked, rejecting revoked sessions with HTTP 401 Unauthorized.
- **FR-008**: The frontend `logout()` function in `audioState.svelte.ts` MUST call the backend session termination endpoint before clearing local storage and updating view state.

### Key Entities *(include if feature involves data)*

- **SessionKey**: The 256-bit cryptographic signing secret used by the Gin cookie session store, loaded from `config.yaml` or `~/.config/abel/session.key`.
- **RevokedSession**: In-memory registry tracking invalidated session IDs to enforce immediate server-side revocation.
- **AuthSession**: The client session containing `authenticated` (bool), `username` (string), and `session_id` (string UUID/hex).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of direct unauthenticated browser visits to `/admin` redirect to `/login` without exposing admin interface panels.
- **SC-002**: Zero occurrences of static hardcoded secrets (such as `"secret"`) in session cookie signing.
- **SC-003**: Existing valid admin sessions continue to authenticate successfully after backend process restarts without prompting the user to re-login.
- **SC-004**: 100% of requests presenting a revoked session cookie are rejected with HTTP 401 Unauthorized within 1ms of revocation.
- **SC-005**: 100% test pass rate across backend Go test suite and frontend unit test suite.

## Assumptions

- The application runs in environments with access to a writable user configuration directory (`~/.config/abel/` or the directory containing `config.yaml`).
- Revoked session IDs can be maintained in thread-safe in-memory state (`sync.Map`) as Abel is a single-instance audio utility server.
- Session cookie lifetime remains up to 7 days (`MaxAge: 86400 * 7`) unless explicitly revoked via logout.
