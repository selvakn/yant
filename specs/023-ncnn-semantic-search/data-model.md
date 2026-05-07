# Data Model: ncnn Semantic Search Runtime

**Feature**: 023-ncnn-semantic-search
**Date**: 2026-05-07

---

## Schema Changes

**None.** The SQLite schema is unchanged. Embedding vectors remain 384-dimensional float32 stored in sqlite-vec. The `note_embeddings` table, `content_hash` column, and all query patterns are identical.

---

## Model Files on Disk (changed)

| Before | After |
|--------|-------|
| `/data/models/model.onnx` | `/data/models/model.ncnn.param` |
| `/data/models/tokenizer.json` | `/data/models/model.ncnn.bin` |
| | `/data/models/tokenizer.json` *(unchanged)* |

- `model.ncnn.param`: ncnn network topology (text format, ~50–200 KB)
- `model.ncnn.bin`: ncnn weight binary (same size as model.onnx, ~22 MB)
- `tokenizer.json`: unchanged; still used by the existing pure-Go tokenizer

**Download URLs** (set in `cmd/server/main.go` constants):
- `model.ncnn.param` and `model.ncnn.bin` pulled from the GitHub Release asset URL produced by the conversion CI pipeline
- `tokenizer.json` continues to be pulled from HuggingFace (unchanged)

---

## Go Interface (unchanged)

The `embedding.Embedder` interface in `backend/internal/embedding/embedder.go` is unchanged:

```go
type Embedder interface {
    Embed(text string) ([]float32, error)
}
```

The `New()` constructor signature changes — it now accepts `.param` and `.bin` paths instead of an ONNX path and lib path:

| Before | After |
|--------|-------|
| `embedding.New(onnxLibPath, modelPath, tokenizerPath)` | `embedding.New(paramPath, binPath, tokenizerPath)` |

The `Handler` struct, `atomic.Pointer[embedding.Embedder]`, `SetEmbedder()`, and all call sites in `notes.go` and `search.go` are unchanged.

---

## Environment Variables (changed)

| Variable | Before | After |
|----------|--------|-------|
| `ONNXRUNTIME_LIB_PATH` | `/usr/lib/libonnxruntime.so` | **Removed** |
| `MODEL_PATH` | `/data/models/model.onnx` | `/data/models/model.ncnn.param` |
| `MODEL_BIN_PATH` | *(not present)* | `/data/models/model.ncnn.bin` *(new)* |
| `TOKENIZER_PATH` | `/data/models/tokenizer.json` | unchanged |
| `SEMANTIC_SEARCH` | `true` | unchanged |

---

## Inference Data Flow (runtime)

```
text string
  │
  ▼
clems4ever/tokenizer  (pure Go, tokenizer.json)
  → []int32 input_ids      (shape: [1, 128])
  → []int32 attention_mask (shape: [1, 128])
  → []int32 token_type_ids (shape: [1, 128])
  │
  ▼
ncnn extractor (CGO bridge → libncnn.a)
  → last_hidden_state output (shape: [1, 128, 384])
  │
  ▼
mean pooling (Go, attention_mask weighted)
  → []float32 embedding (shape: [384])
  │
  ▼
L2 normalise (Go)
  → []float32 unit vector (shape: [384])
  │
  ▼
sqlite-vec cosine similarity search
```

**Note**: Mean pooling and L2 normalisation are implemented in Go, not inside the ncnn model graph, because PNNX may not export the pooling head cleanly. This matches the approach used by the current ONNX implementation.
