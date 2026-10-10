# Feature Specification: Admin UI Unauthenticated Session Redirect

**Feature Branch**: `008-admin-unauth-redirect`

**Created**: 2026-10-10

**Status**: Ready

**Input**: User description: \"The Admin UI does not redirect to login when session is unauthenticated. This is an issue with frontend logic\"

## Clarifications

### Session 2026-10-10

- Q: How should the frontend guard the `/admin` route on initial page load to prevent flashing unauthorized content? → A: Implement a route guard at the SvelteKit layout level with a brief loading state while validating with backend session check.
- Q: When an active admin session expires or is revoked mid-use, how should the login page inform the user? → A: Display a clear banner/alert on the login page stating "Your session has expired. Please log in again." (via query param `?reason=expired`).
- Q: How do Constitution v1.2.0 principles from 007 apply to 008? → A: In accordance with Constitution Principles VII (Explicit Functional Decomposition) and VIII (State Encapsulation), frontend session state mutations must be strictly encapsulated inside `SystemStore` methods (`clearSession()`, `setAuthenticated()`, `validateSession()`), and route guarding logic must be decomposed into explicit, single-responsibility functions.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Direct Navigation to Admin When Unauthenticated (Priority: P1)

An unauthenticated visitor navigates directly to the administrative dashboard URL (`/admin`). The SvelteKit layout-level route guard intercepts the route before rendering protected components, performs a fast check against the session API, and immediately navigates the browser to `/login?redirect=/admin` with no flash of unauthenticated admin controls.

**Why this priority**: Unauthenticated users must never see administrative panels or encounter cryptic network failure states. This is the primary gatekeeper for the administrative user experience.

**Independent Test**:
Can be fully tested by opening a fresh browser context with no stored credentials, navigating directly to `/admin`, and verifying the URL transitions to `/login?redirect=/admin` before any admin UI components render.

**Acceptance Scenarios**:

1. **Given** an unauthenticated visitor with no active session, **When** they navigate to `/admin`, **Then** the layout guard blocks component rendering, displays a brief loading state if validating, and redirects to `/login?redirect=/admin`.
2. **Given** an unauthenticated visitor is redirected to `/login?redirect=/admin`, **When** the login page loads, **Then** the redirect destination is preserved for post-login return.

---

### User Story 2 - Stale LocalStorage Session Eviction and Redirect (Priority: P1)

A user previously logged in, leaving an old `session_id` in their browser's local storage, but their server session cookie has expired, been cleared, or been revoked. When navigating to `/admin`, the application validates the session against the server rather than assuming validity. Upon receiving an unauthorized response or recognizing the expired session, the client evicts the stale credentials from local storage, marks authentication as false, and redirects to `/login?redirect=/admin&reason=expired`.

**Why this priority**: Stale local storage credentials currently cause the client to falsely assume `isAuthenticated = true`, loading administrative panels that fail silently or error out with cascaded HTTP 401s.

**Independent Test**:
Can be tested by setting an invalid `session_id` in local storage, navigating to `/admin`, and verifying that the invalid session is pruned, the user is redirected to `/login?redirect=/admin&reason=expired`, and an alert banner informs them of expiration.

**Acceptance Scenarios**:

1. **Given** a browser with a stale or invalid `session_id` in local storage, **When** navigating to `/admin`, **Then** the system detects the session is unauthorized, clears local storage, and redirects to `/login?redirect=/admin&reason=expired`.
2. **Given** a user on the administrative interface whose session expires or is revoked on the backend, **When** an administrative API call or WebSocket connection returns `401 Unauthorized`, **Then** the application triggers session cleanup and routes the user to `/login?redirect=/admin&reason=expired`.
3. **Given** a user redirected with `reason=expired`, **When** the login page loads, **Then** a visible alert banner explains that the session has expired and requests re-authentication.

---

### User Story 3 - Post-Authentication Return to Target View (Priority: P2)

A user redirected from `/admin` to `/login?redirect=/admin` logs in with valid credentials. Upon successful authentication, the system inspects the `redirect` parameter, routes the user directly to `/admin`, and initializes the real-time admin view and WebSockets.

**Why this priority**: Preserves user workflow continuity so administrators do not have to re-navigate manually to the admin panel after authenticating.

**Independent Test**:
Navigate to `/login?redirect=/admin`, submit valid credentials, and verify automatic navigation to `/admin` with the administrative dashboard actively rendering.

**Acceptance Scenarios**:

1. **Given** a user is on `/login?redirect=/admin`, **When** valid admin credentials are submitted and verified, **Then** the application navigates to `/admin` rather than the default landing page.
2. **Given** an already-authenticated user navigates to `/login`, **When** the login view mounts, **Then** the application automatically forwards them to `/admin`.

---

### Edge Cases

- **Slow Network / Backend Unreachable**: If the server returns a temporary network error (`502`, `503`, `NetworkError`) rather than `401 Unauthorized`, the client must show a connection retry banner rather than prematurely destroying legitimate credentials.
- **Open Redirect Sanitization**: The `redirect` parameter must strictly require relative paths starting with a single `/` and reject protocol-relative URLs (`//...`) or scheme-prefixed URLs (`https://...`).
- **Concurrent API Invalidation**: If multiple admin requests fail with `401` in parallel, the client must trigger a single clean redirect rather than repeated redundant navigation calls.

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST implement an authentication route guard at the SvelteKit layout level to intercept unauthenticated access before admin components mount.
- **FR-002**: The system MUST NOT treat client `localStorage` presence of a `session_id` as definitive proof of authentication without server validation.
- **FR-003**: The system MUST intercept HTTP `401 Unauthorized` responses from any administrative endpoint, clear local authentication state via encapsulated store methods, and redirect to `/login?redirect=/admin&reason=expired`.
- **FR-004**: The system MUST preserve the intended destination in a sanitized `redirect` query parameter when routing unauthenticated users to `/login`.
- **FR-005**: The login page MUST display an informational alert banner when the `reason=expired` query parameter is present.
- **FR-006**: The system MUST navigate the user to the target specified in `redirect` upon successful authentication, falling back to `/admin` if unspecified.
- **FR-007**: The system MUST terminate administrative WebSocket connections and event subscriptions when authentication is revoked or invalid.
- **FR-008**: In accordance with Constitution v1.2.0 (Principles VII & VIII), frontend session state MUST be encapsulated in `SystemStore` with private internal mutations accessible only through explicit methods (`clearSession()`, `setAuthenticated()`, `validateSession()`).

### Key Entities

- **Session State**: Encapsulates whether the current user holds a verified active administrative session (`isAuthenticated`), the active identifier (`sessionId`), and the username.
- **Authentication Route Guard**: Frontend navigation handler located at the layout level controlling access to protected routes based on verified session state.

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of unauthenticated visits to `/admin` redirect to `/login?redirect=/admin` with zero flash of admin components.
- **SC-002**: 0 unauthorized API error cascades rendered to users attempting to access administrative views without credentials.
- **SC-003**: 100% of stale local storage sessions are purged upon encountering a `401 Unauthorized` API or WebSocket handshake failure.
- **SC-004**: 100% of successful logins with a valid `redirect` query parameter return to the requested destination.
- **SC-005**: 100% of mid-session expirations redirect with `reason=expired` and render the session expiration banner on `/login`.

---

## Assumptions

- The backend `/api/auth/session` endpoint remains the authoritative check for session cookie validity.
- Modern browser navigation (`goto` from SvelteKit) handles client-side route transitions without requiring a hard window reload.
- The default protected route is `/admin`.
