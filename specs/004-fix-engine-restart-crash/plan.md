# Implementation Plan: Fix Audio Engine Restart Crash

**Branch**: `004-fix-engine-restart-crash` | **Date**: 2026-10-10 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/004-fix-engine-restart-crash/spec.md`

## Summary

The backend server crashes on audio engine restart (`POST /api/audio/restart`) due to non-thread-safe concurrent access to native PortAudio driver initialization (`SIGSEGV` in `Pa_Initialize`), non-deterministic stream shutdown while reading audio buffers, and unsynchronized channel closures. The technical approach introduces synchronized lifecycle management with an engine mutex (`engineMu`), deterministic goroutine teardown synchronization (`StopAudioEngine` awaiting stream closure), safe channel management, and HTTP handler conflict guardrail against overlapping restart bursts.

## Technical Context

**Language/Version**: Go 1.27.2 / 1.25+
**Primary Dependencies**: `github.com/gordonklaus/portaudio` (C PortAudio wrapper), `github.com/gin-gonic/gin`
**Storage**: N/A
**Testing**: Go standard testing (`go test -race ./src/backend/...`)
**Target Platform**: Linux (ALSA / PulseAudio / PipeWire)
**Project Type**: Web Service / Audio Server
**Performance Goals**: Audio engine restarts complete within 500ms - 2s; zero server crashes or drops in HTTP availability under concurrent restart requests.
**Constraints**: PortAudio C functions (`Pa_Initialize`, `Pa_Terminate`, `Pa_OpenStream`) must be strictly serialized across all threads.
**Scale/Scope**: Core audio engine lifecycle (`src/backend/lib/audioengine/engine.go`, `src/backend/lib/web/handlers_audio.go`).

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Principle I: Radical Simplicity & Code Minimization**: PASS. Reuses standard sync primitives (`sync.Mutex`, channels) without complex state machines or third-party locking libraries.
- **Principle II: Streamlined & Maintainable Unit Tests**: PASS. Table-driven and deterministic mock tests using existing `MockStreamer` without bulky fixtures.
- **Principle III: Converged Architecture & Cohesion**: PASS. Consolidates audio engine lifecycle operations into `audioengine` package rather than leaking locking or driver calls across web handlers.
- **Principle IV: Hierarchical Domain Organization**: PASS. Retains domain logic in `src/backend/lib/audioengine/`.
- **Principle V: Zero Duplication**: PASS. Unifies stop and teardown logic into `StopAudioEngine`.
- **Principle VI: Frequent Atomic Commits**: PASS. Each phase and task is committed atomically with conventional commit messages.

## Project Structure

### Documentation (this feature)

```text
specs/004-fix-engine-restart-crash/
├── plan.md              # This file (/speckit-plan output)
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   └── audio-restart.md
└── tasks.md             # Phase 2 output (/speckit-tasks output)
```

### Source Code Layout

```text
src/backend/
├── lib/
│   ├── audioengine/
│   │   ├── engine.go          # Lifecycle mutex, StopAudioEngine, synchronized RestartEngine
│   │   └── engine_test.go     # Concurrency and restart unit tests
│   ├── state/
│   │   └── app_state.go       # Engine lifecycle channel/done coordination
│   └── web/
│       ├── handlers_audio.go  # Guardrail against concurrent restart requests
│       └── handlers_audio_test.go
```

**Structure Decision**: Confined entirely to backend engine package (`src/backend/lib/audioengine`) and its corresponding HTTP handler (`src/backend/lib/web`).

## Complexity Tracking

No constitution violations detected.
