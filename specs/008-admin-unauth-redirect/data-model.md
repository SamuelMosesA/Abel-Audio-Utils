# Data Model: Admin UI Unauthenticated Session Redirect

**Branch**: `008-admin-unauth-redirect` | **Date**: 2026-10-10 | **Spec**: [spec.md](./spec.md)

## Entities

### 1. Client Session State (`SystemStore`)

Represents the client-side authentication posture encapsulated within `src/frontend/src/lib/audioState.svelte.ts`.

| Field | Type | Description | Visibility / Mutability |
| :--- | :--- | :--- | :--- |
| `isAuthenticated` | `boolean` | Whether an active verified server session exists | Getter only (`$state`), mutated exclusively via store methods |
| `isValidating` | `boolean` | Whether initial or background session validation is in progress | Getter only (`$state`), mutated during validation lifecycle |
| `sessionId` | `string` | Cryptographic session identifier returned by backend | Read-only state |
| `adminUser` | `string` | Logged-in admin username | Read-only state |

#### State Transitions:
```
[Uninitialized]
       │
       ▼ (validateSession)
[Validating] ──(200 OK)───────► [Authenticated]
       │                               │
       │ (401 / Network Error)         │ (401 Intercept / Logout)
       ▼                               ▼
[Unauthenticated] ◄────────────────────┘
  (clearSession: purge localStorage, close WS/SSE, redirect to /login)
```

---

### 2. Session Validation Contract (`SessionStatus`)

Backend entity serialized over HTTP for `GET /api/auth/session`.

```typescript
export interface SessionStatus {
    status: "authenticated";
    username: string;
    session: string;
}

export interface SessionError {
    error: string;
}
```

---

### 3. Redirection Parameters (`RedirectContext`)

URL search query parameters processed by the login route guard and redirect helpers.

| Parameter | Type | Validation Rule | Purpose |
| :--- | :--- | :--- | :--- |
| `redirect` | `string` | Must begin with single `/`, cannot contain `//` or protocol scheme | Preserves target path across authentication boundary |
| `reason` | `string` | Enum: `"expired"` \| `"unauthorized"` \| undefined | Triggers contextual feedback banner on login page |
