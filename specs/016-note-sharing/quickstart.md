# Quickstart: Note Sharing

## Prerequisites

- Go 1.25+, `make deps` completed
- On branch `016-note-sharing`

## Build & Test

```bash
make build && make run
make test
```

## Feature Usage

### Sharing a Note

1. Open your note in the reader view
2. Click **Share** in the topbar — a dialog opens
3. Type the collaborator's username, pick `Read` or `Edit`, click **Grant**
4. The collaborator appears in the dialog's list. Change permission or click **Revoke** anytime

### Accessing Shared Notes

- Visit `/shared` (or click **Shared with me** in the sidebar, which shows a count badge)
- The list groups notes by owner. Click to open
- Edit-permission notes have an **Edit** button; read-permission notes only have the reader view

### Attribution

When a collaborator edits a shared note, the git version history entry shows their username. Open the note's **History** page to see per-edit attribution.

### What Collaborators Cannot Do

- Archive, delete, or publish the note
- Change the note's share configuration (add/remove collaborators)
- Re-share the note with someone else

## Key Files

| Purpose                       | Path                                       |
| ----------------------------- | ------------------------------------------ |
| Share model + queries         | `backend/internal/models/shares.go`        |
| Share handlers                | `backend/internal/handlers/shares.go`      |
| Shared-note reader template   | `frontend/templates/shared/reader.html`    |
| Shared-note editor template   | `frontend/templates/shared/editor.html`    |
| Shared list template          | `frontend/templates/shared/list.html`      |
| Schema                        | `backend/internal/models/models.go`        |
| Routes                        | `backend/cmd/server/main.go`               |
| Git attribution               | `backend/internal/versioning/git.go`       |
