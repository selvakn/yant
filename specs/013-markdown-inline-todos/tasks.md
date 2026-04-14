# Tasks: Inline Markdown Todos

**Input**: Design documents from `/specs/013-markdown-inline-todos/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/api.md

**Tests**: Required by constitution (Principles IV and VI). Tests included for all phases.

**Organization**: Tasks grouped by user story for independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup

**Purpose**: No new project setup needed — project exists. This phase is empty.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Todo parsing, DB schema, and goldmark configuration that ALL user stories depend on.

**CRITICAL**: No user story work can begin until this phase is complete.

- [ ] T001 Add `note_todos` table schema to `InitSchema()` in `backend/internal/models/models.go`
- [ ] T002 [P] Create `ParseTodos()` function (regex-based, extracts line number, text, due date, completed status) in `backend/internal/models/todos.go`
- [ ] T003 [P] Create `SyncTodos()` function (delete + re-insert pattern, matching `SyncTags`) in `backend/internal/models/todos.go`
- [ ] T004 Write tests for `ParseTodos()` and `SyncTodos()` in `backend/internal/models/todos_test.go`
- [ ] T005 Configure goldmark with `extension.TaskList` in `Handler` struct, replace `goldmark.Convert()` with instance method in `backend/internal/handlers/notes.go`
- [ ] T006 Hook `SyncTodos()` into the note update flow (alongside existing `SyncTags` and `SyncLinks` calls) in `backend/internal/handlers/notes.go`
- [ ] T007 Add `RebuildTodos` to the `RebuildDB` function for rebuilding from markdown files in `backend/internal/models/models.go`

**Checkpoint**: Todo parsing, storage, and goldmark config ready. All user stories can now proceed.

---

## Phase 3: User Story 1 — Add a Todo to a Note (Priority: P1) MVP

**Goal**: Users write `- [ ]`/`- [x]` lines with optional `@due(YYYY-MM-DD)` in markdown. Reader view renders interactive checkboxes with due date badges. Clicking a checkbox toggles the todo in the markdown source.

**Independent Test**: Write a note with todo lines, view it, click a checkbox, verify it toggles in the markdown.

### Implementation for User Story 1

- [ ] T008 [US1] Add custom goldmark HTML renderer that removes `disabled` from checkboxes and adds `data-slug`/`data-line` attributes in `backend/internal/handlers/notes.go`
- [ ] T009 [US1] Add `@due(YYYY-MM-DD)` post-processing on rendered HTML — replace with `<span class="todo-due" data-date="...">formatted date</span>` in `backend/internal/handlers/notes.go`
- [ ] T010 [US1] Add `ToggleTodoInMarkdown()` function (read markdown, toggle specific line, write back, re-sync) in `backend/internal/models/todos.go`
- [ ] T011 [US1] Create `TodoTogglePUT` handler (`PUT /notes/{slug}/todo`, accepts `{"line": N, "checked": bool}`) in `backend/internal/handlers/todos.go`
- [ ] T012 [US1] Register `PUT /notes/{slug}/todo` route in `backend/cmd/server/main.go`
- [ ] T013 [US1] Add checkbox click handler in reader template — htmx PUT on click, swap none, update checkbox state in `frontend/templates/notes/reader.html`
- [ ] T014 [P] [US1] Add CSS styles for todo checkboxes, `.todo-due` badge, `.todo-overdue` badge, and checked strikethrough in `frontend/static/css/app.css`
- [ ] T015 [US1] Write handler tests for `TodoTogglePUT` (toggle on/off, invalid line, note not found) in `backend/internal/handlers/handlers_test.go`
- [ ] T016 [US1] Write tests for `ToggleTodoInMarkdown()` in `backend/internal/models/todos_test.go`

**Checkpoint**: User Story 1 fully functional — todos render as interactive checkboxes in reader view with one-click toggle.

---

## Phase 4: User Story 2 — View All Pending Todos Across Notes (Priority: P2)

**Goal**: Dedicated `/todos` page aggregating all pending items from non-archived notes, sorted by due date, with tag display, note links, and tag filtering.

**Independent Test**: Create multiple notes with todos, navigate to `/todos`, verify all pending items appear sorted with tags and clickable note links. Filter by tag.

### Implementation for User Story 2

- [ ] T017 [US2] Create `ListPendingTodos()` query (join note_todos + notes + note_tags, filter by user/archived/completed, sort by due date, optional tag filter) in `backend/internal/models/todos.go`
- [ ] T018 [US2] Create `TodosListGET` handler (full page) and `TodosSearchGET` handler (htmx partial for tag filter) in `backend/internal/handlers/todos.go`
- [ ] T019 [US2] Register `GET /todos` route in `backend/cmd/server/main.go`
- [ ] T020 [US2] Create `frontend/templates/todos/list.html` — list of todo items with checkbox, task text, due date badge, overdue highlight, note title link, note tags, tag filter links
- [ ] T021 [P] [US2] Add CSS styles for todos view (todo list items, overdue row highlight, tag filter bar) in `frontend/static/css/app.css`
- [ ] T022 [US2] Write tests for `ListPendingTodos()` (sorting, filtering, archived exclusion) in `backend/internal/models/todos_test.go`
- [ ] T023 [US2] Write handler tests for `TodosListGET` in `backend/internal/handlers/handlers_test.go`

**Checkpoint**: User Stories 1 AND 2 both work independently. Users can view and filter all pending todos.

---

## Phase 5: User Story 3 — Mark Complete from Todos View (Priority: P3)

**Goal**: Checkboxes in the todos view toggle completion via the same `PUT /notes/{slug}/todo` endpoint. Completed items fade out.

**Independent Test**: Open todos view, click a checkbox, verify the item completes and the source note markdown is updated.

### Implementation for User Story 3

- [ ] T024 [US3] Add htmx PUT attributes to checkboxes in `frontend/templates/todos/list.html` — on click, send toggle request, on success remove item from list with fade-out
- [ ] T025 [P] [US3] Add fade-out transition CSS for completed todo items in `frontend/static/css/app.css`
- [ ] T026 [US3] Write handler test verifying toggle from todos view updates markdown and removes item from pending list in `backend/internal/handlers/handlers_test.go`

**Checkpoint**: All interactive features work — toggle from both reader and todos view.

---

## Phase 6: User Story 4 — Navigate to Todos View (Priority: P4)

**Goal**: Todos view accessible from sidebar with pending count and via keyboard shortcut.

**Independent Test**: Open sidebar, see "Todos" link with count, click to navigate. Press `d` to navigate.

### Implementation for User Story 4

- [ ] T027 [US4] Add `CountPendingTodos()` query in `backend/internal/models/todos.go`
- [ ] T028 [US4] Include todo count in sidebar data — update `TagsListGET` handler to pass `TodoCount` in `backend/internal/handlers/notes.go`
- [ ] T029 [US4] Add "Todos" link with pending count badge to `frontend/templates/tags/sidebar.html`
- [ ] T030 [US4] Add `d` keyboard shortcut for `/todos` navigation in `frontend/static/js/app.js`
- [ ] T031 [US4] Write test for `CountPendingTodos()` in `backend/internal/models/todos_test.go`

**Checkpoint**: Full feature complete — all 4 user stories functional.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Final validation and cleanup.

- [ ] T032 Run full test suite (`make test`), verify all tests pass
- [ ] T033 Run coverage check (`make coverage`), verify ≥75% gate passes
- [ ] T034 Run `make lint` and fix any issues
- [ ] T035 Validate quickstart.md scenarios manually

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: Empty — project exists
- **Foundational (Phase 2)**: No external deps — can start immediately. BLOCKS all user stories.
- **US1 (Phase 3)**: Depends on Phase 2 completion
- **US2 (Phase 4)**: Depends on Phase 2 completion. Can run in parallel with US1 (different files) but builds on the toggle endpoint from US1.
- **US3 (Phase 5)**: Depends on US1 (toggle endpoint) and US2 (todos view template)
- **US4 (Phase 6)**: Depends on US2 (todos route exists)
- **Polish (Phase 7)**: Depends on all stories complete

### Within Each User Story

- Models/queries before handlers
- Handlers before templates
- Core logic before tests (constitution requires tests pass before commit, but implementation can come first)
- Each story committed independently when tests pass

### Parallel Opportunities

- T002 and T003 can run in parallel (both write to `todos.go` but different functions)
- T014 (CSS) can run in parallel with any US1 implementation task
- T021 (CSS) can run in parallel with any US2 implementation task
- T025 (CSS) can run in parallel with US3 logic tasks

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 2: Foundational (T001–T007)
2. Complete Phase 3: User Story 1 (T008–T016)
3. **STOP and VALIDATE**: Test checkboxes render and toggle in reader view
4. Commit and push

### Incremental Delivery

1. Foundational → Todo parsing and storage works
2. US1 → Interactive checkboxes in reader → Commit
3. US2 → Aggregated todos view → Commit
4. US3 → Toggle from todos view → Commit
5. US4 → Sidebar + shortcut → Commit
6. Polish → Full validation → Commit

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- **Constitution**: Principle VI requires full test suite green before every commit. Fix failures before new work.
- **Constitution**: Principle IV requires integration tests and ≥90% backend coverage.
- Commit after each phase or logical group (only when tests pass).
