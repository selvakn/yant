# Tasks: ncnn Semantic Search Runtime

**Input**: Design documents from `specs/023-ncnn-semantic-search/`
**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓

**Build strategy**: ncnn CGO code is gated behind `//go:build ncnn` build tag.  
`make test` (no tag) compiles stub — CI stays green without ncnn installed.  
`make docker-build` passes `-tags ncnn` — ncnn inference active in Docker.

## Format: `[ID] [P?] [Story] Description`

---

## Phase 1: Setup

**Purpose**: Create the model conversion CI pipeline (must exist before runtime code can be tested end-to-end).

- [ ] T001 Create .github/workflows/convert-model.yml — PNNX conversion pipeline that downloads all-MiniLM-L6-v2, converts via PNNX, validates output shape and cosine similarity, and publishes model.ncnn.param + model.ncnn.bin as GitHub Release assets on tag `model-v*`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: CGO bridge files and compile-time stub that all ncnn embedding code depends on.

**⚠️ CRITICAL**: No US1 work can begin until T002–T004 are complete and `make test` passes.

- [ ] T002 [P] Create backend/internal/embedding/ncnn_bridge.h — C header declaring `ncnn_embed_init`, `ncnn_embed_run`, `ncnn_embed_destroy` wrapping the ncnn C API
- [ ] T003 [P] Create backend/internal/embedding/ncnn_bridge.c — C implementation calling ncnn_net/extractor C API; loads .param + .bin once at init; sets num_threads=1; returns last_hidden_state float* for mean pooling in Go
- [ ] T004 Create backend/internal/embedding/stub.go (`//go:build !ncnn`) — stub `Embedder` struct and `New()` that returns `(nil, ErrNotAvailable)`; allows `go test ./...` to pass on CI without ncnn installed

**Checkpoint**: `make test` passes, `make lint` passes, stub compiles cleanly.

---

## Phase 3: User Story 1 — Semantic Search Works with ncnn (Priority: P1) 🎯 MVP

**Goal**: Replace the ONNX Runtime embedder with an ncnn-backed embedder; search experience identical to before.

**Independent Test**: Build Docker image with `-tags ncnn`, run server, embed a query and verify the search returns semantically related notes.

### Tests for User Story 1

- [ ] T005 [P] [US1] Create backend/internal/embedding/embedder_test.go — test stub returns ErrNotAvailable when not built with ncnn tag; test normalize() correctness; test that Embed("") returns zero vector of correct dimension; add a golden vector test (skipped if `TEST_NCNN_MODEL_PARAM` env not set) that embeds "hello world" and checks cosine sim > 0.999 vs pre-computed reference

### Implementation for User Story 1

- [ ] T006 [US1] Replace backend/internal/embedding/embedder.go — rewrite with `//go:build ncnn` tag; keep `Embedder` struct (swap `*ort.DynamicAdvancedSession` for a `*ncnnNet` CGO handle); update `New(paramPath, binPath, tokenizerPath string)` to load ncnn model; keep `Embed()` logic (tokenize → CGO bridge → mean pool → normalize); keep `Close()`, `normalize()`, `Dimensions` constant
- [ ] T007 [US1] Update backend/cmd/server/main.go — replace `--onnx-lib` / `ONNXRUNTIME_LIB_PATH` with `--model-param` / `MODEL_PATH` (param file) and `--model-bin` / `MODEL_BIN_PATH` (bin file); update `initEmbedder()` to download model.ncnn.param and model.ncnn.bin from GitHub Release asset URLs; remove modelDownloadURL constant for model.onnx; keep tokenizerDownloadURL unchanged

**Checkpoint**: `make test` passes; `go build -tags ncnn` compiles (if ncnn headers available in Docker).

---

## Phase 4: User Story 2 — Smaller Docker Image (Priority: P2)

**Goal**: Remove ONNX Runtime from the image; ncnn static-linked into Go binary; runtime layer drops to ~15 MB.

**Independent Test**: `make docker-build && docker image inspect yant:latest --format '{{.Size}}'` — total < 65 MB.

### Implementation for User Story 2

- [ ] T008 [US2] Update Dockerfile — add `ncnn-builder` stage (alpine:edge, cmake build-base git, clone ncnn, cmake with DNCNN_SHARED_LIB=OFF DNCNN_BUILD_TESTS=OFF, build); update `backend-builder` stage to COPY libncnn.a + c_api.h from ncnn-builder, add g++, set CGO_LDFLAGS, add `-tags ncnn` to go build; update `runtime` stage to remove `onnxruntime-dev`, rename MODEL_PATH env to model.ncnn.param path, add MODEL_BIN_PATH env
- [ ] T009 [US2] Update .github/workflows/ci.yml — remove `ONNXRUNTIME_LIB_PATH` env var; verify `CGO_ENABLED=1` remains; no other changes needed (docker build inherits Dockerfile tags)

**Checkpoint**: `make docker-build` succeeds; image size < 65 MB; `docker run` starts server without errors.

---

## Phase 5: User Story 3 — Graceful Degradation (Priority: P3)

**Goal**: Server falls back to FTS5 text search when ncnn embedder unavailable; no panics or errors shown to user.

**Independent Test**: Start server with `SEMANTIC_SEARCH=false`; search returns text-based results without error.

### Implementation for User Story 3

- [ ] T010 [US3] Verify backend/cmd/server/main.go — confirm `initEmbedder` failure (download error or ncnn init error) is logged and does NOT call `h.SetEmbedder`; confirm `h.embedder.Load()` nil-check in notes.go and handlers.go falls back cleanly; no code changes expected — add comment if logic was unclear

**Checkpoint**: `make test` passes; graceful degradation path is covered by existing test or T005 golden test skip logic.

---

## Phase N: Polish & Cleanup

- [ ] T011 [P] Remove `github.com/yalue/onnxruntime_go` from backend/go.mod — run `go mod tidy` in backend/; verify no remaining imports of the package
- [ ] T012 [P] Remove `github.com/ebitengine/purego` indirect dep if it was only needed by onnxruntime_go — verify after T011 go mod tidy
- [ ] T013 Update CLAUDE.md active technologies — replace onnxruntime_go with ncnn CGO bridge; note build tag `-tags ncnn` required for embedding; note conversion pipeline

---

## Dependencies & Execution Order

- **Phase 1** (T001): Independent — create CI workflow file only
- **Phase 2** (T002–T004): Independent of Phase 1; must complete before Phase 3
- **Phase 3** (T005–T007): Depends on Phase 2 complete; T005 and T006 can start in parallel with T007
- **Phase 4** (T008–T009): Depends on Phase 3 complete (needs updated embedder + main.go)
- **Phase 5** (T010): Can run alongside Phase 4 (only verification, no code changes)
- **Phase N** (T011–T013): After Phase 4 complete

### Parallel Opportunities

- T002 and T003 (C bridge files): fully parallel (different files)
- T001 and T002/T003/T004: fully parallel (different concerns)
- T005 and T007: partial parallel (different files; T006 depends on T005)
- T011 and T012 and T013: fully parallel

---

## Implementation Strategy

### MVP (US1 only)
1. Phase 2: CGO bridge + stub → `make test` green
2. Phase 3: ncnn embedder + main.go changes → `go build -tags ncnn` compiles
3. `make docker-build` → verify end-to-end in Docker

### Full delivery order
Phase 1 → Phase 2 → Phase 3 → Phase 4 → Phase 5 → Phase N

---

## Notes

- `[P]` = can run in parallel (different files, no blocking dependency)
- Build tag `ncnn`: add to Go build with `-tags ncnn`; controls whether CGO ncnn code compiles
- `make test` must stay green throughout (stub satisfies interface when `!ncnn`)
- Run `make test && make lint` before every commit (Constitution Principle VI)
- The conversion workflow (T001) must run and publish model files before end-to-end ncnn embedding can be tested
