# Tasks: Diagram Drawing in Notes

**Input**: `/specs/004-tldraw-diagrams/` (spec, plan, data-model)
**Prerequisites**: plan.md, spec.md, research.md

## Phase 1: Build System Setup

- [x] T001 Create `frontend-build/` directory with `package.json` (tldraw, react, react-dom, vite, typescript)
- [x] T002 Add `vite.config.ts` for library build outputting to `../frontend/static/vendor/`
- [x] T003 Add `tsconfig.json` for TypeScript/React JSX
- [x] T004 Add `.gitignore` for `node_modules/` in `frontend-build/`
- [x] T005 Update root `Makefile` with `build-frontend` target

## Phase 2: tldraw Island Bundle

- [x] T006 Create `frontend-build/src/tldraw-island.tsx` with `initTldrawIsland()` global
- [x] T007 Run `npm run build` and verify output in `frontend/static/vendor/tldraw-bundle.*`
- [x] T008 Built bundle committed to repo (not gitignored, so users don't need Node)

## Phase 3: Backend Storage

- [x] T009 Create `backend/internal/storage/drawings.go` with `ReadDrawing`, `WriteDrawing`, `DeleteDrawing`
- [x] T010 Add unit tests in `backend/internal/storage/drawings_test.go`

## Phase 4: Backend API

- [x] T011 Create `backend/internal/handlers/drawings.go` with `DrawingGET`, `DrawingPUT`, `DrawingDELETE`
- [x] T012 Register routes in `backend/cmd/server/main.go`: `/notes/{slug}/drawing`
- [x] T013 Extend `noteDelete` in `handlers/notes.go` to cascade-delete drawing file
- [x] T014 Add integration tests in `backend/internal/handlers/drawings_test.go`

## Phase 5: Frontend Integration

- [x] T015 Load tldraw bundle in editor.html and reader.html (conditional per HasDrawing)
- [x] T016 Update `frontend/templates/notes/editor.html` with drawing section and init script
- [x] T017 Update `frontend/templates/notes/reader.html` with drawing preview
- [x] T018 Add CSS for drawing container in `frontend/static/css/app.css`

## Phase 6: Verification

- [x] T019 Run `make test` and ensure ≥90% coverage maintained
- [ ] T020 Manual quickstart validation per `quickstart.md`
