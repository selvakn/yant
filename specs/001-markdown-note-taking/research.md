# Research: Markdown Note Taking App

**Date**: 2026-04-05
**Branch**: `001-markdown-note-taking`

## Backend Language & Framework

**Decision**: Go 1.22+ with Chi v5 router

**Rationale**: Go is the user-mandated backend language. Chi v5 is
chosen over the stdlib `net/http` ServeMux because it provides route
grouping, URL parameter extraction, and middleware chaining without
introducing framework-specific handler signatures. Every Chi handler
is a plain `http.HandlerFunc` — no lock-in, `httptest` works directly.
Echo and Gin impose their own Context types, complicating testing and
violating the Simplicity principle.

**Alternatives considered**:
- stdlib `net/http` (Go 1.22 ServeMux): Lacks route grouping and
  subtree mounting; reimplementing those adds more complexity than
  importing Chi.
- Echo / Gin: Custom Context types lock in framework abstractions.
  YAGNI for an internal app at this scale.

## Markdown Editor (Frontend)

**Decision**: EasyMDE 2.x (vendored JS+CSS, no build step)

**Rationale**: The user requested StackEdit
(github.com/benweet/stackedit), but StackEdit is a full Vue.js SPA
and is not embeddable as an editor widget in a server-rendered page.
The companion `stackedit.js` library provides only an iframe shim
pointing at the externally-hosted SPA — architecturally fragile,
requires external network access, and offers no upload customization.
StackEdit uses CodeMirror 5 internally; EasyMDE is built on the same
CodeMirror 5 engine, actively maintained (v2.20.0, March 2025), and
ships as a single JS+CSS file. It supports drag-and-drop image upload
natively via `imageUploadFunction` callback and requires zero build
tooling.

**Alternatives considered**:
- StackEdit SPA (iframe via stackedit.js): External dependency, no
  image upload customization, fragile.
- CodeMirror 5 directly: Raw editor with no markdown toolbar or
  preview — would require reimplementing EasyMDE.
- Milkdown: ProseMirror-based WYSIWYG, requires npm build step.

## SQLite Driver (Go)

**Decision**: modernc.org/sqlite (pure Go, no CGO)

**Rationale**: `go build` works without a C compiler. Cross-
compilation works without a CGO cross-toolchain. For a metadata-
only SQLite index the performance difference vs. mattn/go-sqlite3
is irrelevant at ≤1,000 notes/user.

**Alternatives considered**:
- mattn/go-sqlite3: Requires CGO_ENABLED=1, complicates CI and
  Docker builds. Violates "minimal friction" spirit of Simplicity.

## Markdown Rendering (Server-Side)

**Decision**: github.com/yuin/goldmark

**Rationale**: Full CommonMark compliance. Blackfriday v2 is not
CommonMark-compliant and has known list-rendering edge cases.
Goldmark's AST is interface-based and extensible; supports XSS-safe
HTML output. Gitea migrated from blackfriday to goldmark for these
reasons.

**Alternatives considered**:
- blackfriday v2: Not CommonMark-compliant, in low-maintenance mode.
- russross/blackfriday v1: Deprecated.

## Session Management

**Decision**: github.com/alexedwards/scs/v2 with in-memory store

**Rationale**: Actively maintained (v2.7.0+). Uses Go context for
automatic session loading/saving via middleware (no explicit Save()
call to forget). In-memory store is ideal for mock auth and tests.
gorilla/sessions has ambiguous maintenance status (archived 2022,
nominally revived 2023, low activity since).

**Alternatives considered**:
- gorilla/sessions: Maintenance concerns; requires explicit Save() on
  every response writer.
- Plain signed cookie (crypto/hmac): Zero-dependency option, viable
  if session needs never expand — retained as fallback.

## Testing Strategy

**Decision**: net/http/httptest + go test -coverprofile

**Rationale**: Standard library approach, no external test framework.
`httptest.NewServer(router)` starts a real HTTP server for integration
tests. `t.TempDir()` provides real filesystem isolation per test.
In-memory SQLite (`:memory:`) for DB tests — fast and isolated.
`go test -cover -coverprofile=coverage.out ./...` + `go tool cover
-func=coverage.out` enforces ≥90%.

**Alternatives considered**:
- testify: Reduces assertion boilerplate but adds a dependency.
  Not justified at this scale per Simplicity principle.
- httpx/anyio (Python patterns): Not applicable to Go.

## Progressive Enhancement

**Decision**: htmx 2.x (vendored JS, no build step)

**Rationale**: Same rationale as prior research — single script tag,
pairs naturally with server-rendered Go templates. `hx-post`,
`hx-swap`, `hx-trigger` handle note list updates and tag sidebar
without custom JavaScript.

**Alternatives considered**:
- Alpine.js: Better for component-level state; doesn't simplify
  server round-trips which is the primary need.
- Vanilla fetch() calls: More boilerplate per interaction.
