# Contract: Shared Note History Route

**Route**: `GET /shared/{username}/{slug}/history`  
**Handler**: `SharedNoteHistoryGET`

## Access Control

- Requires authenticated session.
- Viewer must have any active share grant (read or edit) for the note, OR be the owner.
- Returns 404 if note not found or viewer has no access.

## Query Parameters

| Parameter | Type | Default | Description |
|---|---|---|---|
| `page` | int | 1 | Page number (1-based) |
| `per_page` | int | 20 | Entries per page (max 100) |

## Template Data

```
Note          *models.Note        the shared note (owner's note)
OwnerUsername string              the note owner's username
Versions      []versioning.Version version list, newest first
Page          int                 current page number
PrevPage      int                 page - 1 (for pagination link)
NextPage      int                 page + 1
HasMore       bool                true if more entries exist
PerPage       int                 entries per page
```

## Behavior

- Uses the note owner's userID (not the viewer's) to construct the git relative path.
- Calls `versioning.Log(notesDir, relPath, limit, offset)` — same as the owner's history handler.
- Renders `shared/history.html`.
- No revert action is available from the shared history view (revert is an owner-only action).
