# Quickstart: Validating Progressive MP3 Live Audio Streaming

**Branch**: `009-replace-hls-with-mp3` | **Date**: 2026-10-10 | **Spec**: [spec.md](./spec.md)

This guide provides automated and manual scenarios to verify that live audio streams directly as progressive chunked MP3 without HLS segment files or playlist delays.

---

## Automated Validation

### 1. In-Memory MP3 Broadcaster Unit Tests

Run unit tests for the live audio broadcaster in Go:

```bash
go test -v -run TestMP3Broadcaster ./src/backend/lib/audioengine/audio_processing
```

**Expected Outcome**:
- In-memory encoder boots without creating temporary directories.
- Encodes PCM chunks into valid MP3 frames.
- Multicasts frames to multiple listener channels without blocking.

### 2. Frontend Svelte Player Component Tests

Run Vitest suite to verify player simplification:

```bash
cd src/frontend
bun test src/lib/components/audio/LiveAudioPlayer.test.ts
```

**Expected Outcome**:
- `LiveAudioPlayer.svelte` renders cleanly without HLS state workarounds.
- Correctly binds audio source to `/api/audio/stream`.

---

## Manual Validation

### 1. Inspect Stream Headers with `curl`

Verify that the stream is chunked `audio/mpeg` with no HLS redirection:

```bash
curl -I http://localhost:8080/api/audio/stream
```

**Expected Response**:
```http
HTTP/1.1 200 OK
Content-Type: audio/mpeg
Transfer-Encoding: chunked
Cache-Control: no-cache, no-store, must-revalidate
Connection: keep-alive
```

### 2. Stream Data Verification

Read a sample of the live stream and verify it is recognized as valid MPEG audio:

```bash
curl -s -N http://localhost:8080/api/audio/stream | head -c 100000 | file -
```

**Expected Output**:
```text
/dev/stdin: Audio file with ID3 version 2... or MPEG ADTS, layer III...
```

### 3. Mobile Browser Playback (iOS Safari & Android Chrome)
1. Open the broadcast URL on iPhone Safari (`http://<ip>:8080/`).
2. Tap "Start Listening".
3. Verify audio plays immediately within 1 second.
4. Lock the iPhone screen.
5. Verify audio continues streaming in the background and lock screen controls appear.
