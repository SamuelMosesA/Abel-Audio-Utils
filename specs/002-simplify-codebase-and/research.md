# Research: Testing Strategies & Safe Code Reduction

## Decision 1: Verification & Safety Methodology for Backend Changes

- **Constraint**: The local host currently does not have `go` or the Docker daemon running in user space, while CI and production compile the Go codebase.
- **Verification Strategy**:
  1. **Syntax & AST Invariant Validation**: Use Python's AST/regex and Go syntax integrity checkers to verify balanced braces, imports, and interface implementations across `src/backend/lib/`.
  2. **Backward-Compatible Refactoring (Non-Destructive)**:
     - Keep public function signatures intact.
     - Move duplicate logic to shared functions (such as `audioengine/conversion/` and `openai/realtime_base.go`) while maintaining delegation shims where existing unit tests or external callers rely on package symbols.
     - Add complete Go docstrings following standard `godoc` formatting without changing function mechanics.
  3. **Table-Driven Test Simplification**:
     - Simplify repetitive assertions using standard Testify table cases without dropping test cases or invariant checks.

## Decision 2: Verification Strategy for Frontend Changes

- **Verification Strategy**:
  1. **Fast Unit Testing with Vitest**:
     - All 5 test suites (12 tests) in `src/frontend` pass cleanly via `bun run test:unit`.
  2. **Type Checking & Svelte Validation**:
     - Run `bun run check` (`svelte-check` + `tsc`) to catch any broken props, missing types, or invalid bindings.
  3. **Production Static Build**:
     - Run `bun run build` to verify Vite bundling, Svelte AST parsing, and `@sveltejs/adapter-static` generation without errors.

## Decision 3: Code Reduction Towards ~75% Size Target

- **Targets for Pruning**:
  - `src/backend/lib/web/recording_processor.go` (~776 lines): Contains duplicate command execution, verbose error wrapping, and repetitive status reporting. Can be condensed by ~150-200 lines.
  - `src/backend/lib/openai/` (`translation.go` & `transcription.go`): Merge duplicate reconnect/session loops into `realtime_base.go`.
  - `src/frontend/src/lib/components/admin/RecordingList.svelte` (~280 lines) & `AudioAdminView.svelte` (~250 lines): Prune duplicate button/wrapper layouts and repetitive inline Tailwind classes.
