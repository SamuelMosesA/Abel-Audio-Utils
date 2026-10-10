# Data Model & Configuration Specifications: OpenTelemetry Build Compatibility

**Feature**: [spec.md](spec.md) | **Feature Directory**: `specs/003-fix-otel-build-failure`

## Entities

### 1. Go Module Dependency Manifest (`go.mod`)

The dependency configuration tracking module requirements for binary compilation.

| Dependency Module | Target Compatible Version | Description |
| :--- | :--- | :--- |
| `go.opentelemetry.io/contrib/bridges/otelslog` | `v0.19.0` | Slog handler bridge to OpenTelemetry log records |
| `go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp` | `v0.20.0` | OTLP HTTP log exporter |
| `go.opentelemetry.io/otel/log` | `v0.20.0` | OpenTelemetry Go log interface definitions |
| `go.opentelemetry.io/otel/sdk/log` | `v0.20.0` | OpenTelemetry Go log provider implementation |
| `go.opentelemetry.io/otel` | `v1.45.0` | Core OpenTelemetry API |
| `go.opentelemetry.io/otel/sdk` | `v1.45.0` | Core OpenTelemetry SDK |

### 2. Runtime Application Configuration (`~/.config/abel/config.yaml`)

Configuration file required at service startup to resolve server ports, audio interfaces, and external API credentials.

| Field | Type | Description |
| :--- | :--- | :--- |
| `port` | `string` | Web server port (default: `"8080"`) |
| `sample_rate` | `int` | Audio sample rate (default: `48000`) |
| `buffer_size` | `int` | Buffer frame length (default: `1024`) |
| `default_ch_l` | `int` | Left audio channel index |
| `default_ch_r` | `int` | Right audio channel index |
| `default_boost` | `float64` | Digital gain boost factor (default: `1.0`) |
| `storage_location` | `string` | Filesystem path for captured WAV files |
| `cloud_drive_location` | `string` | Destination path for automated cloud delivery |
| `otlp_endpoint` | `string` | OTLP telemetry receiver URL (default: `"http://localhost:4318"`) |
