# API Contracts: Progressive MP3 Live Audio Streaming

**Branch**: `009-replace-hls-with-mp3` | **Date**: 2026-10-10 | **Spec**: [spec.md](../spec.md)

## Live Audio Stream Endpoints

### 1. Main Broadcast Audio Stream

Stream real-time captured audio (original language) as continuous MP3 frames.

```http
GET /api/audio/stream
```

#### Headers
- `Accept: */*` or `Accept: audio/*`

#### Response Headers
- `Content-Type: audio/mpeg`
- `Transfer-Encoding: chunked`
- `Cache-Control: no-cache, no-store, must-revalidate`
- `Connection: keep-alive`

#### Body
Continuous binary stream of MPEG-1/2 Audio Layer III (MP3) frames. Connection remains open until client disconnects or audio engine stops.

---

### 2. Translated Audio Stream

Stream real-time translated audio for a specific language channel.

```http
GET /api/audio/stream/:lang
```

#### URL Parameters
- `lang`: Language code (e.g. `es`, `fr`, `de`, `nl`, `default`)

#### Response Headers
- `Content-Type: audio/mpeg`
- `Transfer-Encoding: chunked`
- `Cache-Control: no-cache, no-store, must-revalidate`
- `Connection: keep-alive`

#### Body
Continuous binary stream of MP3 audio frames generated from the live AI translation stream.

---

### 3. Removed / Deprecated Endpoints

The following HLS endpoints are deprecated and permanently removed:
- `GET /api/audio/hls/:lang/index.m3u8`
- `GET /api/audio/hls/:lang/:segment`
