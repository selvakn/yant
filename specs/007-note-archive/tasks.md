# Tasks: Note Archive

**Input**: Design documents from `/specs/007-note-archive/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md

**Tests**: Required per Constitution Principle IV (integration testing) and Principle VI (test-before-commit).

**Organization**: Tasks grouped by user story for independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story (US1, US2, US3)
- Exact file paths included

---

## Phase 1: Setup

**Purpose**: Schema migration and model updates

- [x] T001 Add `archived` column to notes table in `backend/internal/models/models.go` InitSchema
- [x] T002 Add `Archived` field to Note struct in `backend/internal/models/models.go`
- [x] T003 Update scanNote and scanNoteRow to include archived field in `backend/internal/models/models.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core model functions that all user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T004 Update ListNotes to accept archived bool parameter in `backend/internal/models/models.go`
- [x] T005 Update all ListNotes call sites to pass archived=false in `backend/internal/handlers/notes.go`
- [x] T006 Add ArchiveNote function in `backend/internal/models/models.go`
- [x] T007 Add RestoreNote function in `backend/internal/models/models.go`
- [x] T008 Update ListTagsForUser to accept archived bool parameter in `backend/internal/models/models.go`
- [x] T009 Update TagsListGET to pass archived=false in `backend/internal/handlers/tags.go`
- [x] T010 Update SearchNotes to accept archived bool parameter in `backend/internal/models/search.go`
- [x] T011 Update NotesSearchGET to pass archived=false in `backend/internal/handlers/notes.go`
- [x] T012 Add unit tests for ArchiveNote and RestoreNote in `backend/internal/models/models_test.go`

**Checkpoint**: All existing functionality works unchanged; new archive functions tested

---

## Phase 3: User Story 1 - Archive a note (Priority: P1) 🎯 MVP

**Goal**: User can archive a note from the notes list or reader/editor views

**Independent Test**: Click "Archive" on a note; note disappears from main list

### Implementation for User Story 1

- [x] T013 [US1] Add NotesArchivePUT handler in `backend/internal/handlers/notes.go`
- [x] T014 [US1] Register PUT /notes/{slug}/archive route in `backend/cmd/server/main.go`
- [x] T015 [US1] Add Archive button to note items in `frontend/templates/notes/list.html`
- [x] T016 [US1] Add Archive button to reader view in `frontend/templates/notes/reader.html`
- [x] T017 [US1] Add Archive button to editor view in `frontend/templates/notes/editor.html`
- [x] T018 [US1] Add integration tests for archive handler in `backend/internal/handlers/handlers_test.go`

**Checkpoint**: Notes can be archived from list/reader/editor; tests pass

---

## Phase 4: User Story 2 - View archived notes (Priority: P1)

**Goal**: User can access Archive section with search and tag filtering

**Independent Test**: Navigate to Archive section; see all archived notes with working search/filter

### Implementation for User Story 2

- [ ] T019 [US2] Create archive.go handler file in `backend/internal/handlers/archive.go`
- [ ] T020 [US2] Add ArchiveListGET handler in `backend/internal/handlers/archive.go`
- [ ] T021 [US2] Add ArchiveSearchGET handler in `backend/internal/handlers/archive.go`
- [ ] T022 [US2] Add ArchiveTagsGET handler for archive-specific tags in `backend/internal/handlers/archive.go`
- [ ] T023 [US2] Register GET /archive route in `backend/cmd/server/main.go`
- [ ] T024 [US2] Register GET /archive/search route in `backend/cmd/server/main.go`
- [ ] T025 [US2] Register GET /archive/tags route in `backend/cmd/server/main.go`
- [ ] T026 [US2] Create archive/list.html template in `frontend/templates/archive/list.html`
- [ ] T027 [US2] Create archive/search-results.html partial in `frontend/templates/archive/search-results.html`
- [ ] T028 [US2] Add Archive link to sidebar in `frontend/templates/base.html`
- [ ] T029 [US2] Add integration tests for archive list and search in `backend/internal/handlers/handlers_test.go`

**Checkpoint**: Archive section displays archived notes with working search and tag filter; tests pass

---

## Phase 5: User Story 3 - Restore an archived note (Priority: P1)

**Goal**: User can restore an archived note back to the active notes list

**Independent Test**: Click "Restore" on archived note; note returns to active list

### Implementation for User Story 3

- [ ] T030 [US3] Add NotesRestorePUT handler in `backend/internal/handlers/notes.go`
- [ ] T031 [US3] Register PUT /notes/{slug}/restore route in `backend/cmd/server/main.go`
- [ ] T032 [US3] Add Restore button to archived notes in `frontend/templates/archive/list.html`
- [ ] T033 [US3] Add integration tests for restore handler in `backend/internal/handlers/handlers_test.go`

**Checkpoint**: Archived notes can be restored to active list; tests pass

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final validation and edge case handling

- [ ] T034 Handle empty archive section (show "No archived notes" message) in `frontend/templates/archive/list.html`
- [ ] T035 Ensure archived notes accessible via direct URL in editor in `backend/internal/handlers/notes.go`
- [ ] T036 Run full test suite and verify ≥90% coverage: `make coverage`
- [ ] T037 Manual validation per quickstart.md scenarios

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - start immediately
- **Foundational (Phase 2)**: Depends on Setup - BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational
- **User Story 2 (Phase 4)**: Depends on Foundational (can parallelize with US1)
- **User Story 3 (Phase 5)**: Depends on Foundational (can parallelize with US1/US2)
- **Polish (Phase 6)**: Depends on all user stories complete

### User Story Dependencies

- **US1 (Archive)**: Independent - core archive action
- **US2 (View Archive)**: Independent - can start after foundational
- **US3 (Restore)**: Independent - can start after foundational

### Parallel Opportunities

```text
After Foundational (Phase 2) completes:
├── User Story 1 (Archive action)
├── User Story 2 (Archive section) [can parallelize]
└── User Story 3 (Restore action) [can parallelize]
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T003)
2. Complete Phase 2: Foundational (T004-T012)
3. Complete Phase 3: User Story 1 (T013-T018)
4. **STOP and VALIDATE**: Test archive action independently
5. Deploy/demo if ready

### Incremental Delivery

1. Setup + Foundational → Schema ready, existing features work
2. Add User Story 1 → Can archive notes
3. Add User Story 2 → Can view archived notes
4. Add User Story 3 → Can restore archived notes
5. Polish → Edge cases handled

---

## Notes

- [P] tasks can run in parallel (different files)
- [Story] label maps task to user story for traceability
- Constitution Principle VI: Run `make test` before each commit; fix failures before proceeding
- Constitution Principle IV: Maintain ≥90% backend coverage
- Commit after each task or logical group (only when tests pass)
- Archived notes remain editable via direct URL (no blocking)
