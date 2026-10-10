# Data Model & Hierarchy Plan: Codebase Simplification

## 1. Domain Hierarchy Realignment

To make the Go backend easy to navigate and read for humans, the package hierarchy is structured as follows:

```text
src/backend/lib/
├── audioengine/          # Real-time hardware capture & audio routing
│   ├── conversion/       # Domain: Stateless format conversion, resamplers, WAV headers
│   │   ├── pcm.go        # Float32 <-> PCM16
│   │   ├── resample.go   # Sample rate conversion (48k -> 24k)
│   │   └── wav.go        # 44-byte RIFF headers & metadata
│   ├── broadcaster.go    # Audio fan-out to WS and HLS
│   ├── engine.go         # PortAudio capture lifecycle
│   ├── hls.go            # FFmpeg HLS segmentation
│   └── storage.go        # Disk WAV recording writer
├── config/               # App configuration & YAML loader
├── openai/               # AI real-time streaming & translation
│   ├── realtime_base.go  # Core WebSocket connection lifecycle, disconnects, token telemetry
│   ├── transcription.go  # Subtitles & speech-to-text
│   └── translation.go    # Multilingual speech synthesis & audio streaming
├── state/                # Thread-safe global application state
├── telemetry/            # OpenTelemetry metrics & traces
└── web/                  # Gin HTTP REST API & WebSockets
    ├── handlers_audio.go
    ├── handlers_auth.go
    ├── handlers_recordings.go
    ├── recording_cloud.go
    └── recording_processor.go
```

## 2. Docstring Standard (Godoc)

Every exported type and function MUST follow this template:

```go
// FunctionName performs <action> on <target>.
// It accepts <params> and returns <results>.
// If <condition>, it returns <specific error>.
func FunctionName(...) (...)
```
