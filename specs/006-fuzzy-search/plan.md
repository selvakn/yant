# Implementation Plan: Fuzzy Search for Notes

**Branch**: `006-fuzzy-search` | **Date**: 2026-04-05 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/006-fuzzy-search/spec.md`

## Summary

Add real-time fuzzy search to filter notes by title, tags, and body content. Users type in a search box and see results update instantly (debounced). Results are ranked by relevance (title > tags > body) with matching text highlighted. Keyboard navigation allows opening results without mouse.

## Technical Context

**Language/Version**: Go 1.22+
**Primary Dependencies**: github.com/lithammer/fuzzysearch/fuzzy (lightweight fuzzy matching)
**Storage**: SQLite (note metadata) + Markdown files (note body)
**Testing**: `go test ./...` with ≥90% coverage
**Target Platform**: Web browser (htmx-driven)
**Project Type**: Web application (monorepo)
**Performance Goals**: Results update within 300ms of typing stop
**Constraints**: Client-side perceived latency <300ms; search scoped to user's notes
**Scale/Scope**: Typical user has <500 notes; server-side search for all sizes

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Confirm alignment with `.specify/memory/constitution.md`:

- [x] **I. Markdown-first storage** — Notes remain portable; search reads from Markdown files on disk.
- [x] **II. Simplicity** — YAGNI; single lightweight dependency (fuzzysearch); no heavy indexing.
- [x] **III. Monorepo** — Frontend/backend layout respected; no new directories.
- [x] **IV. Integration testing** — Plan covers handler tests for search endpoint; ≥90% coverage maintained.
- [x] **V. Simple web UI** — UI approach uses htmx for live search; no heavy JS frameworks.
- [x] **VI. Commit & test discipline** — Implementation will use frequent commits; full test suite green before each commit.

## Project Structure

### Documentation (this feature)

```text
specs/006-fuzzy-search/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
└── tasks.md             # Phase 2 output (via /speckit.tasks)
```

### Source Code (repository root)

```text
backend/
├── internal/
│   ├── handlers/
│   │   ├── notes.go        # Add NotesSearchGET handler
│   │   └── handlers_test.go # Add search handler tests
│   ├── models/
│   │   ├── models.go       # Add SearchNote struct, SearchNotes function
│   │   └── models_test.go  # Add search model tests
│   └── storage/
│       └── storage.go      # ReadNote already exists (reuse)
└── cmd/server/
    └── main.go             # Register search route

frontend/
├── templates/
│   └── notes/
│       ├── list.html       # Add search input + results container
│       └── search-results.html # New partial for htmx swap
└── static/
    └── css/
        └── app.css         # Search box + highlight styles
```

**Structure Decision**: Extend existing `handlers/notes.go` with search handler. Add new template partial for search results. No new packages needed.

## Complexity Tracking

> No violations; complexity tracking not required.
