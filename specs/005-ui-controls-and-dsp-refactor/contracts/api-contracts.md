# API Contracts: UI Controls, Per-Language AI Killswitch, and DSP Refactor

**Branch**: `005-ui-controls-and-dsp-refactor`
**Date**: 2026-10-10

## 1. AI Streams Management API

### POST `/api/ai/streams`

Allows administrators to toggle the master AI switch, stop a translation session, or toggle an individual language killswitch.

#### Request Schema:

```json
{
  "action": "toggle_language",   // "toggle_master" | "stop_translation" | "toggle_language"
  "enabled": true,               // optional bool for toggle_master
  "language": "es",              // language code for stop_translation or toggle_language
  "blocked": true                // boolean for toggle_language
}
```

#### Response:
- Status: `200 OK`
```json
{
  "status": "AI action completed"
}
```
- Status: `400 Bad Request`
```json
{
  "error": "Invalid request body"
}
```

---

### GET `/api/ai/streams`

Returns master AI state, all configured languages with active client counts and blocked statuses, and raw session objects.

#### Response:
- Status: `200 OK`
```json
{
  "masterEnabled": true,
  "languages": [
    {
      "code": "es",
      "name": "Spanish",
      "blocked": false,
      "active": true,
      "listeners": 3
    },
    {
      "code": "fr",
      "name": "French",
      "blocked": true,
      "active": false,
      "listeners": 0
    }
  ],
  "sessions": [
    {
      "language": "Spanish",
      "listeners": 3,
      "subtitles": true
    }
  ]
}
```

---

### GET `/api/ai/subtitles?lang={code}` (SSE Stream)

- If language is blocked via killswitch:
  - Returns `503 Service Unavailable` with `{"error": "Language translation is blocked"}` or sends event and closes connection.
- If language is allowed:
  - Streams SSE subtitle events `data: {"text": "..."}` with keep-alives.

---

### GET `/api/audio/hls/{lang}/index.m3u8`

- If language is blocked via killswitch:
  - Returns `503 Service Unavailable` with `{"error": "translation audio is blocked"}`.
- If language is allowed:
  - Serves HLS m3u8 playlist.

---

## 2. Recording Library API

### GET `/api/recordings/library`

- Triggered on demand via UI Refresh button.
- Status: `200 OK`
- Returns recordings and exports payload.

---

## 3. Go Internal DSP Contract

### Package `abel/src/backend/lib/audioengine/conversion`

```go
// ExtractStereoChunk extracts left and right channels from multi-channel interleaved audio,
// applies digital gain boost, and clamps samples strictly to [-1.0, 1.0].
// If boost <= 0, unity gain (1.0) is used.
func ExtractStereoChunk(in []float32, bufferSize int, openedChannels int, chL int, chR int, boost float32) []float32
```
