# Feature Specification: Brew Service Lifecycle Management & Docker Compose Orchestration

**Feature Branch**: `010-brew-service-docker-lifecycle`

**Created**: 2026-10-10

**Status**: Draft

**Input**: User description: "add docker and starting up docker compose as part of the brew services script. and on stop it should also stop the docker service and the containers. Change the version in brew. Ipm going to create v0.2.0 as the new version. Now create a branching from origin main and create the MR. Use the spec to creat the changelog"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Automatic Observability Stack Startup on Service Launch (Priority: P1)

As a system operator running Abel via `brew services start abel`, I want the Abel service to automatically verify Docker daemon availability, start the Docker daemon if it is stopped, and run `docker compose up -d` before starting the Abel server, so that Prometheus, Grafana, Loki, and OpenTelemetry collector are immediately capturing metrics and logs without requiring manual container management.

**Why this priority**: Eliminates manual steps to start the observability infrastructure and ensures out-of-the-box telemetry functionality when Abel runs as a background service.

**Independent Test**: Can be tested by starting the service on a system with Docker installed; verify that `docker compose ps` shows all 4 containers healthy and `GET /api/audio/config` responds on port 8080.

**Acceptance Scenarios**:

1. **Given** Docker is installed but not running, **When** `brew services start abel` executes, **Then** the service starts the Docker daemon, awaits responsiveness, runs `docker compose up -d`, and launches Abel.
2. **Given** Docker daemon is already active, **When** the service executes, **Then** it immediately executes `docker compose up -d` without restarting the daemon.
3. **Given** Docker is not installed on the host, **When** the service executes, **Then** it logs a warning notice and starts Abel in standalone mode without error.

---

### User Story 2 - Clean Teardown of Docker Compose Services on Service Stop (Priority: P2)

As a system operator stopping the Abel service via `brew services stop abel`, I want Abel's Docker Compose services to be cleanly brought down with `docker compose down` while leaving the Docker daemon active for other system workloads, so that Abel's system resources (RAM, CPU, and network ports 3001, 3100, 9090, 4317) are completely freed without disrupting external containers.

**Why this priority**: Leaving background containers running indefinitely after stopping the host service wastes memory and locks ports needed by other workflows, while preserving the daemon keeps other developer workloads intact.

**Independent Test**: Can be tested by sending a termination signal (SIGTERM/SIGINT) to the service wrapper; verify that `docker compose down` terminates all 4 containers, the Docker daemon remains active, and the Abel process exits with code 0.

**Acceptance Scenarios**:

1. **Given** Abel and the Docker Compose containers are actively running, **When** `brew services stop abel` sends a termination signal, **Then** the service wrapper traps the signal, gracefully terminates Abel, executes `docker compose down`, leaves the Docker daemon intact, and cleanly exits.

---

### User Story 3 - Homebrew Formula Version Bump to v0.2.0 and Asset Packaging (Priority: P3)

As a packager and end user, I want the Homebrew formula `abel.rb` to target release `v0.2.0`, packaging the service lifecycle wrapper script, the `docker-compose.yaml` definition, and the `observability/` configurations into Homebrew's shared directory (`pkgshare`), so that users installing `brew install abel` get the latest version with full service lifecycle capabilities.

**Why this priority**: Ensures the formula installs all files required by the service lifecycle and points to the `v0.2.0` release tag.

**Independent Test**: Can be tested by running `brew audit --formula` or inspecting `abel.rb` syntax and verifying that `v0.2.0` tag, assets, and service script paths are defined correctly.

**Acceptance Scenarios**:

1. **Given** the repository is tagged at `v0.2.0`, **When** inspecting `abel.rb`, **Then** the git url tag is `v0.2.0`.\
2. **Given** `abel.rb` install routine, **When** Homebrew installs the formula, **Then** `abel-service` wrapper is installed into `bin/`, and `docker-compose.yaml` + `observability/` are installed into `pkgshare`.

---

### Edge Cases

- **Docker not installed**: If Docker is not found in `$PATH` or applications, the service wrapper MUST log an informational warning to the error log and proceed running Abel without crashing.
- **Docker startup timeout**: If the Docker daemon fails to respond within a reasonable threshold (e.g., 30 seconds), the script MUST log a timeout error, avoid blocking forever, and continue starting Abel.
- **External Docker containers**: The teardown MUST only terminate Abel's compose stack via `docker compose -f "$COMPOSE_FILE" down`, preserving any unrelated containers or daemon state.
- **Port collisions**: If ports 3001, 3100, 4317, or 9090 are occupied by external processes, `docker compose up -d` errors are logged to `var/log/abel.errors.log` without aborting Abel audio capture.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide an executable service wrapper script (`bin/abel-service`) designed to be executed by Homebrew's `service` block.
- **FR-002**: The service script MUST detect whether the Docker daemon is accessible; if not accessible, it MUST attempt to start the Docker daemon (supporting macOS Docker Desktop / OrbStack / Colima and Linux systemd/service) and poll for daemon readiness up to 30 seconds.
- **FR-003**: The service script MUST locate the packaged `docker-compose.yaml` and execute `docker compose up -d` to launch the observability stack.
- **FR-004**: The service script MUST trap termination signals (`SIGTERM`, `SIGINT`, `SIGHUP`) and on receiving a signal, stop Abel and execute `docker compose down` for Abel's services, leaving the Docker daemon itself running.
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
- **SC-002**: Stopping the service via SIGTERM terminates all Abel processes and compose containers within 10 seconds without stopping the Docker daemon.
- **SC-003**: `abel.rb` targets release `v0.2.0` and references the service wrapper.
- **SC-004**: On systems without Docker installed, Abel launches successfully with zero fatal errors.

## Assumptions

- Docker Desktop, OrbStack, Colima, or Docker engine on Linux is installed if the user desires observability containers.
- Standard signal propagation from `launchctl` (macOS) and `systemd` (Linux) delivers `SIGTERM` to the process group / main process.
- User configuration in `~/.config/abel/config.yaml` remains the primary runtime configuration for Abel.
