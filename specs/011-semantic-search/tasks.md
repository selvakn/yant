# Tasks: Semantic Search

**Input**: Design documents from `/specs/011-semantic-search/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: The constitution (Principle IV, VI) mandates integration tests and ≥90% backend coverage. Test tasks are included. The spec explicitly requires API-level integration tests via testcontainers.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add new dependencies, create new packages, update build configuration

- [x] T001 Add `modernc.org/sqlite/vec` dependency to backend/go.mod (blank import for sqlite-vec support)
- [x] T002 Add `github.com/yalue/onnxruntime_go` dependency to backend/go.mod for ONNX Runtime inference
- [x] T003 Add `github.com/testcontainers/testcontainers-go` dependency to backend/go.mod for integration tests
- [x] T004 Create backend/internal/embedding/ package directory structure (onnx.go, tokenizer.go)
- [x] T005 Download all-MiniLM-L6-v2 ONNX model and vocab.txt; add `make download-model` target to Makefile that downloads the ONNX model file to models/all-MiniLM-L6-v2.onnx and vocab.txt to backend/internal/embedding/vocab.txt
- [x] T006 Add `models/` to .gitignore
- [x] T007 Add new server flags (`-semantic-search`, `-search-debounce`, `-model-path`) with env var fallbacks (`SEMANTIC_SEARCH`, `SEARCH_DEBOUNCE_MS`, `MODEL_PATH`) to backend/cmd/server/main.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T008 Add sqlite-vec blank import (`_ "modernc.org/sqlite/vec"`) to backend/internal/models/models.go and add schema migration in migrateSchema for `note_embeddings` table (note_id INTEGER PRIMARY KEY REFERENCES notes(id) ON DELETE CASCADE, content_hash TEXT NOT NULL, updated_at TEXT NOT NULL) and `vec_note_embeddings` virtual table (`CREATE VIRTUAL TABLE IF NOT EXISTS vec_note_embeddings USING vec0(note_id INTEGER PRIMARY KEY, embedding FLOAT[384] distance_metric=cosine)`)
- [x] T009 [P] Implement WordPiece tokenizer in backend/internal/embedding/tokenizer.go: load vocab.txt (Go embed), tokenize text into WordPiece token IDs with [CLS]/[SEP] tokens, handle unknown tokens with ##subword splitting, truncate to max sequence length (512 tokens)
- [x] T010 [P] Implement ONNX Runtime inference wrapper in backend/internal/embedding/onnx.go: Embedder struct with Init(modelPath string)/Close()/Embed(text string) ([]float32, error) methods; load ONNX session, run inference with tokenized input (input_ids, attention_mask, token_type_ids), extract pooled 384-dim output, normalize to unit vector
- [x] T011 Write unit tests for WordPiece tokenizer in backend/internal/embedding/tokenizer_test.go: test basic tokenization, subword splitting, [CLS]/[SEP] insertion, truncation, empty input, special characters
- [x] T012 Write unit tests for ONNX embedder in backend/internal/embedding/onnx_test.go: test initialization with valid/invalid model path, embedding generation produces 384-dim normalized vector, embedding of similar texts produces high cosine similarity

**Checkpoint**: Foundation ready — embedding infrastructure and vector storage schema in place

---

## Phase 3: User Story 2 — Embedding Generation on Note Save (Priority: P1)

**Goal**: Automatically generate and store vector embeddings when notes are created or updated. This is a prerequisite for semantic search (US1).

**Independent Test**: Save a note and verify that an embedding record exists in note_embeddings with the correct content_hash, and a corresponding row in vec_note_embeddings.

### Implementation for User Story 2

- [x] T013 [US2] Create backend/internal/models/embeddings.go: implement ContentHash(title, body string) string (SHA-256), UpsertEmbedding(db *DB, noteID int64, embedding []float32, contentHash string) error (insert/replace into both note_embeddings and vec_note_embeddings), DeleteEmbedding(db *DB, noteID int64) error (delete from both tables), GetContentHash(db *DB, noteID int64) (string, bool, error), NeedsEmbedding(db *DB, noteID int64, currentHash string) bool
- [x] T014 [US2] Write unit tests for embedding CRUD in backend/internal/models/embeddings_test.go: test UpsertEmbedding creates record, UpsertEmbedding updates on content change, DeleteEmbedding removes from both tables, NeedsEmbedding returns true for new/changed notes and false for unchanged
- [x] T015 [US2] Update backend/internal/handlers/handlers.go: add Embedder field (interface with Embed method) and SemanticSearchEnabled bool to Handler struct; update New() constructor to accept embedder and config
- [x] T016 [US2] Update backend/internal/handlers/notes.go: in NotesCreatePOST and noteUpdate, after SyncTags/SyncLinks, call ContentHash on title+body, check NeedsEmbedding, if true call Embedder.Embed and UpsertEmbedding; handle embedding failures gracefully (log error, don't fail the save)
- [x] T017 [US2] Update backend/internal/handlers/notes.go: in NoteDeletePOST, call DeleteEmbedding to remove embedding when note is deleted
- [x] T018 [US2] Update backend/cmd/server/main.go: initialize Embedder with model path, pass to handlers.New(); if model file not found, log warning and set embedder to nil (graceful degradation)
- [x] T019 [US2] Update existing handler tests in backend/internal/handlers/handlers_test.go: update newTestApp to pass nil embedder (or mock embedder interface); verify that note create/update still works when embedder is nil; add test that embedding is generated on save when embedder is provided (using a fake embedder that returns fixed vectors)
- [x] T020 [US2] Handle edge cases in embedding generation: title-only notes (embed title alone), very long content (truncate to model's max input ~512 tokens worth of text), in backend/internal/models/embeddings.go add PrepareEmbeddingText(title, body string) string with truncation logic

**Checkpoint**: Notes generate and store embeddings on save. Embedding failures don't break note saving.

---

## Phase 4: User Story 1 — Semantic Note Search (Priority: P1)

**Goal**: Users can search notes by meaning rather than exact keywords. Semantically related notes appear in results ranked by cosine similarity.

**Independent Test**: Create notes about "Docker deployment" and "Kubernetes setup", search for "container orchestration", verify both appear. Create a "cooking recipes" note, verify it doesn't appear.

**Depends on**: User Story 2 (embeddings must exist to search against)

### Implementation for User Story 1

- [x] T021 [US1] Create backend/internal/models/semantic_search.go: implement SemanticSearch(db *DB, notesDir string, userID int64, query string, queryEmbedding []float32, archived bool, threshold float64, maxResults int) ([]SearchResult, error) — KNN query against vec_note_embeddings with cosine distance, join with notes table, filter by user_id and archived, apply similarity threshold, cap results
- [x] T022 [US1] Implement text-based fallback in backend/internal/models/semantic_search.go: add TextFallbackSearch(db *DB, notesDir string, userID int64, query string, archived bool, excludeNoteIDs []int64) ([]SearchResult, error) that searches notes without embeddings using existing fuzzy matching logic, excluding notes already found by semantic search
- [x] T023 [US1] Implement merged search in backend/internal/models/semantic_search.go: add MergedSearch(db *DB, notesDir string, userID int64, query string, queryEmbedding []float32, archived bool, threshold float64, maxResults int) ([]SearchResult, error) that combines semantic results with text-fallback results for notes lacking embeddings
- [x] T024 [US1] Modify backend/internal/models/search.go: update SearchNotes to accept a semanticEnabled bool, embedder, threshold, and maxResults parameters; when enabled, call MergedSearch; when disabled, call existing fuzzy search logic (preserve current behavior as fallback)
- [x] T025 [US1] Update backend/internal/handlers/notes.go NotesSearchGET: embed the query string, call updated SearchNotes with semantic flag from handler config, pass threshold and maxResults
- [x] T026 [US1] Write unit tests for semantic search in backend/internal/models/semantic_search_test.go: test KNN returns semantically similar results (using pre-computed fixed embeddings), test threshold filtering excludes low-similarity notes, test max results cap, test fallback for notes without embeddings, test empty query returns all notes
- [x] T027 [US1] Update search handler tests in backend/internal/handlers/handlers_test.go: test search with semantic enabled returns results, test search with semantic disabled uses text matching, test feature toggle behavior

**Checkpoint**: Semantic search works end-to-end for active notes. Users can find notes by meaning.

---

## Phase 5: User Story 3 — Bulk Embedding Rebuild (Priority: P2)

**Goal**: On startup, generate embeddings for all notes that don't have them yet. Ensures existing notes become searchable.

**Independent Test**: Start the app with 10 existing notes that have no embeddings. After startup completes, verify all 10 have embeddings.

### Implementation for User Story 3

- [x] T028 [US3] Implement BackfillEmbeddings in backend/internal/models/embeddings.go: query all notes that have no entry in note_embeddings, read their body from disk, generate embeddings via Embedder, insert into both tables; log progress (processed N of M notes)
- [x] T029 [US3] Call BackfillEmbeddings during server startup in backend/cmd/server/main.go: after DB initialization and schema migration, run backfill if embedder is available; log completion time
- [x] T030 [US3] Write unit tests for BackfillEmbeddings in backend/internal/models/embeddings_test.go: test that notes without embeddings get backfilled, notes with existing embeddings are skipped, backfill handles embedding failures gracefully (skips failed notes, continues with rest)

**Checkpoint**: Existing notes get embeddings on startup. New deployments with existing data work seamlessly.

---

## Phase 6: User Story 4 — Search Across Active and Archived Notes (Priority: P2)

**Goal**: Semantic search respects active/archived separation, matching current behavior.

**Independent Test**: Create active and archived notes about the same topic. Search in active section returns only active notes. Search in archive section returns only archived notes.

### Implementation for User Story 4

- [x] T031 [US4] Update backend/internal/handlers/archive.go ArchiveSearchGET: use the same semantic search path as NotesSearchGET but with archived=true parameter
- [x] T032 [US4] Write tests for archive semantic search in backend/internal/handlers/handlers_test.go: test that archive search with semantic enabled only returns archived notes, active notes are excluded

**Checkpoint**: Semantic search works identically in both active and archive sections.

---

## Phase 7: User Story 5 — Docker Distribution with All Dependencies (Priority: P2)

**Goal**: Docker image bundles ONNX Runtime, embedding model, and all dependencies. Semantic search works out of the box.

**Independent Test**: Build Docker image, run container, create notes, verify semantic search returns meaningful results without any external setup.

### Implementation for User Story 5

- [x] T033 [US5] Update Dockerfile: change Go build stage to CGO_ENABLED=1 with necessary C toolchain (gcc, musl-dev if Alpine); install ONNX Runtime shared library in build stage
- [x] T034 [US5] Update Dockerfile: add a model download stage or COPY the ONNX model file into the runtime image at /app/models/all-MiniLM-L6-v2.onnx; copy ONNX Runtime shared library to runtime stage; set LD_LIBRARY_PATH if needed
- [x] T035 [US5] Update Dockerfile: add SEMANTIC_SEARCH and MODEL_PATH environment variables with defaults; update CMD/ENTRYPOINT to include `-model-path /app/models/all-MiniLM-L6-v2.onnx`
- [x] T036 [US5] Update Makefile: update docker-build target if needed; verify `make docker-build && make docker-run` produces a working image with semantic search
- [x] T037 [US5] Test Docker image locally: build image, run container, create a few notes via the UI, search with a conceptual query, verify results are semantically relevant

**Checkpoint**: Docker image is self-contained. No external API keys, services, or setup needed.

---

## Phase 8: User Story 6 — API-Level Integration Tests (Priority: P1)

**Goal**: All API endpoints are covered by integration tests running against the real Docker image via testcontainers-go. No mocking.

**Independent Test**: Run `make integration-test` and all tests pass.

### Implementation for User Story 6

- [x] T038 [US6] Create backend/internal/integration/helpers_test.go with `//go:build integration` tag: implement TestMain that starts the app Docker image via testcontainers-go (WithExposedPorts "8080/tcp", wait.ForHTTP("/login").WithPort("8080/tcp")), resolve base URL, create shared HTTP client; implement helper functions for authentication (login flow), creating notes (POST), reading notes (GET), searching (GET /notes/search?q=), archiving, deleting
- [x] T039 [US6] Create backend/internal/integration/integration_test.go with `//go:build integration` tag: implement test cases for note CRUD — TestCreateNote (POST create, verify 200/redirect), TestReadNote (GET /notes/{slug}, verify content), TestUpdateNote (PUT update, verify changes), TestDeleteNote (DELETE, verify gone), TestArchiveNote (archive, verify in archive list, not in active list), TestUnarchiveNote (unarchive, verify back in active list)
- [x] T040 [US6] Add semantic search integration tests in backend/internal/integration/integration_test.go: TestSemanticSearch (create 3 notes about different topics, search with conceptual query, verify relevant notes ranked higher than irrelevant), TestSemanticSearchThreshold (verify low-similarity notes excluded), TestSemanticSearchArchived (verify archive search returns only archived notes), TestSearchToggleDisabled (if toggle mechanism is testable, verify text fallback works)
- [x] T041 [US6] Add `make integration-test` target to Makefile: `cd backend && go test -tags=integration -v -timeout=5m ./internal/integration/...`
- [x] T042 [US6] Update .github/workflows/ci.yml: add integration-test job that depends on build-scan-push (uses the built Docker image), runs `make integration-test`

**Checkpoint**: Full API test coverage. `make integration-test` passes consistently.

---

## Phase 9: Frontend — Search Debounce

**Purpose**: Update search UI to use debounced triggering instead of keystroke-by-keystroke

- [x] T043 Update frontend/templates/notes/list.html: change the search input's hx-trigger from `keyup` to `keyup changed delay:{{.SearchDebounceMS}}ms` where SearchDebounceMS is injected from server config (default 300)
- [x] T044 Update frontend/templates/archive/list.html: apply the same debounce change as T043
- [x] T045 Update backend/internal/handlers/notes.go and archive.go: pass SearchDebounceMS to template data from handler config
- [x] T046 Update existing search-related handler tests to verify the debounce value is passed to templates

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: Final cleanup, documentation, CI integration

- [x] T047 Run `make test` and `make coverage` — verify ≥90% backend line coverage with all new code; add any missing unit tests to reach threshold
- [x] T048 Run `make lint` — fix any go vet warnings
- [x] T049 Update README.md: add semantic search section describing the feature, configuration flags, and how to enable/disable
- [x] T050 Run full validation: `make docker-build && make test && make integration-test` — all must pass
- [x] T051 Update CLAUDE.md if new technologies need to be reflected

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion — BLOCKS all user stories
- **US2 Embedding Generation (Phase 3)**: Depends on Foundational — BLOCKS US1
- **US1 Semantic Search (Phase 4)**: Depends on US2 (needs embeddings to search against)
- **US3 Bulk Rebuild (Phase 5)**: Depends on US2 (uses same embedding infrastructure)
- **US4 Active/Archived (Phase 6)**: Depends on US1 (extends semantic search to archives)
- **US5 Docker Packaging (Phase 7)**: Depends on US1 + US2 (packages the working feature)
- **US6 Integration Tests (Phase 8)**: Depends on US5 (tests against Docker image)
- **Debounce (Phase 9)**: Can start after Foundational, independent of other stories
- **Polish (Phase 10)**: Depends on all phases being complete

### User Story Dependencies

```
Phase 1: Setup
    │
    ▼
Phase 2: Foundational (tokenizer, ONNX, schema)
    │
    ├──────────────────┐
    ▼                  ▼
Phase 3: US2        Phase 9: Debounce (independent)
(Embed on save)
    │
    ├──────────┐
    ▼          ▼
Phase 4:    Phase 5:
US1         US3
(Search)    (Backfill)
    │
    ▼
Phase 6: US4
(Archive search)
    │
    ▼
Phase 7: US5
(Docker packaging)
    │
    ▼
Phase 8: US6
(Integration tests)
    │
    ▼
Phase 10: Polish
```

### Within Each User Story

- Models/data layer before handlers
- Handlers before tests that depend on them
- Unit tests alongside implementation (same phase)
- Story complete before moving to next priority

### Parallel Opportunities

- T009 (tokenizer) and T010 (ONNX wrapper) can run in parallel in Phase 2
- T011 and T012 (tests) can run in parallel in Phase 2
- Phase 9 (debounce) can run in parallel with Phases 3-8
- T033/T034/T035 (Dockerfile stages) are sequential but scoped to a single file

---

## Parallel Example: Phase 2 (Foundational)

```
# These can be implemented in parallel (different files):
Task T009: "Implement WordPiece tokenizer in backend/internal/embedding/tokenizer.go"
Task T010: "Implement ONNX Runtime wrapper in backend/internal/embedding/onnx.go"

# Then tests in parallel:
Task T011: "Unit tests for tokenizer in backend/internal/embedding/tokenizer_test.go"
Task T012: "Unit tests for ONNX embedder in backend/internal/embedding/onnx_test.go"
```

---

## Implementation Strategy

### MVP First (User Stories 1 + 2)

1. Complete Phase 1: Setup (dependencies, flags)
2. Complete Phase 2: Foundational (tokenizer, ONNX, schema)
3. Complete Phase 3: US2 — Embedding on save
4. Complete Phase 4: US1 — Semantic search
5. **STOP and VALIDATE**: Search works end-to-end for active notes
6. Deploy/demo if ready

### Incremental Delivery

1. Setup + Foundational → Infrastructure ready
2. Add US2 (embed on save) → Notes generate embeddings
3. Add US1 (semantic search) → **MVP — search by meaning works!**
4. Add US3 (backfill) → Existing notes become searchable
5. Add US4 (archive search) → Full search parity
6. Add US5 (Docker) → Self-contained distribution
7. Add US6 (integration tests) → Full confidence
8. Debounce + Polish → Production ready

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- **Constitution**: Principle VI in YANT requires running the full test suite before every commit and fixing failures before new work or further commits
- Commit after each task or logical group (only when tests pass)
- Stop at any checkpoint to validate story independently
- The embedder interface allows nil/mock embedders in tests — no ONNX Runtime needed for unit tests
