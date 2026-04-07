# API Contracts: Semantic Search

**Feature**: 011-semantic-search
**Date**: 2026-04-05

## Modified Endpoints

### GET /notes/search?q={query}

**Change**: Backend switches from fuzzy matching to semantic similarity search (when semantic search toggle is enabled). Response format is unchanged.

**Request**: Same as current — query string parameter `q`.

**Behavior change**:
- Previously: fuzzy string matching against title, tags, body (keystroke-by-keystroke).
- Now: debounced query → backend generates embedding for query → KNN against stored note embeddings → results ranked by cosine similarity.
- Notes without embeddings fall back to text-based title/tag matching and are appended after semantic results.
- Results filtered by minimum similarity threshold, capped at maximum count.
- When semantic search toggle is disabled: uses existing text-based matching (same behavior as before this feature).

**Response**: Same HTML fragment (htmx swap). No API contract change.

### GET /archive/search?q={query}

**Change**: Same semantic search behavior as `/notes/search`, but filtered to archived notes only.

### POST /notes (create note)

**Side effect added**: After note is saved and tags synced, embedding is generated asynchronously (or synchronously if fast enough) and stored. No change to request/response contract.

### PUT /notes/{slug} (update note)

**Side effect added**: After note is updated and tags synced, embedding is re-generated if content hash has changed. No change to request/response contract.

### DELETE /notes/{slug} (delete note)

**Side effect added**: Embedding is removed from both `note_embeddings` and `vec_note_embeddings`. No change to request/response contract.

### POST /notes/{slug}/archive (archive note)

**No change**: Embedding is retained. Search filtering by `archived` column already handles this.

### POST /notes/{slug}/unarchive (unarchive note)

**No change**: Same reasoning as archive.

## New Endpoints

None. Semantic search is integrated into existing search endpoints.

## Configuration Flags

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `-semantic-search` | `SEMANTIC_SEARCH` | `true` | Enable/disable semantic search. When disabled, text-based matching is used. Embeddings are still generated. |
| `-search-debounce` | `SEARCH_DEBOUNCE_MS` | `300` | Frontend search debounce delay in milliseconds. |
| `-model-path` | `MODEL_PATH` | `/app/models/all-MiniLM-L6-v2.onnx` | Path to the ONNX embedding model file. |

## Frontend Contract Changes

### Search box behavior

The search input's htmx trigger changes from `keyup` to a debounced trigger:

```
hx-trigger="keyup changed delay:300ms"
```

The delay value (300ms) is injected from server configuration via the template.

No other frontend changes. The search results HTML fragment format is identical.
