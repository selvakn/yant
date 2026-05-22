# Data Model: Resource Limits & Abuse Prevention

## Schema Changes

### `notes` table — add `size_bytes`

```sql
ALTER TABLE notes ADD COLUMN size_bytes INTEGER NOT NULL DEFAULT 0;
```

**Purpose**: Derived cache of markdown content size in bytes. Populated on create and update. Allows efficient per-user storage aggregation in admin queries without reading files.

**Source of truth**: The markdown file on disk. `size_bytes` is a rebuildable cache.

**Populated by**: `models.CreateNote` and `models.UpdateNote` — `len([]byte(body))` computed before write.

---

## No other schema changes

| Existing table | Why it already suffices |
|----------------|------------------------|
| `images` | Tracks every uploaded image with `note_id`. `COUNT(*) WHERE note_id = ?` gives lifetime upload count. No deletion of individual image rows (only cascade-deleted with the note). |
| `users` | `is_admin INTEGER` already exists. Admin exemption reads this column directly. |
| `notes` | `user_id` + new `size_bytes`. Count query for note limit: `SELECT COUNT(*) FROM notes WHERE user_id = ?`. |

---

## Constants (hardcoded, not stored in DB)

```go
const (
    MaxNoteSizeBytes   = 5 * 1024 * 1024  // 5 MB — markdown text content
    MaxImageSizeBytes  = 1 * 1024 * 1024  // 1 MB — per uploaded image
    MaxImagesPerNote   = 10               // lifetime upload count per note
    MaxNotesPerUser    = 25               // regular users only; admins exempt
)
```

These live in `backend/internal/handlers/` (image + note handlers) or a shared `limits` package if reuse warrants it.

---

## Queries

### Count notes for user (for limit check)
```sql
SELECT COUNT(*) FROM notes WHERE user_id = ?
```
Used in `models.CreateNoteIfBelowLimit` within a `BEGIN IMMEDIATE` transaction.

### Count lifetime images for note (for limit check)
```sql
SELECT COUNT(*) FROM images WHERE note_id = ?
```
Used in `ImageUploadPOST` before `models.CreateImage`.

### Total note storage per user (admin dashboard)
```sql
SELECT u.id, u.username, u.is_admin, u.disabled,
       COUNT(n.id) AS note_count,
       COALESCE(SUM(n.size_bytes), 0) AS total_size_bytes,
       MAX(n.updated_at) AS last_active
FROM users u
LEFT JOIN notes n ON n.user_id = u.id
GROUP BY u.id
```
Used in `models.ListAllUsers`.
