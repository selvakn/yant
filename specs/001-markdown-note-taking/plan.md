# Implementation Plan: Markdown Note Taking App

**Branch**: `001-markdown-note-taking` | **Date**: 2026-04-05 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-markdown-note-taking/spec.md`

## Summary

A multi-user note-taking web app where users author notes in Markdown
with a live reader mode, drag-and-drop image support, and hashtag-based
tag navigation. Notes are stored as plain Markdown files on the
filesystem (source of truth) with SQLite as a metadata index. Mock
authentication via username-only login backed by signed-cookie sessions.

Backend is Go with Chi v5 router serving Go `html/template` pages with
htmx for progressive enhancement and EasyMDE as the Markdown editor.

> **Note on StackEdit**: StackEdit (github.com/benweet/stackedit) is a
> full Vue.js SPA and cannot be embedded as a standalone editor widget
> in a server-rendered page. `stackedit.js` provides only an iframe
> shim pointing at the hosted SPA — architecturally fragile and
> dependent on external hosting. StackEdit uses CodeMirror 5 internally,
> which is the same engine powering **EasyMDE** — the actively
> maintained editor chosen here. EasyMDE provides the same editing
> experience (split preview, toolbar, drag-and-drop images) as a single
> vendored JS+CSS file with no build step required.

## Technical Context

**Language/Version**: Go 1.22+
**Primary Dependencies**: github.com/go-chi/chi/v5, github.com/yuin/goldmark, github.com/alexedwards/scs/v2, modernc.org/sqlite, EasyMDE 2.x (JS vendored), htmx 2.x (JS vendored)
**Storage**: Markdown files (source of truth) + SQLite via modernc.org/sqlite (pure Go, no CGO)
**Testing**: go test + net/http/httptest + go test -coverprofile (≥90% coverage enforced)
**Target Platform**: Modern web browsers (Chrome, Firefox, Safari, Edge — latest 2 versions)
**Project Type**: web-service (monorepo: backend + frontend)
**Performance Goals**: Note CRUD <2s, editor/reader toggle <1s, tag filtering <1s for ≤1,000 notes
**Constraints**: No CGO (pure Go SQLite), no build step for frontend, no heavy JS frameworks, markdown files portable across OS
**Scale/Scope**: Single-instance deployment, up to 1,000 notes per user

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Evidence |
|-----------|--------|----------|
| I. Markdown-First Storage | PASS | Notes stored as `.md` files per user; modernc/sqlite is derived metadata-only index, rebuildable from filesystem |
| II. Simplicity | PASS | Chi v5 (net/http-compatible, no custom Context), raw database/sql + modernc/sqlite (no ORM), EasyMDE + htmx vendored — zero build toolchain |
| III. Monorepo Structure | PASS | `backend/` and `frontend/` in same repo; Go binary serves templates from `frontend/templates/` at runtime |
| IV. Integration Testing | PASS | httptest.NewServer exercises real HTTP stack; go test -coverprofile enforces ≥90%; t.TempDir() for real filesystem in tests |
| V. Simple Web Interface | PASS | html/template server-rendered pages, htmx for progressive enhancement, EasyMDE drop-in editor — no Vue/React/Angular |

No violations. Complexity Tracking table not required.

## Project Structure

### Documentation (this feature)

```text
specs/001-markdown-note-taking/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (REST API)
└── tasks.md             # Phase 2 output (/speckit.tasks)
```

### Source Code (repository root)

```text
backend/
├── cmd/
│   └── server/
│       └── main.go              # Entry point: wires Chi router, serves static + templates
├── internal/
│   ├── auth/
│   │   └── auth.go              # Session middleware (scs), login/logout handlers, mock auth
│   ├── handlers/
│   │   ├── notes.go             # Note CRUD HTTP handlers
│   │   ├── tags.go              # Tag navigation handler
│   │   ├── images.go            # Image upload + serve handler
│   │   └── handlers_test.go     # Integration tests (httptest.NewServer)
│   ├── models/
│   │   ├── models.go            # SQLite schema init + all DB query functions
│   │   └── models_test.go       # DB layer tests with in-memory SQLite
│   └── storage/
│       ├── storage.go           # Markdown file read/write/delete helpers
│       └── storage_test.go      # Unit tests using t.TempDir()
├── notes/                       # Markdown file storage root (gitignored in prod)
│   └── {user_id}/
│       └── {slug}.md
├── uploads/                     # Image storage root (gitignored in prod)
│   └── {user_id}/
│       └── {uuid}.{ext}
├── go.mod
└── go.sum

frontend/
├── static/
│   ├── vendor/
│   │   ├── easymde.min.js
│   │   ├── easymde.min.css
│   │   └── htmx.min.js
│   ├── js/
│   │   └── app.js               # EasyMDE init + imageUploadFunction callback
│   └── css/
│       └── app.css
└── templates/
    ├── base.html                 # Layout: nav, script/link includes, content block
    ├── login.html                # Username-only login form
    ├── notes/
    │   ├── list.html             # Home: note list with tags + updated_at
    │   ├── editor.html           # Editor mode (EasyMDE textarea)
    │   └── reader.html           # Reader mode (goldmark-rendered HTML)
    └── tags/
        └── sidebar.html          # Tag nav partial (htmx swap target)
```

**Structure Decision**: Go binary at `backend/cmd/server/main.go` wires
the Chi router and serves static files from `../frontend/static/` and
templates from `../frontend/templates/` via relative path (or
`go:embed` for single-binary distribution). All logic under
`backend/internal/` — no exported packages needed.

## Complexity Tracking

> No violations detected. Table not required.
