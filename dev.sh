#!/bin/bash
set -e

# Change to script directory
cd "$(dirname "$0")"

# Ensure brew is in PATH
for path in /home/linuxbrew/.linuxbrew/bin/brew /opt/homebrew/bin/brew /usr/local/bin/brew; do
    [ -x "$path" ] && eval "$($path shellenv)" && break
done
command -v brew &>/dev/null || { echo "Homebrew not found!"; exit 1; }

# Package source code
export ABEL_DEV_TARBALL="${ABEL_DEV_TARBALL:-/tmp/abel-source.tar.gz}"
trap 'rm -f "$ABEL_DEV_TARBALL"' EXIT
echo "==> Packaging source code..."
tar -czf "$ABEL_DEV_TARBALL" --exclude=.go --exclude=node_modules --exclude=src/frontend/node_modules .

# Ensure local tap exists
TAP_DIR="$(brew --repository local/abel 2>/dev/null || true)"
if [ -z "$TAP_DIR" ] || [ ! -d "$TAP_DIR" ]; then
    echo "==> Initializing local Homebrew tap..."
    brew tap-new local/abel
    TAP_DIR="$(brew --repository local/abel)"
fi

# Copy and patch formula in the tap to use local tarball
mkdir -p "$TAP_DIR/Formula"
cp ./abel.rb "$TAP_DIR/Formula/abel.rb"
SHA_SUM=$(sha256sum "$ABEL_DEV_TARBALL" | awk '{print $1}')
ruby -i -pe "sub(/url \".*\", tag: \".*\"/, \"url \\\"file://$ABEL_DEV_TARBALL\\\"\\n  sha256 \\\"$SHA_SUM\\\"\\n  version \\\"1.0.0-dev\\\"\")" "$TAP_DIR/Formula/abel.rb"

# Reinstall/install from the local tap
echo "==> Installing/updating Abel via Homebrew..."
if brew list local/abel/abel &>/dev/null; then
    brew reinstall --build-from-source local/abel/abel
else
    brew install --build-from-source local/abel/abel
fi

echo "==> Starting Abel server..."
exec abel
