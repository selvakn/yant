# Tasks: Note Version Control

**Input**: Design documents from `/specs/014-note-version-control/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

## Phase 1: Foundational — Versioning Package

**Purpose**: Core git operations package that all user stories depend on

- [X] T001 [US1] Write tests for versioning package in `backend/internal/versioning/git_test.go` — cover Init, CommitFile, CommitDelete, Log, Show, Diff, ParseDiff
- [X] T002 [US1] Implement versioning package in `backend/internal/versioning/git.go` — Init, CommitFile, CommitDelete, Log, Show, Diff, ParseDiff, Version/DiffLine types
- [X] T003 [US1] Integrate versioning into server startup in `backend/cmd/server/main.go` — call Init(notesDir) after directory setup
- [X] T004 [US1] Integrate commit calls into existing note handlers in `backend/internal/handlers/notes.go` — CommitFile on create/update, CommitDelete on delete
- [X] T005 [P] [US1] Integrate commit calls into drawing handlers in `backend/internal/handlers/drawings.go` — CommitFile on save, CommitDelete on delete

**Checkpoint**: Git tracks all note/drawing saves. No UI yet.

---

## Phase 2: User Story 1 — View Note Change History (P1)

**Goal**: User can see a chronological list of all versions for a note with dates and change stats.

- [X] T006 [US1] Write handler tests for NoteHistoryGET in `backend/internal/handlers/history_test.go`
- [X] T007 [US1] Implement NoteHistoryGET handler in `backend/internal/handlers/history.go` — paginated version list
- [X] T008 [US1] Create history template `frontend/templates/notes/history.html` — version list with dates, change summary, pagination
- [X] T009 [US1] Add "History" link to note reader top bar in `frontend/templates/notes/reader.html`
- [X] T010 [US1] Register `/notes/{slug}/history` route in `backend/cmd/server/main.go`
- [X] T011 [US1] Add diff styling CSS to `frontend/static/css/app.css`

**Checkpoint**: User can browse version history for any note.

---

## Phase 3: User Story 2 — View Note at Specific Version (P2)

**Goal**: User can select a version from history and see the full rendered note as it was at that point.

- [X] T012 [US2] Write handler tests for NoteVersionGET in `backend/internal/handlers/history_test.go`
- [X] T013 [US2] Implement NoteVersionGET handler in `backend/internal/handlers/history.go` — render markdown at commit
- [X] T014 [US2] Create version view template `frontend/templates/notes/version.html` — rendered note with version banner
- [X] T015 [US2] Register `/notes/{slug}/history/{commit}` route in `backend/cmd/server/main.go`

**Checkpoint**: User can view any historical version of a note.

---

## Phase 4: User Story 3 — Compare Versions / Diff View (P3)

**Goal**: User can see a source-level unified diff between versions with color-coded additions and deletions.

- [X] T016 [US3] Write handler tests for NoteVersionDiffGET in `backend/internal/handlers/history_test.go`
- [X] T017 [US3] Implement NoteVersionDiffGET handler in `backend/internal/handlers/history.go` — unified diff view
- [X] T018 [US3] Create diff template `frontend/templates/notes/diff.html` — colored unified diff with line numbers
- [X] T019 [US3] Register `/notes/{slug}/history/{commit}/diff` route in `backend/cmd/server/main.go`
- [X] T020 [US3] Implement NoteVersionDrawingGET handler in `backend/internal/handlers/history.go` — serve tldraw JSON at version
- [X] T021 [US3] Add side-by-side tldraw rendering to diff template when drawing changed
- [X] T022 [US3] Register `/notes/{slug}/history/{commit}/drawing` route in `backend/cmd/server/main.go`

**Checkpoint**: User can see diffs for markdown and drawings.

---

## Phase 5: User Story 4 — Revert to Previous Version (P4)

**Goal**: User can restore a note to any previous version with confirmation and full history preserved.

- [X] T023 [US4] Write handler tests for NoteVersionRevertPOST in `backend/internal/handlers/history_test.go`
- [X] T024 [US4] Implement NoteVersionRevertPOST handler in `backend/internal/handlers/history.go` — revert with metadata sync
- [X] T025 [US4] Add revert button with confirmation dialog to version.html and history.html templates
- [X] T026 [US4] Register `POST /notes/{slug}/history/{commit}/revert` route in `backend/cmd/server/main.go`

**Checkpoint**: User can revert any note and history is fully preserved.

---

## Phase 6: Polish & Cross-Cutting

- [X] T027 [P] Update Dockerfile to add git to runtime stage
- [X] T028 [P] Update existing handler tests to account for git operations (ensure tests init git in temp dirs)
- [X] T029 Run full test suite and verify ≥90% coverage on `internal/...`

---

## Dependencies & Execution Order

- **Phase 1** (T001–T005): Foundation — must complete before any UI work
- **Phase 2** (T006–T011): US1 History — depends on Phase 1
- **Phase 3** (T012–T015): US2 Version View — depends on Phase 1
- **Phase 4** (T016–T022): US3 Diff View — depends on Phase 1
- **Phase 5** (T023–T026): US4 Revert — depends on Phase 1
- **Phase 6** (T027–T029): Polish — after all user stories

Phases 2–5 can run in parallel after Phase 1. Sequential order (P1→P4) recommended for single developer.
