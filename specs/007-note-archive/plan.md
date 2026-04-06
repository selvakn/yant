# Implementation Plan: Note Archive

**Branch**: `007-note-archive` | **Date**: 2026-04-05 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/007-note-archive/spec.md`

## Summary

Add archive functionality to notes allowing users to archive notes (soft-delete) and restore them. Archived notes are accessible in a separate Archive section with full search and tag filtering capabilities.

## Technical Context

**Language/Version**: Go 1.22+
**Primary Dependencies**: Existing stack (chi, goldmark, htmx)
**Storage**: SQLite (note metadata with new `archived` column) + Markdown files (unchanged)
**Testing**: `go test ./...` with ≥90% coverage
**Target Platform**: Web browser (htmx-driven)
**Project Type**: Web application (monorepo)
**Performance Goals**: Archive/restore actions complete in <300ms
**Constraints**: Markdown files remain source of truth; archived status is metadata only
**Scale/Scope**: All existing notes can be archived/restored

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Confirm alignment with `.specify/memory/constitution.md`:

- [x] **I. Markdown-first storage** — Archived status stored in SQLite as metadata; Markdown files unchanged. Files remain portable.
- [x] **II. Simplicity** — No new dependencies; simple boolean column addition.
- [x] **III. Monorepo** — Frontend/backend layout respected; shared templates.
- [x] **IV. Integration testing** — Plan covers handler tests for archive/restore; ≥90% coverage maintained.
- [x] **V. Simple web UI** — UI uses htmx for archive actions; no heavy JS.
- [x] **VI. Commit & test discipline** — Frequent commits; tests pass before each commit.

## Project Structure

### Documentation (this feature)

```text
specs/007-note-archive/
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
│   │   ├── notes.go        # Add ArchivePUT, RestorePUT handlers
│   │   ├── archive.go      # Archive section handlers (list, search)
│   │   └── handlers_test.go # Add archive/restore tests
│   ├── models/
│   │   ├── models.go       # Add archived column, update ListNotes
│   │   └── models_test.go  # Add archive model tests
│   └── storage/
│       └── storage.go      # No changes (Markdown files unchanged)
└── cmd/server/
    └── main.go             # Register archive routes

frontend/
├── templates/
│   ├── base.html           # Add Archive nav link
│   └── archive/
│       ├── list.html       # Archived notes list
│       └── search-results.html # Archive search partial
└── static/
    └── css/
        └── app.css         # Archive section styles (minimal)
```

**Structure Decision**: Archive handlers in separate `archive.go` for clarity. Reuse existing search logic with archived filter.

## Complexity Tracking

> No violations; complexity tracking not required.
