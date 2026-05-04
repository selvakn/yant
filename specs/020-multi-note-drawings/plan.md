# Implementation Plan: Multiple Drawings Per Note

**Branch**: `020-multi-note-drawings` | **Date**: 2026-05-04 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/020-multi-note-drawings/spec.md`

## Summary

Extend the drawing system from one-drawing-per-note to many-drawings-per-note. Each drawing gets a stable auto-generated ID, a user-provided display name, and a position marker (`![[draw:<id>]]`) in the markdown. Drawings render inline at marker positions in reader mode. Metadata lives in a new `note_drawings` SQLite table; files use `<slug>--<id>.<tool>.json` naming. Legacy single-drawing notes are lazily migrated on first interaction.

## Technical Context

**Language/Version**: Go 1.25  
**Primary Dependencies**: chi/v5, goldmark (+ custom extension), scs/v2, modernc.org/sqlite, htmx, EasyMDE  
**Storage**: SQLite (`note_drawings` table) + markdown files (markers) + JSON files (drawing content)  
**Testing**: `go test ./...` with ≥90% line coverage on `internal/...`  
**Target Platform**: Linux server (Docker)  
**Project Type**: Web application (monorepo: `backend/` + `frontend/`)  
**Performance Goals**: 10 drawings per note with no degradation; each drawing save/load <3s  
**Constraints**: No new external dependencies; goldmark extension is stdlib + goldmark API only  
**Scale/Scope**: Single-user to small-team; ~1-50 drawings across all notes

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [x] **I. Markdown-first storage** — Drawing content stored as JSON files on disk. SQLite `note_drawings` is a derived cache, rebuildable from files via `--rebuild-db`. Marker positions live in the markdown itself.
- [x] **II. Simplicity** — No new external dependencies. Goldmark extension uses goldmark's public API. File naming uses simple double-dash convention. 8-char random IDs from stdlib `crypto/rand`.
- [x] **III. Monorepo** — All changes within existing `backend/` and `frontend/` structure. New goldmark extension in `backend/internal/markdown/`.
- [x] **IV. Integration testing** — Each new endpoint and handler gets integration tests. Storage functions get unit tests. Existing drawing tests updated for multi-drawing. Coverage target ≥90%.
- [x] **V. Simple web UI** — UI stays lightweight: EasyMDE plain text markers, vanilla JS hydration for drawing placeholders (same pattern as mermaid diagrams). No new frameworks.
- [x] **VI. Commit & test discipline** — Each task is a small commit. Full test suite passes before each commit.

## Project Structure

### Documentation (this feature)

```text
specs/020-multi-note-drawings/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── drawing-api.md
├── checklists/
│   └── requirements.md
└── tasks.md
```

### Source Code (repository root)

```text
backend/
├── cmd/server/main.go              # new route registrations
├── internal/
│   ├── handlers/
│   │   ├── drawings.go             # multi-drawing handlers (new + refactored)
│   │   ├── drawings_test.go        # updated tests
│   │   ├── notes.go                # updated reader data
│   │   ├── history.go              # updated for multi-drawing version display
│   │   ├── public.go               # updated for multi-drawing public notes
│   │   └── shares.go               # updated for multi-drawing shared notes
│   ├── markdown/
│   │   ├── drawingext.go           # NEW: goldmark inline parser for ![[draw:id]]
│   │   └── drawingext_test.go      # NEW: unit tests
│   ├── models/
│   │   ├── models.go               # note_drawings schema + CRUD
│   │   ├── models_test.go          # updated tests
│   │   └── rebuild.go              # updated rebuild-db
│   ├── storage/
│   │   ├── drawings.go             # multi-drawing file ops
│   │   └── drawings_test.go        # updated tests
│   └── versioning/
│       └── git.go                  # no changes needed (already file-path-based)
│
frontend/
├── templates/
│   ├── notes/reader.html           # multi-drawing placeholder hydration
│   ├── notes/editor.html           # multi-drawing management UI
│   ├── public/note.html            # multi-drawing public rendering
│   └── shared/reader.html          # multi-drawing shared rendering
└── static/
    ├── css/app.css                 # styles for drawing list, placeholders
    └── js/app.js                   # drawing marker click handler
```

**Structure Decision**: All changes fit within the existing `backend/internal/` and `frontend/` layout. One new package `backend/internal/markdown/` for the goldmark extension keeps rendering concerns separate from handlers.
