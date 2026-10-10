# Quickstart: Validating Crash-Resilient Audio Engine Restart

## Overview
This guide describes how to validate the audio engine restart fix, specifically verifying that single, rapid, and concurrent restart requests execute safely without crashes (`SIGSEGV`) or unhandled panics.

## Validation Steps

### 1. Automated Test Suite (Race Detection)
Run the Go unit tests for `audioengine` with the race detector enabled:

```bash
go test -v -race ./src/backend/lib/audioengine/...
```

Expected result: PASS with zero race warnings.

### 2. Live Process Engine Restart Test
Launch the backend server, select an audio interface, and issue a restart request:

```bash
# Start server in background
/tmp/abel-test &
SERVER_PID=$!
sleep 2

# Authenticate
curl -s -c /tmp/cookies.txt -b /tmp/cookies.txt -X POST http://localhost:8080/api/auth/session \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "admin"}'

# Select an audio device
curl -s -b /tmp/cookies.txt -X PATCH http://localhost:8080/api/audio/config \
  -H "Content-Type: application/json" \
  -d '{"deviceID": 0}'

# Trigger restart
curl -s -b /tmp/cookies.txt -X POST http://localhost:8080/api/audio/restart

# Verify server process is still alive
kill -0 $SERVER_PID
```

Expected result: Server process remains running; returns HTTP 200 with refreshed device list and reconnected device name.

### 3. Concurrent Burst Restart Stress Test
Fire 5 simultaneous restart requests while the engine is streaming:

```bash
for i in {1..5}; do
  curl -s -b /tmp/cookies.txt -X POST http://localhost:8080/api/audio/restart &
done
wait

kill -0 $SERVER_PID
```

Expected result: No `SIGSEGV`, no channel closure panic, zero crashes. Server continues running normally.
