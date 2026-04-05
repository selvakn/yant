# Tasks: Diagram Drawing in Notes

**Input**: `/specs/004-tldraw-diagrams/` (spec, plan, data-model)
**Prerequisites**: plan.md, spec.md, research.md

## Phase 1: Build System Setup

- [ ] T001 Create `frontend-build/` directory with `package.json` (tldraw, react, react-dom, vite, typescript)
- [ ] T002 Add `vite.config.ts` for library build outputting to `../frontend/static/vendor/`
- [ ] T003 Add `tsconfig.json` for TypeScript/React JSX
- [ ] T004 Add `.gitignore` for `node_modules/` in `frontend-build/`
- [ ] T005 Update root `Makefile` with `build-frontend` target

## Phase 2: tldraw Island Bundle

- [ ] T006 Create `frontend-build/src/tldraw-island.tsx` with `initTldrawIsland()` global
- [ ] T007 Run `npm run build` and verify output in `frontend/static/vendor/tldraw-bundle.*`
- [ ] T008 Add `tldraw-bundle.js` and `tldraw-bundle.css` to `.gitignore` (build artifacts)

## Phase 3: Backend Storage

- [ ] T009 Create `backend/internal/storage/drawings.go` with `ReadDrawing`, `WriteDrawing`, `DeleteDrawing`
- [ ] T010 Add unit tests in `backend/internal/storage/drawings_test.go`

## Phase 4: Backend API

- [ ] T011 Create `backend/internal/handlers/drawings.go` with `DrawingGET`, `DrawingPUT`, `DrawingDELETE`
- [ ] T012 Register routes in `backend/cmd/server/main.go`: `/notes/{slug}/drawing`
- [ ] T013 Extend `noteDelete` in `handlers/notes.go` to cascade-delete drawing file
- [ ] T014 Add integration tests in `backend/internal/handlers/drawings_test.go`

## Phase 5: Frontend Integration

- [ ] T015 Update `frontend/templates/base.html` to conditionally load tldraw bundle
- [ ] T016 Update `frontend/templates/notes/editor.html` with drawing section and init script
- [ ] T017 Update `frontend/templates/notes/reader.html` with drawing preview
- [ ] T018 Add CSS for drawing container in `frontend/static/css/app.css`

## Phase 6: Verification

- [ ] T019 Run `make test` and ensure ≥90% coverage maintained
- [ ] T020 Manual quickstart validation per `quickstart.md`
