# Technical Research: OpenTelemetry Dependency Resolution & Build Packaging

**Feature**: [spec.md](spec.md) | **Feature Directory**: `specs/003-fix-otel-build-failure`

## Research Overview

Investigation into the build failure encountered during `go build -o /home/linuxbrew/.linuxbrew/Cellar/abel/1.0.0-dev/bin/abel src/backend/main.go` and the associated Homebrew Ruby JSON error.

---

### Decision 1: Align OpenTelemetry Logging Sub-module Versions

- **Decision**: Lock `go.opentelemetry.io/otel/log` and `go.opentelemetry.io/otel/sdk/log` to `v0.20.0` to match `otelslog@v0.19.0` and `otlploghttp@v0.20.0`.
- **Rationale**: 
  - `go.opentelemetry.io/contrib/bridges/otelslog@v0.19.0` was designed for and requires `otel/log@v0.20.0`.
  - `go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp@v0.20.0` also operates against `otel/log@v0.20.0`.
  - In `otel/log v0.21.0`, breaking type changes occurred in the experimental OpenTelemetry log attributes (`log.Int64Value`, `log.Value`, `api.KeyValue` signatures changed or moved), causing the undefined symbol errors during compilation.
  - Aligning all experimental logging dependencies to `v0.20.0` completely eliminates all undefined symbol errors and enables a clean zero exit status during `go build`.
- **Alternatives Considered**:
  - *Upgrading all packages to latest (`v1.47.0`)*: `v1.47.0` introduces breaking changes across metric and trace providers, requiring substantial code rewrites in `telemetry.go`.
  - *Disabling OTel logging bridge*: Violates the observability requirements and removes structured log forwarding.

---

### Decision 2: Prevent Homebrew Ruby Pathname Serializer Crash

- **Decision**: In [`abel.rb`](file:///home/samuelmoses/Workspace/Church/Abel-Audio-Utils/abel.rb), ensure path arguments in build commands are explicitly stringified (e.g., `(bin/"abel").to_s`).
- **Rationale**: 
  - In Homebrew's Ruby runtime, `bin/"abel"` returns an instance of `Pathname`.
  - When Homebrew's execution logger logs steps or serializes state to JSON, `JSON.generate` throws `Pathname not allowed in JSON (JSON::GeneratorError)` if an unstringified `Pathname` is present in step arguments.
  - Ensuring clean build execution (exit code 0) prevents Homebrew from entering the error logging path, and stringifying paths prevents serializer errors under all conditions.
- **Alternatives Considered**:
  - *Hardcoding bin paths*: Unsafe across varied Homebrew prefixes (`/home/linuxbrew/.linuxbrew` vs `/opt/homebrew` on macOS).

---

### Decision 3: Ensure Embedded Static Asset Staging

- **Decision**: Verify that build scripts and recipes ensure `src/backend/static` is populated before `go build` is invoked, adhering to `//go:embed all:static` in `main.go`.
- **Rationale**: 
  - Go's `embed` directive requires matching directory contents at compile time.
  - `abel.rb` already contains steps 1 and 2 to build the frontend and copy to `src/backend/static/`. Maintaining this sequencing ensures standalone builds and brew builds succeed deterministically.
