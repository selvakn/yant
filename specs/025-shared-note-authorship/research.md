# Research: Shared Note Authorship & Indicators

**Feature**: 025-shared-note-authorship  
**Date**: 2026-05-15

## Findings

### Decision: Author storage mechanism
- **Decision**: Read author from git commit metadata (`%an` field in git log format), recorded at save time via `CommitFileAs()`.
- **Rationale**: Git already stores author information per commit. `CommitFileAs()` already exists and is used by `SharedNoteUpdate` (handlers/shares.go:323). No new DB column is needed. Author is captured server-side from the session (cannot be spoofed).
- **Alternatives considered**: Storing `last_editor_username` in the `notes` SQLite table. Rejected — adds a DB column and migration for information git already holds. Would still need a separate mechanism for per-history-entry attribution.

### Decision: Owner note saves currently use anonymous "yant" identity
- **Finding**: `noteUpdate()` and `NotesCreatePOST()` call `versioning.CommitFile()` (handlers/notes.go:271,71), which uses the git repo's default "yant" user. `DrawingByIDPUT()` also uses `CommitFile()` (drawings.go:326).
- **Decision**: Change these three call sites to `CommitFileAs()` with the session username. Same pattern as shared edits.
- **Impact**: Pre-deployment commits will show "yant" (handled as legacy "—" in the UI). Post-deployment commits show the actual owner username.

### Decision: Last-editor on reader page — derive from git, not DB
- **Decision**: Call `versioning.Log(notesDir, relPath, 1, 0)` in the reader handler to get the most recent commit, extract `AuthorName`. Pass as `LastEditor string` to template.
- **Rationale**: One git call per page load is acceptable for a single note view. Avoids a DB schema change. Falls back cleanly if history is empty or AuthorName is "".
- **Alternatives considered**: Caching last editor in `notes.last_editor_username` column. Rejected — adds DB complexity for information accessible from git. Could revisit if reader latency proves unacceptable.

### Decision: Share indicators — outgoing on /notes list, incoming already on /shared list
- **Finding**: `/notes` (NotesListGET) shows only the user's owned notes via `ListNotes()` with no share join. `/shared` (SharedNotesListGET) shows notes shared with the user; template already shows "Shared by OwnerUsername" text.
- **Decision**:
  - `/notes` list: add outgoing share badge ("↑ Shared with N") by joining `note_shares` per query for the current user's notes.
  - `/shared` list: refine existing "Shared by OwnerUsername" into a consistent incoming badge style.
- **Implementation**: New `ListShareCountsForOwner(db, userID)` model function returns `map[int64]int` (noteID → active collaborator count) via a single JOIN query. Passed as `ShareStates` to the list template.

### Decision: Collaborator access to full version history
- **Finding**: No `/shared/{username}/{slug}/history` route exists. The owner's `/notes/{slug}/history` requires ownership (filters by session userID). Collaborators currently have no way to view history.
- **Decision**: Add `GET /shared/{username}/{slug}/history` route and `SharedNoteHistoryGET` handler. Handler uses the owner's userID (from `models.GetNoteForViewer`) to call `versioning.Log()`. Renders the same `shared/history.html` template (new file).
- **Rationale**: The spec (US1) explicitly states "when either user opens the version history". Without this route, P1 user story is only half-satisfied.

### Decision: parseGitLog format extension — backward compatible
- **Finding**: `parseGitLog()` parses `%H|%h|%aI|%s` → splits on `|` with `SplitN(..., 4)`. If we change format to `%H|%h|%aI|%s|%an`, we need to split into 5 fields. Old commits in the git repo have no `%an` field in the commit object — the format string is applied at read time, so ALL commits will return 5 fields immediately after the format string change. No backward compat issue for commits.
- **Caveat**: If AuthorName is the default "yant" (pre-deployment owner edits) or "" (unexpected), the UI shows "—" as the fallback per FR-002.

### Decision: CSS styling for indicators
- **Decision**: Add minimal CSS classes: `.share-badge-out`, `.share-badge-in`, `.last-editor` in the existing `frontend/static/css/main.css`. Small additions; no new stylesheets.
