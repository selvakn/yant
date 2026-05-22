# Implementation Plan: Resource Limits & Abuse Prevention

**Branch**: `026-resource-limits-defense` | **Date**: 2026-05-22 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `specs/026-resource-limits-defense/spec.md`

## Summary

Add four server-side limits to prevent storage abuse: (1) note content capped at 5 MB, (2) image uploads capped at 1 MB per file with a lifetime maximum of 10 images per note, (3) regular users capped at 25 notes (atomically enforced), and (4) admin dashboard extended to show per-user total storage. All checks are server-side; limits are hardcoded constants; no schema changes beyond adding `size_bytes` to the `notes` table.

## Technical Context

**Language/Version**: Go 1.25+  
**Primary Dependencies**: chi/v5 (routing), modernc.org/sqlite (database), scs/v2 (sessions)  
**Storage**: Markdown files (source of truth), SQLite `notes` table (derived cache including new `size_bytes` column)  
**Testing**: `go test ./backend/...`, integration tests via `testcontainers-go`  
**Target Platform**: Linux server (single-instance, self-hosted)  
**Project Type**: Web service (Go backend + htmx frontend)  
**Performance Goals**: Limit checks add < 1 ms to save/upload operations  
**Constraints**: No new external dependencies; SQLite serialized writes for atomicity  
**Scale/Scope**: Personal/small-team app; single SQLite instance

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [x] **I. Markdown-first storage** — `size_bytes` in SQLite is a derived cache; markdown files remain the source of truth and are unchanged.
- [x] **II. Simplicity** — Four constants, one new column, count queries on existing tables. No new abstractions or dependencies.
- [x] **III. Monorepo** — All changes within `backend/` and `frontend/templates/`; no new top-level directories.
- [x] **IV. Integration testing** — All new limit paths covered by integration tests; coverage maintained at ≥90%.
- [x] **V. Simple web UI** — Error messages via existing flash/redirect pattern; admin dashboard table extended with one new column.
- [x] **VI. Commit & test discipline** — Each step committed separately with tests green.

No violations. Complexity Tracking table not needed.

## Project Structure

### Documentation (this feature)

```text
specs/026-resource-limits-defense/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
└── tasks.md             # Phase 2 output
```

### Source Code (affected files)

```text
backend/
├── internal/
│   ├── handlers/
│   │   ├── images.go           # Reduce maxImageSize; add image count check
│   │   └── notes.go            # Add note size check; add note count check (atomic)
│   ├── models/
│   │   ├── models.go           # Add DB migration for size_bytes; add CountNotesForUser,
│   │   │                       # CountImagesForNote; update CreateNote/UpdateNote to populate size_bytes
│   │   └── admin.go            # Extend ListAllUsers to include TotalSizeBytes
│   └── storage/
│       └── (no changes)
frontend/
└── templates/
    └── admin/
        └── users.html          # Add "Storage" column to user table
```

---

## Implementation Steps

### Step 1 — DB migration: add `size_bytes` to `notes`

**File**: `backend/internal/models/models.go`

Add a new migration step to the `runMigrations` function (or equivalent):

```sql
ALTER TABLE notes ADD COLUMN size_bytes INTEGER NOT NULL DEFAULT 0;
```

The `DEFAULT 0` handles existing rows gracefully. No backfill needed for the limit check (only new saves populate it); admin totals will show 0 for notes not yet resaved, which is acceptable.

**Commit message**: `feat(limits): add size_bytes column to notes table`

---

### Step 2 — Model helpers: note count and image count

**File**: `backend/internal/models/models.go`

Add two read functions:

```go
// CountNotesForUser returns the number of notes owned by userID.
func CountNotesForUser(db *DB, userID int64) (int, error) {
    var n int
    err := db.QueryRow(`SELECT COUNT(*) FROM notes WHERE user_id = ?`, userID).Scan(&n)
    return n, err
}

// CountImagesForNote returns the lifetime number of images uploaded to noteID.
func CountImagesForNote(db *DB, noteID int64) (int, error) {
    var n int
    err := db.QueryRow(`SELECT COUNT(*) FROM images WHERE note_id = ?`, noteID).Scan(&n)
    return n, err
}
```

**Commit message**: `feat(limits): add CountNotesForUser and CountImagesForNote helpers`

---

### Step 3 — Update `CreateNote` to populate `size_bytes` atomically

**File**: `backend/internal/models/models.go`

Modify `CreateNote` to:
1. Accept `sizeBytes int64` parameter (or compute from body length passed in).
2. Wrap the count check + insert in a `BEGIN IMMEDIATE` transaction so concurrent requests cannot both see count < 25 and both succeed.

```go
// CreateNote creates a note for userID, enforcing maxNotes for non-admin users.
// Returns ErrNoteLimitReached if the user is at the limit.
var ErrNoteLimitReached = errors.New("note limit reached")

func CreateNote(db *DB, userID int64, slug, title string, sizeBytes int64, isAdmin bool) (*Note, error) {
    tx, err := db.Begin() // SQLite: BEGIN IMMEDIATE via pragma or explicit
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()

    if !isAdmin {
        var count int
        if err := tx.QueryRow(`SELECT COUNT(*) FROM notes WHERE user_id = ?`, userID).Scan(&count); err != nil {
            return nil, err
        }
        if count >= MaxNotesPerUser {
            return nil, ErrNoteLimitReached
        }
    }

    now := time.Now().UTC()
    res, err := tx.Exec(
        `INSERT INTO notes (user_id, slug, title, size_bytes, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
        userID, slug, title, sizeBytes, now, now,
    )
    if err != nil {
        return nil, err
    }
    id, _ := res.LastInsertId()
    if err := tx.Commit(); err != nil {
        return nil, err
    }
    return &Note{ID: id, UserID: userID, Slug: slug, Title: title}, nil
}
```

> Note: Check current `CreateNote` signature and adjust the parameter list to match — the goal is to add `sizeBytes` and `isAdmin` without breaking callers. If the existing signature differs substantially, adapt accordingly while preserving all existing fields.

**Commit message**: `feat(limits): enforce max-notes limit atomically in CreateNote`

---

### Step 4 — Update `UpdateNote` to populate `size_bytes`

**File**: `backend/internal/models/models.go`

Add `sizeBytes int64` to `UpdateNote` and include it in the UPDATE statement:

```sql
UPDATE notes SET title = ?, size_bytes = ?, updated_at = ? WHERE id = ? AND user_id = ?
```

**Commit message**: `feat(limits): track size_bytes on note update`

---

### Step 5 — Note content size check in handlers

**File**: `backend/internal/handlers/notes.go`

Add a constant at package or file level:

```go
const maxNoteSizeBytes = 5 * 1024 * 1024 // 5 MB
```

In `NotesCreatePOST`, before calling `storage.WriteNote`:

```go
body := r.FormValue("body")
if len([]byte(body)) > maxNoteSizeBytes {
    // flash error and redirect back
    session.Put(r.Context(), "flash_error", "Note is too large. Maximum size is 5 MB.")
    http.Redirect(w, r, "/notes/new", http.StatusSeeOther)
    return
}
```

Same check in `noteUpdate` before the write, redirecting back to `/notes/{slug}/edit`.

Pass `int64(len([]byte(body)))` as `sizeBytes` to `models.CreateNote` / `models.UpdateNote`.

**Commit message**: `feat(limits): reject note saves exceeding 5 MB`

---

### Step 6 — Note count limit error handling in handler

**File**: `backend/internal/handlers/notes.go`

In `NotesCreatePOST`, after calling `models.CreateNote`, check for `ErrNoteLimitReached`:

```go
note, err := models.CreateNote(h.db, userID, slug, title, sizeBytes, user.IsAdmin)
if errors.Is(err, models.ErrNoteLimitReached) {
    session.Put(r.Context(), "flash_error",
        fmt.Sprintf("You have reached the maximum of %d notes. Delete a note to create a new one.", models.MaxNotesPerUser))
    http.Redirect(w, r, "/notes", http.StatusSeeOther)
    return
}
```

**Commit message**: `feat(limits): surface note count limit error to user`

---

### Step 7 — Image upload: reduce size limit and add count check

**File**: `backend/internal/handlers/images.go`

Change the existing constant:

```go
// Before:
const maxImageSize = 10 << 20  // 10 MB

// After:
const maxImageSize = 1 << 20   // 1 MB
```

Add constants:

```go
const maxImagesPerNote = 10
```

Before calling `models.CreateImage`, look up the note ID and check the lifetime count:

```go
imgCount, err := models.CountImagesForNote(h.db, note.ID)
if err != nil {
    http.Error(w, "Internal error", http.StatusInternalServerError)
    return
}
if imgCount >= maxImagesPerNote {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusUnprocessableEntity)
    json.NewEncoder(w).Encode(map[string]string{
        "error": fmt.Sprintf("This note has reached the maximum of %d images.", maxImagesPerNote),
    })
    return
}
```

**Commit message**: `feat(limits): reduce image size limit to 1 MB and cap images per note at 10`

---

### Step 8 — Admin dashboard: per-user total storage

**File**: `backend/internal/models/admin.go`

Add `TotalSizeBytes int64` to the user list struct (whatever `ListAllUsers` returns per user). Extend the query to include:

```sql
COALESCE(SUM(n.size_bytes), 0) AS total_size_bytes
```

as part of the `LEFT JOIN notes n ON n.user_id = u.id` aggregation already present in the query.

**File**: `frontend/templates/admin/users.html`

Add a "Storage" column header and per-row cell displaying the total in a human-readable format (e.g., `1.2 MB`, `340 KB`). Add a helper template function or format in Go handler if needed, or use a simple JS/Go formatting approach consistent with the rest of the admin templates.

**Commit message**: `feat(limits): show per-user total note storage in admin dashboard`

---

### Step 9 — Integration tests

**File**: `backend/internal/handlers/notes_test.go` (or equivalent integration test file)

Cover:
- `POST /notes` with body > 5 MB → HTTP 303 redirect with flash error, note not created
- `POST /notes` at exactly 25 notes for a regular user → rejected; admin with 25+ notes → succeeds
- `POST /notes` concurrent creates at limit (two simultaneous requests) → exactly one succeeds, one fails
- `POST /notes/{slug}` update with body > 5 MB → rejected

**File**: `backend/internal/handlers/images_test.go` (or equivalent)

Cover:
- Image upload > 1 MB → HTTP 413
- Image upload when note already has 10 images → HTTP 422 with JSON error
- Image upload within limits → succeeds

**File**: `backend/internal/handlers/admin_test.go` (or equivalent)

Cover:
- `GET /admin/users` response includes `total_size_bytes` (or formatted equivalent) per user

**Commit message**: `test(limits): integration tests for all resource limit enforcement`

---

## Constants Reference

| Constant | Value | Location |
|----------|-------|----------|
| `maxNoteSizeBytes` | 5 × 1024 × 1024 (5 MB) | `handlers/notes.go` |
| `maxImageSize` | 1 × 1024 × 1024 (1 MB) | `handlers/images.go` |
| `maxImagesPerNote` | 10 | `handlers/images.go` |
| `MaxNotesPerUser` | 25 | `models/models.go` (exported for use in handler error message) |

---

## Rollout Notes

- The `size_bytes DEFAULT 0` migration is safe to apply on an existing database; no notes break.
- Existing notes with > 10 images or > 5 MB content remain readable; only new saves/uploads are blocked.
- Existing users with > 25 notes can continue using their notes; only new note creation is blocked once they exceed the limit (they were created before the limit existed and the limit only fires on `CreateNote`).
