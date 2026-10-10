#!/bin/bash
set -e

# Change to script directory
cd "$(dirname "$0")"

# Check required build tools
command -v go &>/dev/null || { echo "Go not found! Please install Go 1.25+"; exit 1; }
command -v node &>/dev/null || { echo "Node.js not found! Please install Node.js 20+"; exit 1; }
command -v npm &>/dev/null || { echo "npm not found! Please install npm"; exit 1; }

# Build frontend
echo "==> Building frontend..."
cd src/frontend
npm install
npm run build
cd ../..

# Copy built assets to backend static directory
echo "==> Syncing static assets..."
mkdir -p src/backend/static
rm -rf src/backend/static/*
cp -r src/frontend/build/* src/backend/static/

# Build backend
echo "==> Building backend..."
mkdir -p bin
go build -o bin/abel src/backend/main.go

# Ensure user config directory and default config exist
mkdir -p "$HOME/.config/abel"
if [ ! -f "$HOME/.config/abel/config.yaml" ]; then
    echo "==> Generating default config at ~/.config/abel/config.yaml..."
    cp config/config-example.yaml "$HOME/.config/abel/config.yaml"
fi

echo "==> Starting Abel server..."
exec ./bin/abel
