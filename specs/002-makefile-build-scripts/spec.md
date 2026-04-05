# Feature Specification: Makefile Build Scripts

**Feature Branch**: `002-makefile-build-scripts`  
**Created**: 2026-04-05  
**Status**: Draft  
**Input**: User description: "add set of build script with make (Makefile) for build, test, run the app, etc"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Build and Run the Application (Priority: P1)

A developer checks out the repository and wants to build and start the application with a single command. Rather than looking up the right flags or invocations, they type `make run` (or `make build`) and the app starts correctly.

**Why this priority**: Getting a working app running locally is the most fundamental developer task. Every other workflow depends on this.

**Independent Test**: A new contributor can clone the repo, run `make run`, and have the server running — without reading any additional documentation.

**Acceptance Scenarios**:

1. **Given** a clean checkout with the required runtime installed, **When** the developer runs `make build`, **Then** a runnable binary is produced in a predictable location (e.g., `bin/server`).
2. **Given** source available, **When** the developer runs `make run`, **Then** the server starts and accepts connections on its configured address.
3. **Given** any working directory within the repo, **When** the developer runs `make help` (or just `make`), **Then** all available targets are listed with short descriptions.

---

### User Story 2 - Run Tests with Coverage Report (Priority: P2)

A developer wants to run the full test suite and verify coverage is above the project threshold (≥90%), without memorising complex test invocation flags.

**Why this priority**: Testing is a core workflow for every contribution; the coverage gate is a stated project requirement.

**Independent Test**: Running `make test` produces a pass/fail result and a coverage summary. Running `make coverage` prints a detailed coverage percentage.

**Acceptance Scenarios**:

1. **Given** all tests passing, **When** the developer runs `make test`, **Then** test results are printed and the command exits successfully.
2. **Given** a test failure, **When** the developer runs `make test`, **Then** the failing test is reported and the command exits with an error code.
3. **Given** passing tests, **When** the developer runs `make coverage`, **Then** a line-coverage percentage is printed; if it is below 90% the command exits with an error code.

---

### User Story 3 - Clean Build Artifacts (Priority: P3)

A developer wants to reset the project to a clean state — removing compiled binaries, coverage files, and any generated artifacts — before a fresh build or to free disk space.

**Why this priority**: Hygiene task; expected in any well-maintained project.

**Independent Test**: After running `make build` and then `make clean`, the binary and all generated files are removed.

**Acceptance Scenarios**:

1. **Given** previously built artifacts exist, **When** the developer runs `make clean`, **Then** all generated files are removed and the working tree is back to source-only state.
2. **Given** no artifacts exist, **When** the developer runs `make clean`, **Then** the command completes without error.

---

### User Story 4 - Install / Tidy Dependencies (Priority: P4)

A developer on a fresh machine wants to pull all dependencies with a single command before building.

**Why this priority**: Convenience target; improves discoverability for new contributors.

**Independent Test**: Running `make deps` completes without error and all required modules are available locally.

**Acceptance Scenarios**:

1. **Given** a machine with the required toolchain but no module cache, **When** the developer runs `make deps`, **Then** all dependencies are downloaded and available.

---

### Edge Cases

- What happens when the required toolchain is not installed? The Makefile should fail with a clear, actionable error message rather than a cryptic one.
- What if the binary already exists when `make build` is run again? It should be rebuilt (fresh compile) or skipped if unchanged, using make's standard dependency tracking.
- What if `make clean` is run when nothing has been built? It MUST exit 0 silently.
- What if `make run` is invoked and the configured port is already in use? The error is surfaced from the application layer, not from the Makefile itself.
- What if a developer overrides a variable (e.g., `make run ADDR=:9090`)? The override MUST be respected by the target.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: A `Makefile` MUST be present at the project root and be invocable with the standard `make` command.
- **FR-002**: `make build` MUST compile the backend binary and place it at a configurable output path, defaulting to `bin/server`.
- **FR-003**: `make run` MUST build (if necessary) and start the application server using sensible default configuration.
- **FR-004**: `make test` MUST execute the full backend test suite and report pass/fail; it MUST exit non-zero on any test failure.
- **FR-005**: `make coverage` MUST run tests with coverage analysis, display the total line-coverage percentage, and exit non-zero if coverage is below 90%.
- **FR-006**: `make clean` MUST remove all generated build artifacts, binaries, and coverage report files without affecting source files.
- **FR-007**: `make deps` MUST download and tidy all project dependencies.
- **FR-008**: Running `make` or `make help` with no target MUST print a human-readable list of all available targets with one-line descriptions.
- **FR-009**: All targets MUST work correctly when invoked from the project root directory.
- **FR-010**: The Makefile MUST expose configurable variables (e.g., output path, server address, test flags) that developers can override at the command line without editing the file.

### Key Entities

- **Makefile**: The single file at the project root containing all named build targets.
- **Binary**: The compiled server executable produced by `make build`.
- **Coverage report**: The output artifact of coverage analysis produced by `make coverage`.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A developer unfamiliar with the project can build and run the application in under 60 seconds using only `make` commands, without consulting any other documentation.
- **SC-002**: `make test` output clearly indicates pass/fail for every test run.
- **SC-003**: `make coverage` prints the total coverage percentage and exits non-zero when below 90%, making it suitable as a CI gate.
- **SC-004**: All defined targets are discoverable from `make help` — zero targets require prior knowledge of the repository internals.
- **SC-005**: `make clean` removes 100% of generated artifacts; running `make clean` twice in a row both succeed without error.

## Assumptions

- The project root contains a `backend/` directory with Go source and a `frontend/` directory with templates and static assets, reflecting the existing project layout.
- The Go toolchain (1.22+) is installed on the developer's machine, consistent with the existing project requirement.
- Primary target platforms are Linux and macOS (POSIX-compatible `make`); native Windows support is out of scope, though WSL is acceptable.
- Default server configuration values (address, database path, notes/uploads directories) will mirror those already defined in the application's entry point.
- Frontend assets are already vendored; no JavaScript package manager build step is required.
- A `lint` target is desirable but not required for the initial version of this feature.
