# Data Model: UI Controls, Per-Language AI Killswitch, and DSP Refactor

**Branch**: `005-ui-controls-and-dsp-refactor`
**Date**: 2026-10-10

## Entities and State

### 1. AIConfig (Runtime Backend State in `state.AIConfig`)

Represents the global AI translation runtime state and per-language blocked states.

```go
type AIConfig struct {
    Enabled          bool            // Master AI toggle
    BlockedLanguages map[string]bool // map of lowercase language code -> blocked boolean
}
```

#### Invariants & Operations:
- `IsEnabled() bool`: Returns true if master AI is enabled.
- `SetEnabled(bool)`: Updates master AI state.
- `IsBlocked(code string) bool`: Returns true if specific language code is blocked.
- `SetBlocked(code string, blocked bool)`: Updates blocked state for given language code (stores normalized lowercase code).
- Thread safety: Protected by `sync.RWMutex` inside `state.AppState`.

---

### 2. LanguageStatus (API Response Model)

Returned in `GET /api/ai/streams` to represent each configured language's status and metrics.

```typescript
interface LanguageStatus {
    code: string;       // e.g. "es", "fr", "de"
    name: string;       // e.g. "Spanish", "French", "German"
    blocked: boolean;   // true if killswitch is active
    active: boolean;    // true if live OpenAI session is running
    listeners: number;  // active listener count (subtitles + audio)
}
```

```go
type LanguageStatus struct {
    Code      string `json:"code"`
    Name      string `json:"name"`
    Blocked   bool   `json:"blocked"`
    Active    bool   `json:"active"`
    Listeners int    `json:"listeners"`
}
```

---

### 3. AIStreamsResponse (API Response for `GET /api/ai/streams`)

```typescript
interface AIStreamsResponse {
    masterEnabled: boolean;
    languages: LanguageStatus[];
    sessions: SessionInfo[]; // Retained for backward compatibility
}
```

---

### 4. StereoChunk (Audio DSP Buffer)

Interleaved 32-bit float audio slice of size `bufferSize * 2`.

```go
// Output of conversion.ExtractStereoChunk
// Index 2*i: Left channel sample clamped to [-1.0, 1.0]
// Index 2*i + 1: Right channel sample clamped to [-1.0, 1.0]
type StereoChunk = []float32
```
