# Implementation Plan: Makefile Build Scripts

**Branch**: `002-makefile-build-scripts` | **Date**: 2026-04-05 | **Spec**: [spec.md](spec.md)  
**Input**: Feature specification from `/specs/002-makefile-build-scripts/spec.md`

## Summary

Add a `Makefile` at the project root with targets for building the Go backend binary (`make build`), running the server (`make run`), executing tests (`make test`), enforcing the ≥90% coverage gate (`make coverage`), cleaning artifacts (`make clean`), tidying dependencies (`make deps`), and self-documenting help (`make help`). All targets accept configurable variables (e.g., `ADDR`, `DB`) for override at the command line.

## Technical Context

**Language/Version**: Go 1.22+ (existing project requirement); POSIX `make`  
**Primary Dependencies**: Standard Go toolchain (`go build`, `go test`, `go tool cover`, `go vet`); POSIX `awk`, `grep`, `rm`  
**Storage**: N/A (build tooling only)  
**Testing**: `go test ./...` (existing test suite, ≥90% coverage requirement)  
**Target Platform**: Linux and macOS (POSIX-compatible `make`); Windows via WSL only  
**Project Type**: Build/developer tooling for an existing Go web application  
**Performance Goals**: N/A — build speed is determined by Go toolchain, not the Makefile  
**Constraints**: Zero new runtime dependencies; must work from project root; all targets PHONY  
**Scale/Scope**: Single `Makefile` file; ~80 lines

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Markdown-First Storage | ✅ PASS | Build tooling does not affect storage layer |
| II. Simplicity | ✅ PASS | `make` is universally available, zero new dependencies; `go vet` for lint rather than external linters |
| III. Monorepo Structure | ✅ PASS | `Makefile` at repo root; leverages existing `backend/`/`frontend/` split |
| IV. Integration Testing | ✅ PASS | `make test` and `make coverage` wrap existing `go test ./...`; coverage gate enforced at ≥90% |
| V. Simple Web Interface | ✅ PASS | Build tooling does not affect the UI |

**Post-design re-check**: All principles still PASS — no violations to track.

## Project Structure

### Documentation (this feature)

```text
specs/002-makefile-build-scripts/
├── plan.md       # This file
├── research.md   # Phase 0 output
└── tasks.md      # Phase 2 output (/speckit.tasks — NOT created here)
```

*(No data-model.md or contracts/ — this feature is pure build tooling with no data entities or external interfaces.)*

### Source Code (repository root)

```text
Makefile                          # NEW: project root — all build targets
bin/                              # NEW: created by make build, gitignored
coverage.out                      # NEW: created by make coverage, gitignored
backend/
├── cmd/server/main.go            # Existing entry point (flags inform make run defaults)
└── ...                           # Unchanged
frontend/                         # Unchanged
.gitignore                        # UPDATE: add bin/ and coverage.out
```

**Structure Decision**: Single `Makefile` at the project root. The Go module lives in `backend/`, so Go commands run as `cd backend && go ...`. The binary is output to `bin/server` at the project root so it can be run with `./bin/server` while having `frontend/` visible in the CWD.

## Makefile Design

### Variables (overridable)

| Variable | Default | Purpose |
|----------|---------|---------|
| `BINARY` | `./bin/server` | Output path for compiled binary |
| `ADDR` | `:8080` | Server listen address |
| `DB` | `./notes.db` | SQLite database file path |
| `NOTES_DIR` | `./notes` | Markdown notes root directory |
| `UPLOADS_DIR` | `./uploads` | Image uploads root directory |
| `COVERAGE_THRESHOLD` | `90` | Minimum required coverage percentage |
| `TEST_FLAGS` | *(empty)* | Extra flags passed to `go test` |

### Targets

| Target | Description | Key behaviour |
|--------|-------------|---------------|
| `all` / `help` | Default: show help | Grep `##` comments; print target table |
| `build` | Compile binary | `cd backend && go build -o ../$(BINARY) ./cmd/server` |
| `run` | Build + start server | Depends on `build`; exec binary with configurable flags |
| `test` | Run test suite | `cd backend && go test $(TEST_FLAGS) ./...` |
| `coverage` | Test + coverage gate | Run with `-coverpkg`; extract %; fail if < threshold |
| `lint` | Static analysis | `cd backend && go vet ./...` |
| `clean` | Remove artifacts | `rm -rf ./bin ./coverage.out` |
| `deps` | Tidy + download | `cd backend && go mod tidy && go mod download` |

### Help implementation pattern

```makefile
help: ## Show this help message
    @grep -E '^[a-zA-Z_-]+:.*?##' $(MAKEFILE_LIST) | \
     awk 'BEGIN{FS=":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'
```

### Coverage gate implementation

```makefile
coverage: ## Run tests and enforce ≥90% line coverage
    cd backend && go test ./internal/... -coverpkg=./internal/... \
        -coverprofile=../coverage.out $(TEST_FLAGS)
    @PCTG=$$(go tool cover -func=coverage.out | tail -1 | \
        awk '{gsub(/%/,""); print int($$3)}'); \
     echo "Coverage: $$PCTG%"; \
     if [ "$$PCTG" -lt "$(COVERAGE_THRESHOLD)" ]; then \
       echo "FAIL: coverage $$PCTG% < $(COVERAGE_THRESHOLD)%"; exit 1; \
     fi
```

## Quickstart

```bash
# 1. Build the binary
make build

# 2. Run the server (default: http://localhost:8080)
make run

# 3. Run on a different port
make run ADDR=:9090

# 4. Run all tests
make test

# 5. Check coverage gate (fails if <90%)
make coverage

# 6. Lint
make lint

# 7. Remove all build artifacts
make clean

# 8. Tidy dependencies
make deps
```
