# Tasks: Fuzzy Search for Notes

**Input**: Design documents from `/specs/006-fuzzy-search/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md

**Tests**: Required per Constitution Principle IV (integration testing) and Principle VI (test-before-commit).

**Organization**: Tasks grouped by user story for independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story (US1, US2, US3)
- Exact file paths included

---

## Phase 1: Setup

**Purpose**: Add fuzzy search dependency

- [ ] T001 Add fuzzysearch dependency: `cd backend && go get github.com/lithammer/fuzzysearch/fuzzy`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core search infrastructure that all user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T002 Add SearchResult struct in `backend/internal/models/search.go`
- [ ] T003 Add SearchNotes function skeleton in `backend/internal/models/search.go` (returns empty slice for now)
- [ ] T004 Add NotesSearchGET handler skeleton in `backend/internal/handlers/notes.go` (calls SearchNotes, renders partial)
- [ ] T005 Register GET /notes/search route in `backend/cmd/server/main.go`
- [ ] T006 Create search-results.html partial template in `frontend/templates/notes/search-results.html`
- [ ] T007 Add search input with htmx to `frontend/templates/notes/list.html`
- [ ] T008 [P] Add search box and highlight styles to `frontend/static/css/app.css`
- [ ] T009 Add integration tests for search endpoint in `backend/internal/handlers/handlers_test.go`

**Checkpoint**: Search input visible, endpoint responds with empty results, tests pass

---

## Phase 3: User Story 1 - Search notes as you type (Priority: P1) 🎯 MVP

**Goal**: User types in search box and sees note list filter in real-time with debounce

**Independent Test**: Type "meet" in search box; notes containing "meeting" appear; others disappear

### Implementation for User Story 1

- [ ] T010 [US1] Implement basic SearchNotes with exact substring matching in `backend/internal/models/search.go`
- [ ] T011 [US1] Update NotesSearchGET to pass results to template in `backend/internal/handlers/notes.go`
- [ ] T012 [US1] Render matching notes in `frontend/templates/notes/search-results.html`
- [ ] T013 [US1] Handle empty query (show all notes) in `backend/internal/models/search.go`
- [ ] T014 [US1] Handle no matches (show "No notes found") in `frontend/templates/notes/search-results.html`
- [ ] T015 [US1] Add unit tests for SearchNotes basic matching in `backend/internal/models/search_test.go`

**Checkpoint**: Typing filters notes by exact match; clearing shows all; tests pass

---

## Phase 4: User Story 2 - Fuzzy matching with ranking (Priority: P2)

**Goal**: Search tolerates typos; results ranked by relevance (title > tags > body)

**Independent Test**: Type "recpie" (misspelled); note titled "Recipe Ideas" appears in results

### Implementation for User Story 2

- [ ] T016 [US2] Replace exact matching with fuzzy.RankMatchFold in `backend/internal/models/search.go`
- [ ] T017 [US2] Implement weighted scoring (title 3x, tags 2x, body 1x) in `backend/internal/models/search.go`
- [ ] T018 [US2] Sort results by score descending in `backend/internal/models/search.go`
- [ ] T019 [US2] Add HighlightMatch helper function in `backend/internal/models/search.go`
- [ ] T020 [US2] Add BodySnippet helper function in `backend/internal/models/search.go`
- [ ] T021 [US2] Update search-results.html to display highlights in `frontend/templates/notes/search-results.html`
- [ ] T022 [US2] Add unit tests for fuzzy matching and scoring in `backend/internal/models/search_test.go`

**Checkpoint**: Typos find matches; title matches rank higher; highlights visible; tests pass

---

## Phase 5: User Story 3 - Keyboard navigation (Priority: P3)

**Goal**: Arrow keys navigate results; Enter opens selected note

**Independent Test**: Type query, press Down Arrow twice, press Enter; third result opens

### Implementation for User Story 3

- [ ] T023 [US3] Add data attributes to result items for JS selection in `frontend/templates/notes/search-results.html`
- [ ] T024 [US3] Add keyboard navigation JS to `frontend/templates/notes/list.html`
- [ ] T025 [US3] Add .selected class styling to `frontend/static/css/app.css`
- [ ] T026 [US3] Handle Escape key to clear search in `frontend/templates/notes/list.html`

**Checkpoint**: Arrow keys move selection; Enter opens note; Escape clears; all tests pass

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final validation and edge case handling

- [ ] T027 Handle very long queries (truncate to 200 chars) in `backend/internal/handlers/notes.go`
- [ ] T028 Handle special characters safely (no regex injection) in `backend/internal/models/search.go`
- [ ] T029 Run full test suite and verify ≥90% coverage: `make coverage`
- [ ] T030 Manual validation per quickstart.md scenarios

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - start immediately
- **Foundational (Phase 2)**: Depends on Setup - BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational
- **User Story 2 (Phase 4)**: Depends on Foundational (can parallelize with US1 if desired)
- **User Story 3 (Phase 5)**: Depends on Foundational (frontend-only, can parallelize)
- **Polish (Phase 6)**: Depends on all user stories complete

### Within Each User Story

- Implementation tasks are sequential within story
- Tests must pass before moving to next story
- Commit after each logical group (per Principle VI)

### Parallel Opportunities

```text
After Foundational (Phase 2) completes:
├── User Story 1 (backend focus)
├── User Story 2 (can start after T010-T011)
└── User Story 3 (frontend focus, independent)
```

---

## Parallel Example: Foundational Phase

```bash
# These can run in parallel (different files):
Task: "Add search box and highlight styles to frontend/static/css/app.css" [T008]
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001)
2. Complete Phase 2: Foundational (T002-T009)
3. Complete Phase 3: User Story 1 (T010-T015)
4. **STOP and VALIDATE**: Test basic search independently
5. Deploy/demo if ready

### Incremental Delivery

1. Setup + Foundational → Basic search endpoint working
2. Add User Story 1 → Exact match filtering works
3. Add User Story 2 → Fuzzy matching + ranking + highlights
4. Add User Story 3 → Keyboard navigation
5. Polish → Edge cases handled

---

## Notes

- [P] tasks can run in parallel (different files)
- [Story] label maps task to user story for traceability
- Constitution Principle VI: Run `make test` before each commit; fix failures before proceeding
- Constitution Principle IV: Maintain ≥90% backend coverage
- Commit after each task or logical group (only when tests pass)
