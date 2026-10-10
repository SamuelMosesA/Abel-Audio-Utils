#!/usr/bin/env bash
# Abel Standalone macOS Installer
# Installs dependencies directly from official vendor packages / binaries:
# - Node.js: official macOS .pkg installer from nodejs.org
# - Go: official macOS .pkg installer from go.dev
# - FFmpeg & FFprobe: official static macOS binaries from evermeet.cx / osxfx
# - PortAudio: compiled & installed directly from official source tarball (portaudio.com)
# - pkg-config: official pkg-config / pkgconf binary
# - Docker Desktop: official .dmg installer from docker.com
# Then compiles Abel, installs it to /usr/local/bin, and configures launchd autostart on boot.

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
TMP_DIR="$(mktemp -d -t abel-install-XXXXXX)"
trap 'rm -rf "$TMP_DIR"' EXIT

ARCH="$(uname -m)"
case "$ARCH" in
    x86_64)
        NODE_ARCH="x64"
        GO_ARCH="amd64"
        DOCKER_ARCH="amd64"
        ;;
    arm64)
        NODE_ARCH="arm64"
        GO_ARCH="arm64"
        DOCKER_ARCH="arm64"
        ;;
    *)
        log_err "Unsupported CPU architecture: $ARCH"
        exit 1
        ;;
esac

export PATH="/usr/local/bin:/usr/local/sbin:/usr/bin:/bin:/usr/sbin:/sbin:$PATH"

log_info "Detected macOS architecture: $ARCH"

# 0. Check for Command Line Tools (clang, make, etc.)
if ! command -v clang >/dev/null 2>&1 || ! command -v make >/dev/null 2>&1; then
    log_info "Installing Apple Command Line Tools..."
    xcode-select --install || true
    echo "Please complete the Command Line Tools installation dialog if prompted, then press Enter to continue."
    read -r
fi

# 1. Install Node.js (v20 LTS) from official website
if ! command -v node >/dev/null 2>&1 || ! command -v npm >/dev/null 2>&1; then
    NODE_VERSION="v20.18.0"
    log_info "Installing Node.js $NODE_VERSION from nodejs.org..."
    NODE_PKG="node-${NODE_VERSION}.pkg"
    curl -fsSL "https://nodejs.org/dist/${NODE_VERSION}/${NODE_PKG}" -o "${TMP_DIR}/${NODE_PKG}"
    sudo installer -pkg "${TMP_DIR}/${NODE_PKG}" -target /
    rm -f "${TMP_DIR}/${NODE_PKG}"
else
    log_info "Node.js already installed: $(node -v)"
fi

# 2. Install Go compiler from official website
if ! command -v go >/dev/null 2>&1; then
    GO_VERSION="1.25.1"
    # Fallback to 1.24.1 if 1.25 is not yet published
    log_info "Downloading Go from go.dev..."
    GO_PKG="go${GO_VERSION}.darwin-${GO_ARCH}.pkg"
    if ! curl -fsSL "https://go.dev/dl/${GO_PKG}" -o "${TMP_DIR}/${GO_PKG}" 2>/dev/null; then
        GO_VERSION="1.24.1"
        GO_PKG="go${GO_VERSION}.darwin-${GO_ARCH}.pkg"
        curl -fsSL "https://go.dev/dl/${GO_PKG}" -o "${TMP_DIR}/${GO_PKG}"
    fi
    sudo installer -pkg "${TMP_DIR}/${GO_PKG}" -target /
    rm -f "${TMP_DIR}/${GO_PKG}"
else
    log_info "Go already installed: $(go version)"
fi

# 3. Install pkg-config (pkgconf) if missing
if ! command -v pkg-config >/dev/null 2>&1; then
    log_info "Building and installing pkgconf from official source..."
    PKGCONF_VER="2.2.0"
    curl -fsSL "https://distfiles.ariadne.space/pkgconf/pkgconf-${PKGCONF_VER}.tar.xz" -o "${TMP_DIR}/pkgconf.tar.xz"
    tar -xf "${TMP_DIR}/pkgconf.tar.xz" -C "${TMP_DIR}"
    (
        cd "${TMP_DIR}/pkgconf-${PKGCONF_VER}"
        ./configure --prefix=/usr/local
        make -j"$(sysctl -n hw.ncpu || echo 2)"
        sudo make install
        sudo ln -sf /usr/local/bin/pkgconf /usr/local/bin/pkg-config
    )
else
    log_info "pkg-config already installed: $(pkg-config --version)"
fi

# 4. Install PortAudio from official source (portaudio.com)
if [ ! -f "/usr/local/include/portaudio.h" ] && [ ! -f "/usr/include/portaudio.h" ]; then
    log_info "Downloading and building PortAudio from official source (files.portaudio.com)..."
    PORTAUDIO_URL="https://files.portaudio.com/archives/pa_stable_v190700_20210406.tgz"
    curl -fsSL "$PORTAUDIO_URL" -o "${TMP_DIR}/portaudio.tgz"
    tar -xzf "${TMP_DIR}/portaudio.tgz" -C "${TMP_DIR}"
    (
        cd "${TMP_DIR}/portaudio"
        ./configure --prefix=/usr/local --disable-mac-universal
        make -j"$(sysctl -n hw.ncpu || echo 2)"
        sudo make install
    )
else
    log_info "PortAudio headers found in /usr/local/include or /usr/include."
fi

# 5. Install FFmpeg & FFprobe from official static builds
if ! command -v ffmpeg >/dev/null 2>&1 || ! command -v ffprobe >/dev/null 2>&1; then
    log_info "Installing FFmpeg & FFprobe official macOS static binaries..."
    # Download static universal/arch ffmpeg binary from evermeet.cx or osxexperts
    if [ "$ARCH" = "arm64" ]; then
        curl -fsSL "https://evermeet.cx/ffmpeg/getrelease/zip" -o "${TMP_DIR}/ffmpeg.zip"
        curl -fsSL "https://evermeet.cx/ffmpeg/getrelease/ffprobe/zip" -o "${TMP_DIR}/ffprobe.zip"
    else
        curl -fsSL "https://evermeet.cx/ffmpeg/getrelease/zip" -o "${TMP_DIR}/ffmpeg.zip"
        curl -fsSL "https://evermeet.cx/ffmpeg/getrelease/ffprobe/zip" -o "${TMP_DIR}/ffprobe.zip"
    fi
    unzip -q "${TMP_DIR}/ffmpeg.zip" -d "${TMP_DIR}"
    unzip -q "${TMP_DIR}/ffprobe.zip" -d "${TMP_DIR}"
    sudo install -m 755 "${TMP_DIR}/ffmpeg" /usr/local/bin/ffmpeg
    sudo install -m 755 "${TMP_DIR}/ffprobe" /usr/local/bin/ffprobe
else
    log_info "FFmpeg already installed: $(ffmpeg -version | head -n1)"
fi

# 6. Install Docker Desktop from official website if Docker is not installed
if [ ! -d "/Applications/Docker.app" ] && ! command -v docker >/dev/null 2>&1; then
    log_info "Downloading Docker Desktop from docker.com..."
    DOCKER_DMG="Docker.dmg"
    DOCKER_URL="https://desktop.docker.com/mac/main/${DOCKER_ARCH}/Docker.dmg"
    curl -fsSL "$DOCKER_URL" -o "${TMP_DIR}/${DOCKER_DMG}"
    log_info "Mounting and installing Docker Desktop to /Applications..."
    hdiutil attach -nobrowse -quiet "${TMP_DIR}/${DOCKER_DMG}" -mountpoint "${TMP_DIR}/docker_mount"
    sudo cp -R "${TMP_DIR}/docker_mount/Docker.app" /Applications/
    hdiutil detach "${TMP_DIR}/docker_mount" -quiet
    rm -f "${TMP_DIR}/${DOCKER_DMG}"
    log_info "Docker Desktop installed in /Applications/Docker.app"
else
    log_info "Docker is already installed."
fi

# Ensure Docker command line symlinks and default socket are available
if [ -d "/Applications/Docker.app" ]; then
    log_info "Configuring Docker CLI binaries and system paths..."
    sudo ln -sf /Applications/Docker.app/Contents/Resources/bin/docker /usr/local/bin/docker || true
    sudo ln -sf /Applications/Docker.app/Contents/Resources/bin/docker-compose /usr/local/bin/docker-compose || true
    sudo ln -sf /Applications/Docker.app/Contents/Resources/bin/docker-compose /usr/local/bin/docker-compose-v1 2>/dev/null || true
    
    # Enable system socket link if user docker.sock exists
    for sock in "$HOME/.docker/run/docker.sock" /Users/*/.docker/run/docker.sock; do
        if [ -S "$sock" ]; then
            sudo ln -sf "$sock" /var/run/docker.sock 2>/dev/null || true
            break
        fi
    done
fi

# 7. Build Abel (Frontend + Backend) and install to /usr/local
log_info "Building and installing Abel from repository..."
cd "$REPO_DIR"

# Ensure pkg-config can find PortAudio
export PKG_CONFIG_PATH="/usr/local/lib/pkgconfig:${PKG_CONFIG_PATH:-}"

sudo make install PREFIX=/usr/local

# 8. Create macOS launchd daemon plist for automatic start at boot
LAUNCHD_PLIST="/Library/LaunchDaemons/com.abel.service.plist"
log_info "Configuring macOS launchd service at $LAUNCHD_PLIST..."

sudo mkdir -p /var/log/abel
sudo chmod 755 /var/log/abel

# Clean up any existing registered service to prevent error 5 (Already loaded / I/O error)
sudo launchctl bootout system/com.abel.service 2>/dev/null || sudo launchctl unload "$LAUNCHD_PLIST" 2>/dev/null || true

sudo tee "$LAUNCHD_PLIST" > /dev/null << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.abel.service</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/abel-service</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/var/log/abel/abel.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/abel/abel.err</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/usr/local/bin:/usr/local/sbin:/usr/bin:/bin:/usr/sbin:/sbin</string>
    </dict>
</dict>
</plist>
EOF

sudo chown root:wheel "$LAUNCHD_PLIST"
sudo chmod 644 "$LAUNCHD_PLIST"

# 9. Load launchd service
log_info "Starting Abel background service with launchd..."
if ! sudo launchctl bootstrap system "$LAUNCHD_PLIST" 2>/dev/null; then
    sudo launchctl load -w "$LAUNCHD_PLIST"
fi

log_info "Installation complete!"
echo ""
echo "=================================================================="
echo " Abel is installed and registered with macOS launchd."
echo " It will automatically start whenever the Mac boots up,"
echo " initialize Docker and the docker-compose stack, and start Abel."
echo ""
echo " Service Management:"
echo "   View logs:   tail -f /var/log/abel/abel.log"
echo "   Stop:        sudo launchctl bootout system/com.abel.service (or sudo launchctl unload /Library/LaunchDaemons/com.abel.service.plist)"
echo "   Start:       sudo launchctl bootstrap system /Library/LaunchDaemons/com.abel.service.plist (or sudo launchctl load -w /Library/LaunchDaemons/com.abel.service.plist)"
echo "   Config:      /etc/abel/config.yaml (or ~/.config/abel/config.yaml)"
echo "=================================================================="
