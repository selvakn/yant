# Quickstart: Note Version Control

**Feature**: 014-note-version-control
**Date**: 2026-04-14

## Prerequisites

- Go 1.25+
- Git 2.x+ installed and on `$PATH`
- Existing YANT development environment (`make build` passes)

## Implementation Order

### 1. Backend: Versioning Package

Create `backend/internal/versioning/` with git operations:

```text
backend/internal/versioning/
├── git.go           # Core git operations (init, commit, log, show, diff)
└── git_test.go      # Unit tests using t.TempDir()
```

Key functions:
- `Init(notesDir)` — git init + seed existing files
- `CommitFile(notesDir, relPath, message)` — add + commit single file
- `CommitDelete(notesDir, relPath, message)` — rm + commit
- `Log(notesDir, relPath, limit, offset)` — history with --follow
- `Show(notesDir, relPath, commit)` — file content at version
- `Diff(notesDir, relPath, commit1, commit2)` — unified diff
- `ParseDiff(raw)` — parse unified diff into DiffLine structs

**Test first**, then integrate.

### 2. Backend: Integrate with Existing Handlers

Modify existing note handlers to commit on save:
- `NotesCreatePOST` → call `CommitFile` after `storage.WriteNote`
- `noteUpdate` → call `CommitFile` after `storage.WriteNote`
- `noteDelete` → call `CommitDelete` after `storage.DeleteNoteFile`
- Drawing PUT/DELETE → call `CommitFile`/`CommitDelete` after drawing storage ops

### 3. Backend: History Handlers

Add new handler methods to `handlers.Handler`:
- `NoteHistoryGET` — version list
- `NoteVersionGET` — view at version
- `NoteVersionDiffGET` — diff view
- `NoteVersionDrawingGET` — drawing JSON at version
- `NoteVersionRevertPOST` — revert action

Register routes in `main.go` inside the protected group.

### 4. Frontend: Templates

Create three new templates:
- `frontend/templates/notes/history.html` — version list with dates, change stats
- `frontend/templates/notes/version.html` — rendered note with version banner
- `frontend/templates/notes/diff.html` — unified diff with color-coded lines

Add diff CSS to `frontend/static/css/app.css`.
Add history link to `reader.html` top bar.

### 5. Frontend: Drawing Diff

Extend `diff.html` to include side-by-side tldraw canvases when `HasDrawingChange` is true. Each canvas loads drawing JSON from the version-specific endpoint.

### 6. Docker

Add `git` package to the Dockerfile runtime stage.

## Verification

```bash
make test        # All existing + new tests pass
make coverage    # ≥90% line coverage on internal/...
make build       # Binary compiles
make docker-build  # Docker image builds with git
```

## Key Design Decisions

- **No new Go dependencies**: Uses `os/exec` to call the git binary.
- **No new database tables**: Git is the sole storage for version data.
- **Atomic commits**: One git commit per save operation.
- **Rename tracking**: `git log --follow` preserves history across slug changes.
- **Non-destructive reverts**: Revert writes old content as a new commit.
