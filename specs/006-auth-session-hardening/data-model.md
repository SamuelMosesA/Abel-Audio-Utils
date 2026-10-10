# Data Model: Admin Authentication Hardening

**Branch**: `006-auth-session-hardening`
**Date**: 2026-10-10

## Entities and State

### 1. SessionKey

Represents the cryptographic signing key for Gin session cookies.

- **Length**: 32 bytes (256-bit binary or hex).
- **Storage**:
  - Primary: `config.Config.SessionSecret` (if defined in `config.yaml`).
  - File: `session.key` adjacent to `config.yaml` (`0600` permissions).
- **Generation**: Generated using `crypto/rand.Read`.

---

### 2. RevokedSessions Registry (Runtime State in `state.AppState`)

Tracks invalidated session identifiers to prevent cookie replay attacks after logout.

```go
// Inside AppState
RevokedSessions sync.Map // map[string]time.Time (sessionId -> revokedAt)
```

#### Operations:
- `RevokeSession(sessionID string)`: Stores `sessionID` with current timestamp.
- `IsSessionRevoked(sessionID string) bool`: Returns `true` if `sessionID` is present in `RevokedSessions`.

---

### 3. AuthSession (Cookie Payload)

The existing signed cookie structure managed by `gin-contrib/sessions`:

```go
session.Set("authenticated", true)
session.Set("username", "admin")
session.Set("session_id", "4a7f9b2c...") // 32-char hex string
```

#### Invariants:
- `authenticated`: Must be boolean `true`.
- `session_id`: Must be non-empty and NOT present in `RevokedSessions`.
- `MaxAge`: 7 days (604,800 seconds).

---

### 4. Client-Side Auth State (`SystemStore`)

Managed by Svelte 5 runes in `src/frontend/src/lib/audioState.svelte.ts`:

```typescript
class SystemStore {
    isAuthenticated = $state(false);
    sessionId = $state("");
    // Persisted in localStorage: "session_id", "admin_user"
}
```
