# Quickstart: Build, Installation & Service Verification

**Feature**: [spec.md](spec.md) | **Feature Directory**: `specs/003-fix-otel-build-failure`

## 1. Direct Backend Binary Build

Verify that Go compiles `main.go` cleanly with exit code 0:

```bash
# Compile binary to temporary test destination
/home/linuxbrew/.linuxbrew/bin/go build -o /tmp/abel-test src/backend/main.go
```

Expected outcome:
- Clean 0 exit status
- Binary `/tmp/abel-test` created without undefined symbol errors

## 2. User Configuration Setup

Ensure user configuration directory and configuration file exist before service boot:

```bash
mkdir -p ~/.config/abel
cp /home/linuxbrew/.linuxbrew/etc/abel/config.yaml ~/.config/abel/config.yaml
```

Expected outcome:
- `~/.config/abel/config.yaml` is accessible

## 3. Launching Abel Service

Run the compiled executable to verify startup and telemetry initialization:

```bash
/home/linuxbrew/.linuxbrew/opt/abel/bin/abel
```

Expected outcome:
- Logs `INFO Starting Abel... component=main`
- Logs `INFO Loading configuration component=main path=~/.config/abel/config.yaml`
- Logs `INFO Web UI active component=main url=http://<host>:8080`
