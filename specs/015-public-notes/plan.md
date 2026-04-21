# Implementation Plan: Public Note Sharing

**Branch**: `015-public-notes` | **Date**: 2026-04-14 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/015-public-notes/spec.md`

## Summary

Add the ability to publish a note at an unguessable public URL (`/p/{token}`) that renders the note without requiring login. Public notes render in read-only mode (no toggleable checkboxes, no edit/archive/delete controls, no sidebar). Wiki-links to private notes render as plain text to avoid information leakage. The public identifier is generated once per note and preserved across public↔private toggles so shared links survive re-publishing.

## Technical Context

**Language/Version**: Go 1.25+ (backend), vanilla JS + htmx (frontend)  
**Primary Dependencies**: chi/v5 (routing), goldmark + GFM extension (markdown), scs/v2 (sessions), modernc.org/sqlite  
**Storage**: Markdown files (source of truth), SQLite `public_notes` table (public ID + published flag)  
**Testing**: Go `testing` package, `httptest` for handler tests  
**Target Platform**: Linux server (Docker/distroless)  
**Project Type**: Web application (Go server + HTML templates + htmx)  
**Performance Goals**: Public page loads in <2s, revocation effective in <1s  
**Constraints**: Unauthenticated access must NOT leak existence of private notes or other owner data  
**Scale/Scope**: Single-user personal app with occasional external share recipients; dozens of concurrent public reads

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [x] **I. Markdown-first storage** — Note body remains in plain markdown. Public share state (flag + public ID) is metadata, stored alongside other note metadata in SQLite. Markdown files remain portable and are the source of truth.
- [x] **II. Simplicity** — No new external dependencies. Public ID is a simple base64-URL-safe random token. Reuses existing goldmark pipeline, storage, and handler patterns.
- [x] **III. Monorepo** — All changes within existing `backend/` and `frontend/` directories.
- [x] **IV. Integration testing** — Plan includes handler tests for public GET endpoint (both published and revoked states), model tests for token generation and lookup, and security tests (URL enumeration, private wiki-link leakage).
- [x] **V. Simple web UI** — Public page is a minimal read-only HTML page. Owner-facing publish toggle uses htmx.
- [x] **VI. Commit & test discipline** — Incremental slices (schema → token generation → public GET route → publish toggle UI → wiki-link scrubbing → public notes list). Each slice is independently testable.

## Project Structure

### Documentation (this feature)

```text
specs/015-public-notes/
├── plan.md
├── spec.md
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/
    └── api.md
```

### Source Code (repository root)

```text
backend/
├── cmd/server/
│   └── main.go                    # Modified: register /p/{token} and /p/{token}/uploads/{filename} as PUBLIC routes
├── internal/
│   ├── handlers/
│   │   ├── public.go              # New: PublicNoteGET, PublicImageServeGET, PublishPUT, UnpublishPUT, PublicNotesListGET
│   │   ├── notes.go               # Modified: pass IsPublic + PublicURL to reader template; unpublish on archive/delete
│   │   └── handlers_test.go       # Modified: tests for new handlers
│   ├── models/
│   │   ├── models.go              # Modified: public_notes schema in InitSchema
│   │   ├── public.go              # New: token generation, publish/unpublish, GetNoteByToken, ListPublic, ResolveWikiLinksPublic
│   │   └── public_test.go         # New: tests
│   └── storage/
│       └── storage.go             # Unchanged

frontend/
├── templates/
│   ├── public/
│   │   ├── note.html              # New: minimal unauthenticated reader (no base.html)
│   │   └── list.html              # New: owner-facing public notes list (uses base.html)
│   ├── notes/
│   │   └── reader.html            # Modified: publish toggle, share URL display + copy
│   └── tags/
│       └── sidebar.html           # Modified: "Public notes" link with count
├── static/
│   ├── css/app.css                # Modified: public page styles, share URL dialog
│   └── js/app.js                  # Modified: copy-to-clipboard helper
```

**Structure Decision**: New `public.go` handler file for all public-facing routes, new `public.go` model file for share-token logic. Public reader template lives under `frontend/templates/public/` with its own minimal layout (no sidebar/nav) to enforce the "no owner UI leakage" requirement at the template level.

## Implementation Slices

### Slice 1: Schema + Token Generation (foundational)

Add `public_notes` table with `(note_id, token, published, published_at)`. The row exists with `published=false` after first unpublish; token persists across toggles. Implement `GenerateShareToken()` (16-byte crypto/rand base64url, ~22 chars).

**Files**: `models/models.go` (schema in `InitSchema`), `models/public.go`, `models/public_test.go`

### Slice 2: Public Reader Route (P1 MVP)

Add `GET /p/{token}` **outside** `RequireLogin` group. Handler: look up note by token, verify `published=true` and note `archived=false`, read markdown, resolve wiki-links with **public-only** variant (private targets → plain text, not hyperlinks), render via goldmark with checkboxes rendered inert (no `data-slug`/`data-line` attributes, no HTMX), rewrite image URLs to `/p/{token}/uploads/...`, render minimal template.

**Files**: `handlers/public.go` (PublicNoteGET), `cmd/server/main.go`, `templates/public/note.html`, `models/public.go` (ResolveWikiLinksPublic)

### Slice 3: Public Image Serving (P1 dependency)

Add `GET /p/{token}/uploads/{filename}`. Verifies token is valid and note is published, and that the image's `note_id` matches the token's note. Serves the file. No session check.

**Files**: `handlers/public.go` (PublicImageServeGET), `cmd/server/main.go`

### Slice 4: Owner-Facing Publish Toggle (P1 owner side)

Add "Publish" / "Unpublish" to the reader page's `...` dropdown. `PUT /notes/{slug}/publish` marks the note published (generating token on first publish). `PUT /notes/{slug}/unpublish` marks it private. Template shows the public URL with a copy button when published.

**Files**: `handlers/public.go` (PublishPUT, UnpublishPUT), `cmd/server/main.go`, `templates/notes/reader.html`, `handlers/notes.go` (NoteReaderGET passes IsPublic + PublicURL), `css/app.css`, `js/app.js` (clipboard)

### Slice 5: Archive/Delete Cascades (P1 safety)

`NotesArchivePUT` also unpublishes. `DELETE` cascades via `ON DELETE CASCADE` on `public_notes.note_id` FK.

**Files**: `handlers/notes.go`

### Slice 6: Public Notes List (P3)

`GET /public` (authenticated, owner-only) lists the owner's currently-published notes with share URLs. Sidebar link.

**Files**: `handlers/public.go` (PublicNotesListGET), `cmd/server/main.go`, `templates/public/list.html`, `templates/tags/sidebar.html`, `handlers/tags.go` (include PublicCount for sidebar badge)

## Complexity Tracking

> No constitution violations. No complexity justification needed.
