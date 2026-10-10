# Data Model & Component Entities: Simplify & Consolidate Audio Codebase

## 1. Domain Entities & Interfaces

### Audio Conversion Package (`audioengine/conversion`)

#### `PCMConverter`
Stateless functions operating on raw floating-point and integer PCM slices:
- `Float32ToPCM16(chunk []float32) []byte`: Converts normalized Float32 values [-1.0, 1.0] to little-endian 16-bit PCM bytes with hard-limiting clamping.
- `PCM16ToFloat32(data []byte) []float32`: Converts little-endian 16-bit PCM byte slices back to normalized Float32.

#### `Resampler`
- `DownsampleStereoToMonoPCM24k(chunk []float32, srcRate int) []byte`: Downsamples stereo float32 buffer at `srcRate` to 24kHz mono 16-bit PCM byte array required by OpenAI Realtime API.
- `ResampleFloat32(chunk []float32, srcRate, dstRate int, channels int) []float32`: Generic ratio accumulator resampler.

#### `WAVHeader`
- `WritePlaceholderHeader(w io.Writer, channels uint16, sampleRate int) error`
- `FinalizeWavHeader(w io.WriteSeeker, channels uint16, dataLength int64, sampleRate int) error`

---

### OpenAI Realtime Session Controller (`lib/openai`)

#### `RealtimeRunner`
Common connection session runner struct in `realtime_base.go`:
```go
type RealtimeRunner struct {
    Session      *RealtimeSession
    APIKey       string
    Model        string
    Instructions string
    Voice        string
    OnDelta      func(eventType string, raw map[string]interface{})
    OnError      func(err error)
}
```
Responsibilities:
- Establishes WebSocket with Authorization headers.
- Emits initial session config.
- Runs concurrent read pump and write pump.
- Traps disconnections, logs structured errors, reports token telemetry on `response.done`.
- Ensures deterministic shutdown on cancellation context.

---

### Frontend State Entities

#### `SimplifiedAudioState`
- Direct Svelte 5 state runes without over-abstracted event dispatchers.
- Lean peak meters, playback buffer status, and translation subtitle queues.
