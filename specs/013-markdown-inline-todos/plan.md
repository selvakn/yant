# Implementation Plan: Inline Markdown Todos

**Branch**: `013-markdown-inline-todos` | **Date**: 2026-04-13 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/013-markdown-inline-todos/spec.md`

## Summary

Add inline todo support to notes: parse `- [ ]`/`- [x]` markdown checkbox lines with optional `@due(YYYY-MM-DD)` dates, render as interactive checkboxes in reader view, and provide a dedicated `/todos` page aggregating all pending items across notes with tag-based filtering, one-click completion, and navigation to source notes. All data stays in plain markdown files; a derived SQLite table enables efficient querying.

## Technical Context

**Language/Version**: Go 1.25+ (backend), vanilla JS + htmx (frontend)  
**Primary Dependencies**: chi/v5 (routing), goldmark + goldmark TaskList extension (markdown), scs/v2 (sessions), modernc.org/sqlite (database)  
**Storage**: Markdown files (source of truth), SQLite `note_todos` table (derived cache for aggregation queries)  
**Testing**: Go `testing` package, `httptest` for handler tests, testcontainers for integration tests  
**Target Platform**: Linux server (Docker/distroless)  
**Project Type**: Web application (Go server + HTML templates + htmx)  
**Performance Goals**: Todos view loads in <2s, toggle completes in <1s  
**Constraints**: Markdown files are the single source of truth. No separate todo storage.  
**Scale/Scope**: Single-user personal app, hundreds of notes, dozens of active todos

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [x] **I. Markdown-first storage** — Todos stored as `- [ ]`/`- [x]` lines in markdown files. SQLite `note_todos` table is a derived, rebuildable cache populated by parsing markdown on save. Markdown files remain the source of truth.
- [x] **II. Simplicity** — Uses goldmark's built-in TaskList extension. Todo parsing follows the established tag-parsing pattern (regex → sync to DB). No new external dependencies beyond goldmark extension (already part of goldmark).
- [x] **III. Monorepo** — All changes within existing `backend/` and `frontend/` directories. No new projects or packages beyond `models/` additions.
- [x] **IV. Integration testing** — Plan includes handler tests for all new endpoints, model tests for todo parsing/querying, and storage tests for line-level toggle. Coverage target ≥90%.
- [x] **V. Simple web UI** — htmx for checkbox toggling and tag filtering. No new JS frameworks. Templates follow existing patterns.
- [x] **VI. Commit & test discipline** — Implementation follows incremental slices (parse → store → render → view → toggle). Each slice is independently testable and committable.

## Project Structure

### Documentation (this feature)

```text
specs/013-markdown-inline-todos/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0: research findings
├── data-model.md        # Phase 1: data model
├── quickstart.md        # Phase 1: quickstart guide
└── contracts/           # Phase 1: API contracts
    └── api.md
```

### Source Code (repository root)

```text
backend/
├── cmd/server/
│   └── main.go                    # Modified: add /todos routes
├── internal/
│   ├── handlers/
│   │   ├── notes.go               # Modified: goldmark config with TaskList, interactive checkbox rendering, @due badge
│   │   ├── todos.go               # New: TodosListGET, TodoTogglePUT handlers
│   │   └── handlers_test.go       # Modified: tests for new handlers
│   ├── models/
│   │   ├── models.go              # Modified: note_todos schema in InitSchema, todo sync in UpdateNote flow
│   │   ├── todos.go               # New: ParseTodos, SyncTodos, ListPendingTodos, ToggleTodoInMarkdown
│   │   └── todos_test.go          # New: tests for todo parsing, queries, toggle
│   └── storage/
│       └── storage.go             # Existing: ReadNote/WriteNote used by toggle flow

frontend/
├── templates/
│   ├── todos/
│   │   └── list.html              # New: todos view template
│   └── tags/
│       └── sidebar.html           # Modified: add Todos link with pending count
├── static/
│   ├── css/app.css                # Modified: todo checkbox, due-date badge, overdue styles
│   └── js/app.js                  # Modified: keyboard shortcut for todos view
```

**Structure Decision**: Follows existing monorepo layout. New handler file `todos.go` for the dedicated view, new model file `todos.go` for todo-specific logic. No new packages — todo logic lives in existing `handlers/` and `models/` packages, consistent with how tags and search were added.

## Implementation Slices

### Slice 1: Todo Parsing and Storage (P1 foundation)

Parse `- [ ]`/`- [x]` lines with optional `@due(YYYY-MM-DD)` from markdown body. Sync parsed todos to `note_todos` SQLite table on note save. Add `RebuildTodos` for rebuilding from markdown files.

**Files**: `models/models.go` (schema), `models/todos.go` (parse + sync + queries), `models/todos_test.go`

### Slice 2: Goldmark Checkbox Rendering (P1 rendering)

Configure goldmark with TaskList extension. Add a custom HTML renderer that:
- Renders checkboxes without `disabled` attribute
- Adds `data-slug` and `data-line` attributes for the toggle endpoint
- Transforms `@due(YYYY-MM-DD)` into a styled `<span class="todo-due">` badge

**Files**: `handlers/notes.go` (goldmark config, rendering), `css/app.css` (todo styles)

### Slice 3: Reader Checkbox Toggle (P1 interactivity)

Add `PUT /notes/{slug}/todo` endpoint that accepts a line number and new checked state, reads the markdown file, toggles the specific `- [ ]`↔`- [x]` on that line, writes it back, and re-syncs todos to DB.

**Files**: `handlers/todos.go` (TodoTogglePUT), `cmd/server/main.go` (route), `handlers_test.go` (tests), reader template JS (htmx click handler)

### Slice 4: Todos Aggregation View (P2)

Add `GET /todos` page listing all pending todos across non-archived notes. Each item shows: checkbox, task text, due date badge, note title link, note tags. Sorted by: overdue first, then by due date ascending, undated last. Tag-based filtering via query parameter.

**Files**: `handlers/todos.go` (TodosListGET), `models/todos.go` (ListPendingTodos query), `templates/todos/list.html`, `cmd/server/main.go` (route), `css/app.css` (styles)

### Slice 5: Complete from Todos View (P3)

The todos list checkboxes send `PUT /notes/{slug}/todo` (same endpoint as Slice 3) via htmx. On success, the item is visually marked complete and fades out.

**Files**: `templates/todos/list.html` (htmx attributes on checkboxes), `css/app.css` (fade-out transition)

### Slice 6: Navigation and Sidebar (P4)

Add "Todos" link to the sidebar with pending count. Add `d` keyboard shortcut to navigate to `/todos`. Update sidebar HTMX endpoint to include todo count.

**Files**: `templates/tags/sidebar.html`, `handlers/notes.go` (TagsListGET — add todo count), `js/app.js` (keyboard shortcut)

## Complexity Tracking

> No constitution violations. No complexity justification needed.
