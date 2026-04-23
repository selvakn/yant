# Implementation Plan: Share Notes with Specific Users

**Branch**: `016-note-sharing` | **Date**: 2026-04-22 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/016-note-sharing/spec.md`

## Summary

Add a second access model alongside ownership: shared access. A `note_shares` table records which users have read or edit permission on another user's note. Note-lookup and list-query code is extended to support a *viewer* who may either own a note or have a share grant. Disk storage stays owner-scoped (all edits write to the owner's file). Git commits on collaborator edits are authored as the collaborator (per-commit author override) so version history attributes correctly.

## Technical Context

**Language/Version**: Go 1.25+ (backend), vanilla JS + htmx (frontend)  
**Primary Dependencies**: chi/v5 (routing), goldmark + GFM extension, scs/v2 (sessions), modernc.org/sqlite  
**Storage**: Markdown files (owner-scoped, unchanged), SQLite `note_shares` table  
**Testing**: Go `testing`, `httptest`  
**Target Platform**: Linux server (Docker/distroless)  
**Project Type**: Web application  
**Performance Goals**: Shared notes appear in recipient's view on next page load; revocation effective in <1s  
**Constraints**: Disk layout must not change; owner-only destructive actions enforced at API layer; git attribution must use collaborator's username  
**Scale/Scope**: Personal use with occasional collaborators

## Constitution Check

- [x] **I. Markdown-first storage** — Note bodies stay in plain markdown at the owner's path. Share grants are metadata.
- [x] **II. Simplicity** — One new table. No new dependencies.
- [x] **III. Monorepo** — All changes within `backend/` and `frontend/`.
- [x] **IV. Integration testing** — Plan includes cross-user authorization tests, wiki-link leakage tests, attribution tests.
- [x] **V. Simple web UI** — Share dialog is a small htmx modal; shared-note list reuses the notes list template.
- [x] **VI. Commit & test discipline** — Incremental slices; tests green before every commit.

## Project Structure

### Documentation (this feature)

```text
specs/016-note-sharing/
├── plan.md
├── spec.md
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/
    └── api.md
```

### Source Code

```text
backend/
├── cmd/server/main.go               # Modified: /notes/{slug}/share routes; /shared routes
├── internal/
│   ├── handlers/
│   │   ├── shares.go                # New: ShareCreatePUT, ShareDeletePUT, ShareListGET, SharedNotesListGET, SharedNoteReaderGET, SharedNoteEditorGET, SharedNoteUpdate
│   │   ├── notes.go                 # Modified: reader passes IsShared/OwnerName to template; pass Collaborators for owner
│   │   ├── tags.go                  # Modified: sidebar data includes SharedCount
│   │   └── handlers_test.go         # Modified: tests
│   ├── models/
│   │   ├── models.go                # Modified: note_shares schema; UpdateNoteByID helper
│   │   ├── shares.go                # New: GrantShare, RevokeShare, ListSharesForNote, ListSharedNotesForUser, CountSharedNotesForUser, GetShareByViewerAndNote, GetNoteForViewer, ResolveWikiLinksForViewer
│   │   └── shares_test.go           # New: tests
│   └── versioning/
│       └── git.go                   # Modified: CommitFileAs(notesDir, relPath, message, name, email)

frontend/
├── templates/
│   ├── notes/
│   │   ├── reader.html              # Modified: Share button, share dialog, shared-by banner
│   │   ├── editor.html              # Modified: "Shared by X" banner for editors; hide owner-only actions
│   │   └── list.html                # Unchanged (only owned notes here)
│   ├── shared/
│   │   ├── list.html                # New: recipient's shared-with-me list
│   │   ├── reader.html              # New: shared-note reader (uses base.html)
│   │   └── editor.html              # New: shared-note editor (for edit-permission collaborators)
│   └── tags/
│       └── sidebar.html             # Modified: "Shared with me" link
├── static/
│   ├── css/app.css                  # Modified: shared-badge, share dialog styles
│   └── js/app.js                    # Modified: share dialog handlers
```

**Structure Decision**: Separate `/shared/{owner-username}/{slug}` URL namespace for notes shared WITH the viewer. This keeps `/notes/*` strictly owner-scoped (simplest authorization model) and prevents slug collisions between Alice's "home" and Bob's "home".

## Implementation Slices

### Slice 1: Schema + Share CRUD (foundational)

Add `note_shares(note_id, user_id, permission, granted_at, granted_by)` in `InitSchema`. Model functions in `models/shares.go`: `GrantShare`, `RevokeShare`, `GetShareByViewerAndNote`, `ListSharesForNote`, `ListSharedNotesForUser`, `CountSharedNotesForUser`.

### Slice 2: Viewer-aware access helper

`GetNoteForViewer(db, viewerID, ownerUsernameOrID, slug) -> (note, role, error)` where role ∈ {owner, editor, reader}. Returns an error if no access. Used by shared-note handlers.

### Slice 3: Share grant/revoke endpoints (US1, US3)

- `PUT /notes/{slug}/share` — body `{"username":..., "permission":"read"|"edit"}` — owner only; upserts share
- `DELETE /notes/{slug}/share/{username}` — owner only; revokes
- `GET /notes/{slug}/shares` — owner only; returns JSON list for the dialog

### Slice 4: Shared-notes list view (US2)

`GET /shared` renders a list of notes shared WITH the viewer, grouped per owner. Sidebar gets a "Shared with me" link with a count badge.

### Slice 5: Shared-note reader + editor (US1, US2, US4)

- `GET /shared/{owner}/{slug}` — requires viewer has any share on that note; renders reader with "Shared by owner" banner. No Archive/Delete/Share controls.
- `GET /shared/{owner}/{slug}/edit` — requires edit permission; renders editor.
- `POST /shared/{owner}/{slug}` — update dispatcher; writes to owner's file via `storage.WriteNote(h.notesDir, ownerID, slug, body)`. Uses `versioning.CommitFileAs(..., viewerUsername, email)` for attribution.

### Slice 6: Share dialog UI (US1, US3)

Reader (owner view): "Share" button in the topbar opens a dialog. Dialog shows a form (username + permission select) and a list of current collaborators with per-row permission dropdown + Revoke. Uses htmx for partial updates.

### Slice 7: Wiki-link safety for shared notes

`ResolveWikiLinksForViewer(db, viewerID, ownerID, body)`:
- If the `[[Title]]` target is another note of the owner also shared with the viewer → link to `/shared/{owner}/{slug}`
- Otherwise → plain text (no hyperlink, no leakage)

### Slice 8: Git attribution (FR-019)

Add `versioning.CommitFileAs(notesDir, relPath, message, authorName, authorEmail)` that uses `git -c user.name=... -c user.email=... commit ...` for a single commit. Update the shared-note update handler to pass the viewer's username/email. Owner edits continue to use the repo default.

## Out of Scope (deferred)

- Including shared-note todos in the `/todos` aggregator (only shown in the note's reader; aggregator stays personal-todos-only to match v1 assumption)
- Search across shared notes (v1: search is per-user; shared notes are accessed by list navigation)
- Notifications when a note is shared
- "Leave share" from the recipient's side

## Complexity Tracking

> No constitution violations. No complexity justification needed.
