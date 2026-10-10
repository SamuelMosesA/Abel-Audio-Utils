# Quickstart Validation Guide: UI Controls, Per-Language AI Killswitch, and DSP Refactor

**Branch**: `005-ui-controls-and-dsp-refactor`
**Date**: 2026-10-10

## Prerequisites
- Go 1.24+ installed
- Bun or Node.js with npm installed
- PortAudio development libraries installed

## Validation Steps

### 1. DSP Extraction Unit Tests
Verify that the stereo chunk extraction and clamping utility passes all boundary tests:
```bash
go test -v ./src/backend/lib/audioengine/conversion -run TestExtractStereoChunk
```
Expected output:
- PASS: Samples properly amplified by boost.
- PASS: Samples clamped within `[-1.0, 1.0]`.
- PASS: Non-positive boost (0 or negative) safely defaults to unity gain (1.0).

### 2. Backend Web AI Endpoints Unit Tests
Verify that the per-language killswitch actions, query statuses, and SSE/HLS blocking behave as expected:
```bash
go test -v ./src/backend/lib/web -run TestAIStreams
```
Expected output:
- PASS: `POST /api/ai/streams` can toggle language block status.
- PASS: `GET /api/ai/streams` reports configured languages with listener metrics.
- PASS: Blocked language terminates active translation session and rejects subtitle / audio requests.

### 3. Frontend Unit Tests
Verify Svelte components and state:
```bash
cd src/frontend
bun run test:unit
```
Expected output:
- All Vitest tests pass without regressions.

### 4. End-to-End Verification in Browser
1. Start backend: `go run src/backend/main.go`
2. Start frontend: `cd src/frontend && bun run dev`
3. Open `http://localhost:5173/admin`:
   - Inspect **Recording Library**: Check network tab. Verify no requests to `/api/recordings/library` occur periodically. Click **Refresh** and verify requests occur on demand.
   - Inspect **AI Translation Panel**: Verify all configured languages are displayed with listener counts.
   - Click the **Killswitch** for a language: verify its status toggles to Blocked, verify any running stream stops, and verify clicking the AI panel Refresh button reflects the persisted runtime state.
