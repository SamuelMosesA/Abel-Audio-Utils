# Abel

Abel (anti-babel) is a high-performance, web-based audio recording, AI translation & streaming interface designed for professional USB audio interfaces. Built with Go (backend), SvelteKit (frontend), structured `slog` logging, and OpenTelemetry integrations.

## Features

- **Real-time Monitoring**: Visual feedback via high-performance dB meters and waveforms.
- **Stereo Recording**: Support for dual-channel recording with configurable routing.
- **Digital Gain Boost**: Adjust input levels digitally before recording.
- **File Management**: List, play back, and manage your recordings directly from the browser.
- **Cloud Integration**: Push recordings to a configured cloud drive location with a click.
- **Multi-Client Sync**: WebSocket-based state synchronization across multiple open tabs.
- **AI Live Subtitles & Translation**: Real-time translation and transcription via **OpenAI Realtime API** with Server-Sent Events (SSE) subtitle delivery.
- **Centralized Telemetry**: Full OpenTelemetry OTLP integration emitting structured application logs (`slog` log handler) and performance metrics (loop latency, write latency, active/dropped connections, AI token consumption).
- **Client Error Hook**: Automatic collection of frontend runtime crashes, pushing stack traces back to the server telemetry pipeline.
- **Production Packaging**: Native macOS Homebrew Tap formula with a background service (`launchd`).

## Prerequisites

- **Go**: 1.25 or higher
- **Node.js & npm**: For building the frontend
- **PortAudio**: Development headers for audio I/O
  - macOS: `brew install portaudio`
  - Linux: `sudo apt-get install portaudio19-dev`

## Installation & Setup

### Local Development

1. **Clone the repository**:
   ```bash
   git clone https://github.com/SamuelMosesA/Abel-Audio-Utils.git
   cd Abel-Audio-Utils
   ```

2. **Run the Dev Script**:
   Run the local helper script to package, compile, and start the app using Homebrew:
   ```bash
   ./dev.sh
   ```
   This script packages the source code, runs `brew install --build-from-source ./abel.rb`, sets up a config template at `~/.config/abel/config.yaml` if not present, and starts the `abel` server.

### Native Installation (Homebrew)

Install via your custom Homebrew Tap:
```bash
brew tap SamuelMosesA/Abel-Audio-Utils
brew install abel
```

Start the application as a background service:
```bash
brew services start abel
```
Configure it by copying the template config to your user directory:
```bash
mkdir -p ~/.config/abel
cp /opt/homebrew/etc/abel/config.yaml ~/.config/abel/config.yaml
```
Then edit `~/.config/abel/config.yaml`.

## Usage

1. **Start the Observability Stack (Optional)**:
   Launch Loki, Prometheus, Grafana, and the OTel Collector to view centralized telemetry:
   ```bash
   docker-compose up -d
   ```
   Navigate to Grafana at `http://localhost:3000`.

2. **Run the Server**:
   ```bash
   brew services start abel
   ```
   The application strictly loads the configuration from your user home directory `~/.config/abel/config.yaml`.

3. **Access the UI**:
   Open `http://localhost:8080` (or your configured port).

## Project Structure

- `src/backend/main.go` - Application entry point with configuration fallback path resolution.
- `src/backend/lib/` - Go backend modules (AI connectors, web routing, audio recording engine, telemetry handlers).
- `src/backend/static/` - Built SvelteKit frontend assets embedded in the backend binary.
- `src/frontend/` - SvelteKit source code.
- `config/` - Local configuration files.
- `observability/` - OpenTelemetry Collector, Prometheus, and Grafana service configurations.

## License

MIT
