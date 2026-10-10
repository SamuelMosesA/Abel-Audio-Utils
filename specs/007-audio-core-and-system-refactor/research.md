# Implementation Research: Audio Core Pipeline, OpenAI Managers, and Network Refactor

**Feature Branch**: `007-audio-core-and-system-refactor`
**Date**: 2026-10-10
**Spec**: [spec.md](./spec.md)

## Executive Summary of Technical Decisions

This research resolves all implementation strategies for consolidating audio processing utilities, making the storage recording pipeline lock-free, streamlining OpenAI WebSocket sessions with typed registries and resilient buffering, discovering reachable network endpoints for mobile QR joining, and unifying UI refresh controls under a Master Refresh paradigm.

---

## Decision 1: Consolidation into `audio_processing` Package

- **Current State**:
  - `src/backend/lib/audioengine/conversion/` contains WAV header generation, downsampling (`DownsampleStereoToMonoPCM24k`), and audio delta decoding (`DecodeAudioDelta`).
  - `src/backend/lib/audioengine/wav.go` contains thin wrappers delegating to `conversion`.
  - `src/backend/lib/audioengine/hls.go` contains HLS segment writing and playlist generation directly in `audioengine`.
  - `src/backend/lib/audioengine/storage.go` contains lines 51-69 inline: loops through float32 samples, clamps them to `[-1.0, 1.0]`, and packs int16 Little-Endian bytes.
- **Decision**:
  - Rename package `conversion` to `audio_processing` (`src/backend/lib/audioengine/audio_processing/`).
  - Move `wav.go` directly into `audio_processing/wav.go`, eliminating the redundant delegate layer.
  - Move `hls.go` into `audio_processing/hls.go` to isolate streaming segment slicing from core engine management.
  - Extract the sample clamping and byte packing into `ConvertStereoFloat32ToPCM16LE(chunk []float32) []byte` inside `audio_processing/pcm.go`.
- **Rationale**:
  - Unifies all DSP, PCM format conversion, WAV packing, and HLS segment slicing in a single cohesive domain package conforming to Constitution Principle IV (Hierarchical Domain Organization) and Principle V (Zero Duplication).
- **Alternatives Considered**:
  - Keep `conversion` and leave `hls.go` in root `audioengine`: Rejected because `audioengine` root should only coordinate the PortAudio engine lifecycle, not detailed file formats or protocol encoding.

---

## Decision 2: Lock-Free Inner Chunk Storage Recording Pipeline

- **Current State**:
  - In `StartStorageWorker`, for every audio chunk received over `recordChan` (~100 times per second), the code calls `appState.Engine().WriteWithFile(func(f *os.File) (int, error) { return WriteAudio(f, chunk) })`.
  - `WriteWithFile` locks `e.fileMu.Lock()` and `Unlock()` on every chunk.
  - Only a single goroutine (the storage worker) writes to the active recording file.
- **Decision**:
  - Refactor `storage.go` so the worker manages its own file descriptor reference. When recording starts, the engine passes the opened file to the storage worker via a dedicated control signal or atomic reference, or the worker maintains file state exclusively.
  - `WriteAudio` writes directly to `io.Writer` using `audio_processing.ConvertStereoFloat32ToPCM16LE`.
  - File closure and finalization coordinate cleanly upon recording stop without requiring a mutex acquisition for each audio chunk.
- **Rationale**:
  - Eliminates unnecessary lock contention in the high-frequency audio recording loop. Audio callbacks and storage writes operate with maximum determinism and zero synchronization stalls.
- **Alternatives Considered**:
  - Retaining `sync.Mutex` on `WriteWithFile`: Rejected as redundant overhead when single-threaded write ownership guarantees safety.

---

## Decision 3: Typed Session Registry & Resilient OpenAI Buffering

- **Current State**:
  - `TranslationManager` and `TranscriptionManager` each maintain untyped `sync.Map` fields: `Sessions`, `Subscribers`, and `LastRestart`.
  - Both managers contain duplicated implementations of `broadcastSubtitle` and identical dial/backoff loops.
  - Both managers have `pendingAudio` buffering, but transcription drops downsampled audio on dead sockets while translation keeps a bounded FIFO.
- **Decision**:
  - Create a shared `RealtimeRegistry` or typed session container:
    ```go
    type TypedSessionMap struct {
        mu       sync.RWMutex
        sessions map[string]*RealtimeSession
    }
    type TypedSubscriberMap struct {
        mu          sync.RWMutex
        subscribers map[string][]chan string
    }
    ```
  - Create a shared `SubtitleBroadcaster` helper that encodes JSON payloads once and safely delivers to all active subscriber channels non-blockingly.
  - Standardize `pendingAudio` (15-second bounded FIFO of 24 kHz mono PCM16) across both managers so neither translation nor transcription loses chunks during brief API reconnections.
- **Rationale**:
  - Completely eliminates untyped map casts (`v.(*RealtimeSession)`, `v.([]chan string)`), prevents runtime panic vectors, deduplicates ~100 lines of broadcasting boilerplate, and ensures rock-solid audio continuity.
- **Alternatives Considered**:
  - Keeping separate `sync.Map` with individual type guards: Rejected because it leaves unsafe type assertions and duplicated code across two managers.

---

## Decision 4: Network Connection Discovery & Mobile QR Endpoint

- **Current State**:
  - `GetWiFiSSID()` on macOS calls `/System/Library/PrivateFrameworks/Apple80211.framework/Resources/airport -I`, which is deprecated/removed in recent macOS versions.
  - `GetLocalIP()` uses UDP dial to `8.8.8.8:80`.
  - Frontend `LandingView.svelte` generates QR code using `window.location.href`, which points to `localhost:5173` if opened locally on the broadcaster's computer, making the QR code useless on mobile phones.
- **Decision**:
  - macOS Wi-Fi command: Execute `ipconfig getsummary en0 | awk -F ' SSID : ' '/ SSID : / {print $2}'`. If empty or failed, fallback to scanning available Wi-Fi interfaces or returning `"N/A"`.
  - Outward-facing IP:
    1. If a client accesses `/api/system/connection`, use incoming request headers (`X-Forwarded-For`, `Host`) if non-localhost.
    2. Otherwise, enumerate non-loopback network interfaces (`net.Interfaces()`) to find active IPv4 addresses on primary LAN interfaces (`en0`, `eth0`, `wlan0`), with UDP fallback.
  - Frontend:
    - Receive `serverUrl` (e.g. `http://192.168.1.50:8080`) from `/api/system/connection`.
    - Generate QR code pointing to `serverUrl`.
    - Render explicit `ip:port` text underneath the QR code box (e.g., `192.168.1.50:8080`).
- **Rationale**:
  - Solves the real-world frustration of attendees being unable to scan and open the broadcast on their mobile phones.
- **Alternatives Considered**:
  - Cloudflare tunnels or external STUN servers: Rejected because Abel Audio Utils is designed to operate seamlessly on local offline church Wi-Fi networks.

---

## Decision 5: Real-Time Recording Updates & Master Console Refresh

- **Current State**:
  - When recordings finish MP3 compression or trimming in `file_watcher.go` / `storage.go`, no SSE notification is pushed.
  - Users have to click "Refresh" inside `RecordingList.svelte`.
  - Multiple distinct "Refresh" buttons exist across different cards.
- **Decision**:
  - In `file_watcher.go` and processing callbacks, invoke `appState.Broadcast(state.SectionRecording)` upon job completion or file export creation.
  - In frontend `audioState.svelte.ts`, the SSE listener for `changelog` triggers `files.fetchFiles()` whenever a `recording` section change arrives.
  - Remove section-specific refresh buttons in `RecordingList.svelte`, and add a prominent "Master Refresh" button in `AudioAdminView.svelte` header that runs:
    ```ts
    await Promise.all([
      audio.fetchDevices(),
      audio.sync(),
      files.fetchFiles(),
      system.fetchConnection(),
    ]);
    ```
- **Rationale**:
  - Streamlines the administrative interface, eliminates manual page refreshing, and provides instant visual confirmation when MP3 encoding completes.

---

## Decision 6: Code Quality Governance & Constitution Updates

- **Decision**:
  - Update Constitution (`.specify/memory/constitution.md`) to version 1.2.0, ratifying Principles VII (Explicit Functional Decomposition), VIII (State Encapsulation), and IX (Auditable & Lock-Free Audio Pipeline).
  - Provide a formal Code Quality Report in `plan.md` auditing:
    1. Large functions broken down into named functions.
    2. State struct field encapsulation.
    3. Separation of state from interfaces.
    4. Auditable engine-to-recording flow.
