# Implementation Plan: Audio Core Pipeline, OpenAI Managers, and Network Refactor

**Branch**: `007-audio-core-and-system-refactor` | **Date**: 2026-10-10 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/007-audio-core-and-system-refactor/spec.md`

## Summary

This feature executes a comprehensive architectural consolidation and refactor across five core operational areas:
1. **Audio Engine Core**: Consolidates WAV, HLS, and PCM clamping/conversion into a dedicated `audio_processing` package, removes inner-loop mutex locking in the single-threaded storage worker, and simplifies the engine-to-recording path for maximum manual auditability.
2. **OpenAI Managers**: Replaces untyped `sync.Map` fields in `translation.go` and `transcription.go` with generic typed `sync_map.Map[K, V]` using `github.com/zolstein/sync-map`, deduplicates broadcasting logic, simplifies connection lifecycle/auditing, and standardizes bounded FIFO audio buffering for seamless reconnects.
3. **Network Discovery & QR Code**: Detects macOS Wi-Fi SSID using `ipconfig getsummary en0 | awk -F ' SSID : ' '/ SSID : / {print $2}'` (with Linux fallback), determines the public/LAN-exposed IP and server port, and renders `ip:port` on the landing page QR code.
4. **Live UI Updates & Master Refresh**: Emits real-time SSE notifications upon completion of normal and trimmed recordings, enables automatic recording list updates, and replaces fragmented card refresh buttons with a single Master Refresh button on the console header.
5. **Code Quality & Constitution**: Audits large functions, enforces private struct field encapsulation, cleanly separates state from interface definitions, creates a Code Quality Report, and codifies principles into the project Constitution (v1.2.0).

---

## Technical Context

**Language/Version**: Go 1.23+ (backend), TypeScript 5+ with Svelte 5 / SvelteKit (frontend)
**Primary Dependencies**: PortAudio, Gin (Web & REST), Gorilla WebSocket, `github.com/zolstein/sync-map`, Lucide Svelte, qrcode
**Storage**: Local filesystem (`recordings/`, `cloud_drive/`) with WAV PCM audio and MP3 exports
**Testing**: Go standard testing with `testify` (`go test -race ./src/backend/...`), Vitest for frontend (`bun run test:unit`)
**Target Platform**: Linux and macOS (with dedicated macOS network utilities)
**Project Type**: Real-time audio streaming daemon and web console
**Performance Goals**:
- Lock-free audio recording write loop with <1ms per-chunk processing latency.
- Resilient OpenAI reconnect buffer holding up to 15 seconds of audio (~720 KB) during socket drops.
**Constraints**:
- Single-threaded PortAudio callback stream safety.
- Zero data races under `go test -race`.
**Scale/Scope**: ~10 backend files touched, ~4 frontend components updated.

---

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Compliance Assessment | Status |
|-----------|-----------------------|--------|
| **I. Radical Simplicity & Code Minimization** | Deletes redundant wrappers (`conversion` layer, duplicated broadcasting loops, inner-loop mutex locking, per-card UI buttons). Uses `github.com/zolstein/sync-map` instead of hand-writing custom map wrappers. | PASS |
| **II. Streamlined & Maintainable Unit Tests** | Table-driven unit tests for PCM clamping, WAV headers, typed registries, and network detection without mocking bloat. | PASS |
| **III. Converged Architecture & Cohesion** | Converges scattered audio conversion and streaming logic into `audio_processing`. Unifies translation and transcription WebSocket management. | PASS |
| **IV. Hierarchical Domain Organization** | Relocates `wav.go`, `hls.go`, and PCM conversion to dedicated `audio_processing/` subpackage. | PASS |
| **V. Zero Duplication (DRY)** | Shared `SubtitleBroadcaster` and `PendingAudioBuffer` eliminate duplicated logic between translation and transcription. | PASS |
| **VI. Frequent Atomic Commits** | Commits executed after each phase and discrete task. | PASS |
| **VII. Explicit Functional Decomposition** | Large multi-step functions broken down into single-responsibility, explicitly named functions. | PASS |
| **VIII. Strict State Encapsulation** | Struct fields made private; state separated from interface contracts. | PASS |
| **IX. Auditable & Lock-Free Audio Pipeline** | Engine-to-recording path simplified to direct, transparent functional calls; inner recording loop is lock-free. | PASS |

---

## Code Quality Report

### 1. Function Sizing and Functional Naming Audit
- **Observed Deficiency**:
  - `WriteAudio` in `storage.go` performed clamp calculations, Little-Endian bit shifts, buffer allocations, and I/O writes all in a single inline loop.
  - `runSession`, `dialBuffering`, and `readLoop` in `translation.go` and `transcription.go` mixed WebSocket protocol handling, token telemetry counting, JSON decoding, subtitle broadcasting, and reconnect timing.
- **Architectural Remedy**:
  - Extract pure calculations into distinct, verb-noun named functions:
    - `audio_processing.ConvertStereoFloat32ToPCM16LE(chunk []float32) []byte`
    - `audio_processing.WritePCM16Buffer(w io.Writer, pcm []byte) (int, error)`
  - Decompose OpenAI session handling into discrete stages:
    - `registry.BroadcastSubtitle(lang, text string)`
    - `session.ReplayPendingAudio(conn)`
    - `session.HandleServerEvent(event)`

### 2. State Encapsulation and Struct Field Protection
- **Observed Deficiency**:
  - `TranslationManager` and `TranscriptionManager` exposed public untyped `sync.Map` fields (`Sessions`, `Subscribers`, `LastRestart`, `Mu`) allowing unrestricted external map access and runtime panic vectors from unchecked type assertions.
- **Architectural Remedy**:
  - Replace untyped `sync.Map` with generic `sync_map.Map[K, V]` from `github.com/zolstein/sync-map` (`sync_map.Map[string, *RealtimeSession]`, `sync_map.Map[string, []chan string]`, `sync_map.Map[string, time.Time]`).
  - Provide domain-level thread-safe operations with zero casting overhead and zero panic vulnerabilities.

### 3. State vs. Interface Separation
- **Observed Deficiency**:
  - In `src/backend/lib/state/`, interface definitions (such as `InterfaceConfig`) and internal state representations were grouped in files without explicit naming distinction.
- **Architectural Remedy**:
  - Organize state definitions into clean data holding types and behavior contracts into separate, dedicated interface definitions.

### 4. Manually Verifiable Engine-to-Recording Pipeline
- **Observed Deficiency**:
  - Path from PortAudio callback to recording WAV file required traversing `engine.go` callback, `recordChan`, `StartStorageWorker`, `WriteWithFile`, `fileMu.Lock`, `WriteAudio`, and sample counter atomic increments under mutex.
- **Architectural Remedy**:
  - Flatten and clarify the pipeline:
    `PortAudio Stream Callback` → `recordChan` → `StorageWorker` → `ConvertStereoFloat32ToPCM16LE` → `Write to File` (Lock-Free).
  - The entire recording path can be audited in under 3 minutes by any engineer.

---

## Project Structure

### Documentation (this feature)

```text
specs/007-audio-core-and-system-refactor/
├── plan.md              # Implementation Plan & Code Quality Report
├── research.md          # Technical decisions & rationale
├── data-model.md        # Data entities & signatures
├── quickstart.md        # Validation guide
├── contracts/           # API contracts (connection & SSE changelog)
│   ├── system-connection.json
│   └── sse-changelog.json
├── checklists/
│   └── requirements.md  # Spec quality checklist
└── spec.md              # Feature specification
```

### Source Code Changes

```text
src/backend/lib/
├── audioengine/
│   ├── audio_processing/       # Consolidated domain package (replaces conversion/)
│   │   ├── wav.go              # Standard WAV header generation & finalization
│   │   ├── hls.go              # HLS segment writing and playlist management
│   │   ├── pcm.go              # Sample clamping and float32-to-int16 LE conversion
│   │   ├── resample.go         # 24 kHz downsampling & delta decoding
│   │   └── audio_processing_test.go
│   ├── engine.go               # PortAudio capture coordinator
│   └── storage.go              # Lock-free storage worker
├── openai/
│   ├── broadcast.go            # Deduplicated subtitle broadcasting
│   ├── buffer.go               # Resilient 15s FIFO pending audio buffer
│   ├── translation.go          # Streamlined translation manager using zolstein/sync-map
│   ├── transcription.go        # Streamlined transcription manager using zolstein/sync-map
│   └── openai_test.go
├── web/
│   ├── handlers_system.go      # External IP/port resolution & macOS en0 SSID
│   └── handlers_audio.go
└── state/
    └── sections.go             # Recording state notification broadcast triggers

src/frontend/src/
├── lib/
│   ├── audioState.svelte.ts    # Auto-refresh on 'recording' SSE event
│   └── components/
│       ├── admin/
│       │   ├── AudioAdminView.svelte  # Master Refresh button on header
│       │   └── RecordingList.svelte   # Removed per-card refresh button
│       └── views/
│           └── LandingView.svelte     # Display external ip:port under QR code
```

---

## Phased Implementation Roadmap

- **Phase 1: Audio Processing Consolidation (`audio_processing`)**
  - Create `audio_processing` package; move and consolidate `wav.go`, `hls.go`, and `resample.go`.
  - Extract `ConvertStereoFloat32ToPCM16LE` clamping utility in `pcm.go`.
  - Add comprehensive unit tests.
- **Phase 2: Lock-Free Audio Engine Storage Worker**
  - Refactor `storage.go` and `engine.go` to eliminate per-chunk mutex locks.
  - Simplify the engine-to-recording call graph.
- **Phase 3: Typed OpenAI Generic Maps via `github.com/zolstein/sync-map` & Buffering**
  - Integrate `github.com/zolstein/sync-map` for `Sessions`, `Subscribers`, and `LastRestart`.
  - Deduplicate broadcasting logic in `broadcast.go`.
  - Standardize 15s bounded FIFO `pendingAudio` buffer.
  - Streamline `translation.go` and `transcription.go`.
- **Phase 4: Network Discovery & QR Code Endpoint**
  - Implement macOS `ipconfig getsummary en0` SSID detection and public IP/port resolution in `handlers_system.go`.
  - Update `LandingView.svelte` to show `ip:port` and render QR code targeting external `serverUrl`.
- **Phase 5: Real-Time UI Updates & Master Refresh**
  - Broadcast `SectionRecording` on normal and trimmed MP3 processing completions.
  - Wire frontend SSE handler to auto-refresh file list.
  - Add Master Refresh button to `AudioAdminView.svelte` and remove duplicate refresh buttons.
- **Phase 6: Quality Verification & Test Suite**
  - Run full test suites: `go test -race ./src/backend/...` and `bun run test:unit`.
  - Verify zero regressions and check against Constitution Principles I-IX.
