# Changelog

All notable changes to the Abel Audio Utils project are documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.1] - 2026-10-10

### Changed
- **Homebrew Formula Version**: Bumped formula tag in `abel.rb` from `v0.2.0` to `v0.2.1`.
- **AI Translation Default State**: Disabled and blocked English translation along with all other configured languages by default on startup.

### Fixed
- **Local Network Host Discovery**: Avoid returning loopback (`127.0.0.1`) and virtual container bridges (`docker0`, `br-*`, `utun*`) by prioritizing physical LAN interface addresses (macOS `en*` and Linux `wlan*`/`eth*`).
- **SSID Resolution**: Removed redundant and redacted Wi-Fi SSID lookups across macOS and Linux, streamlining endpoint connectivity.

## [0.2.0] - 2026-10-10

### Added
- **Homebrew Service Docker & Compose Lifecycle (`010-brew-service-docker-lifecycle`)**:
  - Implemented `scripts/abel-service.sh` service wrapper supporting automated Docker daemon startup (macOS Docker Desktop / OrbStack / Colima and Linux systemd).
  - Automated `docker compose up -d` execution for the full observability stack (OpenTelemetry Collector, Loki, Prometheus, Grafana).
  - Deterministic signal trapping (`SIGTERM`, `SIGINT`, `SIGHUP`) on `brew services stop abel`, ensuring graceful shutdown of Abel, teardown of compose containers via `docker compose down`, and Docker daemon stoppage.
  - Packaged `docker-compose.yaml` and `observability/` configs into Homebrew's `pkgshare` achievements.
- **In-Memory Progressive MP3 Live Streaming (`009-replace-hls-with-mp3`)**:
  - Replaced disk-based HLS playlist and segmenting with high-throughput, low-latency in-memory MP3 broadcaster (`LiveAudioBroadcaster`).
  - Added concurrent circular chunk ring buffer with zero disk I/O and instant HTTP chunked transfer (`GET /api/audio/stream`).
  - Added native HTML5 `<audio>` playback across iOS/Safari, Android, macOS, and Windows with sub-50ms latency.
  - Added per-language translation audio killswitch controls with disabled-by-default initial state.
- **Client-Side Admin Route Protection (`008-admin-unauth-redirect`)**:
  - Implemented SvelteKit layout route guard enforcing authentication checks on all `/admin/*` views.
  - Automatically redirects unauthenticated sessions to `/admin/login` with sanitized return parameters.
- **Lock-Free Audio Pipeline & Concurrency Hardening (`007-audio-core-and-system-refactor`)**:
  - Streamlined audio engine capture loop directly into a dedicated single-threaded recording worker without per-chunk mutex locking.
  - Adopted generic typed concurrency maps (`zolstein/sync-map`) across OpenAI translation and transcription managers.
  - Added Wi-Fi SSID network discovery via macOS `ipconfig getsummary en0` and external IP/port resolution.
  - Added dynamic QR code generation displaying `ip:port` for easy client onboarding.
- **Session Authentication Hardening (`006-auth-session-hardening`)**:
  - Added cryptographically random dynamic session secret generation on startup.
  - Added server-side session revocation endpoint (`DELETE /api/auth/session`) and cookie invalidation.
- **DSP Audio Boost & UI Refresh Controls (`005-ui-controls-and-dsp-refactor`)**:
  - Extracted chunk amplification and soft sample clamping into `audio_processing`.
  - Added per-language AI listener count indicators and persistent translation blocking.
  - Replaced continuous recording list polling loops with on-demand manual refresh triggers.
- **Audio Engine Restart Synchronization (`004-fix-engine-restart-crash`)**:
  - Added mutex-guarded lifecycle transitions and PortAudio re-initialization safety.
  - Added HTTP 409 Conflict rejection preventing concurrent restart collisions.

### Changed
- **Homebrew Formula Version**: Bumped formula tag in `abel.rb` from `v0.1.0` to `v0.2.0`.
- **Frontend Architecture & Design Tokens (`002-simplify-codebase-and`)**:
  - Consolidated scattered CSS utility classes into centralized CSS variables and tokens in `app.css`.
  - Audited and refactored UI components to follow flat shadcn/ui design conventions.
- **Audio Conversion Architecture (`001-simplify-and-consolidate`)**:
  - Consolidated scattered audio conversion, resampling, PCM transformations, and WAV header encoding into cohesive `audio_processing` domain package.
  - Retired deprecated conversion utilities.

### Removed
- **HLS Segmenting & Disk Artifacts (`009-replace-hls-with-mp3`)**:
  - Removed `hls.go`, `hls_test.go`, HLS playlist generation, and disk segment cleaner routines.
  - Eliminated "HLS playlist not ready" warnings and disk write overhead.
- **Redundant Authentication Aliases (`006-auth-session-hardening`)**:
  - Removed duplicate `/auth/logout` endpoints in favor of restful session management.

### Fixed
- **OpenTelemetry Build Failure (`003-fix-otel-build-failure`)**:
  - Resolved `undefined: log.Int64Value` compiler error by updating `otelslog` bridge dependency for Go 1.23+ compatibility.
  - Fixed pathname JSON serialization in Homebrew formula.
- **PortAudio Stream Teardown Order**:
  - Fixed audio engine shutdown sequence by enforcing `stream.Stop()` prior to `stream.Close()`, eliminating PortAudio ALSA/CoreAudio termination timeout warnings.
