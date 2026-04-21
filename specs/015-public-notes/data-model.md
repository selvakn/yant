# Data Model: Public Note Sharing

## Entities

### Public Note Share

A row per note that has ever been published. The row persists across toggle cycles so that re-publishing reuses the same token.

| Field          | Type        | Description                                                                 |
| -------------- | ----------- | --------------------------------------------------------------------------- |
| note_id        | INTEGER     | FK → notes(id), CASCADE on delete, UNIQUE (one public row per note)         |
| token          | TEXT        | URL-safe random identifier (~22 chars, 128 bits entropy), UNIQUE            |
| published      | BOOLEAN     | Current public state. `true` = publicly accessible; `false` = private        |
| published_at   | TEXT        | Timestamp of first publish (ISO 8601)                                       |
| updated_at     | TEXT        | Timestamp of most recent publish/unpublish toggle                           |

**Primary Key**: `note_id`  
**Unique Index**: `token` (for efficient token lookup)  
**Foreign Key**: `note_id` REFERENCES `notes(id)` ON DELETE CASCADE

### Schema DDL

```sql
CREATE TABLE IF NOT EXISTS public_notes (
    note_id      INTEGER PRIMARY KEY REFERENCES notes(id) ON DELETE CASCADE,
    token        TEXT    NOT NULL UNIQUE,
    published    BOOLEAN NOT NULL DEFAULT 1,
    published_at TEXT    NOT NULL,
    updated_at   TEXT    NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_public_notes_token ON public_notes (token);
```

## Relationships

```
notes 1──? public_notes   (a note has zero or one public share row)
notes 1──* images         (existing; images are scoped to a note)
```

When a visitor loads `/p/{token}`:
1. `public_notes` row is fetched by token
2. `notes` row is fetched by `note_id` from the public row
3. Must satisfy: `public_notes.published = 1 AND notes.archived = 0` — otherwise 404

## State Transitions

```
[no row] ──publish──→ Published (row created, token generated)
Published ──unpublish──→ Revoked (published=false, token preserved)
Published ──archive──→ Revoked (archive flow also sets published=false)
Published ──delete──→ [no row] (CASCADE removes public_notes row)
Revoked ──publish──→ Published (same token reused)
```

## Go Structs

```go
// PublicNote represents the publishing state of a note.
type PublicNote struct {
    NoteID      int64
    Token       string
    Published   bool
    PublishedAt time.Time
    UpdatedAt   time.Time
}
```

## Key Queries

### Generate or retrieve token for a note, mark published

```sql
-- If row doesn't exist, insert with generated token
-- If row exists, flip published to true
INSERT INTO public_notes (note_id, token, published, published_at, updated_at)
VALUES (?, ?, 1, ?, ?)
ON CONFLICT(note_id) DO UPDATE SET
    published = 1,
    updated_at = excluded.updated_at;
```

### Unpublish (preserve token)

```sql
UPDATE public_notes
SET published = 0, updated_at = ?
WHERE note_id = ?;
```

### Public reader lookup (by token)

```sql
SELECT n.id, n.user_id, n.slug, n.title, n.archived, n.created_at, n.updated_at
FROM public_notes p
JOIN notes n ON n.id = p.note_id
WHERE p.token = ?
  AND p.published = 1
  AND n.archived = 0;
```

### Is note public? (for owner UI)

```sql
SELECT token, published
FROM public_notes
WHERE note_id = ?;
```

### Owner's public notes list

```sql
SELECT n.slug, n.title, p.token, p.published_at
FROM public_notes p
JOIN notes n ON n.id = p.note_id
WHERE n.user_id = ? AND p.published = 1 AND n.archived = 0
ORDER BY p.published_at DESC;
```

## Security Considerations

- **Token generation**: 16 bytes from `crypto/rand`, base64url-encoded. Must never use `math/rand`.
- **Token uniqueness**: UNIQUE constraint at the DB level. On rare collision (extremely unlikely at 128 bits), INSERT fails and the caller retries with a fresh token.
- **No enumeration**: Invalid/unknown tokens return a plain 404 with no hint about whether the token is malformed, expired, or never existed.
- **Revocation is atomic**: A single `UPDATE` on `published` flips the flag. Next request returns 404. No caching.
