#!/usr/bin/env bash
# Abel Service Lifecycle Wrapper
# Manages Docker daemon startup (if needed), Docker Compose stack, Abel binary lifecycle, and clean teardown.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Locate Abel binary
if [ -n "$ABEL_BIN" ] && [ -x "$ABEL_BIN" ]; then
    BINARY="$ABEL_BIN"
elif [ -x "$SCRIPT_DIR/abel" ]; then
    BINARY="$SCRIPT_DIR/abel"
elif command -v abel >/dev/null 2>&1; then
    BINARY="$(command -v abel)"
else
    echo "[abel-service] ERROR: Unable to locate 'abel' executable." >&2
    exit 1
fi

# Locate docker-compose.yaml
COMPOSE_FILE=""
CANDIDATE_PATHS=(
    "$ABEL_COMPOSE_FILE"
    "$SCRIPT_DIR/../share/abel/docker-compose.yaml"
    "/opt/homebrew/share/abel/docker-compose.yaml"
    "/usr/local/share/abel/docker-compose.yaml"
    "$SCRIPT_DIR/../docker-compose.yaml"
    "$SCRIPT_DIR/docker-compose.yaml"
    "./docker-compose.yaml"
)

for p in "${CANDIDATE_PATHS[@]}"; do
    if [ -n "$p" ] && [ -f "$p" ]; then
        COMPOSE_FILE="$p"
        break
    fi
done

COMPOSE_STARTED=0
ABEL_PID=""

is_docker_responsive() {
    docker info >/dev/null 2>&1
}

start_docker_daemon() {
    echo "[abel-service] Docker daemon is not active. Attempting to start..."
    OS_TYPE="$(uname -s)"
    if [ "$OS_TYPE" = "Darwin" ]; then
        if [ -d "/Applications/Docker.app" ]; then
            open -a Docker --args --unattended 2>/dev/null || true
        elif [ -d "/Applications/OrbStack.app" ]; then
            open -a OrbStack 2>/dev/null || true
        elif command -v colima >/dev/null 2>&1; then
            colima start 2>/dev/null || true
        fi
    elif [ "$OS_TYPE" = "Linux" ]; then
        if systemctl --user is-enabled docker >/dev/null 2>&1 || systemctl --user status docker >/dev/null 2>&1; then
            systemctl --user start docker 2>/dev/null || true
        elif command -v systemctl >/dev/null 2>&1; then
            sudo systemctl start docker 2>/dev/null || true
        elif command -v service >/dev/null 2>&1; then
            sudo service docker start 2>/dev/null || true
        fi
    fi

    # Wait up to 30 seconds for Docker daemon
    local elapsed=0
    while [ $elapsed -lt 30 ]; do
        if is_docker_responsive; then
            echo "[abel-service] Docker daemon started successfully."
            return 0
        fi
        sleep 1
        elapsed=$((elapsed + 1))
    done

    echo "[abel-service] WARNING: Timed out waiting for Docker daemon; continuing without Docker." >&2
    return 1
}

cleanup() {
    trap - SIGTERM SIGINT SIGHUP EXIT
    echo "[abel-service] Initiating graceful shutdown..."

    if [ -n "$ABEL_PID" ] && kill -0 "$ABEL_PID" 2>/dev/null; then
        echo "[abel-service] Stopping Abel binary (PID $ABEL_PID)..."
        kill -TERM "$ABEL_PID" 2>/dev/null || true
        wait "$ABEL_PID" 2>/dev/null || true
    fi

    # Stop Abel's Docker Compose services (preserves external Docker containers and daemon)
    if [ "$COMPOSE_STARTED" -eq 1 ] && [ -n "$COMPOSE_FILE" ]; then
        echo "[abel-service] Stopping Docker Compose services..."
        docker compose -f "$COMPOSE_FILE" down 2>/dev/null || true
    fi

    echo "[abel-service] Teardown complete."
    exit 0
}

trap cleanup SIGTERM SIGINT SIGHUP EXIT

# 1. Manage Docker and Docker Compose
if command -v docker >/dev/null 2>&1; then
    if ! is_docker_responsive; then
        start_docker_daemon || true
    fi

    if is_docker_responsive && [ -n "$COMPOSE_FILE" ]; then
        echo "[abel-service] Starting observability stack using $COMPOSE_FILE..."
        if docker compose -f "$COMPOSE_FILE" up -d; then
            COMPOSE_STARTED=1
        else
            echo "[abel-service] WARNING: 'docker compose up -d' encountered errors; proceeding with Abel." >&2
        fi
    elif [ -z "$COMPOSE_FILE" ]; then
        echo "[abel-service] NOTICE: docker-compose.yaml not found; skipping container orchestration."
    fi
else
    echo "[abel-service] NOTICE: Docker CLI not found in PATH; skipping container orchestration."
fi

# 2. Launch Abel binary
echo "[abel-service] Starting Abel binary: $BINARY"
"$BINARY" "$@" &
ABEL_PID=$!

wait "$ABEL_PID"
