# Data Model: Replace HLS with Cross-Platform Progressive MP3 Streaming

**Branch**: `009-replace-hls-with-mp3` | **Date**: 2026-10-10 | **Spec**: [spec.md](./spec.md)

## Entities

### 1. In-Memory MP3 Broadcaster (`LiveAudioBroadcaster`)

Manages the lifecycle of the in-memory encoding pipeline and active listener subscriptions.

| Field | Type | Description | Visibility / Mutability |
| :--- | :--- | :--- | :--- |
| `language` | `string` | Language code (e.g., `"default"`, `"es"`, `"fr"`) | Read-only |
| `sampleRate` | `int` | Input audio sample rate (e.g. 48000 or 24000) | Read-only |
| `channels` | `int` | Number of channels (1 for mono translation, 2 for stereo capture) | Read-only |
| `listeners` | `map[chan []byte]struct{}` | Set of active HTTP listener output channels | Protected by `sync.RWMutex` |
| `inPipe` | `io.WriteCloser` | Stdin pipe writing raw PCM chunks to the encoder | Private |
| `outPipe` | `io.ReadCloser` | Stdout pipe reading encoded MP3 frames | Private |

#### Lifecycle:
```
[Engine Capture / AI Translation]
            │
            ▼ (Write PCM float32/int16)
   [LiveAudioBroadcaster]
            │
            ▼ (In-Memory stdin/stdout pipe)
    [FFmpeg MP3 Encoder]
            │
            ▼ (Continuous MP3 frames)
     [Fan-Out Buffer]
      ├──► Client A (iOS Safari)
      ├──► Client B (Android Chrome)
      └──► Client C (Desktop Web)
```

---

### 2. HTTP Audio Listener Connection (`StreamListener`)

Represents a single active HTTP client connection streaming audio.

| Field | Type | Description |
| :--- | :--- | :--- |
| `channel` | `chan []byte` | Bounded FIFO channel (buffer capacity 32 chunks) |
| `ctx` | `context.Context` | HTTP request context; cancelled when client disconnects |
| `flusher` | `http.Flusher` | Flushes chunked MP3 bytes immediately to network socket |
