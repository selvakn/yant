# Data Model: Markdown Note Taking App

**Date**: 2026-04-05
**Branch**: `001-markdown-note-taking`

## SQLite Tables

### users

| Column     | Type    | Constraints                |
|------------|---------|----------------------------|
| id         | INTEGER | PRIMARY KEY AUTOINCREMENT  |
| username   | TEXT    | UNIQUE NOT NULL            |
| created_at | TEXT    | NOT NULL (ISO 8601)        |

**Notes**: Mock auth — no password stored. Account auto-created on
first login with unrecognized username. `id` is used as the per-user
directory name for filesystem storage.

### notes

| Column     | Type    | Constraints                                    |
|------------|---------|------------------------------------------------|
| id         | INTEGER | PRIMARY KEY AUTOINCREMENT                      |
| user_id    | INTEGER | NOT NULL REFERENCES users(id)                  |
| slug       | TEXT    | NOT NULL                                       |
| title      | TEXT    | NOT NULL DEFAULT 'Untitled Note'               |
| created_at | TEXT    | NOT NULL (ISO 8601)                            |
| updated_at | TEXT    | NOT NULL (ISO 8601)                            |

**Unique constraint**: `UNIQUE(user_id, slug)` — slugs are unique per
user, not globally.

**Notes**: Markdown body is NOT stored in SQLite. It lives on the
filesystem at `notes/{user_id}/{slug}.md`. SQLite stores metadata only
and is a derived, rebuildable cache (Constitution Principle I).

### note_tags

| Column   | Type    | Constraints                        |
|----------|---------|------------------------------------|
| note_id  | INTEGER | NOT NULL REFERENCES notes(id)      |
| tag_name | TEXT    | NOT NULL (stored lowercase)        |

**Primary key**: `(note_id, tag_name)`

**Notes**: Tags parsed from note body on each save. Always stored
lowercase (case-insensitive, FR-008). Distinct tag names per user
derived by JOIN with notes table.

### images

| Column    | Type    | Constraints                        |
|-----------|---------|------------------------------------|
| id        | INTEGER | PRIMARY KEY AUTOINCREMENT          |
| note_id   | INTEGER | NOT NULL REFERENCES notes(id)      |
| filename  | TEXT    | NOT NULL (UUID-based, e.g. uuid.png) |
| original  | TEXT    | NOT NULL (original upload filename) |
| mime_type | TEXT    | NOT NULL                           |
| size      | INTEGER | NOT NULL (bytes)                   |

**Notes**: Image binary stored at `uploads/{user_id}/{filename}`.
On note deletion, orphan images (those belonging to the deleted note)
are cleaned up from both the table and filesystem.

## Indexes

```sql
CREATE INDEX idx_note_user ON notes(user_id);
CREATE INDEX idx_tag_name_note ON note_tags(tag_name, note_id);
CREATE INDEX idx_image_note ON images(note_id);
```

## Relationships

```text
users 1──* notes
notes 1──* note_tags
notes 1──* images
```

## Go Struct Representations

```go
type User struct {
    ID        int64
    Username  string
    CreatedAt time.Time
}

type Note struct {
    ID        int64
    UserID    int64
    Slug      string
    Title     string
    CreatedAt time.Time
    UpdatedAt time.Time
    Tags      []string // populated by JOIN query, not a DB column
}

type Image struct {
    ID       int64
    NoteID   int64
    Filename string
    Original string
    MimeType string
    Size     int64
}
```

## State Transitions

Notes have no explicit status. Lifecycle is implicit:

1. **Created** → INSERT into `notes` + write `{slug}.md` file +
   INSERT into `note_tags` (if body contains tags)
2. **Updated** → UPDATE `notes.updated_at` + overwrite `.md` file +
   DELETE + re-INSERT `note_tags` for this note
3. **Deleted** → DELETE from `notes` (cascades to `note_tags`) +
   DELETE from `images` + remove `.md` file + remove image files

## Rebuild Strategy

Per Constitution Principle I, SQLite MUST be rebuildable from
Markdown source files:

1. Scan `notes/` for all `{user_id}/{slug}.md` files
2. Parse each file: first `# heading` → title (fallback: filename),
   `#word` patterns → tags (lowercased), file mtime → timestamps
3. Scan `uploads/` for image files, match to note references in body
4. TRUNCATE all tables, re-INSERT from parsed data

Rebuild triggered by: `./server --rebuild-db` flag in main.go.
