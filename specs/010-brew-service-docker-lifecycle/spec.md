# Feature Specification: Homebrew Service Docker & Compose Lifecycle Management & v0.2.0 Release

**Feature Branch**: `010-brew-service-docker-lifecycle`

**Created**: 2026-10-10

**Status**: Ready for Planning

**Input**: User description: "add docker and starting up docker compose as part of the brew services script. and on stop it should also stop the docker service and the containers. Change the version in brew. Ipm going to create v0.2.0 as the new version. Now create a branching from origin main and create the MR. Use the spec to creat the changelog"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Automated Observability Stack Startup with Homebrew Service (Priority: P1)

As a system operator deploying Abel with Homebrew, I want `brew services start abel` to automatically launch the Docker service and run `docker compose up -d` for the observability stack (Prometheus, Loki, Grafana, OpenTelemetry Collector) alongside Abel, so that I don't have to manually start Docker and compose containers every time the service starts.

**Why this priority**: Users currently have to remember to start Docker and run compose independently. Unifying this under `brew services start` guarantees that logs, metrics, and dashboards are immediately available whenever Abel runs.

**Independent Test**: Can be tested on macOS and Linux by running the service startup script; verify that Docker daemon is active, compose containers (`otel-collector`, `loki`, `prometheus`, `grafana`) reach running state, and the Abel web server begins listening.

**Acceptance Scenarios**:

1. **Given** Docker is not running, **When** the Homebrew service starts, **Then** it launches the Docker daemon/app, waits until Docker is responsive, runs `docker compose up -d`, and launches the Abel binary.
2. **Given** Docker is already running, **When** the Homebrew service starts, **Then** it immediately executes `docker compose up -d` without restarting the Docker daemon, and launches the Abel binary.

---

### User Story 2 - Clean Teardown of Docker Compose and Docker Daemon on Service Stop (Priority: P2)

As a system operator stopping the Abel service via `brew services stop abel`, I want all Docker containers to be cleanly brought down with `docker compose down` and the Docker service/daemon stopped (if initiated by Abel or configured for full teardown), so that system resources (RAM, CPU, and network ports 3001, 3100, 9090, 4317) are completely freed and no orphaned containers remain.

**Why this priority**: Leaving background containers running indefinitely after stopping the host service wastes memory and locks ports needed by other workflows.

**Independent Test**: Can be tested by sending a termination signal (SIGTERM/SIGINT) to the service wrapper; verify that `docker compose down` terminates all 4 containers and the Abel process exits with code 0.

**Acceptance Scenarios**:

1. **Given** Abel and the Docker Compose containers are actively running, **When** `brew services stop abel` sends a termination signal, **Then** the service wrapper traps the signal, gracefully terminates Abel, executes `docker compose down`, optionally quits Docker if it was started by the script, and cleanly exits.

---

### User Story 3 - Homebrew Formula Version Bump to v0.2.0 and Asset Packaging (Priority: P3)

As a packager and end user, I want the Homebrew formula `abel.rb` to target release `v0.2.0`, packaging the service lifecycle wrapper script, the `docker-compose.yaml` definition, and the `observability/` configurations into Homebrew's shared directory (`pkgshare`), so that users installing `brew install abel` get the latest version with full service lifecycle capabilities.

**Why this priority**: Ensures the formula installs all files required by the service lifecycle and points to the `v0.2.0` release tag.

**Independent Test**: Can be tested by running `brew audit --formula` or inspecting `abel.rb` syntax and verifying that `v0.2.0` tag, assets, and service script paths are defined correctly.

**Acceptance Scenarios**:

1. **Given** the repository is tagged at `v0.2.0`, **When** inspecting `abel.rb`, **Then** the git url tag is `v0.2.0`.
2. **Given** `abel.rb` install routine, **When** Homebrew installs the formula, **Then** `abel-service` wrapper is installed into `bin/`, and `docker-compose.yaml` + `observability/` are installed into `pkgshare`.

---

### Edge Cases

- **Docker not installed**: If Docker is not found in `$PATH` or applications, the service wrapper MUST log an informational warning to the error log and proceed running Abel without crashing.
- **Docker startup timeout**: If the Docker daemon fails to respond within a reasonable threshold (e.g., 30 seconds), the script MUST log a timeout error, avoid blocking forever, and continue starting Abel.
- **Pre-existing Docker daemon**: If Docker was already running before Abel service started, the teardown should avoid terminating the entire Docker daemon unless explicitly requested, preventing disruption to unrelated container workloads.
- **Port collisions**: If ports 3001, 3100, 4317, or 9090 are occupied by external processes, `docker compose up -d` errors are logged to `var/log/abel.errors.log` without aborting Abel audio capture.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide an executable service wrapper script (`bin/abel-service`) designed to be executed by Homebrew's `service` block.
- **FR-002**: The service script MUST detect whether the Docker daemon is accessible; if not accessible, it MUST attempt to start the Docker daemon (supporting macOS Docker Desktop / OrbStack / Colima and Linux systemd/service) and poll for daemon readiness up to 30 seconds.
- **FR-003**: The service script MUST locate the packaged `docker-compose.yaml` and execute `docker compose up -d` to launch the observability stack.
- **FR-004**: The service script MUST trap termination signals (`SIGTERM`, `SIGINT`, `SIGHUP`) and on receiving a signal, stop Abel, execute `docker compose down`, and stop the Docker daemon if it was spawned by the wrapper.
- **FR-005**: The Homebrew formula `abel.rb` MUST be updated with version tag `v0.2.0`.
- **FR-006**: The Homebrew formula `abel.rb` MUST install the `abel-service` wrapper into `bin/` and `docker-compose.yaml` + `observability/` into `pkgshare/"abel"` or `etc/"abel"`.
- **FR-007**: The Homebrew formula's `service` block MUST configure `run [opt_bin/"abel-service"]` with log output redirected to `var/"log/abel.log"` and `var/"log/abel.errors.log"`.

### Key Entities

- **Service Wrapper (`abel-service`)**: Shell script managing process lifecycle, signal trapping, Docker daemon activation, Docker Compose orchestration, and Abel process execution.
- **Observability Stack (`docker-compose.yaml`)**: Definition of Prometheus, Loki, Grafana (port 3001), and OpenTelemetry Collector services.
- **Formula (`abel.rb`)**: Ruby specification for Homebrew package installation and launchd/systemd service configuration.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Running `abel-service` starts the background containers and Abel server within 15 seconds on a machine with Docker available.
- **SC-002**: Stopping the service via SIGTERM terminates all Abel processes and compose containers within 10 seconds.
- **SC-003**: `abel.rb` targets release `v0.2.0` and references the service wrapper.
- **SC-004**: On systems without Docker installed, Abel launches successfully with zero fatal errors.

## Assumptions

- Docker Desktop, OrbStack, Colima, or Docker engine on Linux is installed if the user desires observability containers.
- Standard signal propagation from `launchctl` (macOS) and `systemd` (Linux) delivers `SIGTERM` to the process group / main process.
- User configuration in `~/.config/abel/config.yaml` remains the primary runtime configuration for Abel.
