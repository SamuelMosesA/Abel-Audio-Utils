# Test Verification Protocol: Ensuring Zero Regressions

## 1. Backend Verification Matrix

Since backend changes must not break any existing contracts, the following verification layers will be enforced:

| Subsystem | Risk Area | Verification Technique | Target Invariant |
| :--- | :--- | :--- | :--- |
| **Audio Conversion** | PCM clamping, WAV headers, downsampling | Static AST analysis & conversion unit parity | Float32 values clipped to [-1.0, 1.0]; exactly 44-byte WAV headers; 48kHz -> 24kHz ratio |
| **Web API Handlers** | Gin route registrations & session auth | Signature & contract preservation in `router.go` | All 22 REST/WS endpoints retain identical HTTP methods, query params, and JSON structures |
| **Recording Processing** | MP3 encoding, trim operations, cloud sync | Parameter normalization & execution guard consolidation | Output filenames (`.mp3`, `.wav`), status fields (`idle`, `processing`, `completed`) unchanged |
| **OpenAI Realtime** | WebSocket disconnects, reconnect timers | Delegation to `realtime_base.go` | Identical token telemetry metrics and event routing for deltas & transcriptions |

---

## 2. Frontend Verification Matrix

Every frontend simplification must pass 3 independent gates before completion:

```bash
# Gate 1: Unit & Component Behavioral Tests
cd src/frontend && bun run test:unit

# Gate 2: SvelteKit Type Checking & Template Syntax
cd src/frontend && bun run check

# Gate 3: Production Client & Static Server Build
cd src/frontend && bun run build
```

| Component | Simplifications | Verification Check |
| :--- | :--- | :--- |
| `AudioPlayer.svelte` | Trimmed playback controls, cleaner CSS | `src/lib/components/admin/AudioPlayer.test.ts` (100% pass) |
| `LiveAudioPlayer.svelte` | Reduced retry markup & error bindings | `src/lib/components/audio/LiveAudioPlayer.test.ts` (100% pass) |
| `audioState.svelte.ts` | Lean reactive state runes | `src/lib/audioState.test.ts` (100% pass) |
| `audioVisuals.svelte.ts`| Simplified peak meter maths | `src/lib/audioVisuals.test.ts` (100% pass) |
| `timecode.ts` | Clean formatting helper | `src/lib/utils/timecode.test.ts` (100% pass) |
