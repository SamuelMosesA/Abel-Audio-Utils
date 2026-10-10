<!--
Sync Impact Report
- Version change: 1.1.0 → 1.2.0
- List of modified principles:
  - PRINCIPLE_6: Frequent Atomic Commits with Descriptive Messages (Maintained)
  - PRINCIPLE_7: Explicit Functional Decomposition & Naming (Added)
  - PRINCIPLE_8: Strict State Encapsulation & Interface Separation (Added)
  - PRINCIPLE_9: Auditable & Lock-Free Audio Pipeline (Added)
- Added sections:
  - Code Quality Governance & Encapsulation Standards
- Removed sections: None
- Follow-up TODOs: None
-->

# Abel Audio Utils Constitution

## Core Principles

### I. Radical Simplicity & Code Minimization
The codebase MUST favor subtraction over addition: less code is strictly better.
- Developers and agents MUST delete obsolete, redundant, or speculative code proactively.
- Every new line of code must justify its existence against maintenance cost.
- Strive for the simplest possible solution; avoid speculative abstraction, unnecessary indirection, and premature generalization.

### II. Streamlined & Maintainable Unit Tests
Testing MUST be straightforward, focused, and minimal.
- Unit tests must test behavior and invariants without bloated setup or convoluted mock hierarchies.
- Tests should be concise and easily readable, using idiomatic table-driven tests and clear assertions.
- Eliminate redundant and brittle test cases that duplicate coverage or hinder rapid refactoring.

### III. Converged Architecture & Cohesion
System design MUST converge related logic into unified, cohesive components.
- Scattered logic handling similar concerns across disjointed packages MUST be consolidated into a single authoritative module.
- System pipelines, state management, and data flow (as represented in `ARCHITECTURE.md`) must be kept radically simple and lean.
- Complex orchestration and tangled dependencies are non-compliant and must be flattened.

### IV. Hierarchical Domain Organization
Specialized domain capabilities MUST be cleanly isolated in structured folder hierarchies.
- Distinct functional domains (such as audio format conversion, DSP/filtering, and streaming protocols) MUST reside in dedicated subdirectories with clear file separations.
- Avoid large monolithic files or dumping multiple distinct utilities into a flat root package.
- Sub-packages must present minimal exported interfaces.

### V. Zero Duplication (DRY & Shared Utilities)
Logic repeated across multiple files MUST be extracted into reusable shared functions.
- Duplicated validation, formatting, transformation, or pipeline logic is prohibited.
- When identical or near-identical logic appears in more than one place, developers MUST extract and converge it into a shared, well-tested function.

### VI. Frequent Atomic Commits with Descriptive Messages (MANDATORY)
Every completed task, refactoring step, or logical change set MUST be committed immediately to git.
- Developers and AI agents MUST perform automatic git commits upon completing each discrete task.
- Commit messages MUST be descriptive and follow conventional commits (e.g. `refactor(audio): delegate wav headers to conversion package`, `feat(ui): simplify recording list styling`).
- Atomic commits ensure changes are easily auditable, safe, and can be cleanly reverted if an issue arises.

### VII. Explicit Functional Decomposition & Descriptive Naming
Functions with large bodies or multi-step logic MUST be decomposed into small, single-responsibility functions.
- Every function name MUST explicitly and unambiguously convey what the function does (verb-noun functional naming).
- Avoid monolithic functions containing unseparated setup, processing, error-handling, and serialization.
- Callers and maintainers must be able to understand execution flow directly from the call sites without reading large blocks of inline code.

### VIII. Strict State Encapsulation & Interface Separation
Mutable state MUST be cleanly encapsulated and shielded from direct outside mutation.
- Struct fields representing internal state MUST be unexported (private) unless public access is strictly required.
- Mutations MUST occur exclusively through explicit, thread-safe accessor and modifier methods.
- State representations and interface contracts MUST reside in separate, distinctly named files to avoid confusing data holding with behavior contracts.

### IX. Auditable & Lock-Free Audio Pipeline (Verifiable Engine-to-Recording Path)
The path from audio engine capture to disk recording MUST be the most direct and manually verifiable flow in the system.
- Audio sample clamping, byte formatting, and header construction MUST be pure, reusable functions in `audio_processing`.
- Recording write loops MUST remain lock-free in the inner chunk path: do not acquire mutexes on each audio chunk when a dedicated single-threaded worker owns file writing.
- Any engineer inspecting the codebase must be able to trace audio capture from input channel to disk file within minutes.

## Architecture & Quality Standards

- **Audio Pipeline Safety**: Audio streams, buffer allocations, and concurrency mechanisms (goroutines, channels) must guarantee zero memory leaks and deterministic shutdown.
- **API & Routing Simplicity**: Gin route handlers and middleware must remain thin, delegating domain logic to cohesive service modules.
- **Observability without Bloat**: Telemetry and structured logging must provide actionable insights with minimal overhead and clean signal-to-noise ratios.

## Refactoring & Review Workflow

- **Refactor Before Feature Addition**: Prioritize consolidating scattered logic and eliminating duplicates before expanding feature footprints.
- **Code Review Gate**: Every change set and pull request MUST be audited for potential code deletion and simplification opportunities.
- **Quality Gates**: All automated workflows (`speckit-plan`, `speckit-tasks`, `speckit-implement`) must enforce simplicity and convergence principles prior to merge.
- **Continuous Commit Gate**: Checkpoints during implementation must commit working increments before proceeding to subsequent tasks.

## Governance

- This Constitution supersedes all informal practices and serves as the highest architectural authority for Abel Audio Utils.
- Amendments require explicit documentation, version increment according to semantic versioning (MAJOR for breaking changes/removals, MINOR for additions, PATCH for clarifications), and migration guidance.
- Compliance is validated across all Spec Kit phases and continuous integration gates.

**Version**: 1.2.0 | **Ratified**: 2026-10-10 | **Last Amended**: 2026-10-10
