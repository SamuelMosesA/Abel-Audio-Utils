#!/usr/bin/env bash
# Abel Service Lifecycle Wrapper
# Manages Docker Compose stack (observability) and Abel binary lifecycle.
# Does NOT attempt to start or stop the Docker daemon itself.

set -e

# Ensure full standard PATH is available across environments (crucial for launchd/systemd at boot)
export PATH="/usr/local/bin:/usr/local/sbin:/opt/local/bin:/opt/local/sbin:/usr/bin:/bin:/usr/sbin:/sbin:$PATH"

# Auto-detect standard Docker sockets if DOCKER_HOST is not explicitly configured
detect_docker_socket() {
    if [ -n "$DOCKER_HOST" ]; then
        return 0
    fi
    if [ -S "/var/run/docker.sock" ]; then
        export DOCKER_HOST="unix:///var/run/docker.sock"
    elif [ -S "$HOME/.docker/run/docker.sock" ]; then
        export DOCKER_HOST="unix://$HOME/.docker/run/docker.sock"
    elif [ -S "$HOME/.colima/default/docker.sock" ]; then
        export DOCKER_HOST="unix://$HOME/.colima/default/docker.sock"
    else
        for user_sock in /Users/*/.docker/run/docker.sock /Users/*/.colima/default/docker.sock; do
            if [ -S "$user_sock" ]; then
                export DOCKER_HOST="unix://$user_sock"
                if [ ! -e "/var/run/docker.sock" ] && [ "$(id -u)" -eq 0 ]; then
                    ln -sf "$user_sock" /var/run/docker.sock 2>/dev/null || true
                fi
                break
            fi
        done
    fi
}
detect_docker_socket

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Locate Abel binary
if [ -n "$ABEL_BIN" ] && [ -x "$ABEL_BIN" ]; then
    BINARY="$ABEL_BIN"
elif [ -x "$SCRIPT_DIR/abel" ]; then
    BINARY="$SCRIPT_DIR/abel"
elif command -v abel >/dev/null 2>&1; then
    BINARY="$(command -v abel)"
elif [ -x "/usr/local/bin/abel" ]; then
    BINARY="/usr/local/bin/abel"
elif [ -x "/opt/local/bin/abel" ]; then
    BINARY="/opt/local/bin/abel"
else
    echo "[abel-service] ERROR: Unable to locate 'abel' executable." >&2
    exit 1
fi

# Locate docker-compose.yaml
COMPOSE_FILE=""
CANDIDATE_PATHS=(
    "$ABEL_COMPOSE_FILE"
    "$SCRIPT_DIR/../share/abel/docker-compose.yaml"
    "/usr/local/share/abel/docker-compose.yaml"
    "/opt/local/share/abel/docker-compose.yaml"
    "/usr/share/abel/docker-compose.yaml"
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

cleanup() {
    trap - SIGTERM SIGINT SIGHUP
    echo "[abel-service] Initiating graceful shutdown..."

    if [ -n "$ABEL_PID" ] && kill -0 "$ABEL_PID" 2>/dev/null; then
        echo "[abel-service] Stopping Abel binary (PID $ABEL_PID)..."
        kill -TERM "$ABEL_PID" 2>/dev/null || true
        wait "$ABEL_PID" 2>/dev/null || true
    fi

    # Stop Abel's Docker Compose services (does NOT stop Docker daemon)
    if [ "$COMPOSE_STARTED" -eq 1 ] && [ -n "$COMPOSE_FILE" ]; then
        echo "[abel-service] Stopping Docker Compose services..."
        docker compose -f "$COMPOSE_FILE" down 2>/dev/null || docker-compose -f "$COMPOSE_FILE" down 2>/dev/null || true
    fi

    echo "[abel-service] Teardown complete."
    exit 0
}

trap cleanup SIGTERM SIGINT SIGHUP

# 1. Manage Docker Compose stack (if docker is available and running)
if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
    if [ -n "$COMPOSE_FILE" ]; then
        echo "[abel-service] Starting observability stack using $COMPOSE_FILE..."
        if docker compose -f "$COMPOSE_FILE" up -d 2>/dev/null || docker-compose -f "$COMPOSE_FILE" up -d 2>/dev/null; then
            COMPOSE_STARTED=1
        else
            echo "[abel-service] WARNING: Docker compose failed to start; continuing with Abel." >&2
        fi
    fi
else
    echo "[abel-service] NOTICE: Docker daemon not running or docker CLI not found; skipping docker compose."
fi

# 2. Launch Abel binary
echo "[abel-service] Starting Abel binary: $BINARY"
"$BINARY" "$@" &
ABEL_PID=$!

EXIT_CODE=0
wait "$ABEL_PID" || EXIT_CODE=$?
echo "[abel-service] Abel binary exited with status $EXIT_CODE"
exit "$EXIT_CODE"
