# Data Model: Shared Note Authorship & Indicators

**Feature**: 025-shared-note-authorship  
**Date**: 2026-05-15

## No database schema changes

This feature introduces no new SQLite tables or columns. All new data is derived from:
1. **Git commit metadata** — author name stored in git commits via `CommitFileAs()`.
2. **Existing `note_shares` table** — already contains note_id, user_id, permission, granted_by.

---

## Modified Go structs

### `versioning.Version` (backend/internal/versioning/git.go)

**Before:**
```go
type Version struct {
    CommitHash string
    ShortHash  string
    Timestamp  time.Time
    Message    string
    Insertions int
    Deletions  int
}
```

**After (added field):**
```go
type Version struct {
    CommitHash string
    ShortHash  string
    Timestamp  time.Time
    Message    string
    Insertions int
    Deletions  int
    AuthorName string  // git commit author; "" for legacy commits
}
```

**Population**: Extracted from the `%an` token in the git log format string. Empty string for commits that predate this change (where the default "yant" identity was used — treated as legacy/unknown in the UI).

**UI fallback**: If `AuthorName == ""` or `AuthorName == "yant"`, the history template renders "—" in the Author column.

---

## New model function

### `models.ListShareCountsForOwner` (backend/internal/models/shares.go)

```go
// ListShareCountsForOwner returns a map of noteID -> active collaborator count
// for all non-archived notes owned by the given user. Notes with zero shares
// are not present in the map (use map lookup with zero-value fallback).
func ListShareCountsForOwner(db *DB, userID int64) (map[int64]int, error)
```

**Query:**
```sql
SELECT ns.note_id, COUNT(*) AS collab_count
FROM note_shares ns
JOIN notes n ON n.id = ns.note_id
WHERE n.user_id = ? AND n.archived = 0
GROUP BY ns.note_id
```

**Returns**: `map[int64]int` — note ID to number of active collaborators. Used by `NotesListGET` to attach share counts to the template context.

---

## Template data contracts

### `notes/list.html` — updated fields

| Field | Type | Description |
|---|---|---|
| `Notes` | `[]*models.Note` | Unchanged |
| `ShareStates` | `map[int64]int` | New: note ID → active collaborator count (0 = not shared out) |

### `notes/reader.html` — updated fields

| Field | Type | Description |
|---|---|---|
| (all existing fields) | — | Unchanged |
| `LastEditor` | `string` | New: username of most recent editor; "" if unknown/legacy |

### `notes/history.html` — updated fields

| Field | Type | Description |
|---|---|---|
| (all existing fields) | — | Unchanged |
| `Versions[].AuthorName` | `string` | New field on Version struct; "" for legacy entries |

### `shared/history.html` — new template

| Field | Type | Description |
|---|---|---|
| `Note` | `*models.Note` | The shared note |
| `OwnerUsername` | `string` | Note owner's username |
| `Versions` | `[]versioning.Version` | Version list with AuthorName populated |
| `Page`, `PrevPage`, `NextPage`, `HasMore`, `PerPage` | `int`/`bool` | Pagination |

### `shared/reader.html` — updated fields

| Field | Type | Description |
|---|---|---|
| (all existing fields) | — | Unchanged |
| `LastEditor` | `string` | New: username of most recent editor; "" if unknown/legacy |

### `shared/list.html` — no data change

No new fields. The existing `SharedNoteSummary.OwnerUsername` is used for the incoming badge label. CSS class styling is the only change.
