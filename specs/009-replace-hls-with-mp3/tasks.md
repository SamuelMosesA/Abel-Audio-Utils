# Tasks: Replace HLS with Cross-Platform Progressive MP3 Streaming

**Branch**: `009-replace-hls-with-mp3` | **Spec**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md)

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Audio streaming constants and pipeline configurations

- [X] T001 Configure MP3 encoding pipeline parameters and constants in `src/backend/lib/audioengine/audio_processing/mp3_stream.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: In-memory MP3 streaming engine and broadcaster infrastructure that MUST be complete before user story endpoints

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T002 [P] Create `LiveAudioBroadcaster` struct and subscriber fan-out management with `sync.RWMutex` and bounded subscriber channels in `src/backend/lib/audioengine/audio_processing/mp3_stream.go`
- [X] T003 Implement in-memory ffmpeg pipe worker (`ffmpeg -f f32le -ar <rate> -ac <channels> -i pipe:0 -f mp3 -b:a 128k -flush_packets 1 pipe:1`) reading continuous MP3 frames to listeners in `src/backend/lib/audioengine/audio_processing/mp3_stream.go`
- [X] T004 [P] Implement unit tests for `LiveAudioBroadcaster` verifying in-memory pipe lifecycle, non-blocking fan-out, and subscriber cleanup in `src/backend/lib/audioengine/audio_processing/mp3_stream_test.go`
- [X] T005 Integrate `LiveAudioBroadcaster` with audio capture pipeline in `src/backend/lib/audioengine/broadcaster.go`

**Checkpoint**: Foundation ready - LiveAudioBroadcaster produces in-memory MP3 streams and multicasts frames to subscribers.

---

## Phase 3: User Story 1 - Instant Progressive MP3 Audio Streaming for Web & Mobile (Priority: P1) 🎯 MVP

**Goal**: Deliver real-time live captured and translated audio over HTTP using progressive chunked MP3 streaming without playlist delays.

**Independent Test**: Send HTTP GET requests to `/api/audio/stream` and `/api/audio/stream/es`; verify `Content-Type: audio/mpeg`, `Transfer-Encoding: chunked`, and immediate continuous MP3 delivery without waiting for segment files.

### Implementation for User Story 1

- [X] T006 [P] [US1] Implement unit tests for chunked MP3 streaming handlers in `src/backend/lib/web/handlers_audio_test.go`
- [X] T007 [US1] Implement `StreamHandler` and `LanguageStreamHandler` delivering progressive chunked MP3 via `c.Writer.Flush()` in `src/backend/lib/web/handlers_audio.go`
- [X] T008 [US1] Register `/api/audio/stream` and `/api/audio/stream/:lang` routes and update `NewRouter` to accept broadcaster in `src/backend/lib/web/router.go`
- [X] T009 [US1] Update router initialization and broadcaster lifecycle in `src/backend/main.go`
- [X] T010 [P] [US1] Update audio source from `/api/audio/hls/default/index.m3u8` to `/api/audio/stream` in `src/frontend/src/lib/components/views/LandingView.svelte`
- [X] T011 [P] [US1] Update audio source from `/api/audio/hls/${lang}/index.m3u8` to `/api/audio/stream/${lang}` in `src/frontend/src/lib/components/views/StreamingView.svelte`

**Checkpoint**: User Story 1 complete — web and mobile listeners receive immediate progressive MP3 streaming.

---

## Phase 4: User Story 2 - Elimination of HLS Complexity & Disk I/O (Priority: P1)

**Goal**: Purge all HLS segment generators, `.m3u8` playlist writers, and temporary directory management, deleting hundreds of lines of obsolete code.

**Independent Test**: Audit backend code and filesystem while running broadcast; verify zero disk files generated and zero HLS warnings logged.

### Implementation for User Story 2

- [X] T012 [US2] Delete obsolete HLS publisher and playlist generator in `src/backend/lib/audioengine/audio_processing/hls.go` and `src/backend/lib/audioengine/audio_processing/hls_test.go`
- [X] T013 [US2] Remove deprecated `/api/audio/hls/:lang/index.m3u8` and `/api/audio/hls/:lang/:segment` route registrations in `src/backend/lib/web/router.go`
- [X] T014 [US2] Remove legacy `HLSPlaylistHandler` and `HLSSegmentHandler` functions in `src/backend/lib/web/handlers_audio.go` and obsolete HLS tests in `src/backend/lib/web/handlers_audio_test.go`

**Checkpoint**: User Story 2 complete — ~450 lines of complex HLS code deleted and zero disk I/O for live streaming.

---

## Phase 5: User Story 3 - Seamless Mobile Background Playback & Lock-Screen Audio (Priority: P2)

**Goal**: Radically simplify Svelte `LiveAudioPlayer` to a native HTML5 `<audio playsinline controls autoplay>` component without brittle HLS retry loops.

**Independent Test**: Render `LiveAudioPlayer.svelte`, verify simplified component DOM, test play/pause controls, and verify background playback compatibility.

### Implementation for User Story 3

- [X] T015 [US3] Radically simplify `LiveAudioPlayer.svelte` to native HTML5 audio with `playsinline` and clean controls in `src/frontend/src/lib/components/audio/LiveAudioPlayer.svelte`
- [X] T016 [P] [US3] Update player unit tests for simplified component structure in `src/frontend/src/lib/components/audio/LiveAudioPlayer.test.ts`

**Checkpoint**: User Story 3 complete — mobile background playback supported with minimal frontend code.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Verification, end-to-end testing, and documentation validation

- [X] T017 [P] Run quickstart validation scenarios per `specs/009-replace-hls-with-mp3/quickstart.md`
- [X] T018 Run full Go test suite (`go test -race ./src/backend/...`)
- [X] T019 [P] Run full frontend test suite (`bun run test:unit` in `src/frontend`)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: Can start immediately.
- **Phase 2 (Foundational)**: Depends on Phase 1; BLOCKS Phase 3.
- **Phase 3 (User Story 1 - MVP)**: Depends on Phase 2; provides core live streaming capability.
- **Phase 4 (User Story 2)**: Depends on Phase 3 completion to ensure MP3 streaming is operational before deleting HLS.
- **Phase 5 (User Story 3)**: Can proceed after or in parallel with Phase 3/4 frontend updates.
- **Phase 6 (Polish)**: Depends on all user stories being complete.

### User Story Dependencies

- **US1 (P1)**: Independent of US2/US3; relies on foundational broadcaster.
- **US2 (P1)**: Executes after US1 to cleanly decommission HLS endpoints without interrupting streaming.
- **US3 (P2)**: Refactors frontend player component after new stream URLs are in place.

---

## Parallel Execution Opportunities

- T002 and T004 can be developed in parallel with T003.
- T006 [US1] and T010 [US1] / T011 [US1] can proceed in parallel once handler interface is defined.
- T015 [US3] and T016 [US3] can proceed in parallel with backend cleanup tasks T012-T014.
- T017, T018, and T019 in Polish phase can run in parallel.

---

## Implementation Strategy (MVP First)

1. **Step 1**: Complete Phase 1 & Phase 2 foundational in-memory broadcaster (`mp3_stream.go`).
2. **Step 2**: Implement Phase 3 (User Story 1) to wire `/api/audio/stream` and verify live sound in browser (MVP reached!).
3. **Step 3**: Execute Phase 4 (User Story 2) to delete obsolete HLS files and prune routes.
4. **Step 4**: Execute Phase 5 (User Story 3) to simplify `LiveAudioPlayer.svelte`.
5. **Step 5**: Run Phase 6 polish tests and verify zero regressions.
