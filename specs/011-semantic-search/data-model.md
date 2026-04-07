# Data Model: Semantic Search

**Feature**: 011-semantic-search
**Date**: 2026-04-05

## Entities

### Note Embedding (new)

One-to-one relationship with `notes` table. Stores the vector representation of a note's content.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| note_id | INTEGER | PRIMARY KEY, REFERENCES notes(id) ON DELETE CASCADE | Links to the parent note |
| embedding | FLOAT[384] | NOT NULL | 384-dimensional vector from all-MiniLM-L6-v2 |
| content_hash | TEXT | NOT NULL | SHA-256 of the text that was embedded (title + body), used to detect when re-embedding is needed |
| updated_at | TEXT | NOT NULL | ISO 8601 timestamp of last embedding generation |

**Lifecycle**:
- Created when a note is saved (create or update) and embedding generation succeeds.
- Updated when note content changes (detected via content_hash comparison).
- Deleted when the parent note is deleted (CASCADE).
- Rebuilt in bulk on startup for notes missing embeddings.

### vec_note_embeddings (sqlite-vec virtual table, new)

sqlite-vec `vec0` virtual table for efficient KNN search.

```sql
CREATE VIRTUAL TABLE vec_note_embeddings USING vec0(
  note_id INTEGER PRIMARY KEY,
  embedding FLOAT[384] distance_metric=cosine
);
```

**Notes**:
- This is a virtual table managed by sqlite-vec, not a regular SQLite table.
- Rows are inserted/updated/deleted in sync with the `note_embeddings` metadata table.
- KNN queries use `WHERE embedding MATCH ?query AND k = ?limit`.
- Cosine distance is used (better for semantic similarity than L2).

## Existing Entities (unchanged)

### notes (existing)

No schema changes. The `archived` column is used to filter search results.

| Field | Type | Used by semantic search |
|-------|------|------------------------|
| id | INTEGER PK | Joined to note_embeddings.note_id |
| user_id | INTEGER | Filter: only search notes owned by current user |
| slug | TEXT | Returned in search results |
| title | TEXT | Used for text-based fallback matching |
| archived | INTEGER | Filter: separate active/archived search |

### note_tags (existing)

No schema changes. Tags are used in text-based fallback matching.

## Relationships

```
notes (1) ──── (0..1) note_embeddings
  │                        │
  │                        │ (synced)
  │                        ▼
  │               vec_note_embeddings (virtual)
  │
  ├── (1) ──── (N) note_tags
  └── (1) ──── (N) note_links
```

## Data Flow

1. **Note save** → extract title + body → hash content → if hash differs from stored → generate embedding → upsert note_embeddings + vec_note_embeddings
2. **Search query** → embed query text → KNN against vec_note_embeddings → join with notes → filter by user_id + archived → merge with text-fallback results for notes without embeddings → apply threshold + cap → return
3. **Note delete** → CASCADE deletes note_embeddings row → manually delete from vec_note_embeddings (virtual tables may not cascade)
4. **Startup backfill** → scan notes missing from note_embeddings → batch generate embeddings → insert
