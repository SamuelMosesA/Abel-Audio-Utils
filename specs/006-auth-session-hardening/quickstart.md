# Quickstart Validation Guide: Admin Authentication Hardening

**Branch**: `006-auth-session-hardening`
**Date**: 2026-10-10

## Validation Scenarios

### 1. Dynamic Session Key Generation & Restart Continuity
Verify that a secure random key is created and that sessions persist across daemon restarts:
```bash
# Run backend auth tests including key generation and restart simulation
go test -v ./src/backend/lib/web -run TestSessionKey
```
Expected:
- PASS: `session.key` is created with `0600` permissions.
- PASS: Session cookie created before simulated restart continues to authenticate after restart.
- PASS: Cookie signed with default `"secret"` is rejected with 401.

### 2. Server-Side Session Revocation
Verify that logging out revokes the session on the server:
```bash
go test -v ./src/backend/lib/web -run TestSessionRevocation
```
Expected:
- PASS: `DELETE /api/auth/session` successfully revokes session.
- PASS: Replayed session cookie is rejected with HTTP 401.

### 3. Frontend Route Guard Verification
Verify that accessing `/admin` unauthenticated redirects to `/login?redirect=/admin`:
```bash
cd src/frontend
bun run test:unit
```
Expected:
- All unit tests pass, confirming auth state management and routing redirection.

### 4. Browser End-to-End Walkthrough
1. In a private/incognito window, open `http://localhost:5173/admin`.
2. Verify browser immediately redirects to `http://localhost:5173/login?redirect=/admin`.
3. Enter administrator credentials (`admin` / password).
4. Verify browser redirects into `http://localhost:5173/admin`.
5. Restart the backend daemon: `go run src/backend/main.go`.
6. Refresh the browser page on `/admin`: verify user is STILL authenticated.
7. Click Logout in the user interface.
8. Verify user is redirected to `/login`.
9. Hit the browser "Back" button: verify `/admin` redirects immediately back to `/login`.
