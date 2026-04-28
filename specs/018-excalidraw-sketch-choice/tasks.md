# Tasks: Excalidraw Sketch Choice

**Input**: `/specs/018-excalidraw-sketch-choice/` (spec, plan, data-model, research)
**Prerequisites**: plan.md, spec.md, research.md

## Phase 1: Excalidraw Island Bundle

- [x] T001 Add `@excalidraw/excalidraw` to `frontend-build/package.json`
- [x] T002 Create `frontend-build/src/excalidraw-island.tsx` with `initExcalidrawIsland()` global (same contract as tldraw island)
- [x] T003 Update `frontend-build/vite.config.ts` for multi-entry build (tldraw + excalidraw bundles)
- [x] T004 Run `npm run build` and verify both bundles output to `frontend/static/vendor/`

## Phase 2: Backend Storage Layer

- [x] T005 Extend `backend/internal/storage/drawings.go` — add `DrawingType` type, `DetectDrawingType`, update `ReadDrawing`/`WriteDrawing`/`DeleteDrawing`/`DrawingExists` to support both file extensions
- [x] T006 Add/update unit tests in `backend/internal/storage/drawings_test.go`

## Phase 3: Backend API Handlers

- [x] T007 Update `backend/internal/handlers/drawings.go` — `DrawingGET` returns type-wrapped response, `DrawingPUT` accepts `?type=` param, `DrawingDELETE` handles both types, `DrawingTypeForNote` returns type
- [x] T008 Update `backend/internal/handlers/notes.go` — pass `DrawingType` to editor and reader templates
- [x] T009 Update `backend/internal/handlers/history.go` — `NoteVersionDrawingGET` detects tool type at commit, diff template data includes drawing type
- [x] T010 Update `backend/internal/handlers/public.go` — `PublicDrawingGET` returns type-wrapped response, public note template gets `DrawingType`
- [x] T011 Update `backend/internal/handlers/shares.go` — pass `DrawingType` to shared reader/editor templates
- [x] T012 Add/update integration tests in `backend/internal/handlers/drawings_test.go` for both tool types

## Phase 4: Frontend Templates

- [x] T013 Update `frontend/templates/notes/editor.html` — tool selection UI (two buttons), conditional bundle loading based on `DrawingType`, init correct island
- [x] T014 Update `frontend/templates/notes/reader.html` — conditional bundle and island init based on `DrawingType`
- [x] T015 Update `frontend/templates/notes/version.html` — conditional bundle and island init based on drawing type at version
- [x] T016 Update `frontend/templates/notes/diff.html` — conditional bundle loading for drawing diff panels
- [x] T017 Update `frontend/templates/shared/reader.html` — conditional bundle based on `DrawingType`
- [x] T018 Update `frontend/templates/public/note.html` — conditional bundle based on `DrawingType`

## Phase 5: Versioning Support

- [x] T019 Update `backend/internal/versioning/git.go` — extend `DiffResult` with `DrawingType` field for old/new versions
- [x] T020 Update version/diff handlers to detect drawing type at each commit and pass to templates

## Phase 6: Verification

- [x] T021 Run `make test` and verify ≥76% coverage (passing)
- [x] T022 Run `make build-frontend` to verify both bundles build cleanly
- [ ] T023 Manual quickstart validation per `quickstart.md`
