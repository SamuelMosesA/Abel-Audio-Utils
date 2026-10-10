# Quickstart & Verification Guide: Audio Core Pipeline, OpenAI Managers, and Network Refactor

**Feature Branch**: `007-audio-core-and-system-refactor`
**Date**: 2026-10-10
**Spec**: [spec.md](./spec.md)

## Runnable Validation Scenarios

This quickstart guides developers and reviewers through verifying each architectural enhancement end-to-end.

---

### Scenario 1: Audio Processing & Clamping Unit Tests
Validate that sample clamping, PCM16 Little-Endian byte serialization, WAV header generation, and HLS packaging are functional and test-covered in the new `audio_processing` package.

```bash
# Run unit tests on audio_processing package
go test -v -race ./src/backend/lib/audioengine/audio_processing/...
```

**Expected Outcome**:
- `TestConvertStereoFloat32ToPCM16LE` verifies clamping strictly bounds inputs to `[-1.0, 1.0]` and correctly converts to 16-bit integer values (`32767` and `-32767`).
- All WAV header tests pass without dependencies on the deprecated `conversion` package.

---

### Scenario 2: Lock-Free Audio Recording Storage
Verify that the storage worker records audio cleanly without lock contention or races.

```bash
# Run audioengine storage tests
go test -v -race ./src/backend/lib/audioengine/...
```

**Expected Outcome**:
- Inner chunk write loop in `storage.go` executes with zero mutex acquisitions.
- Bit-exact WAV captures are written and finalized cleanly upon stop.
- Race detector reports 0 data races.

---

### Scenario 3: Typed OpenAI Session Registry & Reconnect Buffering
Verify that `TranslationManager` and `TranscriptionManager` operate with typed session registries and keep bounded audio during socket reconnections.

```bash
# Run openai manager tests
go test -v -race ./src/backend/lib/openai/...
```

**Expected Outcome**:
- Sessions and subscriber registrations are managed through `TypedSessionMap` and `TypedSubscriberMap` without untyped `sync.Map` casts.
- 15-second bounded FIFO preserves pending audio during simulated WebSocket drops.

---

### Scenario 4: Network Discovery & QR Code Endpoint
Verify the `/api/system/connection` endpoint returns the external IP, port, display endpoint, and Wi-Fi SSID.

```bash
# Query the connection endpoint
curl -s http://localhost:8080/api/system/connection | jq .
```

**Expected Output**:
```json
{
  "serverUrl": "http://192.168.1.120:8080",
  "host": "192.168.1.120",
  "port": "8080",
  "displayEndpoint": "192.168.1.120:8080",
  "ssid": "Auditorium-WiFi"
}
```

Verify that on the landing page:
- The QR code renders targeting `serverUrl`.
- The text underneath reads `192.168.1.120:8080` (or `ip:port`).

---

### Scenario 5: Real-Time Recording Updates & Master Refresh
Verify the admin console updates automatically on job completion and handles master refresh.

1. Start audio recording and stop it.
2. Observe SSE stream on `/api/system/changelog`:
   ```bash
   curl -N -H "Accept: text/event-stream" http://localhost:8080/api/system/changelog
   ```
   Event received: `data: {"section":"recording"}`.
3. In the UI, notice the recording list automatically updates with the newly completed recording without manual click.
4. Click the Master Refresh button in the console header. Verify all panels refresh simultaneously.
