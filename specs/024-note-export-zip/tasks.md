# Tasks: Note Export as ZIP

**Feature**: 024-note-export-zip  
**Total tasks**: 7  
**Date**: 2026-05-08

## Phase 1: Setup

- [ ] T001 Verify `archive/zip` import path and existing handler file structure in `backend/internal/handlers/`

## Phase 2: Core Implementation (US1 — Export ZIP)

- [ ] T002 [US1] Create `backend/internal/handlers/export.go` with `NoteExportZIP` handler: reads note, assembles ZIP (note.md + drawings), sets Content-Disposition header, writes to response
- [ ] T003 [US1] Register route `r.Get("/notes/{slug}/export", h.NoteExportZIP)` in `backend/cmd/server/main.go`
- [ ] T004 [P] [US1] Add integration test for `GET /notes/{slug}/export` in `backend/internal/handlers/export_test.go` (or nearest integration test location)

## Phase 3: UI (US2 — Export Button in Reader)

- [ ] T005 [US2] Add Export button to reader topbar in `frontend/templates/notes/reader.html` — anchor tag with `href="/notes/{{.Note.Slug}}/export"` styled as topbar action button

## Phase 4: Polish

- [ ] T006 [P] Verify full test suite passes (`make test`)
- [ ] T007 Verify build compiles cleanly (`make build`)

## Dependencies

- T003 depends on T002
- T005 is independent
- T006, T007 depend on T002, T003, T005

## Suggested MVP

T002 + T003 + T005 deliver the full feature. T004 adds test coverage.
