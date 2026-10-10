# Research: UI Controls, Per-Language AI Killswitch, and DSP Refactor

**Branch**: `005-ui-controls-and-dsp-refactor`
**Date**: 2026-10-10

## Decision 1: Per-Language AI Killswitch State Management

- **Decision**: Store the blocked languages state in-memory inside `state.AIConfig` in backend `AppState`, protected by RWMutex, mirroring the existing `Enabled` master AI flag.
- **Rationale**:
  - The user explicitly requested: "The killswithc should be like the current master AI switch. Just that it is per language. So the state is runtime stored in the backend appstate".
  - This avoids unnecessary disk I/O, file locks, or YAML schema serialization complexity for runtime state.
  - Aligns with Constitution Principle I (Radical Simplicity) and Principle III (Converged Architecture).
- **Alternatives Considered**:
  - Persisting to `config.yaml`: Rejected because `config.yaml` is human-edited static configuration and modifying it dynamically introduces format corruption risk and comments loss.
  - Persisting to separate `.state.json`: Rejected because the user explicitly specified runtime storage in backend AppState like the master AI switch.

## Decision 2: Recording Library On-Demand Refresh Pattern

- **Decision**: Remove the 2000ms `setInterval` polling in `RecordingList.svelte` and provide an explicit manual Refresh button on the recording panel toolbar next to existing controls.
- **Rationale**:
  - Continuous 2-second background polling creates needless network traffic and server log noise, and causes UI jumps or disruptions during file trimming/renaming.
  - Svelte 5 runes (`$state`) in `audioState.svelte.ts` make on-demand re-fetching immediate and clean when the Refresh button is clicked.
- **Alternatives Considered**:
  - WebSockets push for file changes: Rejected as overly complex (Constitution Principle I) when church audio operators only record once or twice a week and an on-demand refresh button completely fulfills the need.

## Decision 3: Audio DSP Stereo Extraction and Boost Function Signature

- **Decision**: Place `ExtractStereoChunk` inside `abel/src/backend/lib/audioengine/conversion`:
  ```go
  func ExtractStereoChunk(in []float32, bufferSize int, openedChannels int, chL int, chR int, boost float32) []float32
  ```
- **Rationale**:
  - Consolidates channel extraction, digital amplification, and hard clamping `[-1.0, 1.0]` into a pure, thread-safe function in the existing `conversion` package.
  - Directly replaces lines 283–311 in `engine.go`.
  - Ensures boost `<= 0.0` safely defaults to unity gain (`1.0`).
- **Alternatives Considered**:
  - Separate `ApplyGainAndClamp` requiring two passes (extracting channels first, then applying gain): Rejected because a single pass avoids allocating intermediate slices and minimizes CPU overhead in the 48kHz audio capture thread.

## Decision 4: Translation Listener Counting

- **Decision**: Expose active listener counts per language through `Translator` interface in `realtime_base.go` / `translation.go`.
  - Number of listeners = count of active SSE subtitle subscribers (`len(subscribers[lang])`) + active HLS stream consumers (if streaming).
  - Return each configured language in `GET /api/ai/streams` with its name, code, blocked status, active session status, and listener count.
- **Rationale**:
  - The UI needs to display all configured languages by default, not only languages with active OpenAI Realtime API sessions.
  - Allows administrators to see listener demand before or during translation sessions and toggle specific language blocks.
- **Alternatives Considered**:
  - Separate polling endpoint `/api/ai/metrics`: Rejected to avoid unnecessary endpoint proliferation; `GET /api/ai/streams` already returns translation stream statuses and can return the enriched language list directly.
