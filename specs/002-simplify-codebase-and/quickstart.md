# Quickstart: Validation & Verification Guide

## 1. Quick Frontend Validation (Run on every change)

```bash
cd src/frontend
bun run test:unit
bun run check
bun run build
```
Expected output:
- `Test Files 5 passed (5)`
- `svelte-check: 0 errors`
- `Wrote site to "build" - ✔ done`

## 2. Line of Code Measurement (Size Reduction Audit)

```bash
# Check current total lines of code
find src -type f \( -name "*.go" -o -name "*.svelte" -o -name "*.ts" \) -exec wc -l {} + | tail -n 1
```
Target: Measurable decrease towards 75% baseline.

## 3. Go Docstring Coverage Validation

```bash
# Inspect docstrings on exported functions
python3 -c '
import os, re
missing = []
for root, _, files in os.walk("src/backend/lib"):
    for f in files:
        if f.endswith(".go") and not f.endswith("_test.go"):
            path = os.path.join(root, f)
            with open(path) as fl:
                lines = fl.readlines()
            for i, line in enumerate(lines):
                if re.match(r"^func [A-Z]", line) or re.match(r"^type [A-Z]", line):
                    prev = lines[i-1].strip() if i > 0 else ""
                    if not prev.startswith("//"):
                        missing.append(f"{path}:{i+1} -> {line.strip()}")
print(f"Exported symbols missing docstrings: {len(missing)}")
'
```
Target: `0` missing docstrings.
