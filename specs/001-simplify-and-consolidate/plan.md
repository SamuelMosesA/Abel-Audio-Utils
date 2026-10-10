# Implementation Plan: Simplify & Consolidate Audio Codebase

**Branch**: `001-simplify-and-consolidate` | **Date**: 2026-10-10 | **Spec**: [spec.md](file:///home/samuelmoses/Workspace/Church/Abel-Audio-Utils/specs/001-simplify-and-consolidate/spec.md)

**Input**: Feature specification from `specs/001-simplify-and-consolidate/spec.md`

## Summary

Consolidate scattered audio conversions and format utilities into a dedicated, cohesive package (`src/backend/lib/audioengine/conversion/`). Unify duplicate OpenAI Realtime connection/disconnection lifecycle management between transcription and translation. Prune unnecessary styling classes and component bloat in the SvelteKit frontend, and streamline backend and frontend unit tests to be concise, table-driven, and easy to read.

## Technical Context

**Language/Version**: Go 1.25.6, TypeScript 5.x / Svelte 5  
**Primary Dependencies**: Gin (`github.com/gin-gonic/gin`), PortAudio, Gorilla WebSocket (`github.com/gorilla/websocket`), Testify (`github.com/stretchr/testify`)  
**Storage**: Local WAV file recordings and memory ring buffers  
**Testing**: Go standard testing with `testify/assert`, Vitest for frontend  
**Target Platform**: Linux / macOS server  
**Project Type**: Embedded Audio Service (Go backend + SvelteKit embedded web UI)  
**Performance Goals**: Real-time zero-copy audio pipeline, <5ms buffer processing latency  
**Constraints**: Zero memory leaks in audio channels and goroutines; strictly maintain existing API endpoints  

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [x] **Principle I: Radical Simplicity & Code Minimization**: Deletes duplicate downsampling and Float32-to-PCM routines; simplifies frontend markup and styling classes.
- [x] **Principle II: Streamlined & Maintainable Unit Tests**: Refactors verbose test assertions into concise table-driven test cases.
- [x] **Principle III: Converged Architecture & Cohesion**: Unifies scattered conversion code and duplicate connection state logic.
- [x] **Principle IV: Hierarchical Domain Organization**: Creates a dedicated `conversion/` subpackage under `audioengine/`.
- [x] **Principle V: Zero Duplication (DRY)**: Replaces identical loops in `transcription.go` and `translation.go` with shared functions.

*Outcome*: **PASSED**. The plan directly fulfills all 5 core constitution principles.

## Project Structure

### Documentation (this feature)

```text
specs/001-simplify-and-consolidate/
├── plan.md              # This file
├── research.md          # Phase 0 research decisions
├── data-model.md        # Phase 1 domain entities & types
├── quickstart.md        # Phase 1 run & validation instructions
├── contracts/           # Phase 1 contract definitions
│   └── internal-contracts.md
├── checklists/
│   └── requirements.md
└── tasks.md             # Phase 2 output (/speckit-tasks)
```

### Source Code Touchpoints

```text
src/
├── backend/
│   └── lib/
│       ├── audioengine/
│       │   ├── conversion/          # NEW: Centralized audio conversion subpackage
│       │   │   ├── pcm.go
│       │   │   ├── resample.go
│       │   │   ├── wav.go
│       │   │   └── conversion_test.go
│       │   ├── hls.go               # Updated to use conversion package
│       │   ├── storage.go           # Updated to use conversion package
│       │   └── wav.go               # Delegated / cleaned
│       └── openai/
│           ├── realtime_base.go     # NEW: Unified session runner & disconnect lifecycle
│           ├── transcription.go     # Streamlined (delegates conversion & connection)
│           └── translation.go       # Streamlined (delegates conversion & connection)
└── frontend/
    └── src/
        ├── lib/
        │   ├── audioState.svelte.ts # Simplified reactive state
        │   └── components/          # Stripped redundant wrappers and classes
```

## Phase 0: Outline & Research
*See [research.md](file:///home/samuelmoses/Workspace/Church/Abel-Audio-Utils/specs/001-simplify-and-consolidate/research.md)*

## Phase 1: Design & Contracts
*See [data-model.md](file:///home/samuelmoses/Workspace/Church/Abel-Audio-Utils/specs/001-simplify-and-consolidate/data-model.md), [contracts/internal-contracts.md](file:///home/samuelmoses/Workspace/Church/Abel-Audio-Utils/specs/001-simplify-and-consolidate/contracts/internal-contracts.md), and [quickstart.md](file:///home/samuelmoses/Workspace/Church/Abel-Audio-Utils/specs/001-simplify-and-consolidate/quickstart.md)*
