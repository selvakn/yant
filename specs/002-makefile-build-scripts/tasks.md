# Tasks: Makefile Build Scripts

**Input**: Design documents from `/specs/002-makefile-build-scripts/`  
**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅

**Organization**: Tasks grouped by user story. This feature has a single deliverable (`Makefile`) so phases are thin — each user story maps to a small set of targets within that one file.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to
- Exact file paths included in every task

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create the file scaffold and protect generated artifacts from version control

- [x] T001 Create `bin/` directory entry and update `.gitignore` to exclude `bin/` and `coverage.out` at project root `.gitignore`
- [x] T002 Create the empty `Makefile` at project root with `.PHONY` declarations and the configurable variables block (`BINARY`, `ADDR`, `DB`, `NOTES_DIR`, `UPLOADS_DIR`, `COVERAGE_THRESHOLD`, `TEST_FLAGS`)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: `help` target must exist first — it is the default target (`all`) and all other targets depend on the `##` comment convention being established

**⚠️ CRITICAL**: The `help` target and variable block must be in place before adding any other target, since the `##` pattern is the documentation convention for all remaining targets.

- [x] T003 Add the `help` target as the default (`all: help`) to `Makefile` using the `grep`+`awk` `##`-comment pattern (FR-008)

**Checkpoint**: `make` and `make help` now work and list zero targets (the table is empty until targets are added in subsequent phases).

---

## Phase 3: User Story 1 — Build and Run the Application (Priority: P1) 🎯 MVP

**Goal**: Developers can compile the binary and start the server with `make build` and `make run`.

**Independent Test**: Clone repo, run `make build` → binary appears at `bin/server`. Run `make run` → server starts on `:8080`. Run `make help` → both targets listed with descriptions.

### Implementation for User Story 1

- [x] T004 [P] [US1] Add `build` target to `Makefile`: `cd backend && go build -o ../$(BINARY) ./cmd/server` with `## Compile the server binary to $(BINARY)` comment (FR-002)
- [x] T005 [P] [US1] Add `run` target to `Makefile` depending on `build`: exec `$(BINARY)` with `-addr $(ADDR) -db $(DB) -notes $(NOTES_DIR) -uploads $(UPLOADS_DIR)` flags with `## Build and start the server (default: :8080)` comment (FR-003)

**Checkpoint**: `make build` produces `bin/server`; `make run` starts the server; `make help` lists both targets.

---

## Phase 4: User Story 2 — Run Tests with Coverage Report (Priority: P2)

**Goal**: Developers can run all tests and verify the ≥90% coverage gate with `make test` and `make coverage`.

**Independent Test**: Run `make test` → exits 0, all tests listed. Run `make coverage` → prints `Coverage: XX%`; exits non-zero if below 90.

### Implementation for User Story 2

- [x] T006 [P] [US2] Add `test` target to `Makefile`: `cd backend && go test $(TEST_FLAGS) ./...` with `## Run the full test suite` comment (FR-004)
- [x] T007 [P] [US2] Add `coverage` target to `Makefile` with the shell arithmetic coverage-gate logic: runs `go test ./internal/... -coverpkg=./internal/... -coverprofile=../coverage.out`, extracts integer percentage with `awk`, prints it, and exits non-zero if below `$(COVERAGE_THRESHOLD)` (FR-005)

**Checkpoint**: `make test` runs all tests and exits per result; `make coverage` enforces ≥90% gate.

---

## Phase 5: User Story 3 — Clean Build Artifacts (Priority: P3)

**Goal**: Developers can remove all generated files with `make clean`; the command is idempotent.

**Independent Test**: Run `make build`, verify `bin/server` exists; run `make clean`, verify `bin/` and `coverage.out` are gone; run `make clean` again → exits 0 without error.

### Implementation for User Story 3

- [x] T008 [US3] Add `clean` target to `Makefile`: `rm -rf ./bin ./coverage.out` with `## Remove build artifacts (bin/, coverage.out)` comment (FR-006)

**Checkpoint**: `make clean` removes artifacts; running it twice exits 0.

---

## Phase 6: User Story 4 — Install / Tidy Dependencies (Priority: P4)

**Goal**: Developers can tidy and download all Go module dependencies with `make deps`.

**Independent Test**: Run `make deps` → `go mod tidy && go mod download` complete without error; exit 0.

### Implementation for User Story 4

- [x] T009 [US4] Add `deps` target to `Makefile`: `cd backend && go mod tidy && go mod download` with `## Tidy and download Go module dependencies` comment (FR-007)

**Checkpoint**: `make deps` completes without error; all module dependencies are available.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Lint target (nice-to-have from spec assumptions) and end-to-end validation

- [x] T010 [P] Add `lint` target to `Makefile`: `cd backend && go vet ./...` with `## Run go vet static analysis` comment (spec Assumption: lint is desirable)
- [x] T011 Validate all targets end-to-end per the Quickstart in `specs/002-makefile-build-scripts/plan.md`: run `make`, `make build`, `make run` (then Ctrl-C), `make test`, `make coverage`, `make lint`, `make clean` (twice), `make deps`, and confirm variable override `make run ADDR=:9090` is respected (FR-010, SC-001–SC-005)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1 — T001–T002)**: No dependencies, start immediately
- **Foundational (Phase 2 — T003)**: Depends on T002; blocks all user story phases
- **US1 (Phase 3 — T004–T005)**: Depends on T003; T004 and T005 are [P] — both can be written in parallel
- **US2 (Phase 4 — T006–T007)**: Depends on T003; T006 and T007 are [P]; independent of US1
- **US3 (Phase 5 — T008)**: Depends on T003; independent of US1/US2
- **US4 (Phase 6 — T009)**: Depends on T003; independent of US1/US2/US3
- **Polish (Phase 7 — T010–T011)**: Depends on all prior phases being complete

### User Story Dependencies

- **US1 (P1)**: Independent after Foundational
- **US2 (P2)**: Independent after Foundational
- **US3 (P3)**: Independent after Foundational
- **US4 (P4)**: Independent after Foundational

All four user stories modify the same `Makefile` but add non-conflicting targets, so they can be implemented sequentially in any order after Phase 2.

---

## Parallel Opportunities

```text
# Phase 3 (US1) — both targets add different lines to Makefile:
T004: add build target
T005: add run target
# These touch different lines; can be drafted in parallel and merged.

# Phase 4 (US2) — same pattern:
T006: add test target
T007: add coverage target
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001–T002)
2. Complete Phase 2: Foundational (T003)
3. Complete Phase 3: US1 — Build and Run (T004–T005)
4. **STOP and VALIDATE**: `make build`, `make run`, `make help` all work
5. Merge as minimal viable Makefile

### Incremental Delivery

1. T001–T003 → scaffold + help target
2. T004–T005 → `make build` + `make run` (MVP ✅)
3. T006–T007 → `make test` + `make coverage`
4. T008 → `make clean`
5. T009 → `make deps`
6. T010–T011 → lint + full validation

---

## Notes

- All targets in `Makefile` must be declared in the `.PHONY` list added in T002
- The `##` comment on each target line is the documentation source for `make help` — every target added in T004–T010 must include it
- `make run` blocks the terminal (server process); Ctrl-C is the expected exit
- Variable overrides (`make run ADDR=:9090`) work automatically via make's standard variable precedence — no special handling needed
- Commit after T003 (help scaffold), again after T005 (MVP), and after each subsequent user story phase
