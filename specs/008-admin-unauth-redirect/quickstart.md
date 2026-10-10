# Quickstart: Validating Admin UI Unauthenticated Session Redirect

**Branch**: `008-admin-unauth-redirect` | **Date**: 2026-10-10 | **Spec**: [spec.md](./spec.md)

This guide provides runnable scenarios to verify that unauthenticated visitors are prevented from viewing `/admin`, stale credentials are automatically evicted, and sessions are properly validated.

---

## Prerequisites

- Go 1.23+ installed.
- Bun / Node installed for frontend assets.
- Working directory: Repository root (`/home/samuelmoses/Workspace/Church/Abel-Audio-Utils`).

---

## Automated Validation

### 1. Frontend Unit & Component Tests

Run Vitest suite to verify route guarding, store encapsulation, and redirect sanitization:

```bash
cd src/frontend
bun test src/lib/audioState.test.ts src/lib/utils/redirect.test.ts
```

**Expected Outcome**:
- `resolveRedirect` correctly rejects open redirects (`//evil.com`, `http://...`) and accepts valid relative paths (`/admin`, `/admin/settings`).
- `SystemStore` initializes `isAuthenticated = false` even if `localStorage` has a string, pending explicit validation.
- `clearSession()` purges `session_id`, `admin_user`, terminates SSE/WS, and resets state.

### 2. Backend Auth Session Endpoint Tests

Run Go API tests to verify `GET /api/auth/session` behavior:

```bash
go test -v -run TestAuthSession ./src/backend/lib/web
```

**Expected Outcome**:
- Unauthenticated request returns `401 Unauthorized` with `{"error": "Unauthorized session"}`.
- Authenticated request with valid cookie returns `200 OK` with session details.
- Request with revoked session returns `401 Unauthorized`.

---

## Manual End-to-End Scenarios

### Scenario 1: Direct Unauthenticated Access
1. Open an incognito browser window.
2. Navigate directly to `http://localhost:8080/admin`.
3. **Verify**:
   - URL immediately transitions to `http://localhost:8080/login?redirect=/admin`.
   - No administrative components, telemetry cards, or controls flash on screen.

### Scenario 2: Stale LocalStorage Eviction
1. In browser DevTools Console on `/login`, set `localStorage.setItem("session_id", "invalid-token-1234")`.
2. Navigate to `http://localhost:8080/admin`.
3. **Verify**:
   - The route guard recognizes the invalid session via `GET /api/auth/session`.
   - `localStorage.getItem("session_id")` is removed.
   - User is redirected to `/login?redirect=/admin&reason=expired`.
   - An alert banner displays: *"Your session has expired. Please log in again."*

### Scenario 3: Login and Intended Destination Return
1. On `/login?redirect=/admin`, submit valid administrative credentials.
2. **Verify**:
   - Successful authentication immediately forwards the user to `/admin`.
   - Admin console renders cleanly and establishes real-time connections.
