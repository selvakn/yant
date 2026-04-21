# API Contracts: Public Note Sharing

## Public (Unauthenticated) Endpoints

### GET /p/{token}

Renders a publicly shared note without requiring authentication.

**Path Parameters**:
- `token`: The public share token (base64url, ~22 chars)

**Response** (200):
- Content-Type: `text/html`
- Body: Minimal HTML document with the note title, rendered markdown body, and (if applicable) drawing in read-only mode.
- Includes `<meta name="robots" content="noindex,nofollow">` and `<meta property="og:title">` / `og:description` for link previews.

**Response** (404):
- Plain "Note not found" page.
- Returned when the token is unknown, the note is unpublished (`published=false`), archived, or deleted.
- The error message does NOT distinguish between these cases.

**No owner-only UI**: no sidebar, no search, no edit/archive/delete buttons, no todo-toggle UI.

---

### GET /p/{token}/uploads/{filename}

Serves an image embedded in a public note.

**Path Parameters**:
- `token`: The public share token
- `filename`: The uploaded image filename

**Response** (200):
- Content-Type: image mime type (png, jpeg, gif, webp)
- Body: image bytes

**Response** (404):
- Returned if the token is unknown, the note is unpublished/archived, or the image does not belong to the note associated with the token.

---

### GET /p/{token}/drawing

Serves the tldraw JSON for a public note's drawing (used by the tldraw reader in the public page).

**Path Parameters**:
- `token`: The public share token

**Response** (200):
- Content-Type: `application/json`
- Body: tldraw snapshot JSON

**Response** (404):
- Returned if the token is unknown, the note is unpublished, or no drawing exists.

---

## Authenticated (Owner) Endpoints

### PUT /notes/{slug}/publish

Publishes a note. Generates a token on first publish; reuses the existing token on subsequent calls.

**Path Parameters**:
- `slug`: The note's private slug

**Response** (200, JSON):
```json
{
  "ok": true,
  "token": "abc123...",
  "public_url": "/p/abc123..."
}
```

**Response** (404):
- Returned if the note doesn't exist or doesn't belong to the authenticated user.

---

### PUT /notes/{slug}/unpublish

Unpublishes a note (token is preserved for future re-publish).

**Path Parameters**:
- `slug`: The note's private slug

**Response** (200, JSON):
```json
{
  "ok": true
}
```

**Response** (404):
- Returned if the note doesn't exist or doesn't belong to the authenticated user.

---

### GET /public

Owner's view of all currently-published notes.

**Response** (200, HTML):
- Full-page HTML with the list of published notes, their titles, and copyable share URLs.

---

## Modified Endpoints

### PUT /notes/{slug}/archive

**Change**: After archiving, also sets `public_notes.published = false` so the public URL stops working immediately.

---

### DELETE /notes/{slug} (via X-HTTP-Method-Override POST)

**Change**: `ON DELETE CASCADE` on `public_notes.note_id` automatically removes the share row. Public URL returns 404.

---

### GET /notes/{slug} (NoteReaderGET — owner's reader view)

**Change**: Response template receives additional data:
- `IsPublic` (bool): whether the note is currently published
- `PublicURL` (string): the public URL if `IsPublic`, empty otherwise

Used by the "Publish / Unpublish" menu item and the share URL display.

---

### GET /tags (sidebar partial)

**Change**: Sidebar data includes `PublicCount` (number of currently-published notes) for the "Public notes" sidebar link badge.
