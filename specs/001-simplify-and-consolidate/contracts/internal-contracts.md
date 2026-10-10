# Contracts: Internal & Domain Interfaces

## 1. Audio Conversion Contract

Package: `abel/src/backend/lib/audioengine/conversion`

```go
package conversion

import (
	"io"
)

// Float32ToPCM16 converts normalized [-1.0, 1.0] float32 audio samples into
// signed 16-bit little-endian PCM byte representation with clamping.
func Float32ToPCM16(chunk []float32) []byte

// PCM16ToFloat32 converts 16-bit signed little-endian PCM bytes into normalized float32.
func PCM16ToFloat32(data []byte) []float32

// DownsampleStereoToMonoPCM24k converts stereo float32 samples at srcRate into
// 24kHz mono 16-bit little-endian PCM bytes.
func DownsampleStereoToMonoPCM24k(chunk []float32, srcRate int) []byte

// WritePlaceholderWavHeader writes an initial 44-byte WAV header with 0 data length.
func WritePlaceholderWavHeader(w io.Writer, channels uint16, sampleRate int) error

// FinalizeWavHeader seeks back and updates the RIFF & data chunk sizes in the WAV header.
func FinalizeWavHeader(ws io.WriteSeeker, channels uint16, totalBytes int64, sampleRate int) error
```

---

## 2. OpenAI Realtime Session Contract

Package: `abel/src/backend/lib/openai`

```go
package openai

import "context"

type EventHandler func(raw map[string]interface{}) error

type SessionRunnerConfig struct {
    Session      *RealtimeSession
    APIKey       string
    Model        string
    Language     string
    Instructions string
    Voice        string
    HandleEvent  EventHandler
}

// RunRealtimeSession runs the full connection, write pump, read pump, and token
// telemetry until context cancellation or terminal error.
func RunRealtimeSession(ctx context.Context, cfg SessionRunnerConfig) error
```
