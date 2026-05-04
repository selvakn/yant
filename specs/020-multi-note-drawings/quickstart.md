# Quickstart: Multiple Drawings Per Note

## Prerequisites

- Go 1.25+, Node.js 24+ (for frontend build), Make
- Existing YANT development environment (`make build && make run`)

## Key Files to Modify

### Backend (Go)

| File | Change |
|------|--------|
| `backend/internal/models/models.go` | Add `note_drawings` table to `InitSchema`, add `NoteDrawing` struct, CRUD functions |
| `backend/internal/models/rebuild.go` | Extend `RebuildDB` to scan drawing files and populate `note_drawings` |
| `backend/internal/storage/drawings.go` | Add multi-drawing file operations: `ListDrawings`, `ReadDrawingByID`, `WriteDrawingByID`, `DeleteDrawingByID`, `DetectLegacyDrawing`, `MigrateLegacyDrawing` |
| `backend/internal/handlers/drawings.go` | Add new multi-drawing HTTP handlers, keep legacy endpoints as compat shims |
| `backend/internal/handlers/notes.go` | Update `NoteReaderGET` to pass drawing list instead of single `HasDrawing` boolean |
| `backend/internal/handlers/handlers.go` | Extend goldmark setup with the drawing marker inline parser |
| `backend/cmd/server/main.go` | Register new routes under `/notes/{slug}/drawings/*` |

### Frontend (Templates + JS)

| File | Change |
|------|--------|
| `frontend/templates/notes/reader.html` | Replace single drawing div with JS hydration of `drawing-embed` placeholders |
| `frontend/templates/notes/editor.html` | Replace single drawing section with multi-drawing management UI |
| `frontend/templates/public/note.html` | Update to hydrate multiple drawing placeholders |
| `frontend/static/js/app.js` | Add drawing marker click handler for editor |

### Goldmark Extension (New)

| File | Purpose |
|------|---------|
| `backend/internal/markdown/drawingext.go` | Custom inline parser for `![[draw:<id>]]` markers |

## Build & Test

```bash
make test          # run all Go tests
make lint          # go vet
make build         # compile server
make run           # build and start on :8080
```

## Development Flow

1. Start with schema + models (no UI changes yet)
2. Add storage layer multi-drawing functions
3. Add goldmark extension for marker parsing
4. Update handlers (new endpoints + reader/editor data)
5. Update templates and JS for multi-drawing UI
6. Add legacy migration support
7. Update rebuild-db
8. Integration tests throughout
