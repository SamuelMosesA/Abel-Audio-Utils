#!/usr/bin/env bash
# Abel Service Lifecycle Wrapper
# Manages Abel binary lifecycle and clean signal forwarding.

set -e

# Ensure full standard PATH is available across environments (crucial for launchd/systemd at boot)
export PATH="/usr/local/bin:/usr/local/sbin:/opt/local/bin:/opt/local/sbin:/usr/bin:/bin:/usr/sbin:/sbin:$PATH"

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

ABEL_PID=""

cleanup() {
    trap - SIGTERM SIGINT SIGHUP
    echo "[abel-service] Initiating graceful shutdown..."

    if [ -n "$ABEL_PID" ] && kill -0 "$ABEL_PID" 2>/dev/null; then
        echo "[abel-service] Stopping Abel binary (PID $ABEL_PID)..."
        kill -TERM "$ABEL_PID" 2>/dev/null || true
        wait "$ABEL_PID" 2>/dev/null || true
    fi

    echo "[abel-service] Teardown complete."
    exit 0
}

trap cleanup SIGTERM SIGINT SIGHUP

# Launch Abel binary
echo "[abel-service] Starting Abel binary: $BINARY"
"$BINARY" "$@" &
ABEL_PID=$!

EXIT_CODE=0
wait "$ABEL_PID" || EXIT_CODE=$?
echo "[abel-service] Abel binary exited with status $EXIT_CODE"
exit "$EXIT_CODE"
