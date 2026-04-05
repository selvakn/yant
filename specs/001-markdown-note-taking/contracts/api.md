# REST API Contracts: Markdown Note Taking App

**Date**: 2026-04-05
**Branch**: `001-markdown-note-taking`

All endpoints serve Go `html/template` pages for browser navigation.
htmx requests receive HTML partials (detected via `HX-Request` header).
Endpoints also support `Accept: application/json` for programmatic
access. Error responses use standard HTTP status codes.

## Authentication

### GET /login

Render username-only login form.

**Response (200)**: Login page HTML.

---

### POST /login

Log in with username. Auto-creates user if username is unrecognized.

**Request**: `application/x-www-form-urlencoded`

| Field    | Type   | Required | Description        |
|----------|--------|----------|--------------------|
| username | string | yes      | Username to log in |

**Response (302)**: Redirect to `GET /notes` on success.
**Response (400)**: Missing or empty username.

---

### POST /logout

Clear session.

**Response (302)**: Redirect to `GET /login`.

---

## Notes

### GET /notes

List all notes for the logged-in user. Supports tag filtering.

**Query params**:

| Param | Type   | Description                        |
|-------|--------|------------------------------------|
| tag   | string | Filter by tag name (optional)      |

**Response (200)**: Note list page (title, tags, updated_at per row).
JSON: `[{"id":1,"slug":"my-note","title":"My Note","tags":["work"],"created_at":"...","updated_at":"..."}]`

---

### POST /notes

Create a new note.

**Request**: `application/x-www-form-urlencoded`

| Field | Type   | Required | Description                            |
|-------|--------|----------|----------------------------------------|
| title | string | no       | Note title (defaults to "Untitled Note") |
| body  | string | no       | Markdown body (defaults to empty)      |

**Response (302)**: Redirect to `GET /notes/{slug}/edit`.
JSON: `201 {"id":1,"slug":"untitled-note","title":"Untitled Note","created_at":"...","updated_at":"..."}`

---

### GET /notes/{slug}

View note in reader mode (goldmark-rendered Markdown).

**Response (200)**: Reader page with rendered HTML body, title,
created_at, updated_at, tags.
JSON: `{"id":1,"slug":"...","title":"...","body":"raw md","body_html":"<p>...","tags":[],"created_at":"...","updated_at":"..."}`

---

### GET /notes/{slug}/edit

Edit note in editor mode (EasyMDE).

**Response (200)**: Editor page with raw Markdown body in textarea,
title input, timestamps, link to reader mode.

---

### PUT /notes/{slug}

Update title and/or body of an existing note.

**Request**: `application/x-www-form-urlencoded` (POST with
`X-HTTP-Method-Override: PUT` header, sent by htmx) or JSON.

| Field | Type   | Required | Description           |
|-------|--------|----------|-----------------------|
| title | string | no       | Updated title         |
| body  | string | no       | Updated Markdown body |

**Response (200)**: Updated editor partial (htmx swap) or full page.
JSON: `{"id":1,"slug":"...","title":"...","tags":[],"updated_at":"..."}`

**Note**: Tags re-parsed from body on save. Slug regenerated from
title if title changes; response includes new slug. If slug changes,
JSON response includes `"redirect":"/notes/{new-slug}/edit"`.

---

### DELETE /notes/{slug}

Delete a note and clean up associated images.

**Request**: POST with `X-HTTP-Method-Override: DELETE` (htmx), or
HTTP DELETE.

**Response (200)**: Empty body with `HX-Redirect: /notes` header
(htmx removes row from list).
JSON: `204 No Content`.

---

## Images

### POST /notes/{slug}/images

Upload an image (called by EasyMDE's `imageUploadFunction`).

**Request**: `multipart/form-data`

| Field | Type | Required | Description                    |
|-------|------|----------|--------------------------------|
| image | file | yes      | Image (PNG / JPEG / GIF / WebP)|

**Response (200)**: `application/json`
```json
{"url": "/uploads/{username}/{filename}"}
```

**Response (400)**: Invalid or missing file / unsupported type.
**Response (413)**: File exceeds 10 MB.

---

### GET /uploads/{username}/{filename}

Serve an uploaded image. Access is restricted: session user MUST
match `{username}`.

**Response (200)**: Image bytes with correct `Content-Type`.
**Response (403)**: Session user does not match `{username}`.
**Response (404)**: File not found.

---

## Tags

### GET /tags

List all tags for the logged-in user with note counts.

**Response (200)**: Tag sidebar partial (htmx swap) or JSON.
JSON: `[{"name":"work","count":3},{"name":"ideas","count":1}]`

---

## Error Responses

| Status | Meaning                                  |
|--------|------------------------------------------|
| 400    | Bad request (missing/invalid field)      |
| 401    | Not authenticated → redirect to /login   |
| 403    | Forbidden (accessing another user's data)|
| 404    | Resource not found                       |
| 413    | Payload too large (image upload)         |
| 500    | Internal server error                    |
