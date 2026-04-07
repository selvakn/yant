# Implementation Plan: Semantic Search

**Branch**: `011-semantic-search` | **Date**: 2026-04-05 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/011-semantic-search/spec.md`

## Summary

Replace the existing fuzzy text search with semantic vector search using sqlite-vec for vector storage and all-MiniLM-L6-v2 (ONNX) for local embedding generation. The system generates 384-dimensional embeddings on note save, stores them in a sqlite-vec virtual table, and performs cosine-similarity KNN queries at search time. A feature toggle allows falling back to text-based matching. Notes without embeddings fall back to text matching automatically. All dependencies (ONNX Runtime, model file, sqlite-vec) are bundled in the Docker image. API-level integration tests using testcontainers-go validate the complete stack against the real Docker image.

## Technical Context

**Language/Version**: Go 1.25 (with CGO_ENABLED=1 for Docker builds)
**Primary Dependencies**: chi/v5, goldmark, scs/v2, modernc.org/sqlite + modernc.org/sqlite/vec, yalue/onnxruntime_go, testcontainers/testcontainers-go
**Storage**: Markdown files (source of truth) + SQLite (modernc.org/sqlite) with sqlite-vec virtual table for vector search
**Testing**: `go test` (unit), `go test -tags=integration` (integration via testcontainers-go), ≥90% backend coverage
**Target Platform**: Linux server (Docker/Alpine), local development on Linux/macOS
**Project Type**: Web service (monorepo: Go backend + HTML/JS frontend)
**Performance Goals**: Search results <2s for 500 notes; embedding generation <3s per note save
**Constraints**: No external API calls for embedding; single Docker image; model <100 MB
**Scale/Scope**: Hundreds of notes, single-user

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Confirm alignment with `.specify/memory/constitution.md`:

- [x] **I. Markdown-first storage** — Notes remain as Markdown files on disk. sqlite-vec stores derived embeddings only, rebuildable from source files at any time via backfill.
- [x] **II. Simplicity** — Minimal new dependencies (sqlite-vec is a blank import; ONNX Runtime is the only significant addition, justified by the core requirement for local embedding inference). No unnecessary abstractions.
- [x] **III. Monorepo** — All changes within existing backend/ and frontend/ directories. Integration tests in backend/internal/integration/.
- [x] **IV. Integration testing** — Integration tests via testcontainers-go exercise the real Docker image end-to-end. Unit tests maintain ≥90% coverage. Existing test infrastructure preserved.
- [x] **V. Simple web UI** — Frontend changes limited to search debounce timing. No new frameworks or heavy JavaScript.
- [x] **VI. Commit & test discipline** — Implementation will follow frequent commits with all tests passing before each commit.

## Project Structure

### Documentation (this feature)

```text
specs/011-semantic-search/
├── plan.md              # This file
├── research.md          # Phase 0 output — technology decisions
├── data-model.md        # Phase 1 output — entity schema
├── quickstart.md        # Phase 1 output — developer guide
├── contracts/
│   └── api.md           # Phase 1 output — API endpoint contracts
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```text
backend/
├── cmd/server/
│   └── main.go                  # MODIFIED: new flags (-semantic-search, -search-debounce, -model-path)
├── internal/
│   ├── models/
│   │   ├── models.go            # MODIFIED: schema migration for note_embeddings + vec_note_embeddings
│   │   ├── search.go            # MODIFIED: dispatch to semantic or text search based on toggle
│   │   ├── embeddings.go        # NEW: embedding CRUD, backfill, content hashing
│   │   ├── semantic_search.go   # NEW: KNN query, threshold/cap filtering, fallback merge
│   │   ├── embeddings_test.go   # NEW: unit tests for embedding storage
│   │   └── semantic_search_test.go # NEW: unit tests for semantic search logic
│   ├── embedding/
│   │   ├── onnx.go              # NEW: ONNX Runtime session management, inference
│   │   ├── tokenizer.go         # NEW: WordPiece tokenizer for all-MiniLM-L6-v2
│   │   ├── vocab.txt            # NEW: embedded vocabulary file (~230 KB)
│   │   ├── onnx_test.go         # NEW: unit tests for ONNX inference
│   │   └── tokenizer_test.go    # NEW: unit tests for tokenizer
│   ├── handlers/
│   │   ├── handlers.go          # MODIFIED: Handler struct gets embedder + semantic search config
│   │   ├── notes.go             # MODIFIED: call embedding on note save; use semantic search
│   │   ├── archive.go           # MODIFIED: use semantic search for archive search
│   │   └── handlers_test.go     # MODIFIED: test semantic search toggle, fallback behavior
│   └── integration/
│       ├── integration_test.go  # NEW: testcontainers-go tests (build tag: integration)
│       └── helpers_test.go      # NEW: test utilities (HTTP client, auth helpers)
├── go.mod                       # MODIFIED: new dependencies
└── go.sum                       # MODIFIED

frontend/
├── templates/
│   ├── notes/list.html          # MODIFIED: debounced search trigger
│   └── archive/list.html        # MODIFIED: debounced search trigger
└── static/                      # No changes

models/                          # NEW: ONNX model directory (gitignored)
└── all-MiniLM-L6-v2.onnx       # Downloaded during build or via make target

Dockerfile                       # MODIFIED: CGO build, ONNX Runtime, model bundled
Makefile                         # MODIFIED: new targets (integration-test, download-model)
.github/workflows/ci.yml        # MODIFIED: add integration-test job
.gitignore                       # MODIFIED: add models/ directory
```

**Structure Decision**: Follows the existing web application layout. New embedding logic gets its own `internal/embedding/` package to keep ONNX/tokenizer concerns separate from data model logic. Integration tests go in `internal/integration/` with a build tag to keep them separate from fast unit tests.

## Complexity Tracking

> No constitution violations. No complexity justifications needed.

## Post-Design Constitution Re-Check

- [x] **I. Markdown-first** — Embeddings are a derived cache. `note_embeddings` is rebuildable from Markdown files. `vec_note_embeddings` is rebuildable from `note_embeddings`.
- [x] **II. Simplicity** — Two new packages (`embedding/`, `integration/`), three new model files, two new dependencies. Each is justified by a concrete requirement (local inference, vector search, integration testing). No speculative abstractions.
- [x] **III. Monorepo** — All code in the same repo. No new top-level directories except `models/` (gitignored, build artifact).
- [x] **IV. Integration testing** — Integration tests in `internal/integration/` with `//go:build integration` tag. Unit tests in respective packages. Coverage threshold maintained.
- [x] **V. Simple web UI** — Only change is htmx trigger timing (debounce). No new JS frameworks.
- [x] **VI. Commit discipline** — Plan calls for incremental implementation with tests passing at each step.
