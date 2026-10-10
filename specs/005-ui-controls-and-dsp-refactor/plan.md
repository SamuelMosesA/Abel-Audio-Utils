# Implementation Plan: UI Controls, Per-Language AI Killswitch, and DSP Refactor

**Branch**: `005-ui-controls-and-dsp-refactor` | **Date**: 2026-10-10 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/005-ui-controls-and-dsp-refactor/spec.md`

## Summary

This plan addresses three primary enhancements in Abel Audio Utils:
1. **Recording Library Refresh**: Eliminates the recurring 2-second background polling loop in the Svelte recording panel and replaces it with an on-demand manual Refresh button.
2. **Per-Language AI Killswitch & Listener Metrics**: Extends backend runtime state (`state.AIConfig`) and the AI streams API to support per-language killswitch toggles, active client count reporting (subtitles + HLS listeners), and updates the admin UI to display all configured languages with individual killswitches and a manual Refresh button.
3. **Modular Audio DSP Extraction**: Extracts channel routing, digital gain multiplication, and sample clamping (`[-1.0, 1.0]`) from `engine.go` into a reusable, tested function `ExtractStereoChunk` inside `src/backend/lib/audioengine/conversion`.

## Technical Context

**Language/Version**: Go 1.24+ (Backend), TypeScript / Svelte 5 (Frontend)

**Primary Dependencies**: Gin (`github.com/gin-gonic/gin`), PortAudio (`github.com/gordonklaus/portaudio`), SvelteKit, Lucide Svelte

**Storage**: In-memory runtime state in `state.AppState` (no database required)

**Testing**: Go standard testing with `testify` (`go test ./...`), Vitest (`bun run test:unit`)

**Target Platform**: Linux / macOS server and web browsers (Admin console)

**Project Type**: Web service + Audio DSP daemon + Web SPA

**Performance Goals**: Audio capture loop latency <= 10ms; zero sample clipping over +/- 1.0; zero unnecessary polling requests

**Constraints**: Audio engine capture loop must remain strictly real-time and allocation-conscious

**Scale/Scope**: ~10 configured translation languages, live church service audience

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Principle I: Radical Simplicity & Code Minimization**: PASS. Replaces 20-line inline DSP loop with a single utility call; removes automated setInterval timer; runtime state stored in existing AppState struct without extra storage layers.
- **Principle II: Streamlined & Maintainable Unit Tests**: PASS. Pure unit tests for DSP stereo extraction and clamping, API route unit tests with mocks.
- **Principle III: Converged Architecture & Cohesion**: PASS. DSP logic resides in `audioengine/conversion`; state resides in `state`; web routes in `web`.
- **Principle IV: Hierarchical Domain Organization**: PASS. Clean package boundaries preserved.
- **Principle V: Zero Duplication**: PASS. Extracted math into `ExtractStereoChunk`.
- **Principle VI: Frequent Atomic Commits**: PASS. Changes will be partitioned into atomic commits.

## Project Structure

### Documentation (this feature)

```text
specs/005-ui-controls-and-dsp-refactor/
├── plan.md              # This plan
├── research.md          # Phase 0 decisions
├── data-model.md        # Phase 1 data entities
├── quickstart.md        # Phase 1 validation instructions
├── contracts/           # Phase 1 interface contracts
│   └── api-contracts.md
└── checklists/
    └── requirements.md  # Requirements quality checklist
```

### Source Code Layout

```text
src/
├── backend/
│   ├── lib/
│   │   ├── audioengine/
│   │   │   ├── engine.go                # Capture loop calls ExtractStereoChunk
│   │   │   └── conversion/
│   │   │       ├── pcm.go
│   │   │       ├── resample.go
│   │   │       ├── wav.go
│   │   │       └── conversion_test.go   # Tests for ExtractStereoChunk
│   │   ├── openai/
│   │   │   ├── realtime_base.go         # Listener counting & session control
│   │   │   └── translation.go           # Subtitle subscriber count tracking
│   │   ├── state/
│   │   │   └── config.go                # AIConfig with BlockedLanguages map
│   │   └── web/
│   │       ├── handlers_ai.go           # Toggle language killswitch, enrich GET /api/ai/streams
│   │       ├── handlers_ai_test.go      # Tests for language killswitch API
│   │       └── handlers_audio.go        # Reject HLS requests if language blocked
└── frontend/
    └── src/
        └── lib/
            ├── audioState.svelte.ts     # Language killswitch methods and polling removal
            └── components/
                └── admin/
                    ├── RecordingList.svelte     # Replace setInterval with manual refresh
                    └── TranslationAdmin.svelte  # Display all languages, listener counts, killswitches
```

**Structure Decision**: Standard Abel Audio Utils architecture separating frontend (SvelteKit) and backend (Go modules).

## Complexity Tracking

No constitution violations or unjustified abstractions introduced.
