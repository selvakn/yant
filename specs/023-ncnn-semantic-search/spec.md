# Feature Specification: ncnn Semantic Search Runtime

**Feature Branch**: `023-ncnn-semantic-search`
**Created**: 2026-05-07
**Status**: Draft
**Input**: User description: "lets attempt to move to ncnn as the runtime for doing semantic search"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Semantic Search Continues to Work (Priority: P1)

A user searches their notes using natural language. The system returns results ranked by meaning rather than just keyword matches, exactly as it did before — the underlying runtime change is invisible to the user.

**Why this priority**: Preserving the existing semantic search experience is the primary goal. If this doesn't work, the migration has no value.

**Independent Test**: Can be fully tested by entering a semantic query (e.g., "car maintenance") and verifying that semantically related notes (e.g., notes tagged "vehicle repair") appear in results, even without exact keyword overlap.

**Acceptance Scenarios**:

1. **Given** notes exist with semantically related content, **When** a user submits a natural language search query, **Then** relevant notes are returned ranked by semantic relevance
2. **Given** the application starts with no pre-downloaded model files, **When** the server initialises, **Then** model files are downloaded automatically and semantic search becomes available without a restart
3. **Given** model files are already present on disk, **When** the server restarts, **Then** semantic search is available immediately without re-downloading

---

### User Story 2 - Smaller Docker Image (Priority: P2)

An operator deploying the application pulls the Docker image and observes a meaningfully smaller image size compared to the previous ONNX Runtime-based image.

**Why this priority**: Reducing image size is the primary motivation for this migration. A successful migration must produce a measurably smaller image.

**Independent Test**: Can be fully tested by building the Docker image and comparing its compressed size against the ONNX Runtime baseline.

**Acceptance Scenarios**:

1. **Given** the application is built as a Docker image, **When** the image size is measured, **Then** it is smaller than the ONNX Runtime-based image
2. **Given** the runtime image, **When** its installed packages are inspected, **Then** no ONNX Runtime packages or their heavy C++/protobuf/ICU transitive dependencies are present

---

### User Story 3 - Graceful Degradation When Model Unavailable (Priority: P3)

A user searches notes before the model download has completed or when the runtime fails to initialise. The application falls back to full-text search automatically rather than returning an error.

**Why this priority**: Matches existing behaviour — the fallback path must remain intact regardless of which runtime is used.

**Independent Test**: Can be fully tested by disabling semantic search or blocking model download and verifying text-based search still returns results.

**Acceptance Scenarios**:

1. **Given** the embedding runtime has not yet initialised, **When** a user performs a search, **Then** text-based search results are returned without an error message
2. **Given** model initialisation fails permanently, **When** a user performs a search, **Then** text-based search results are returned and the failure is logged

---

### Edge Cases

- What happens if ncnn cannot load the converted model file on startup?
- What happens if embedding generation for a note fails mid-save?
- What happens if the model file is corrupted or truncated during download?
- How does the system behave when disk space runs out during model download?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST generate 384-dimensional sentence embeddings compatible with the existing sqlite-vec vector search schema
- **FR-009**: Tokenization MUST use the existing `tokenizer.json` file loaded by a pure-Go or CGO HuggingFace-compatible tokenizer, producing identical token IDs to the ONNX Runtime baseline
- **FR-002**: The system MUST load the embedding model using ncnn as the inference runtime instead of ONNX Runtime
- **FR-003**: A dedicated CI pipeline MUST convert the all-MiniLM-L6-v2 ONNX model to ncnn format (`.param` + `.bin`) and publish the converted files as GitHub Release assets
- **FR-004**: The system MUST download the ncnn model files from their GitHub Release URL to the persistent `/data/models` volume on first start if not already present
- **FR-005**: The system MUST fall back to full-text search when the ncnn embedder is not yet available
- **FR-006**: The system MUST produce embedding vectors that yield equivalent or better semantic search relevance compared to the ONNX Runtime baseline
- **FR-007**: The Docker image MUST NOT include ONNX Runtime or its transitive dependencies (protobuf, ICU, abseil-cpp)
- **FR-010**: If ncnn embedding quality fails the SC-002 relevance gate, the implementation MUST attempt tract (pure Rust ONNX runtime) as the fallback before the feature is considered blocked
- **FR-008**: The Go server binary MUST call ncnn via a thin CGO wrapper against the ncnn C API (`ncnn.h`); no third-party Go binding; must compile cleanly on Alpine musl

### Key Entities

- **Embedding Model**: The sentence transformer weights in ncnn format (`.param` + `.bin`), produced by a CI conversion pipeline, published as GitHub Release assets, and downloaded to the persistent data volume on first start
- **Embedder**: The runtime component responsible for tokenising input text and producing a 384-dim float32 vector; backed by ncnn instead of ONNX Runtime
- **Embedding Vector**: A 384-dimensional float32 vector stored in sqlite-vec; used for cosine similarity search; schema is unchanged

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The production Docker image is smaller than the ONNX Runtime-based image (target: under 60 MB runtime layer for the apk-installed packages, down from 75 MB)
- **SC-002**: Semantic search queries return results with equivalent relevance to the ONNX Runtime baseline — verified by running the same query set against both runtimes and comparing top-5 results
- **SC-003**: Time from server start (with model files already present) to first successful embedding generation is under 5 seconds
- **SC-004**: Embedding generation throughput is sufficient for backfill — at least 10 notes per second on a single CPU core
- **SC-005**: The full test suite passes with the ncnn-backed embedder in place

## Clarifications

### Session 2026-05-07

- Q: Where do the ncnn model files (.param + .bin) come from / how are they distributed at runtime? → A: Converted offline via a dedicated GitHub Actions pipeline; the resulting `.param` and `.bin` files are published as assets on a GitHub Release and downloaded on first server start (same pattern as the current ONNX model download).
- Q: How should Go call into ncnn? → A: Write a thin CGO wrapper (~150 lines) directly against the ncnn C API (`ncnn.h`); no third-party Go binding dependency.
- Q: How should tokenization (text → token IDs) be handled? → A: Reuse the existing `tokenizer.json` with a pure-Go or CGO HuggingFace-compatible tokenizer; guarantees identical token IDs to the ONNX baseline.
- Q: What is the contingency if ncnn cannot accurately convert or run the all-MiniLM-L6-v2 model? → A: If ncnn embeddings fail the relevance gate (SC-002), attempt tract (Rust-based pure ONNX runtime, no C++ deps) as the next candidate before declaring the migration blocked.

## Assumptions

- The all-MiniLM-L6-v2 model can be converted to ncnn format with acceptable accuracy loss (conversion is a one-time offline step, not done at runtime)
- ncnn's ONNX-to-ncnn conversion tool (`onnx2ncnn`) correctly handles the transformer architecture used by all-MiniLM-L6-v2; if it does not, tract (pure Rust ONNX runtime) is the next candidate
- A thin CGO wrapper against the ncnn C API (`ncnn.h`) is sufficient to run inference and extract output tensors; no third-party Go binding is used
- Tokenisation uses the existing `tokenizer.json` file via a HuggingFace-compatible pure-Go or CGO tokenizer; ncnn handles only the neural network inference layer
- The persistent `/data/models` volume pattern (already in place) is reused for ncnn model files
- Alpine edge's C++ stdlib (libstdc++) is still required for ncnn but its transitive deps (protobuf, ICU, abseil-cpp) are not
