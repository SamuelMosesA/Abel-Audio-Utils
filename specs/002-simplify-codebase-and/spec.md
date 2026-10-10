# Feature Specification: Codebase Simplification, Documentation & Reduction

**Feature Branch**: `002-simplify-codebase-and`

**Created**: 2026-10-10

**Status**: Ready for Planning

**Input**: User description: "I find a lot of the golang backend code to be long and hard to read for a humna. It doesn not have a hierearchy. it does not have docsstrings. The frontend is too complex. I need to reduce the code in this project to 75 of the size. First git pull the main branch and fix all the conflicts. Mainly target css customization. Either put them in one place that sets the styles. And try to reduce the styling. Make sure you take screenshots and analyze."

## Clarifications

### Session 2026-10-10
- Q: How should CSS styling customization be consolidated and simplified across the frontend? → A: Consolidate into `src/frontend/src/app.css` semantic design tokens and prune scattered inline Tailwind utility strings across Svelte components, removing unused classes (.glow-primary, .card-premium) and repetitive visual clutter while validating with screenshots.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Human-Readable, Hierarchical & Well-Documented Go Backend (Priority: P1)

As a human engineer reading and extending the Abel Go backend,
I need clean package hierarchies, descriptive docstrings on exported types and functions, and short, focused functions,
So that I can comprehend the architecture, understand function responsibilities immediately, and navigate code without parsing hundreds of lines of inline boilerplate.

**Why this priority**: Directly addresses human readability, modular domain hierarchy, and maintainability (Constitution Principle I & IV).

**Independent Test**: Can be validated by inspecting package docstrings, running `go doc ./...`, and verifying that monolithic files (e.g. `recording_processor.go`, `handlers_recordings.go`, `translation.go`) are structured with clear function boundaries and self-explanatory godoc comments.

**Acceptance Scenarios**:

1. **Given** exported structs, interfaces, and public functions across `src/backend/lib/`, **When** inspected, **Then** every exported symbol has a clear, idiomatic Go docstring explaining its purpose, parameters, and return values.
2. **Given** long monolithic Go functions exceeding 50 lines, **When** refactored, **Then** they are split into clear, single-purpose helper functions adhering to single-responsibility principles.
3. **Given** scattered or ad-hoc domain logic (e.g. recording processing, cloud upload, AI streaming), **When** reviewed, **Then** each package presents a clear internal hierarchy with zero circular dependencies.

---

### User Story 2 - Codebase Volume Reduction (~25% Deletion / Pruning) (Priority: P2)

As a project maintainer,
I need unnecessary boilerplate, dead code, verbose logging wrappers, and over-abstracted utilities removed,
So that total codebase size is reduced to ~75% of its current size while preserving 100% of functional capabilities.

**Why this priority**: Directly enforces Constitution Principle I (Radical Simplicity & Code Minimization: "less code is strictly better").

**Independent Test**: Measure line counts before and after refactoring using `wc -l`, confirming total LOC drops towards the 75% target without breaking any application features or API routes.

**Acceptance Scenarios**:

1. **Given** over-engineered processing abstractions (e.g., redundant parameter validation chains, duplicated error wrappers, dead helper methods), **When** simplified, **Then** the logic collapses into compact, idiomatic Go routines.
2. **Given** the frontend codebase, **When** audited, **Then** redundant CSS class strings, overly nested wrapper `div`s, and unused component props are deleted.
3. **Given** full system execution, **When** running recordings, HLS broadcasting, translation, and admin controls, **Then** all workflows operate flawlessly with zero regressions.

---

### User Story 3 - Consolidated CSS & Streamlined Frontend Styling (Priority: P3)

As a frontend developer or user,
I want CSS styling customization centralized in `src/frontend/src/app.css` with semantic design tokens and concise component markup,
So that styling is maintainable in a single authoritative location, inline utility bloat is dramatically reduced across Svelte components, and the UI remains clean and validated via visual screenshots.

**Why this priority**: Directly satisfies the user directive to centralize CSS customization in one place, prune repetitive inline Tailwind styling, and eliminate visual complexity.

**Independent Test**: Can be validated by inspecting `app.css` for centralized semantic tokens, verifying that Svelte components have minimal inline classes, comparing headless browser screenshots (`landing_preview.png`, `admin_preview.png`), and verifying passing tests via `bun run test:unit`, `bun run check`, and `bun run build`.

**Acceptance Scenarios**:

1. **Given** custom visual effects and recurring element styles across Svelte components, **When** refactored, **Then** they are consolidated into `app.css` design tokens instead of being repeated in 10-line inline class strings.
2. **Given** unused custom CSS classes in `app.css` (e.g., `.glow-primary`, `.card-premium`), **When** cleaned up, **Then** dead classes are removed completely.
3. **Given** UI views (Landing, Admin Console, Streaming), **When** captured via browser screenshots before and after refactoring, **Then** the layout, peak meters, and controls render cleanly without visual breakage.

---

### User Story 4 - Lean & Readable Test Suites (Priority: P4)

As a developer running test suites,
I want unit tests to be concise, human-readable, and free of bloated mock scaffolding,
So that tests read like clear executable specifications of system invariants without drowning the reader in boilerplate.

**Why this priority**: Enforces Constitution Principle II (Streamlined & Maintainable Unit Tests).

**Independent Test**: Run test suites and verify all test assertions pass while test files are compact and table-driven.

**Acceptance Scenarios**:

1. **Given** verbose test files with multi-line manual assertions, **When** refactored, **Then** they use compact table-driven structures (`[]struct{...}`).
2. **Given** duplicate test fixtures in recording processor and audio handler tests, **When** reviewed, **Then** shared helpers in `test_helpers.go` are utilized to eliminate repetition.

---

### Edge Cases

- **Godoc Formatting**: Ensuring docstrings follow standard `// FunctionName does ...` format for compatibility with `pkgsite` / IDE tooltips.
- **CSS Specificity & Theme Overrides**: Ensuring centralized rules in `app.css` integrate smoothly with Tailwind v4 `@theme` variables without causing cascade conflicts.
- **Backwards Compatibility**: Preserving all existing HTTP and WebSocket contract routes (`/api/audio/...`, `/api/recordings/...`, `/api/ai/...`).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Every exported package, struct, interface, and function in `src/backend/lib/` MUST have an idiomatic Go docstring summarizing its role and contract.
- **FR-002**: Backend code MUST be organized into clear domain hierarchies (`audioengine/`, `audioengine/conversion/`, `openai/`, `recording/`, `web/`), avoiding sprawling single-file monoliths.
- **FR-003**: The project codebase MUST be aggressively pruned (eliminating dead code, redundant wrappers, bloated boilerplate) aiming towards 75% of original line volume.
- **FR-004**: Frontend CSS customizations MUST be centralized in `src/frontend/src/app.css` using semantic tokens, eliminating scattered inline utility bloat across Svelte templates.
- **FR-005**: All UI changes MUST be audited and validated using visual browser screenshots and test passes (`bun run test:unit`, `bun run check`, `bun run build`).
- **FR-006**: All existing functionality (PortAudio capture, live HLS streaming, MP3 processing, cloud delivery, WebSocket metering, OpenAI translation/transcription) MUST remain fully operational.

### Key Entities

- **CentralizedStylesheet**: `src/frontend/src/app.css` containing the authoritative semantic theme and component design tokens.
- **DocumentedBackend**: Idiomatic Go codebase with docstrings, clear package interfaces, and clean separation of concerns.
- **CompactFrontend**: Lean SvelteKit client application with minimal CSS footprint and concise state runes.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of exported Go identifiers in `src/backend/lib/` have complete, accurate godoc comments.
- **SC-002**: Total codebase line count is reduced towards 75% of original volume without loss of functionality.
- **SC-003**: 100% of custom CSS rules are consolidated in `app.css`, eliminating redundant classes and multi-line utility strings in components.
- **SC-004**: Frontend passes all automated gates: `bun run test:unit` (12/12 passing), `bun run check` (0 errors), and `bun run build` (successful static bundle).
- **SC-005**: Visual verification passes: UI screenshots confirm clean, uncorrupted layout rendering across desktop and mobile viewports.

## Assumptions

- Tailwind CSS v4 `@theme` block in `src/frontend/src/app.css` serves as the primary system design token repository.
- Standard Go toolchain conventions apply for godoc documentation.
