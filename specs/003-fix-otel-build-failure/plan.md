# Implementation Plan: Fix OpenTelemetry Dependency Compatibility & Homebrew Build Failure

**Branch**: `003-fix-otel-build-failure` | **Date**: 2026-10-10 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/003-fix-otel-build-failure/spec.md`

## Summary

Resolve Go compilation failures caused by mismatched OpenTelemetry logging sub-modules (`undefined: log.Int64Value`, `undefined: api.KeyValue`) during binary compilation and Homebrew package installation. Lock matching `v0.20.0` dependencies in `go.mod`, stringify binary paths in `abel.rb`, and document user configuration setup in `~/.config/abel/config.yaml`.

## Technical Context

**Language/Version**: Go 1.25.6 / 1.27.2 (Linux x86_64), Ruby (Homebrew)  
**Primary Dependencies**: OpenTelemetry Go (`otelslog`, `otlploghttp`, `otel/log`, `otel/sdk/log`), PortAudio, Gin  
**Storage**: Local recordings (`./recordings`), Cloud drive (`./cloud_drive`), Config (`~/.config/abel/config.yaml`)  
**Testing**: `go build`, `bun run test:unit`, `bun run check`  
**Target Platform**: Linux (amd64, Homebrew)  
**Project Type**: Standalone service & CLI binary with embedded web assets  
**Performance Goals**: Clean deterministic build in under 10 seconds  
**Constraints**: Zero changes to public telemetry architecture or runtime logging interfaces  
**Scale/Scope**: 2 manifest files (`go.mod`, `go.sum`), 1 formula file (`abel.rb`), 1 configuration path (`~/.config/abel/config.yaml`)  

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Principle I (Radical Simplicity & Code Minimization)**: PASS. No unnecessary abstractions or wrapper packages added; version alignment directly in `go.mod`.
- **Principle II (Streamlined & Maintainable Tests)**: PASS. Build verification validated via `go build` and frontend verification via unit test suites.
- **Principle III (Converged Architecture & Cohesion)**: PASS. Resolves split versions across OpenTelemetry sub-packages into a cohesive `v0.20.0` companion lock.
- **Principle IV (Hierarchical Domain Organization)**: PASS. Project architecture untouched.
- **Principle V (Zero Duplication)**: PASS. Single source of truth maintained in `go.mod`.
- **Principle VI (Frequent Atomic Commits)**: PASS. Discrete commits per feature increment.

## Project Structure

### Documentation (this feature)

```text
specs/003-fix-otel-build-failure/
├── plan.md              # Implementation plan
├── research.md          # Technical research & decisions
├── data-model.md        # Dependency manifest & config specifications
├── quickstart.md        # Build & execution verification guide
├── checklists/
│   └── requirements.md  # Specification quality checklist
```

### Source Code

```text
├── abel.rb              # Homebrew distribution formula
├── go.mod               # Go module dependencies manifest
├── go.sum               # Go module checksums
└── src/
    ├── backend/
    │   ├── main.go      # Server entrypoint with //go:embed
    │   └── lib/telemetry/ # OpenTelemetry logging & metrics initialization
    └── frontend/        # Web client source
```

## Complexity Tracking

Zero constitution violations. No additional complexity introduced.
