# Quickstart & Verification Guide: Simplify & Consolidate Audio Codebase

## Verification Steps

### 1. Build Verification
Confirm the entire Go backend builds cleanly with no unresolved imports or circular dependencies:
```bash
go build -v ./src/backend/...
```

### 2. Backend Unit Test Parity
Execute the backend test suite, verifying all conversion, audio engine, and HTTP handlers pass:
```bash
go test -v ./src/backend/lib/...
```

### 3. Frontend Validation
Run frontend linting, tests, and build check:
```bash
cd src/frontend
npm run check
npm run test
npm run build
```

### 4. End-to-End Sanity Check
Start Abel and confirm audio metering and status endpoints respond:
```bash
go run ./src/backend/main.go &
PID=$!
sleep 2
curl -i http://localhost:8080/api/system/health
kill $PID
```
