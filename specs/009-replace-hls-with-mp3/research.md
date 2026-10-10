# Research: Replace HLS with Cross-Platform Progressive MP3 Streaming

**Branch**: `009-replace-hls-with-mp3` | **Date**: 2026-10-10 | **Spec**: [spec.md](./spec.md)

## Research Topics & Decisions

### 1. In-Memory Progressive MP3 Live Encoding

#### Context
The legacy HLS implementation (`HLSPublisher`) spawned an `ffmpeg` process writing `.ts` Transport Stream chunks and `.m3u8` playlists into an OS temporary directory (`os.MkdirTemp("", "abel-hls-")`). This created high disk I/O, file synchronization races, and frequent `"HLS playlist not ready"` warnings when listeners connected before segments were written.

#### Decision
Implement an in-memory `LiveAudioBroadcaster` in `src/backend/lib/audioengine/audio_processing/`.
- Spawns an in-memory streaming process using `ffmpeg` connected strictly via pipes:
  ```bash
  ffmpeg -f f32le -ar <sampleRate> -ac <channels> -i pipe:0 -f mp3 -b:a 128k -flush_packets 1 pipe:1
  ```
- Writes raw float32/int16 PCM audio directly to `stdin`.
- Reads continuous MP3 frames from `stdout` and multicasts them to registered client listener channels via an in-memory ring buffer.
- **Zero disk files, zero temporary directories, zero segment cleanups.**

#### Rationale
- Complies with Abel Constitution Principle I (Radical Simplicity) and Principle IX (Auditable Audio Pipeline).
- Completely eliminates disk I/O for live streaming.
- Startup latency drops from 4–8 seconds to <500ms.

#### Alternatives Considered
- *Chunked WAV (`audio/wav`)*: Rejected because iOS Safari AVPlayer hangs on unseekable chunked WAV streams with `0xFFFFFFFF` header lengths (`ios-safari-audio-blocked`).
- *Chunked OGG Opus (`audio/ogg`)*: Rejected because iOS Safari lacks native OGG container support.
- *WebSockets Binary PCM*: Rejected because iOS suspends Web Audio API AudioContext when screen locks.

---

### 2. HTTP Streaming Endpoints & Multicast Distribution

#### Context
`GET /api/audio/stream` was previously redirecting to `/api/audio/hls/default/index.m3u8`.

#### Decision
Refactor `GET /api/audio/stream` and `GET /api/audio/stream/:lang` to directly stream MP3 bytes using standard HTTP chunked transfer:
```http
HTTP/1.1 200 OK
Content-Type: audio/mpeg
Transfer-Encoding: chunked
Cache-Control: no-cache, no-store, must-revalidate
Connection: keep-alive
```
- Each incoming HTTP connection registers an active channel with the broadcaster.
- The handler loops reading MP3 chunks, writing to `c.Writer`, and calling `c.Writer.Flush()`.
- When the client disconnects, `c.Request.Context().Done()` unregisters the listener immediately.
- Delete legacy `/api/audio/hls/:lang/index.m3u8` and `/api/audio/hls/:lang/:segment` endpoints.

---

### 3. Frontend Player Simplification (Svelte)

#### Context
`LiveAudioPlayer.svelte` contained retry state machines, error strings, and workarounds specifically tailored to HLS 404s and segment readiness issues.

#### Decision
Radically simplify `LiveAudioPlayer.svelte`:
- Use native HTML5 `<audio>` element with `playsinline` and standard controls.
- Provide simple Play/Stop button and volume/mute handling.
- Point URLs directly to:
  - Original Audio: `/api/audio/stream`
  - Translated Audio: `/api/audio/stream/${lang}`
- Remove all HLS-related error handling and timer hacks.
