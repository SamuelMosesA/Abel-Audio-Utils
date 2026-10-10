#!/usr/bin/env bash
# Abel Service Lifecycle Wrapper
# Manages Docker daemon startup (if needed), Docker Compose stack, Abel binary lifecycle, and clean teardown.

set -e

# Ensure full standard PATH is available across MacPorts and system paths (crucial for launchd/systemd at boot)
export PATH="/opt/local/bin:/opt/local/sbin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin:$PATH"

# Auto-detect standard Docker sockets if DOCKER_HOST is not explicitly configured
detect_docker_socket() {
    if [ -n "$DOCKER_HOST" ]; then
        return 0
    fi
    # 1. Check system default socket first (used by Docker Desktop privileged helper & native Linux)
    if [ -S "/var/run/docker.sock" ]; then
        export DOCKER_HOST="unix:///var/run/docker.sock"
    elif [ -S "$HOME/.docker/run/docker.sock" ]; then
        export DOCKER_HOST="unix://$HOME/.docker/run/docker.sock"
    elif [ -S "$HOME/.colima/default/docker.sock" ]; then
        export DOCKER_HOST="unix://$HOME/.colima/default/docker.sock"
    else
        # If running as root under launchd/daemon, look for active user's socket in /Users/*/
        for user_sock in /Users/*/.docker/run/docker.sock /Users/*/.colima/default/docker.sock; do
            if [ -S "$user_sock" ]; then
                export DOCKER_HOST="unix://$user_sock"
                # If /var/run/docker.sock does not exist, symlink it so root tools find it immediately
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
elif [ -x "/opt/local/bin/abel" ]; then
    BINARY="/opt/local/bin/abel"
elif [ -x "/usr/local/bin/abel" ]; then
    BINARY="/usr/local/bin/abel"
else
    echo "[abel-service] ERROR: Unable to locate 'abel' executable." >&2
    exit 1
fi

# Locate docker-compose.yaml
COMPOSE_FILE=""
CANDIDATE_PATHS=(
    "$ABEL_COMPOSE_FILE"
    "$SCRIPT_DIR/../share/abel/docker-compose.yaml"
    "/opt/local/share/abel/docker-compose.yaml"
    "/usr/local/share/abel/docker-compose.yaml"
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

is_docker_responsive() {
    docker info >/dev/null 2>&1
}

start_docker_daemon() {
    echo "[abel-service] Docker daemon is not active. Attempting to start..."
    OS_TYPE="$(uname -s)"
    if [ "$OS_TYPE" = "Darwin" ]; then
        # Identify currently logged in GUI user (if running as root under launchd)
        CONSOLE_USER="$(stat -f "%Su" /dev/console 2>/dev/null || echo "")"
        if [ "$CONSOLE_USER" = "root" ] || [ -z "$CONSOLE_USER" ]; then
            CONSOLE_USER="$(users 2>/dev/null | awk '{print $1}' || echo "")"
        fi

        if [ -d "/Applications/Docker.app" ]; then
            if [ -n "$CONSOLE_USER" ] && [ "$CONSOLE_USER" != "root" ] && [ "$(id -u)" -eq 0 ]; then
                echo "[abel-service] Launching Docker Desktop as user $CONSOLE_USER..."
                sudo -u "$CONSOLE_USER" open -a Docker --args --unattended 2>/dev/null || open -a Docker --args --unattended 2>/dev/null || true
            else
                open -a Docker --args --unattended 2>/dev/null || true
            fi
        elif [ -d "/Applications/OrbStack.app" ]; then
            if [ -n "$CONSOLE_USER" ] && [ "$CONSOLE_USER" != "root" ] && [ "$(id -u)" -eq 0 ]; then
                sudo -u "$CONSOLE_USER" open -a OrbStack 2>/dev/null || open -a OrbStack 2>/dev/null || true
            else
                open -a OrbStack 2>/dev/null || true
            fi
        elif command -v colima >/dev/null 2>&1; then
            if [ -n "$CONSOLE_USER" ] && [ "$CONSOLE_USER" != "root" ] && [ "$(id -u)" -eq 0 ]; then
                sudo -u "$CONSOLE_USER" colima start 2>/dev/null || colima start 2>/dev/null || true
            else
                colima start 2>/dev/null || true
            fi
            detect_docker_socket
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

    # Wait up to 45 seconds for Docker daemon to become responsive
    local elapsed=0
    while [ $elapsed -lt 45 ]; do
        detect_docker_socket
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
    trap - SIGTERM SIGINT SIGHUP
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

trap cleanup SIGTERM SIGINT SIGHUP

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

EXIT_CODE=0
wait "$ABEL_PID" || EXIT_CODE=$?
echo "[abel-service] Abel binary exited with status $EXIT_CODE"
exit "$EXIT_CODE"
