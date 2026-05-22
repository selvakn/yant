# Research: Resource Limits & Abuse Prevention

## Decision: Image upload size limit enforcement point

- **Decision**: Reduce `maxImageSize` constant in `handlers/images.go:19` from 10 MB to 1 MB (1 << 20). Enforcement already uses `http.MaxBytesReader` + `ParseMultipartForm` with a 413 response.
- **Rationale**: The mechanism is already in place; only the constant changes.
- **Alternatives considered**: Middleware-level byte counting — unnecessary since per-handler enforcement already exists.

## Decision: Lifetime image count per note

- **Decision**: Query `SELECT COUNT(*) FROM images WHERE note_id = ?` before every upload. The `images` table persists all uploaded images with `note_id`; records are never deleted individually (only when a note is deleted), so this naturally gives the lifetime count.
- **Rationale**: The images table already tracks every upload with `note_id`. No new table or column needed.
- **Alternatives considered**: Separate `note_image_count` column — unnecessary extra state to keep in sync.

## Decision: Note content size check

- **Decision**: Check `len([]byte(body))` against `5 * 1024 * 1024` bytes in both `NotesCreatePOST` and `noteUpdate` (handlers/notes.go) before the `storage.WriteNote` call. Return HTTP 413 with a flash error.
- **Rationale**: The body is already in memory as a string at the point of handling; a `len()` check is zero-cost.
- **Alternatives considered**: Streaming byte-count middleware — overkill for a simple text body limit.

## Decision: Atomic note count limit enforcement

- **Decision**: Wrap the count-check + insert in a single SQLite `BEGIN IMMEDIATE` transaction in `models.CreateNote`. SQLite's write lock is held for the duration, preventing concurrent creates from both seeing count < 25 and both succeeding.
- **Rationale**: SQLite serializes writes; `BEGIN IMMEDIATE` acquires the write lock upfront, so no two goroutines can race through the count check simultaneously.
- **Alternatives considered**: Application-level mutex — unnecessary given SQLite's own serialization; Redis-based distributed lock — far too complex for a single-instance personal app.

## Decision: Admin per-user total note size

- **Decision**: Add a `size_bytes INTEGER NOT NULL DEFAULT 0` column to the `notes` table. Populate it on create and update (`len([]byte(body))`). The `ListAllUsers` admin query sums it with `SUM(n.size_bytes)`.
- **Rationale**: Avoids filesystem stats at query time (which would require reading every note file). The value is a derived cache (markdown files remain source of truth per constitution).
- **Alternatives considered**: On-demand filesystem stat — too slow for a user list with many users; separate summary table — over-engineered.

## Decision: Error response format

- **Decision**: Return HTTP 413 (Content Too Large) for size-related rejections and HTTP 422 (Unprocessable Entity) for count-related rejections. For HTML form submissions, flash an error message and redirect back. For non-form requests (image upload which is multipart), return JSON `{"error": "..."}` with the appropriate status.
- **Rationale**: Consistent with existing error patterns in the codebase.
- **Alternatives considered**: All 400 Bad Request — less semantically precise.
