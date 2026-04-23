# API Contracts: Note Sharing

All endpoints require authentication.

## Share Management (owner)

### PUT /notes/{slug}/share

Create or update a share on an owned note.

**Body** (JSON):
```json
{
  "username": "bob",
  "permission": "read" | "edit"
}
```

**Responses**:
- `200` — `{ "ok": true, "username": "bob", "permission": "edit" }`
- `400` — invalid permission / missing fields
- `400` — attempting to share with self
- `404` — note not found OR note not owned by viewer OR target username not found

---

### DELETE /notes/{slug}/share/{username}

Revoke the share.

**Responses**:
- `200` — `{ "ok": true }`
- `404` — note not found, not owned, or no existing share

---

### GET /notes/{slug}/shares

Return the list of collaborators on an owned note.

**Response** (JSON):
```json
{
  "collaborators": [
    { "username": "bob",   "permission": "edit", "granted_at": "2026-04-22T..." },
    { "username": "carol", "permission": "read", "granted_at": "..." }
  ]
}
```

- `404` — note not found or not owned

---

## Shared-with-me (recipient)

### GET /shared

Owner's list view of notes shared WITH the viewer. Renders HTML (full page).

Data: notes grouped by owner, with title, permission badge, owner username, updated_at, tags.

---

### GET /shared/{username}/{slug}

Shared-note reader. Requires viewer has any share grant on this note.

**Responses**:
- `200` — rendered reader HTML with "Shared by {username}" banner and a Permission indicator. No Archive/Delete/Share/Publish controls.
- `404` — no grant, or note archived/deleted

---

### GET /shared/{username}/{slug}/edit

Shared-note editor. Requires edit permission.

**Responses**:
- `200` — editor HTML with "Shared by {username}" banner. No Archive/Delete/Share controls.
- `403` — read-only grant
- `404` — no grant

---

### POST /shared/{username}/{slug}

Update dispatcher (mirror of `/notes/{slug}`).

**Form fields**: `title`, `body`  
**Behavior**: writes to the owner's on-disk file; syncs tags/links/todos/embedding; commits to git with the viewer's username as author.

- `200` — success
- `403` — no edit permission
- `404` — no grant

---

## Modified Endpoints

### GET /notes/{slug}

**Change**: The owner's reader view passes `Collaborators []NoteCollaborator` to the template for rendering the Share dialog's current-collaborators list.

### GET /tags (HX-Request → sidebar partial)

**Change**: Response includes `SharedCount` — number of notes shared with the viewer (for the sidebar nav badge).

---

## Route Registration Summary

All routes protected by `auth.RequireLogin`:

```
PUT    /notes/{slug}/share
DELETE /notes/{slug}/share/{username}
GET    /notes/{slug}/shares

GET    /shared
GET    /shared/{username}/{slug}
GET    /shared/{username}/{slug}/edit
POST   /shared/{username}/{slug}
```
