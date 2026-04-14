# Implementation Plan: Note Version Control

**Branch**: `014-note-version-control` | **Date**: 2026-04-14 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/014-note-version-control/spec.md`

## Summary

Add version control to notes by using git to track every save operation on markdown and tldraw files. Expose history browsing, version viewing, unified diff comparison, and non-destructive revert through new server-rendered pages. No new Go dependencies — uses `os/exec` to call the git binary. No new database tables — git is the sole storage for version data.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: chi/v5, goldmark, scs/v2, modernc.org/sqlite (existing); `os/exec` + git binary (new, stdlib)
**Storage**: Markdown files (source of truth) + git repository at notes directory root (version history) + SQLite (metadata cache, unchanged)
**Testing**: `go test` with `t.TempDir()` for versioning unit tests; `httptest.Server` for handler tests; testcontainers for integration
**Target Platform**: Linux server (Docker container with Debian bookworm-slim)
**Project Type**: Web application (monorepo: Go backend + server-rendered HTML frontend)
**Performance Goals**: History/version/diff pages render within 2-3 seconds for notes with up to 500 versions
**Constraints**: Git binary must be available at runtime; single-writer semantics (no concurrent edit handling)
**Scale/Scope**: Personal note-taking — hundreds of notes, dozens of versions per note on average

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [x] **I. Markdown-first storage** — Git tracks the markdown files directly on the filesystem. No new derived cache. Markdown files remain portable and the source of truth.
- [x] **II. Simplicity** — Zero new Go dependencies. Uses stdlib `os/exec` to call the git binary. Simple functional package design matching existing `storage` package patterns.
- [x] **III. Monorepo** — All changes within existing `backend/` and `frontend/` directories. New `versioning` package under `backend/internal/`. New templates under `frontend/templates/notes/`.
- [x] **IV. Integration testing** — Plan includes unit tests for the versioning package, handler-level tests with real git repos in temp dirs, and integration tests. Coverage target ≥90%.
- [x] **V. Simple web UI** — Server-rendered HTML templates with htmx for interactions. Diff view uses CSS-styled lines, no JS diff libraries. Drawing diff reuses existing vendored tldraw in read-only mode.
- [x] **VI. Commit & test discipline** — Implementation follows frequent commits with full test suite green before each.

## Project Structure

### Documentation (this feature)

```text
specs/014-note-version-control/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── routes.md
└── tasks.md
```

### Source Code (repository root)

```text
backend/
├── cmd/server/
│   └── main.go                          # Add versioning.Init() call, new routes
├── internal/
│   ├── versioning/
│   │   ├── git.go                       # NEW: Git operations (init, commit, log, show, diff, parse)
│   │   └── git_test.go                  # NEW: Unit tests
│   ├── handlers/
│   │   ├── notes.go                     # MODIFY: Add commit calls to create/update/delete
│   │   ├── history.go                   # NEW: History, version, diff, revert handlers
│   │   ├── history_test.go              # NEW: Handler tests
│   │   └── drawings.go                  # MODIFY: Add commit calls to drawing save/delete
│   └── ...

frontend/
├── templates/
│   └── notes/
│       ├── reader.html                  # MODIFY: Add "History" link to top bar
│       ├── history.html                 # NEW: Version history list
│       ├── version.html                 # NEW: View note at specific version
│       └── diff.html                    # NEW: Unified diff view with drawing comparison
├── static/
│   └── css/
│       └── app.css                      # MODIFY: Add diff styling

Dockerfile                               # MODIFY: Add git to runtime stage
```

**Structure Decision**: Follows existing monorepo layout. New `versioning` package mirrors the `storage` package pattern (functional, no struct, operates on paths). New handlers file `history.go` groups all version-control endpoints, consistent with `archive.go`, `drawings.go`, `todos.go` pattern of domain-grouped handler files.

## Complexity Tracking

No constitution violations. No entries needed.
