# Abel

Abel (anti-babel) is a high-performance, web-based audio recording, AI translation & streaming interface designed for professional USB audio interfaces. Built with Go (backend), SvelteKit (frontend), structured `slog` logging, and OpenTelemetry integrations.

## Features

- **Real-time Monitoring**: Visual feedback via high-performance dB meters and waveforms.
- **Stereo Recording**: Support for dual-channel recording with configurable routing.
- **Digital Gain Boost**: Adjust input levels digitally before recording.
- **Recording Processing**: On stop or import, trim edge silence, normalize, and export a mono MP3. The admin library shows FFmpeg progress and supports optional manual trimming.
- **File Management**: Play processed MP3s and create revised trims from the browser. Original recordings and imported audio remain in storage.
- **Cloud Integration**: Automatically push the initial and manually trimmed MP3s to the configured cloud drive directory, using readable date-based filenames and numbered copies when names collide.
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
- **FFmpeg and FFprobe**: Runtime AAC/HLS packaging and recording processing, including an FFmpeg build with `libmp3lame`, `dynaudnorm`, and `loudnorm`
  - macOS: `brew install ffmpeg`
  - Linux: `sudo apt-get install ffmpeg`

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
brew install SamuelMosesA/Abel-Audio-Utils/abel
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

### Processing recordings

Stopping a recording queues background processing. You can also import WAV, MP3, M4A, FLAC, AAC, or OGG files from your device with **Choose files**, drag and drop, or clipboard paste in the admin recordings list. Imports use the same automatic processing and push flow. Abel conservatively trims silence at the beginning and end, preserves pauses in speech, applies the same cleanup chain as [process-sermon](https://github.com/vavilovm/process-sermon), and writes a `-processed.mp3` export beside the source audio. Once complete, it pushes the MP3 to `cloud_drive_location` with a date-based name. Abel also checks for generated MP3s missing from cloud at startup and every minute, then pushes them automatically. The admin list shows upload and processing progress, cloud paths, push status, and failures. Use **Stop** to cancel queued or running processing. If processing a WAV fails, the original WAV can be pushed from the admin library.

To make a manual cut, open **Trim** after the processed MP3 is ready. Enter start and end timecodes in `mm:ss` or `hh:mm:ss`, or use the current playback position. **Create and push trimmed MP3** creates a revised export and pushes it automatically. Cloud names start with the recording date and time: the initial copy is `YYYY-MM-DD_HH-mm-ss.mp3`, and a cut is `YYYY-MM-DD_HH-mm-ss-trimmed.mp3`. If a name is taken, Abel appends `-2`, `-3`, and so on. The assigned names are recorded in `.abel-cloud-names.json` in the recordings folder so retries and restarts keep the same destination. Existing cloud copies retain their older names. The original audio remains in storage for future revisions. Processing jobs are kept in memory; finished MP3s and their cloud copies are discovered from disk after a restart.

## Spec-Driven Development & AI Workflows (Spec Kit)

This repository uses [GitHub Spec Kit](https://github.com/github/spec-kit) (`specify`) for specification-driven development, AI coding agent workflows (Claude Code, OpenAI Codex, Antigravity, etc.), and structured bug triage.

### 1. Spec Kit Installation

Install the `specify` CLI tool using `uv` (recommended) or `pipx`:

```bash
# Recommended (via uv)
uv tool install specify-cli --from git+https://github.com/github/spec-kit.git

# Alternative (via pipx)
pipx install specify-cli

# Verify installation
specify --version
```

> **Note**: Ensure Python 3.11+ is installed and your tool binary path (e.g. `~/.local/bin`) is included in your `$PATH`.

### 2. Project Initialization & Agent Setup

#### Initializing in a Fresh Clone
To initialize Spec Kit in this project for your preferred AI coding agent with the bug extension pre-configured (using `--force` to merge/overwrite in the existing repository without interactive prompts):

```bash
# For Claude Code
specify init --here --force --integration claude --extension bug

# For OpenAI Codex CLI
specify init --here --force --integration codex --extension bug

# For Antigravity (AGY)
specify init --here --force --integration agy --extension bug
```

#### Adding Agent Integrations to an Existing Project
If Spec Kit is already initialized in your working copy, you can install or switch between agent integrations:

```bash
# Install Claude Code integration
specify integration install claude

# Install Codex CLI integration
specify integration install codex

# Install Antigravity integration
specify integration install agy

# List installed and available integrations
specify integration list

# Switch active integration
specify integration switch <claude|codex|agy>
```

### 3. Adding & Managing the Bug Extension

The **Bug Triage Workflow Extension** provides a standardized 3-stage workflow (`assess`, `fix`, `test`) where reports are stored under `.specify/bugs/<slug>/`.

To install and enable the bug extension:

```bash
# Install the bundled bug extension
specify extension add bug

# Verify installed extensions
specify extension list
```

To disable or re-enable the extension:
```bash
specify extension disable bug
specify extension enable bug
```

### 4. Bug Triage Workflow

Coding agents (Claude Code, Codex, Antigravity, etc.) can drive the 3-step bug triage lifecycle:

1. **Assess a Bug**:
   Analyze a bug report from an issue URL or error description, locate suspected code paths, and propose a remediation plan without modifying source code:
   ```text
   /speckit.bug.assess "Issue description or stack trace" slug=<kebab-case-slug>
   # or with an issue URL:
   /speckit.bug.assess https://github.com/SamuelMosesA/Abel-Audio-Utils/issues/123 slug=<slug>
   ```
   *(Generates `.specify/bugs/<slug>/assessment.md`)*

2. **Fix the Bug**:
   Apply the proposed remediation from the assessment:
   ```text
   /speckit.bug.fix slug=<slug>
   ```
   *(Generates `.specify/bugs/<slug>/fix.md`)*

3. **Validate & Test**:
   Run reproductions and automated tests to verify the fix:
   ```text
   /speckit.bug.test slug=<slug>
   ```
   *(Generates `.specify/bugs/<slug>/test.md`)*

*(Note: Depending on your agent and integration, slash commands can also be invoked with hyphens, e.g. `/speckit-bug-assess`.)*

## Project Structure

- `src/backend/main.go` - Application entry point with configuration fallback path resolution.
- `src/backend/lib/` - Go backend modules (AI connectors, web routing, audio recording engine, telemetry handlers).
- `src/backend/static/` - Built SvelteKit frontend assets embedded in the backend binary.
- `src/frontend/` - SvelteKit source code.
- `config/` - Local configuration files.
- `observability/` - OpenTelemetry Collector, Prometheus, and Grafana service configurations.

## License

MIT
