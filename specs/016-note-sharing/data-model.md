# Data Model: Note Sharing

## Entities

### Note Share (new)

Represents one user (the collaborator) having been granted permission on another user's note.

| Field       | Type     | Description                                                            |
| ----------- | -------- | ---------------------------------------------------------------------- |
| note_id     | INTEGER  | FK → notes(id), CASCADE on delete                                      |
| user_id     | INTEGER  | FK → users(id), CASCADE on delete — the collaborator                   |
| permission  | TEXT     | `'read'` or `'edit'`                                                   |
| granted_at  | TEXT     | ISO 8601 timestamp                                                     |
| granted_by  | INTEGER  | FK → users(id) — the owner who granted the share                       |

**Primary Key**: `(note_id, user_id)` — one row per (note, collaborator) pair (re-sharing upserts the row).  
**Indexes**: `idx_note_shares_user ON (user_id)` for fast recipient-side listing.

### Schema DDL

```sql
CREATE TABLE IF NOT EXISTS note_shares (
    note_id     INTEGER NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission  TEXT    NOT NULL CHECK (permission IN ('read','edit')),
    granted_at  TEXT    NOT NULL,
    granted_by  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (note_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_note_shares_user ON note_shares (user_id);
```

## Relationships

```
notes 1──* note_shares         (a note can be shared with 0..N collaborators)
users 1──* note_shares (user)  (a user can be a collaborator on 0..N notes)
users 1──* note_shares (grantor)
```

## State Transitions

```
(no row) ──grant──→ Shared (row inserted)
Shared ──update-permission──→ Shared (row updated)
Shared ──revoke──→ (no row; row deleted)
(note deleted) ──CASCADE──→ (all shares for that note removed)
(user deleted) ──CASCADE──→ (all that user's shares removed)
```

## Go Structs

```go
// NoteShare represents a single share grant on a note.
type NoteShare struct {
    NoteID     int64
    UserID     int64    // the collaborator
    Permission string   // "read" or "edit"
    GrantedAt  time.Time
    GrantedBy  int64
}

// NoteCollaborator used when listing collaborators on a note (for the share dialog).
type NoteCollaborator struct {
    Username   string
    Permission string
    GrantedAt  time.Time
}

// SharedNoteSummary used when listing notes shared with a user.
type SharedNoteSummary struct {
    Slug          string
    Title         string
    OwnerUsername string
    Permission    string
    UpdatedAt     time.Time
    Tags          []string
}
```

## Role Resolution

```
ResolveAccess(db, viewerID, noteID) -> (role, ok)
  if notes.user_id == viewerID:          → ("owner",  true)
  else if row in note_shares matching:
       permission == "edit":              → ("editor", true)
       permission == "read":              → ("reader", true)
  else:                                   → ("", false)
```

Handler permission checks:
- **Read**: any role → allow
- **Edit (body/title/tags/todos/drawing/images)**: owner or editor → allow
- **Archive / Restore / Delete / Share config**: owner only

## Key Queries

### Grant or update a share

```sql
INSERT INTO note_shares (note_id, user_id, permission, granted_at, granted_by)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(note_id, user_id) DO UPDATE SET
    permission = excluded.permission,
    granted_at = excluded.granted_at,
    granted_by = excluded.granted_by;
```

### Revoke

```sql
DELETE FROM note_shares WHERE note_id = ? AND user_id = ?;
```

### List collaborators on a note (for the share dialog)

```sql
SELECT u.username, s.permission, s.granted_at
FROM note_shares s
JOIN users u ON u.id = s.user_id
WHERE s.note_id = ?
ORDER BY u.username;
```

### List notes shared with a viewer

```sql
SELECT n.slug, n.title, u.username AS owner_username,
       s.permission, n.updated_at
FROM note_shares s
JOIN notes n ON n.id = s.note_id
JOIN users u ON u.id = n.user_id
WHERE s.user_id = ? AND n.archived = 0
ORDER BY n.updated_at DESC;
```

### Count

```sql
SELECT COUNT(*)
FROM note_shares s
JOIN notes n ON n.id = s.note_id
WHERE s.user_id = ? AND n.archived = 0;
```

### Resolve viewer access for a given note

```sql
SELECT n.id, n.user_id, n.slug, n.title, n.archived, n.created_at, n.updated_at,
       CASE
         WHEN n.user_id = ? THEN 'owner'
         WHEN s.permission = 'edit' THEN 'editor'
         WHEN s.permission = 'read' THEN 'reader'
         ELSE NULL
       END AS role
FROM notes n
LEFT JOIN note_shares s ON s.note_id = n.id AND s.user_id = ?
JOIN users u ON u.id = n.user_id
WHERE u.username = ? AND n.slug = ?
  AND (n.user_id = ? OR s.user_id IS NOT NULL);
```

(Parameters: viewerID, viewerID, ownerUsername, slug, viewerID.)

## Security Considerations

- **Owner-only enforcement at API layer**: archive/delete/share handlers MUST check `note.user_id == viewerID` even when the note was fetched via `GetNoteForViewer`.
- **Cascading deletes**: removing a note or either user auto-cleans share rows.
- **No self-share**: at the grant handler, reject if `target_user == owner`.
- **Wiki-link leakage**: shared-note rendering uses `ResolveWikiLinksForViewer`, which only links to targets also accessible to the viewer.
