# Implementation Plan: ncnn Semantic Search Runtime

**Branch**: `023-ncnn-semantic-search` | **Date**: 2026-05-07 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `specs/023-ncnn-semantic-search/spec.md`

## Summary

Replace the ONNX Runtime inference backend with ncnn for sentence embedding generation. ncnn is not packaged for Alpine — it is compiled from source in a Docker builder stage and static-linked into the Go binary, eliminating onnxruntime and its protobuf/ICU/abseil-cpp transitive deps (~60 MB) from the runtime image. A GitHub Actions pipeline converts the all-MiniLM-L6-v2 model from PyTorch to ncnn format via PNNX and publishes the `.param` + `.bin` files as GitHub Release assets for download on first server start. The tokenizer layer (pure-Go, `tokenizer.json`) is unchanged.

## Technical Context

**Language/Version**: Go 1.25, CGO enabled, Alpine musl toolchain
**Primary Dependencies**: ncnn (static, built from source in Docker), `github.com/clems4ever/tokenizer` (existing pure-Go tokenizer, unchanged), `modernc.org/sqlite/vec` (unchanged)
**Storage**: Markdown files (unchanged), SQLite (unchanged, no schema changes)
**Testing**: `go test ./...`, `make coverage` (≥75% gate on internal/..., ≥90% constitution target), `testcontainers-go` integration tests
**Target Platform**: Linux/Alpine musl (Docker), `linux/amd64`
**Project Type**: Web service backend + CI pipeline
**Performance Goals**: First embedding after model-present start ≤5s; backfill ≥10 notes/sec
**Constraints**: Runtime image must not contain ONNX Runtime or its deps; ncnn must static-link into the Go binary; Alpine musl CGO
**Scale/Scope**: Personal notes app, single-user

## Constitution Check

- [x] **I. Markdown-first storage** — No changes to note storage; SQLite remains a derived cache only.
- [x] **II. Simplicity** — ncnn build complexity is justified: it directly replaces ONNX Runtime and saves ~60 MB from the image. Static-linking avoids a runtime shared-lib dependency. No new abstractions beyond the existing `Embedder` interface.
- [x] **III. Monorepo** — New code lives in `backend/internal/embedding/`; new CI workflow in `.github/workflows/`; no structural changes.
- [x] **IV. Integration testing** — Existing embedding integration tests extended for ncnn; new embedder unit test with a fixed test vector; coverage gate maintained.
- [x] **V. Simple Web UI** — No frontend changes.
- [x] **VI. Commit & test discipline** — Full test suite must pass before each commit; no skipping failing tests.

## Project Structure

### Documentation (this feature)

```text
specs/023-ncnn-semantic-search/
├── plan.md          ← this file
├── research.md      ← Phase 0 output
├── data-model.md    ← Phase 1 output
└── tasks.md         ← Phase 2 output (/speckit-tasks)
```

### Source Code Changes

```text
backend/
├── internal/
│   └── embedding/
│       ├── embedder.go          # interface — UNCHANGED
│       ├── embedding.go         # New() constructor — update signature (param+bin paths)
│       ├── ncnn_bridge.c        # NEW: thin C shim calling ncnn C API
│       ├── ncnn_bridge.h        # NEW: C header for the bridge
│       └── ncnn.go              # NEW: CGO embedder implementation
│           (replaces onnx.go / onnxruntime_go usage)
└── cmd/server/
    └── main.go                  # Update download URLs + env var names

Dockerfile                       # Add ncnn build stage; remove ONNX stages
.github/workflows/
├── ci.yml                       # Remove ONNXRUNTIME_LIB_PATH env; minor cleanup
└── convert-model.yml            # NEW: PNNX conversion + GitHub Release publish
```

### Files Removed

```text
backend/internal/embedding/onnx.go       (or equivalent ONNX implementation file)
```

## Implementation Phases

### Phase A: Model Conversion CI Pipeline

**Goal**: Produce and publish `model.ncnn.param` + `model.ncnn.bin` to a GitHub Release before writing any Go code. Gate: downloaded files run through a Python validation step confirming output shape `[1, 384]` and cosine similarity >0.999 vs Python baseline.

Steps:
1. Create `.github/workflows/convert-model.yml`
   - Trigger: manual dispatch + on tag `model-v*`
   - Runner: `ubuntu-latest` with Python 3.11
   - Steps:
     a. `pip install sentence-transformers pnnx torch`
     b. Download and export all-MiniLM-L6-v2 to TorchScript (`.pt`)
     c. Run `pnnx model.pt` with `inputshape=[1,128]i32` × 3 (input_ids, attention_mask, token_type_ids)
     d. Run `ncnnoptimize` on output
     e. Validate: load ncnn output in Python, run a test sentence, compare cosine similarity to HuggingFace reference (must be >0.999)
     f. Upload `model.ncnn.param` + `model.ncnn.bin` as GitHub Release assets tagged `model-v1`

2. Run the pipeline and confirm asset URLs before proceeding to Phase B.

### Phase B: ncnn Docker Build Stage

**Goal**: Compile ncnn from source in the Docker builder and produce a static library (`libncnn.a`) for the Go linker to consume.

Changes to `Dockerfile`:
```dockerfile
# NEW stage: build ncnn static library
FROM alpine:edge AS ncnn-builder
RUN apk add --no-cache cmake build-base git
RUN git clone --depth 1 --branch 20240410 https://github.com/Tencent/ncnn.git /ncnn
RUN cmake -S /ncnn -B /ncnn/build \
      -DNCNN_SHARED_LIB=OFF \
      -DNCNN_BUILD_TESTS=OFF \
      -DNCNN_BUILD_TOOLS=OFF \
      -DNCNN_BUILD_EXAMPLES=OFF \
      -DNCNN_ENABLE_LTO=ON \
      -DCMAKE_BUILD_TYPE=Release && \
    cmake --build /ncnn/build -j$(nproc)
# output: /ncnn/build/src/libncnn.a + /ncnn/src/c_api.h

# In backend-builder stage: add ncnn headers + lib
COPY --from=ncnn-builder /ncnn/build/src/libncnn.a /usr/local/lib/
COPY --from=ncnn-builder /ncnn/src/c_api.h /usr/local/include/
RUN apk add --no-cache gcc musl-dev g++
# CGO_LDFLAGS picks up -lncnn -lgomp -lstdc++
RUN CGO_ENABLED=1 CGO_LDFLAGS="-L/usr/local/lib -lncnn -lstdc++ -lgomp -static-libstdc++ -static-libgcc" \
    go build -ldflags="-s -w" -o /server ./cmd/server

# Runtime stage: REMOVE onnxruntime-dev; no new packages
FROM alpine:edge AS runtime
RUN apk add --no-cache git ca-certificates && \   # onnxruntime-dev REMOVED
    adduser -D -u 65532 nonroot && ...
```

### Phase C: ncnn Go Embedder

**Goal**: Replace `onnxruntime_go`-backed embedder with ncnn-backed embedder implementing the same `Embedder` interface.

**`backend/internal/embedding/ncnn_bridge.h`**:
```c
#ifndef NCNN_BRIDGE_H
#define NCNN_BRIDGE_H
int ncnn_embed(const char* param_path, const char* bin_path,
               const int* input_ids, const int* attn_mask,
               const int* token_type_ids, int seq_len,
               float* out_embedding, int embed_dim);
#endif
```

**`backend/internal/embedding/ncnn_bridge.c`**:
- Wraps `ncnn_net_create/load_param/load_model`, `ncnn_extractor_*`
- Calls net once per `Embed()` invocation (stateless extractor; net loaded once at init)
- Sets `num_threads=1`
- Returns last_hidden_state output as `float*`; mean pooling + L2 norm done in Go

**`backend/internal/embedding/ncnn.go`**:
```go
// #cgo LDFLAGS: -lncnn -lstdc++
// #include "ncnn_bridge.h"
import "C"

type ncnnEmbedder struct {
    paramPath string
    binPath   string
    tok       *tokenizer.Tokenizer
}

func New(paramPath, binPath, tokenizerPath string) (*ncnnEmbedder, error)
func (e *ncnnEmbedder) Embed(text string) ([]float32, error)
```

**`cmd/server/main.go`** changes:
- Replace `MODEL_PATH` / `ONNXRUNTIME_LIB_PATH` with `MODEL_PARAM_PATH` / `MODEL_BIN_PATH`
- Update download URLs to GitHub Release asset URLs from Phase A
- Remove `downloadFile(tokenizerPath, ...)` if tokenizer.json is still from HuggingFace (keep as-is)

### Phase D: Tests & Validation

1. **Unit test** (`backend/internal/embedding/ncnn_test.go`):
   - Golden vector test: embed `"hello world"` → compare against pre-computed 384-dim reference vector (cosine sim >0.999 vs ONNX baseline)
   - Requires model files present; skip with `t.Skip` if `TEST_NCNN_MODEL` env not set

2. **Integration test** extension:
   - Extend existing semantic search integration tests to run against the ncnn embedder
   - Verify search result order is equivalent to ONNX baseline for a fixed test corpus

3. **Image size validation**:
   - `make docker-build && docker image inspect ... --format '{{.Size}}'`
   - Assert total < 60 MB (down from 108 MB)

### Phase E: Cleanup

- Remove `github.com/yalue/onnxruntime_go` (or `github.com/clems4ever/all-minilm-l6-v2-go`) from `go.mod`
- Remove `ONNXRUNTIME_LIB_PATH` env var from Dockerfile and CI
- Update `CLAUDE.md` active technologies list
- Update README / deployment docs with new env vars

## Complexity Tracking

| Addition | Why Needed | Simpler Alternative Rejected Because |
|----------|------------|--------------------------------------|
| ncnn build stage in Dockerfile | ncnn not in Alpine repos; must compile from source | Using pre-built binary: no official Alpine musl build; shipping a third-party binary is a security risk |
| CGO bridge (~150 lines C) | ncnn has no maintained Go bindings; C API is stable | Third-party `goncnn`: dormant (9 stars, no recent commits), not production-safe |
| GitHub Actions conversion pipeline | Model must be converted offline (PNNX) and hosted | Runtime conversion: too slow and too complex for a server startup |
