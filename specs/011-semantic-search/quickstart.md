# Quickstart: Semantic Search

**Feature**: 011-semantic-search
**Date**: 2026-04-05

## Prerequisites

- Go 1.25+
- Docker (for building images and running integration tests)
- Node.js 24+ (for frontend build, unchanged)
- GCC / C compiler (for CGO, needed for ONNX Runtime)
- ONNX Runtime shared library (`libonnxruntime.so`) installed locally or available in the build environment

## Local Development

### Build (without embedding support)

For fast local iteration on non-embedding code, continue using CGO_ENABLED=0:

```bash
make build          # builds without ONNX Runtime support
make run            # runs server; semantic search auto-disables if model not found
```

### Build (with embedding support)

```bash
# Download model (one-time)
make download-model

# Build with CGO for ONNX Runtime
CGO_ENABLED=1 make build

# Run with semantic search
make run
```

### Docker Build (full stack, recommended)

```bash
make docker-build   # builds image with all dependencies bundled
make docker-run     # runs container with semantic search working out of the box
```

## Testing

### Unit Tests (fast, no Docker)

```bash
make test           # runs all unit tests (CGO not required)
make coverage       # runs tests with ≥90% coverage gate
```

### Integration Tests (Docker required)

```bash
make docker-build          # build the app image first
make integration-test      # runs integration tests against Docker container
```

Integration tests use testcontainers-go to:
1. Start the application Docker image
2. Wait for HTTP readiness
3. Exercise all API endpoints (notes CRUD, search, archive)
4. Validate semantic search results
5. Tear down container

### All Tests

```bash
make docker-build && make test && make integration-test
```

## Configuration

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `-semantic-search` | `SEMANTIC_SEARCH` | `true` | Toggle semantic vs text search |
| `-search-debounce` | `SEARCH_DEBOUNCE_MS` | `300` | Search debounce delay (ms) |
| `-model-path` | `MODEL_PATH` | `./models/all-MiniLM-L6-v2.onnx` | ONNX model file path |

## Architecture Overview

```
User types query
       │
       ▼ (debounced, 300ms default)
  Search Handler
       │
       ├── Semantic search enabled?
       │     ├── YES: embed query → KNN via sqlite-vec → rank by cosine similarity
       │     │         └── merge text-fallback results for notes without embeddings
       │     └── NO:  text-based matching (title/tag, existing behavior)
       │
       ▼
  Apply threshold + cap → Return results
```

```
Note saved/updated
       │
       ▼
  Hash title+body → compare with stored hash
       │
       ├── Changed: generate embedding → upsert note_embeddings + vec_note_embeddings
       └── Unchanged: skip (embedding already current)
```

## Key Files (new/modified)

```
backend/
├── internal/
│   ├── models/
│   │   ├── embeddings.go       # Embedding generation, storage, backfill
│   │   ├── semantic_search.go  # Semantic search query logic
│   │   └── search.go           # Modified: delegate to semantic or text search
│   ├── embedding/
│   │   ├── onnx.go             # ONNX Runtime inference wrapper
│   │   ├── tokenizer.go        # WordPiece tokenizer
│   │   └── vocab.txt           # Tokenizer vocabulary (embedded)
│   ├── handlers/
│   │   └── notes.go            # Modified: call embedding on save, use semantic search
│   └── integration/
│       └── integration_test.go # Integration tests (build tag: integration)
├── go.mod                      # New deps: modernc.org/sqlite/vec, yalue/onnxruntime_go
└── cmd/server/
    └── main.go                 # New flags: -semantic-search, -search-debounce, -model-path

models/                         # ONNX model files (gitignored, downloaded or Docker-bundled)
└── all-MiniLM-L6-v2.onnx

Dockerfile                      # Updated: CGO build, ONNX Runtime, model bundled
Makefile                        # New targets: integration-test, download-model
```
