# Data Model: Audio Core Pipeline, OpenAI Managers, and Network Refactor

**Feature Branch**: `007-audio-core-and-system-refactor`
**Date**: 2026-10-10
**Spec**: [spec.md](./spec.md)

## Domain Entities and Models

### 1. Audio Processing Domain (`src/backend/lib/audioengine/audio_processing`)

#### `WavHeader`
Represents the standard 44-byte RIFF/WAVE header for uncompressed PCM audio.
- `Channels`: `uint16` (Stereo = 2)
- `SampleRate`: `int` (e.g., 44100, 48000)
- `BitsPerSample`: `uint16` (16)
- `ByteRate`: `uint32` (`SampleRate * Channels * (BitsPerSample / 8)`)
- `BlockAlign`: `uint16` (`Channels * (BitsPerSample / 8)`)
- `DataSize`: `uint32` (audio payload byte count)

#### `PCMConverter`
Pure functional transformation signatures:
- `ConvertStereoFloat32ToPCM16LE(chunk []float32) []byte`:
  - Input: Interleaved stereo float32 slice `[L, R, L, R, ...]`
  - Processing: Clamps each float32 to `[-1.0, 1.0]`, scales by `32767`, encodes into Little-Endian `uint16`.
  - Output: Contiguous byte slice of length `(len(chunk) / 2) * 4`.
- `DownsampleStereoToMonoPCM24k(chunk []float32, srcRate int) []byte`:
  - Input: Interleaved stereo float32 slice and source sample rate.
  - Processing: Averages stereo channels to mono, resamples to 24 kHz, converts to 16-bit PCM Little-Endian.
  - Output: Downsampled byte slice.

---

### 2. Audio Engine Recording Storage (`src/backend/lib/audioengine`)

#### `StorageWorker`
Dedicated worker goroutine consuming from `recordChan <-chan []float32`:
- `activeFile`: `*os.File` (owned exclusively by the storage worker while recording is active)
- `samplesWritten`: `int64` (tracked monotonically)
- `isRecording`: `bool`
- Execution:
  - Consumes audio chunks from input channel.
  - If active recording, calls `audio_processing.ConvertStereoFloat32ToPCM16LE(chunk)`.
  - Performs direct unbuffered/synchronous write to `activeFile` without mutex locking in the chunk loop.
  - Updates sample counter.

---

### 3. OpenAI Session & Subscriber Registry (`src/backend/lib/openai`)

#### `RealtimeSession`
Active bidirectional WebSocket session with OpenAI:
- `language`: `string`
- `audioIn`: `chan []byte` (buffered queue for downsampled 24 kHz audio)
- `audioOut`: `chan []float32` (decoded audio stream for client listeners)
- `ctx`: `context.Context`
- `cancel`: `context.CancelFunc`
- `lastTokens`: `int64` (tracked usage tokens)

#### `SessionRegistry`
Thread-safe typed registry replacing untyped `sync.Map`:
- `mu`: `sync.RWMutex`
- `sessions`: `map[string]*RealtimeSession`
- Methods:
  - `Get(lang string) (*RealtimeSession, bool)`
  - `Store(lang string, s *RealtimeSession)`
  - `Delete(lang string)`
  - `Range(fn func(lang string, s *RealtimeSession) bool)`
  - `Count() int`

#### `SubscriberRegistry`
Thread-safe typed subscriber collection replacing untyped `sync.Map`:
- `mu`: `sync.RWMutex`
- `subscribers`: `map[string][]chan string`
- Methods:
  - `Subscribe(lang string, ch chan string)`
  - `Unsubscribe(lang string, ch chan string)`
  - `Broadcast(lang string, message string)`
  - `Count(lang string) int`

#### `PendingAudioBuffer`
Bounded FIFO ring buffer of 24 kHz PCM chunks held during socket reconnect:
- `chunks`: `[][]byte`
- `totalBytes`: `int`
- `maxBytes`: `int` (default: `24000 * 2 * 15` bytes ~ 15 seconds)
- Methods:
  - `Push(chunk []byte)`: Enqueues chunk, evicting oldest chunks if `totalBytes > maxBytes`.
  - `Pop() []byte`: Dequeues oldest chunk.
  - `Len() int`: Number of pending chunks.
  - `Clear()`: Drops all buffered audio.

---

### 4. System Connection & Network Info (`src/backend/lib/web`)

#### `SystemConnectionResponse`
DTO returned by `GET /api/system/connection`:
- `serverUrl`: `string` (e.g., `"http://192.168.1.120:8080"`)
- `host`: `string` (e.g., `"192.168.1.120"`)
- `port`: `string` (e.g., `"8080"`)
- `displayEndpoint`: `string` (e.g., `"192.168.1.120:8080"`)
- `ssid`: `string` (e.g., `"Church_Auditorium_5G"` or `"N/A"`)

---

### 5. SSE State Notification (`src/backend/lib/state`)

#### `StateChange`
Event payload broadcast over `/api/system/changelog`:
- `section`: `string` (`"recording"`, `"interface"`, `"ai"`, `"locations"`)
- `timestamp`: `int64` (Unix millisecond timestamp)
