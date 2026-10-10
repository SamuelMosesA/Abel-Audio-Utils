# Feature Specification: Fix OpenTelemetry Dependency Compatibility & Homebrew Build Failure

**Feature Branch**: `003-fix-otel-build-failure`

**Created**: 2026-10-10

**Status**: Draft

**Input**: User description: "/speckit-specify thre is a bug go build -o /home/linuxbrew/.linuxbrew/Cellar/abel/1.0.0-dev/bin/abel src/backend/main.go Last 15 lines from /home/samuelmoses/.cache/Homebrew/Logs/abel/04.go.log: /home/samuelmoses/.cache/Homebrew/go_mod_cache/pkg/mod/go.opentelemetry.io/contrib/bridges/otelslog@v0.19.0/convert.go:31:14: undefined: log.Int64Value ... /home/linuxbrew/.linuxbrew/Homebrew/Library/Homebrew/vendor/portable-ruby/4.0.7_1/lib/ruby/4.0.0/json/common.rb:1077:in 'JSON::Ext::Generator::State#generate': Pathname not allowed in JSON (JSON::GeneratorError)"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Reliable Binary Compilation via Package Manager (Priority: P1)

A user or system administrator installing or compiling the Abel package (via Homebrew or direct binary compilation) expects the build pipeline to compile all source components without failing on incompatible module versions or internal compilation crashes.

**Why this priority**: Without a functional build step, the software cannot be installed, packaged, deployed, or upgraded through automated distribution channels.

**Independent Test**: Can be fully tested by executing the binary build command against the application source tree (`go build -o <output_binary> src/backend/main.go`) and confirming a clean zero exit status with an operational executable produced.

**Acceptance Scenarios**:

1. **Given** a clean build environment with the project source code, **When** invoking the backend build command, **Then** all package modules compile successfully without symbol resolution or type definition mismatch errors.
2. **Given** a Homebrew installation pipeline invoking compilation, **When** executing the compilation phase, **Then** all build artifacts are generated and transferred to the designated installation directory without triggering failure handlers.

---

### User Story 2 - Consistent Observability & Telemetry Runtime (Priority: P2)

A service operator running Abel in production expects application telemetry, logging bridges, and metrics exporters to initialize and function harmoniously with matching runtime library contracts.

**Why this priority**: Resolving compilation conflicts must not disable or corrupt runtime structured logging (`slog`), metric reporting, or trace propagation in live environments.

**Independent Test**: Can be fully tested by running application startup with telemetry initialization and verifying that structured log records and system metrics are emitted without runtime panics.

**Acceptance Scenarios**:

1. **Given** an operational application binary, **When** the service boots with configured telemetry parameters, **Then** the logging provider and metric provider initialize without API incompatibility exceptions.

---

### Edge Cases

- What happens when embedded static frontend assets have not yet been placed into the backend static directory before compilation? The build workflow must ensure prerequisites (frontend bundle generation and asset placement) are satisfied before binary compilation executes.
- What happens when external module caches contain stale or transitive pinned versions of experimental telemetry packages? Module declarations must explicitly lock compatible companion versions.
- How does the packaging formula handle path objects when logging execution steps? The packaging recipe must stringify path objects when passing arguments to subshells to prevent serializer exceptions.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The project dependency manifest MUST pin compatible versions across all telemetry, logging bridge, and metric exporter sub-modules.
- **FR-002**: The backend compilation pipeline MUST complete successfully under standard Go toolchains (Go 1.25+ / 1.27+) with zero compilation errors.
- **FR-003**: The packaging installation recipe MUST provide string-safe path arguments during build invocation to prevent serialization errors.
- **FR-004**: Embedded application asset requirements MUST be satisfied prior to invoking backend binary compilation in all build scripts.
- **FR-005**: All existing backend and frontend test suites MUST continue passing with zero regressions after dependency adjustments.

### Key Entities *(include if feature involves data)*

- **Dependency Manifest**: Defines the explicit module boundaries and version constraints for all external libraries used across runtime and compilation.
- **Distribution Formula**: Directs the automated build sequence (frontend generation, asset staging, API documentation generation, binary compilation, and configuration installation).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Clean binary compilation completes with a 0 exit code across local toolchains and automated package managers.
- **SC-002**: 100% of telemetry compilation errors (`undefined: log.Int64Value`, `undefined: api.KeyValue`) are eliminated.
- **SC-003**: The Homebrew installation command completes all execution steps through to final installation without crashing.
- **SC-004**: Automated test suites maintain a 100% pass rate.

## Assumptions

- The target build platform has compatible C library dependencies (such as PortAudio) available in standard link paths during compilation.
- The build toolchain possesses network access to download declared dependency packages during resolution if not already cached.
- OpenTelemetry log and metric functionality in the application uses standardized bridge APIs provided by compatible companion releases.
