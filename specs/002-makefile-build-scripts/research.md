# Research: Makefile Build Scripts

**Branch**: `002-makefile-build-scripts`  
**Date**: 2026-04-05

## Summary

No external unknowns. All decisions are resolved from existing project structure and POSIX make best practices.

---

## Decision 1: Makefile Location

**Decision**: Single `Makefile` at the project root (`/home/selva/projects/selvakn/yant/Makefile`).

**Rationale**: All make commands should be invocable from the repo root without `cd`. The project already has `backend/` and `frontend/` at the root, so the Makefile belongs there.

**Alternatives considered**: Separate `backend/Makefile` — rejected because `make run` needs access to both `backend/` (for building) and `frontend/` (for serving templates), and the spec requires root-invocability (FR-009).

---

## Decision 2: Binary Output Path

**Decision**: `bin/server` relative to the project root.

**Rationale**: Standard Go convention (`bin/` directory at repo root). Predictable, easy to gitignore. `bin/` will be added to `.gitignore`.

**Alternatives considered**: `backend/bin/server` — rejected because it complicates the run invocation (the server CWD needs to be the project root for `resolveFrontend()` to find `frontend/`).

---

## Decision 3: Go Build and Test Commands

**Decision**: All Go commands run with `cd backend && go ...` since the `go.mod` is in `backend/`.

**Rationale**: The Go module root is `backend/` (contains `go.mod`). Running `go build` or `go test` from the repo root would require `-C backend` (Go 1.21+) or explicit paths. Using `cd backend && ...` is more portable across make versions.

**Actual commands**:
- Build: `go build -o ../bin/server ./cmd/server`
- Test: `go test ./...`
- Coverage: `go test ./internal/... -coverpkg=./internal/... -coverprofile=../coverage.out && go tool cover -func=../coverage.out | tail -1`
- Deps: `go mod tidy && go mod download`

---

## Decision 4: Server Run Invocation

**Decision**: `./bin/server` run from the project root with sensible defaults.

**Rationale**: The server's `resolveFrontend()` function searches for `frontend/` relative to the current working directory. Running from the project root ensures it finds `frontend/` at `./frontend`. Default flags match `main.go` defaults.

**Run command**: `./bin/server -addr $(ADDR) -db $(DB) -notes $(NOTES_DIR) -uploads $(UPLOADS_DIR)`

**Default variable values**:
- `ADDR = :8080`
- `DB = ./notes.db`
- `NOTES_DIR = ./notes`
- `UPLOADS_DIR = ./uploads`

---

## Decision 5: Coverage Gate Implementation

**Decision**: Extract the percentage from `go tool cover -func` output and compare against 90 using shell arithmetic.

**Rationale**: The constitution mandates ≥90% coverage; `make coverage` must exit non-zero if below threshold. Shell arithmetic (`$(shell ...)`) can extract the integer part and compare.

**Implementation approach**:
```makefile
COVERAGE_THRESHOLD := 90

coverage:
    cd backend && go test ./internal/... -coverpkg=./internal/... -coverprofile=../coverage.out
    @PCTG=$$(go tool cover -func=coverage.out | tail -1 | awk '{gsub(/%/,""); print int($$3)}'); \
     echo "Coverage: $$PCTG%"; \
     if [ "$$PCTG" -lt "$(COVERAGE_THRESHOLD)" ]; then \
       echo "FAIL: coverage $$PCTG% < $(COVERAGE_THRESHOLD)%"; exit 1; \
     fi
```

---

## Decision 6: Self-Documenting Help Target

**Decision**: Use the `##` comment pattern — each target line is followed by `## Description`. `make help` uses `grep` + `awk` to extract and print the table.

**Rationale**: Zero extra tooling. Works with standard POSIX tools. Widely adopted (used by projects like kubernetes, docker-compose). Keeps documentation next to the target.

**Pattern**:
```makefile
build: ## Build the server binary (output: bin/server)
    ...

help: ## Show this help message
    @grep -E '^[a-zA-Z_-]+:.*?##' $(MAKEFILE_LIST) | \
     awk 'BEGIN{FS=":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'
```

---

## Decision 7: .PHONY Declarations

**Decision**: All non-file targets declared as `.PHONY`.

**Rationale**: Prevents make from treating target names as file paths. Ensures targets always run regardless of file/directory existence.

**Targets to declare**: `all`, `build`, `run`, `test`, `coverage`, `clean`, `deps`, `help`, `lint`.

---

## Decision 8: lint Target (Nice-to-have)

**Decision**: Include a basic `lint` target using `go vet` (no external tools required).

**Rationale**: Spec assumption states lint is "desirable but not required". `go vet` is part of the standard toolchain — zero extra dependencies. Can be extended to `golangci-lint` later without breaking the interface.

**Alternatives considered**: `golangci-lint` — deferred; requires separate install, violates Simplicity principle for MVP.
