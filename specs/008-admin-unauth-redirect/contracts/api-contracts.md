# API Contracts: Session Authentication & Redirects

**Branch**: `008-admin-unauth-redirect` | **Date**: 2026-10-10 | **Spec**: [spec.md](../spec.md)

## Endpoints

### 1. Session Status Verification

Retrieve the status of the currently authenticated session based on the incoming session cookie.

```http
GET /api/auth/session
```

#### Headers
- `Cookie: <session-cookie>` (Credentials: `include`)

#### Responses

##### 200 OK
Session is active and valid.
```json
{
  "status": "authenticated",
  "username": "admin",
  "session": "9f7b189a03bc21e4"
}
```

##### 401 Unauthorized
Session cookie is missing, invalid, or revoked.
```json
{
  "error": "Unauthorized session"
}
```

---

### 2. Session Revocation / Logout

Explicit logout request. Revokes the session identifier server-side and clears the cookie.

```http
DELETE /api/auth/session
```

#### Responses

##### 200 OK
```json
{
  "status": "logged_out"
}
```

---

### 3. Route Guard URL Contracts

#### Login Route Query Parameters
```
GET /login?redirect=/admin&reason=expired
```

| Parameter | Allowed Values | Effect |
| :--- | :--- | :--- |
| `redirect` | Any path matching `/^\/[^\/\\].*$/` | On successful login, user is forwarded to this path instead of `/admin` |
| `reason` | `expired` | Login view displays alert banner: `"Your session has expired. Please log in again."` |
| `reason` | `unauthorized` | Login view displays alert banner: `"Please log in to access this area."` |
