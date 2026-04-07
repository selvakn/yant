# Research: Semantic Search

**Feature**: 011-semantic-search
**Date**: 2026-04-05

## R1: Vector Storage — sqlite-vec Integration with modernc.org/sqlite

### Decision
Use `modernc.org/sqlite` with `modernc.org/sqlite/vec` (blank import). This keeps the existing pure-Go SQLite driver and adds sqlite-vec support **without CGO** and without switching to `mattn/go-sqlite3`.

### Rationale
- The project already uses `modernc.org/sqlite v1.48.1` with `CGO_ENABLED=0`.
- `modernc.org/sqlite/vec` is a Go-native port of sqlite-vec that integrates via blank import — no build chain changes required for the vector storage layer itself.
- This avoids the complexity of switching SQLite drivers (`mattn/go-sqlite3` would require CGO and a full C toolchain in Docker).
- sqlite-vec provides `vec0` virtual tables with KNN support and cosine distance — exactly what we need.

### Alternatives Considered
- **mattn/go-sqlite3 + CGO sqlite-vec bindings**: Maximum compatibility with upstream sqlite-vec, but requires switching the entire SQLite driver and enabling CGO for all builds. Rejected: too disruptive for one feature.
- **ncruces/go-sqlite3 (WASM)**: sqlite-vec support exists but the driver API differs from `database/sql`. Rejected: would require rewriting all DB access code.
- **Separate vector database (Qdrant, Milvus)**: Overkill for hundreds of notes. Rejected: violates simplicity principle, adds operational dependency.

## R2: Embedding Model and Inference

### Decision
Use `all-MiniLM-L6-v2` via ONNX Runtime, loaded through `github.com/yalue/onnxruntime_go`. The ONNX model file (~80 MB) and ONNX Runtime shared library are bundled in the Docker image. The tokenizer is implemented in Go using a WordPiece vocabulary file extracted from the model.

### Rationale
- `all-MiniLM-L6-v2` produces 384-dimensional embeddings, is ~23M parameters, and runs fast on CPU (~5-20ms per sentence). Well-established for semantic search.
- ONNX Runtime is the most mature inference runtime, with Go bindings available via `yalue/onnxruntime_go`.
- Bundling the model in Docker ensures zero external dependencies (no API keys, no network calls, no sidecar processes).
- 384 dimensions is small enough for efficient sqlite-vec KNN queries over hundreds of notes.
- This approach requires CGO for ONNX Runtime shared library loading. Per clarification, CGO is approved.

### Alternatives Considered
- **Ollama sidecar**: Simple HTTP API, but requires running a separate daemon, complicates Docker image, and adds ~1-4 GB to image size for the Ollama binary + model. Rejected: violates self-contained distribution requirement.
- **Python subprocess**: Correct tokenization guaranteed, but adds Python runtime to Docker image (~200+ MB), subprocess overhead per embedding, and packaging complexity. Rejected: too heavy.
- **go-llama.cpp with GGUF model**: Complex build (CMake + CGO), designed primarily for LLM inference not embeddings. Rejected: wrong tool for the job.
- **Pure-Go inference (GoMLX, pure-onnx)**: Immature ecosystem, operator coverage uncertain for transformer models. Rejected: too risky.

### Build Impact
- CGO_ENABLED=1 required for ONNX Runtime shared library.
- Docker build stage needs: Go compiler with CGO, ONNX Runtime shared library for Linux (x86_64).
- Runtime stage needs: ONNX Runtime shared library, model file, tokenizer vocabulary.
- Estimated Docker image size increase: ~100-150 MB (ONNX Runtime ~30 MB + model ~80 MB + C runtime dependencies).

## R3: Tokenization

### Decision
Implement WordPiece tokenization in Go. The vocabulary file (`vocab.txt`) is extracted from the `all-MiniLM-L6-v2` model and bundled as a Go embed asset or file in the Docker image.

### Rationale
- `all-MiniLM-L6-v2` uses a standard BERT WordPiece tokenizer with a well-documented algorithm.
- Go implementations of WordPiece exist and the algorithm is straightforward (~200 lines).
- Avoids dependency on Python tokenizers or Hugging Face libraries at runtime.
- Vocabulary file is ~230 KB — trivial to bundle.

### Alternatives Considered
- **Hugging Face tokenizers via CGO**: The `tokenizers` Rust library has C bindings, but adding Rust build toolchain to Docker is excessive. Rejected.
- **Pre-tokenize in Python, call from Go**: Adds Python dependency. Rejected: same reasons as R2.

## R4: Integration Testing with testcontainers-go

### Decision
Use `github.com/testcontainers/testcontainers-go` to spin up the application Docker image in tests. Tests are organized in a separate `backend/internal/integration/` package with `//go:build integration` build tags. A new `make integration-test` Makefile target runs these tests.

### Rationale
- testcontainers-go is the Go standard for Docker-based integration testing. Mature, well-documented.
- Build tags separate integration tests from fast unit tests — `make test` stays fast, `make integration-test` runs the full suite.
- Tests exercise the real Docker image, ensuring the deployed artifact matches what's tested.
- HTTP wait strategy ensures tests don't start until the application is ready.

### Alternatives Considered
- **docker-compose + test script**: Manual orchestration, harder to parallelize, no programmatic port mapping. Rejected: testcontainers is better.
- **In-process httptest with real DB**: Already exists for unit tests. Doesn't validate Docker packaging, ONNX Runtime loading, or the full startup flow. Kept for unit tests, but insufficient for integration.
- **Custom Docker orchestration in shell**: Fragile, not portable. Rejected.

### Test Organization
- `backend/internal/integration/` — integration test package
- `//go:build integration` build tag on all files
- `TestMain` starts one container, shares across tests in the package
- Tests use `net/http.Client` against the container's mapped port
- Authentication handled by creating a test user via the API or bypassing OAuth for test mode

## R5: CGO Build Chain Changes

### Decision
Enable CGO only in the Docker build. Local development can continue with CGO_ENABLED=0 for fast iteration on non-embedding code. The Dockerfile switches to a CGO-enabled build with static linking via musl.

### Rationale
- ONNX Runtime requires CGO for shared library loading.
- `modernc.org/sqlite/vec` may or may not require CGO (needs verification during implementation — if it's pure Go like the base driver, CGO is only needed for ONNX Runtime).
- Local development without embedding features can stay CGO-free.
- Docker uses Alpine with musl for a small runtime image.

### Build Changes
- Dockerfile Stage 2: `golang:1.25-bookworm` → `golang:1.25-alpine` or keep bookworm with `gcc` installed.
- `CGO_ENABLED=0` → `CGO_ENABLED=1` in Dockerfile.
- ONNX Runtime shared library copied into build and runtime stages.
- Makefile `build` target remains CGO_ENABLED=0 for local dev (embedding features gracefully degrade or use a flag).

## R6: Feature Toggle Design

### Decision
A server-side configuration flag (`-semantic-search` / `SEMANTIC_SEARCH=true|false`, default `true`) controls whether search uses semantic matching or falls back to text-based matching. Embeddings are always generated regardless of the toggle.

### Rationale
- Simple boolean flag, consistent with existing `-addr`, `-db` flag patterns.
- Allows operators to disable semantic search if performance issues arise, without losing embedding data.
- Toggle takes effect immediately on restart — no rebuild needed since embeddings are maintained continuously.

## R7: Search Debounce

### Decision
Frontend JavaScript debounce with a configurable delay (default 300ms). The delay value is injected into the template from a server-side configuration flag.

### Rationale
- Debounce is a frontend concern — backend doesn't need to change for this.
- 300ms is the industry standard for search-as-you-type debounce.
- Making it configurable allows tuning for slower hardware or network conditions.
