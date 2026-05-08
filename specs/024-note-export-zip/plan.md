# Implementation Plan: Note Export as ZIP

**Branch**: `024-note-export-zip` | **Date**: 2026-05-08 | **Spec**: [spec.md](spec.md)

## Summary

Users can export a note as a ZIP file containing its markdown source, an SVG render of each sketch, and each sketch's raw JSON source file. The ZIP downloads in the browser without page navigation. Implementation adds a single Go handler (`GET /notes/{slug}/export`), an Export button in the reader topbar, and no new database tables or dependencies.

## Technical Context

**Language/Version**: Go 1.25+ (backend), vanilla JS + htmx (frontend)  
**Primary Dependencies**: `archive/zip` (standard library), `chi/v5` (routing — existing)  
**Storage**: Markdown files + existing drawing JSON/SVG files (no new tables)  
**Testing**: `go test` (existing suite); integration test for the endpoint  
**Target Platform**: Linux server (same as all other handlers)  
**Project Type**: Web application (monorepo: backend/ + frontend/)  
**Performance Goals**: Under 3 seconds for notes with up to 10 sketches  
**Constraints**: No new external dependencies; ZIP built in memory (notes are small)  
**Scale/Scope**: Per-note operation, single user session

## Constitution Check

- [x] **I. Markdown-first storage** — Reads existing markdown/drawing files; writes nothing new to disk
- [x] **II. Simplicity** — One handler, one route, one button; uses only stdlib `archive/zip`
- [x] **III. Monorepo** — Handler in `backend/internal/handlers/`, template change in `frontend/templates/notes/`
- [x] **IV. Integration testing** — Integration test added for the export endpoint
- [x] **V. Simple web UI** — Button added inline in existing topbar; no new JS framework
- [x] **VI. Commit & test discipline** — Full test suite green before commit

## Project Structure

### Documentation (this feature)

```text
specs/024-note-export-zip/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   └── export-endpoint.md
└── tasks.md             # Phase 2 output
```

### Source Code

```text
backend/
├── internal/
│   └── handlers/
│       └── export.go        # NEW: NoteExportZIP handler
└── cmd/
    └── server/
        └── main.go          # MODIFIED: add route GET /notes/{slug}/export

frontend/
└── templates/
    └── notes/
        └── reader.html      # MODIFIED: add Export button in topbar-actions
```
