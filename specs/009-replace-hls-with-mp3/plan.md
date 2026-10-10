# Implementation Plan: Replace HLS with Cross-Platform Progressive MP3 Streaming

**Branch**: `009-replace-hls-with-mp3` | **Date**: 2026-10-10 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/009-replace-hls-with-mp3/spec.md` + User guidance: "Also simplify the players code in Svelte when possible"

## Summary

Eliminate all HLS segmenting, `.m3u8` playlist files, disk I/O, and `"HLS stream is not ready"` warnings by implementing an in-memory progressive chunked MP3 streaming broadcaster (`LiveAudioBroadcaster`) in Go. The broadcaster streams continuous MP3 frames over HTTP chunked transfer to native browser `<audio>` elements across iOS Safari, Android Chrome, and desktop browsers. Concurrently, radically simplify the Svelte `LiveAudioPlayer` component by removing obsolete HLS retry workarounds, state hacks, and playlist timing logic.

## Technical Context

**Language/Version**: Go 1.23+ for backend streaming; TypeScript 5.6 / Svelte 5 for frontend player.

**Primary Dependencies**: `ffmpeg` (system binary used purely as an in-memory pipe encoder), Gin (`github.com/gin-gonic/gin`) for chunked HTTP responses.

**Storage**: None. 100% in-memory streaming with zero temporary files on disk.

**Testing**: Vitest (`bun test`) for Svelte player components; `go test` for audio broadcaster and HTTP handlers.

**Target Platform**: iOS Safari (iOS 15+), Android Chrome, macOS, Linux, desktop web browsers.

**Project Type**: Real-time Audio Streaming Web Application.

**Performance Goals**: Time-to-first-sound <1 second on mobile and desktop; 0 disk I/O operations for streaming; memory bounded per listener.

**Constraints**: Compliant with Abel Constitution v1.2.0 (Radical Simplicity, Zero Duplication, Code Minimization, State Encapsulation).

**Scale/Scope**: Replaces backend `hls.go` (~450 lines) and frontend `LiveAudioPlayer.svelte`, `StreamingView.svelte`, `LandingView.svelte` audio URLs.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Compliance Assessment | Status |
| :--- | :--- | :---: |
| **I. Radical Simplicity & Code Minimization** | Deletes complex HLS segmenter, playlist writer, disk cleanup routines, and directory watchers (~450 lines removed, net reduction). | **PASS** |
| **II. Streamlined & Maintainable Unit Tests** | Directly tests in-memory MP3 frame delivery and Svelte `<audio>` component without mock hierarchies. | **PASS** |
| **III. Converged Architecture & Cohesion** | Unifies live audio delivery into a single direct `GET /api/audio/stream` endpoint for both original and translated audio. | **PASS** |
| **IV. Hierarchical Domain Organization** | Audio streaming broadcaster resides cleanly in `audio_processing`. | **PASS** |
| **V. Zero Duplication (DRY)** | Shared in-memory MP3 streaming pipeline used for both main broadcast and AI translated language streams. | **PASS** |
| **VI. Frequent Atomic Commits** | Commits will be executed atomically per task. | **PASS** |
| **VII. Explicit Functional Decomposition (from 007)** | Broadcaster broken into clean functions (`writeAudio`, `broadcastFrames`, `addSubscriber`, `removeSubscriber`). | **PASS** |
| **VIII. Strict State Encapsulation (from 007)** | Broadcaster internal subscriber maps and encoder pipes are strictly unexported and mutex-protected. | **PASS** |
| **IX. Auditable & Lock-Free Audio Pipeline (from 007)** | Live capture into broadcaster buffer remains decoupled from file recording; listener slow-downs do not block audio engine. | **PASS** |

## Project Structure

### Documentation (this feature)

```text
specs/009-replace-hls-with-mp3/
├── spec.md                 # Feature specification
├── plan.md                 # This file (/speckit-plan output)
├── research.md             # In-memory MP3 streaming decisions
├── data-model.md           # Broadcaster and subscriber entities
├── quickstart.md           # Runnable validation steps
├── contracts/
│   └── api-contracts.md    # MP3 chunked HTTP contracts
└── checklists/
    └── requirements.md     # Quality checklist
```

### Source Code (repository root)

```text
src/
├── backend/
│   └── lib/
│       ├── audioengine/
│       │   ├── audio_processing/
│       │   │   ├── mp3_stream.go       # In-memory MP3 broadcaster (NEW)
│       │   │   ├── mp3_stream_test.go  # Unit tests for MP3 streaming (NEW)
│       │   │   ├── hls.go              # DELETED (obsolete HLS publisher)
│       │   │   └── hls_test.go         # DELETED
│       │   └── broadcaster.go          # Route live capture into MP3 broadcaster
│       └── web/
│           ├── handlers_audio.go       # Update StreamHandler for chunked MP3; remove HLS handlers
│           ├── handlers_audio_test.go  # Verify chunked MP3 HTTP responses
│           └── router.go               # Wire MP3 streaming routes; remove HLS endpoints
└── frontend/
    └── src/
        ├── lib/
        │   └── components/
        │       ├── audio/
        │       │   ├── LiveAudioPlayer.svelte   # Radically simplified HTML5 player
        │       │   └── LiveAudioPlayer.test.ts  # Updated player tests
        │       └── views/
        │           ├── LandingView.svelte       # Point audio source to /api/audio/stream
        │           └── StreamingView.svelte     # Point audio source to /api/audio/stream/${lang}
```

## Complexity Tracking

*No constitutional violations. Radical subtraction of code (~300+ lines net reduction).*
