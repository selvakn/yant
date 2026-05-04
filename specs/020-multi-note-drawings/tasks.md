# Tasks: Multiple Drawings Per Note

## Task 1: Schema + Models — note_drawings table and CRUD

**Files**: `backend/internal/models/models.go`, `backend/internal/models/models_test.go`

Add `note_drawings` table to `InitSchema`. Add `NoteDrawing` struct. Implement:
- `CreateDrawing(db, noteID, displayName, toolType) (NoteDrawing, error)` — generates 8-char ID, inserts row
- `GetDrawing(db, noteID, drawingID) (*NoteDrawing, error)`
- `ListDrawings(db, noteID) ([]NoteDrawing, error)`
- `RenameDrawing(db, noteID, drawingID, newName) error`
- `DeleteDrawing(db, noteID, drawingID) error`
- `GenerateDrawingID() string` — 8-char alphanumeric from crypto/rand

Tests: unit tests for all CRUD operations, ID generation uniqueness, validation (empty name, max length).

## Task 2: Storage — multi-drawing file operations

**Files**: `backend/internal/storage/drawings.go`, `backend/internal/storage/drawings_test.go`

Refactor storage layer to support multi-drawing file naming:
- `ListDrawingFiles(root, userID, slug) []DrawingFile` — returns all drawing files (new + legacy format)
- `ReadDrawingByID(root, userID, slug, drawingID) ([]byte, DrawingType, error)`
- `WriteDrawingByID(root, userID, slug, drawingID, dt, data) error`
- `DeleteDrawingByID(root, userID, slug, drawingID, dt) error`
- `DetectLegacyDrawing(root, userID, slug) (DrawingType, bool)` — checks for old-format files
- `MigrateLegacyDrawing(root, userID, slug, newID) (DrawingType, error)` — renames file to new format
- `DrawingFilePathByID(root, userID, slug, drawingID, dt) string`
- `DrawingRelPathByID(userID, slug, drawingID, dt) string`

Keep existing single-drawing functions for backward compat (they become thin wrappers).

Tests: file creation/read/delete with new naming, legacy detection, migration rename.

## Task 3: Goldmark extension — drawing marker parser

**Files**: `backend/internal/markdown/drawingext.go`, `backend/internal/markdown/drawingext_test.go`

Create a goldmark inline parser extension:
- Matches `![[draw:<id>]]` pattern in markdown
- Renders as `<div class="drawing-embed" data-drawing-id="<id>"></div>`
- Register in `handlers.go` goldmark setup

Tests: parse single marker, multiple markers, markers mixed with text, malformed markers ignored.

## Task 4: Handlers — new multi-drawing API endpoints

**Files**: `backend/internal/handlers/drawings.go`, `backend/internal/handlers/drawings_test.go`, `backend/cmd/server/main.go`

Add new HTTP handlers per the API contract:
- `DrawingsListGET` — `GET /notes/{slug}/drawings`
- `DrawingsCreatePOST` — `POST /notes/{slug}/drawings`
- `DrawingByIDGET` — `GET /notes/{slug}/drawings/{drawingID}`
- `DrawingByIDPUT` — `PUT /notes/{slug}/drawings/{drawingID}`
- `DrawingByIDRenamePATCH` — `PATCH /notes/{slug}/drawings/{drawingID}`
- `DrawingByIDDELETE` — `DELETE /notes/{slug}/drawings/{drawingID}`

Register routes in `main.go`. Keep existing single-drawing endpoints as legacy compat (delegate to first drawing).

Tests: handler tests for all new endpoints, validation errors, 404 cases.

## Task 5: Update reader — multi-drawing placeholder hydration

**Files**: `frontend/templates/notes/reader.html`, `backend/internal/handlers/notes.go`, `frontend/static/css/app.css`

Update `NoteReaderGET` to:
- Pass `Drawings []NoteDrawing` (with metadata) to template instead of single `HasDrawing`/`DrawingType`
- The goldmark extension already inserts `<div class="drawing-embed" data-drawing-id="...">` in the HTML

Update `reader.html`:
- Remove single-drawing div and script
- Add JS that queries all `.drawing-embed` elements, fetches drawing metadata from the drawings list data, loads appropriate bundles, and hydrates each placeholder

Update CSS for `.drawing-embed` placeholder styling and headings.

Tests: handler test verifying template data includes drawings list.

## Task 6: Update editor — multi-drawing management UI

**Files**: `frontend/templates/notes/editor.html`, `frontend/static/css/app.css`

Replace single drawing section with:
- "Add drawing" button that triggers: tool selection → name prompt → POST to create → insert marker at cursor
- Drawing list showing all drawings with edit/rename/delete actions
- Click on marker line opens drawing canvas for that drawing
- Remove drawing button deletes via API

Keep the existing EasyMDE + auto-save flow unchanged. Drawing management is additive.

Tests: manual verification + handler tests for the data passed to template.

## Task 7: Legacy migration support

**Files**: `backend/internal/storage/drawings.go`, `backend/internal/handlers/drawings.go`, `backend/internal/models/models.go`

Implement lazy migration:
- When a legacy drawing is detected on first edit via new API, call `MigrateLegacyDrawing` to rename file
- Insert `note_drawings` row with display name "Drawing 1"
- Append `![[draw:<id>]]` marker to markdown body
- Version-control the rename

Tests: integration test — create note with legacy drawing format, access via new API, verify migration occurs.

## Task 8: Update shared/public/history templates

**Files**: `frontend/templates/public/note.html`, `frontend/templates/shared/reader.html`, `backend/internal/handlers/public.go`, `backend/internal/handlers/shares.go`, `backend/internal/handlers/history.go`

Update all non-editor views to hydrate multiple drawing placeholders:
- Public note: load drawing data for all markers
- Shared reader: load drawing data respecting permissions
- History version view: load drawing data for that version's state

Tests: handler tests for public/shared/history endpoints with multi-drawing notes.

## Task 9: Rebuild-DB support for note_drawings

**Files**: `backend/internal/models/rebuild.go`, `backend/internal/models/rebuild_test.go`

Extend `RebuildDB` to:
- Scan `<userID>/<slug>--<id>.<tool>.json` files → insert into `note_drawings`
- Scan legacy `<userID>/<slug>.<tool>.json` files → generate ID, insert with display name = drawing ID
- Handle both formats coexisting

Tests: rebuild test with mixed legacy and new-format drawing files.

## Task 10: Integration tests

**Files**: `backend/internal/integration/integration_test.go`

Add integration tests covering:
- Create note → add 2 drawings → verify list returns both
- Save drawing content → reload → verify content intact
- Rename drawing → verify metadata update, file unchanged
- Delete one drawing → verify other unaffected
- Legacy single-drawing → access via new API → verify lazy migration
- Reader mode → verify goldmark renders markers as placeholders
- Public note with multiple drawings → verify all accessible

## Notes

- Constitution Principle VI: each task is a commit boundary. Run `make test && make lint` before each commit.
- Tasks 1-3 are foundational (no UI changes). Tasks 4-6 build the user-facing feature. Tasks 7-9 handle edge cases and compat. Task 10 validates end-to-end.
