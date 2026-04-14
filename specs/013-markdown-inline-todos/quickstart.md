# Quickstart: Inline Markdown Todos

## Prerequisites

- Go 1.25+ installed
- Repository cloned, on branch `013-markdown-inline-todos`
- `make deps` run successfully

## Build & Run

```bash
make build && make run
```

## Test

```bash
make test          # Unit + handler tests
make coverage      # With ≥75% coverage gate
make integration-test  # Docker-based integration tests
```

## Feature Usage

### Writing Todos

In any note's markdown body, write standard checkbox lines:

```markdown
- [ ] Review the quarterly report @due(2026-04-20)
- [ ] Send meeting notes to the team
- [x] Book conference room @due(2026-04-10)
```

- `- [ ]` = pending todo
- `- [x]` = completed todo
- `@due(YYYY-MM-DD)` = optional target date (shown as a badge)

### Viewing Todos

- **In a note**: Checkboxes render as interactive elements in reader view. Click to toggle.
- **Aggregated view**: Navigate to `/todos` (or click "Todos" in the sidebar, or press `d`).

### Marking Complete

- **From reader view**: Click the checkbox next to a todo item.
- **From todos view**: Click the checkbox — the item is marked complete and fades out.

### Filtering

In the todos view, click any tag to filter todos to notes with that tag.

## Key Files

| Purpose              | Path                                       |
| -------------------- | ------------------------------------------ |
| Todo parsing/queries | `backend/internal/models/todos.go`         |
| Todo handlers        | `backend/internal/handlers/todos.go`       |
| Markdown rendering   | `backend/internal/handlers/notes.go`       |
| Todos view template  | `frontend/templates/todos/list.html`       |
| Sidebar (todo count) | `frontend/templates/tags/sidebar.html`     |
| Styles               | `frontend/static/css/app.css`              |
| Routes               | `backend/cmd/server/main.go`               |
