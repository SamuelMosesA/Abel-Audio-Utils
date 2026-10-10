# API Contracts: Admin Authentication Hardening

**Branch**: `006-auth-session-hardening`
**Date**: 2026-10-10

## 1. Authentication Endpoints

### POST `/api/auth/session` (Login)

Establishes an authenticated session.

#### Request:
- Headers: `Content-Type: application/json`
- Body:
```json
{
  "username": "admin",
  "password": "secretpassword"
}
```

#### Response:
- Status: `200 OK`
- Headers: `Set-Cookie: abel_session=...; Path=/; Max-Age=604800; HttpOnly; SameSite=Lax`
- Body:
```json
{
  "status": "success",
  "session": "9e1c2d3f4a5b6c7d8e9f0a1b2c3d4e5f"
}
```
- Status: `401 Unauthorized`
```json
{
  "error": "Invalid username or password"
}
```

---

### DELETE `/api/auth/session` (Logout / Revocation)

Terminates an authenticated session and revokes the session ID server-side.

#### Request:
- Headers: `Cookie: abel_session=...`

#### Response:
- Status: `200 OK`
- Headers: `Set-Cookie: abel_session=; Path=/; Max-Age=-1; HttpOnly; SameSite=Lax`
- Body:
```json
{
  "status": "logged_out"
}
```

---

## 2. Protected Admin Middleware Contract

### `SessionAuthMiddleware(appState *state.AppState)`

Applied to all admin routes (`/api/recordings/upload`, `/api/recordings/process`, `/api/audio/restart`, `/api/audio/config`, `/api/ai/streams`, etc.).

#### Verification Steps:
1. Decode cookie session `abel_session`.
2. Verify HMAC signature against active installation `session.key`.
3. Check `session.Get("authenticated") == true`.
4. Extract `session_id`. If empty or `appState.IsSessionRevoked(session_id) == true`, return:
   - Status: `401 Unauthorized`
   - Body: `{"error": "Unauthorized session"}`
5. Otherwise, call `c.Next()`.

---

## 3. Frontend Routing Contract

### Route: `/admin`
- Unauthenticated access:
  - Dispatches redirect to `/login?redirect=/admin`.
- Authenticated access:
  - Renders `AudioAdminView`.

### Route: `/login`
- Query parameter: `?redirect=/admin` (optional target path).
- On login success: Navigates to `redirect` param (or default `/admin`).
