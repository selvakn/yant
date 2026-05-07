# Research: ncnn Semantic Search Runtime

**Feature**: 023-ncnn-semantic-search
**Date**: 2026-05-07

---

## Decision 1: ncnn availability on Alpine

**Decision**: Build ncnn from source in a dedicated Dockerfile builder stage; static-link into the Go binary.

**Rationale**: ncnn is not available in any Alpine Linux package repository (main, community, or edge). Building from source with `cmake` + `build-base` in the Docker builder stage and static-linking (`-DNCNN_SHARED_LIB=OFF`) bakes the inference library into the Go binary. The runtime image then needs zero new packages — only `git` and `ca-certificates` remain, dropping the apk-installed layer from 75 MB to ~15 MB.

**Alternatives considered**:
- Wait for Alpine to package ncnn: no timeline, not viable
- Ship `libncnn.so` in the runtime image: requires copying the .so file manually; static linking is simpler and slightly smaller

---

## Decision 2: Model conversion pipeline — PNNX over onnx2ncnn

**Decision**: Use PNNX (PyTorch → ncnn direct) in the GitHub Actions conversion pipeline, not `onnx2ncnn`.

**Rationale**: The all-MiniLM-L6-v2 ONNX export uses `Shape`, `Gather`, `Unsqueeze`, and dynamic `Reshape` ops that `onnx2ncnn` cannot reliably convert. These are fundamental to BERT/transformer attention masking. PNNX converts directly from PyTorch TorchScript, applies 509 graph transformation passes, and has explicit support for transformer attention patterns. It is the official ncnn recommendation for transformer models.

**Conversion pipeline**:
1. Load `sentence-transformers/all-MiniLM-L6-v2` via Python `sentence_transformers`
2. Export to TorchScript (`.pt`)
3. Run `pnnx model.pt inputshape=[1,128]i32 inputshape2=[1,128]i32 inputshape3=[1,128]i32` (input_ids, attention_mask, token_type_ids)
4. Run `ncnnoptimize` on the output
5. Verify output shape is `[1, 384]` float32 and cosine similarity against Python baseline is >0.999
6. Upload `.param` + `.bin` to GitHub Release

**Alternatives considered**:
- `onnx2ncnn` + `onnx-simplifier`: Simplifier removes redundant ops but cannot resolve LayerNorm subgraph or dynamic shape issues
- Pre-converted files from community: None found for this specific model

---

## Decision 3: Go tokenizer — keep existing sugarme/tokenizer

**Decision**: No tokenizer changes. The app already uses `github.com/clems4ever/tokenizer` (fork of `sugarme/tokenizer`), a pure-Go HuggingFace-compatible tokenizer that loads `tokenizer.json` directly. This is fully decoupled from ONNX Runtime.

**Rationale**: The tokenizer is an independent layer that produces token IDs; ncnn only receives the token ID tensors. The existing tokenizer already produces correct WordPiece token IDs identical to the HuggingFace Python reference. No change minimises risk and preserves FR-009 compliance.

**Alternatives considered**:
- `github.com/daulet/tokenizers` (CGO Rust): Adds Rust toolchain dependency to the build; no benefit over existing pure-Go solution
- Custom WordPiece tokenizer: Unnecessary given a working solution exists

---

## Decision 4: CGO wrapper — thin custom bridge against ncnn C API

**Decision**: Write a ~150-line C/Go CGO bridge directly against ncnn's stable `c_api.h`. No third-party Go binding.

**Rationale**: ncnn ships a stable C API (`src/c_api.h`) covering the full inference pipeline. Key functions:
```c
ncnn_net_t ncnn_net_create();
int ncnn_net_load_param(ncnn_net_t net, const char* path);
int ncnn_net_load_model(ncnn_net_t net, const char* path);
ncnn_extractor_t ncnn_extractor_create(ncnn_net_t net);
int ncnn_extractor_input(ncnn_extractor_t ex, const char* name, ncnn_mat_t mat);
int ncnn_extractor_extract(ncnn_extractor_t ex, const char* name, ncnn_mat_t* mat);
void ncnn_extractor_destroy(ncnn_extractor_t ex);
ncnn_mat_t ncnn_mat_create_1d(int w, ncnn_allocator_t alloc);
float* ncnn_mat_get_data(ncnn_mat_t mat);
```
Existing third-party Go bindings (`goncnn`: 9 stars, dormant) are unmaintained. `sherpa-ncnn` bindings are tightly coupled to speech recognition. Writing a minimal wrapper against the C API is ~150 lines, has no external dependency, and compiles cleanly against musl.

**Alternatives considered**:
- `goncnn`: Dormant, unmaintained, not production-safe
- `sherpa-ncnn` Go bindings: Designed for ASR, not embedding inference; too coupled

---

## Decision 5: ncnn thread count

**Decision**: Configure ncnn with `num_threads=1` for the embedding use case.

**Rationale**: Embedding generation is not latency-sensitive (async background process). Single-threaded avoids contention with the Go HTTP server's goroutine pool. ncnn's `ncnn_net_opt_set_num_threads(opt, 1)` sets this before model load.

---

## Risk: MiniLM model accuracy after PNNX conversion

**Status**: Unverified until the conversion CI pipeline runs.
**Mitigation**: SC-002 requires top-5 result equivalence against the ONNX baseline. The conversion pipeline must output a cosine similarity verification step. If accuracy fails, tract is the defined fallback (FR-010).

**PNNX known limitation**: Mean pooling (the sentence embedding aggregation step) may need to be handled outside the model graph — PNNX may not export it cleanly. If so, mean pooling is implemented in Go after extracting the last hidden state output.
