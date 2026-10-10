#!/usr/bin/env bash
# Abel macOS Installer using MacPorts
# Installs all build and runtime dependencies, sets up Docker via Colima,
# builds Abel, installs system-wide via MacPorts, and configures launchd autostart on boot.

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${BLUE}==>${NC} ${GREEN}$1${NC}"; }
log_warn() { echo -e "${YELLOW}==> WARNING:${NC} $1"; }
log_err() { echo -e "${RED}==> ERROR:${NC} $1"; }

if [ "$(uname -s)" != "Darwin" ]; then
    log_err "This installer is designed for macOS (Darwin)."
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# 1. Verify MacPorts installation
log_info "Checking MacPorts installation..."
if ! command -v port >/dev/null 2>&1; then
    if [ -x "/opt/local/bin/port" ]; then
        export PATH="/opt/local/bin:/opt/local/sbin:$PATH"
    else
        log_err "MacPorts ('port' command) was not found."
        echo "Please install MacPorts from https://www.macports.org/install.php and re-run this script."
        exit 1
    fi
fi

# Ensure MacPorts paths are in current PATH
export PATH="/opt/local/bin:/opt/local/sbin:$PATH"

# 2. Package List Definition
# Dependencies from README.md and Portfile:
# - go: Go compiler (1.25+)
# - nodejs20, npm10: Frontend toolchain
# - pkgconfig: Required by CGO & PortAudio build
# - portaudio: Audio I/O library headers
# - ffmpeg: Audio post-processing, normalization, and streaming
# - docker, docker-compose-plugin: Docker CLI and compose tool
# - colima: Container runtime providing the Docker daemon on macOS
PACKAGES=(
    "go"
    "nodejs20"
    "npm10"
    "pkgconfig"
    "portaudio"
    "ffmpeg"
    "docker"
    "docker-compose-plugin"
    "colima"
)

log_info "Updating MacPorts tree..."
sudo port selfupdate || log_warn "Port selfupdate had warnings, continuing..."

log_info "Installing required packages via MacPorts:"
for pkg in "${PACKAGES[@]}"; do
    echo "  - $pkg"
done

sudo port install "${PACKAGES[@]}"

# 3. Ensure Docker / Colima autostart service is configured
log_info "Configuring Docker runtime (Colima)..."
if ! command -v docker >/dev/null 2>&1; then
    log_err "Docker CLI was not installed successfully."
    exit 1
fi

# Start Colima if Docker daemon is not active
if ! docker info >/dev/null 2>&1; then
    log_info "Starting Colima VM..."
    colima start || log_warn "Colima start exited with a warning, continuing..."
fi

# Enable Colima background service via MacPorts or launchd if supported
if sudo port installed colima 2>/dev/null | grep -q "(active)"; then
    sudo port load colima 2>/dev/null || true
fi

# 4. Install Abel via MacPorts Portfile
log_info "Installing Abel using Portfile in $REPO_DIR..."
cd "$REPO_DIR"

# Clean any existing build artifacts
sudo port clean abel 2>/dev/null || true

# Install Abel from local Portfile
sudo port install

# 5. Enable Abel launchd service to start on boot
log_info "Enabling Abel auto-start service (launchd)..."
sudo port load abel

log_info "Installation completed successfully!"
echo ""
echo "=================================================================="
echo " Abel is installed and loaded as a system service."
echo " It will automatically start when the computer boots up,"
echo " initialize the Docker daemon/compose stack, and start Abel."
echo ""
echo " Status & Management Commands:"
echo "   View Logs:   tail -f /opt/local/var/log/abel/abel.log"
echo "   Stop:        sudo port unload abel"
echo "   Start:       sudo port load abel"
echo "   Config:      /opt/local/etc/abel/config.yaml"
echo "=================================================================="
