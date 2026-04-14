# Route Contracts: Note Version Control

**Feature**: 014-note-version-control
**Date**: 2026-04-14

All routes are **protected** (require authentication, inside the existing `auth.RequireLogin` group).

## Endpoints

### GET /notes/{slug}/history

**Purpose**: Display the version history list for a note.

**Query Parameters**:

| Param | Type | Default | Description |
| ----- | ---- | ------- | ----------- |
| page | int | 1 | Page number (1-indexed) |
| per_page | int | 20 | Versions per page (max 100) |

**Response**: HTML page rendered from `notes/history.html` template.

**Template Data**:

| Field | Type | Description |
| ----- | ---- | ----------- |
| Note | models.Note | The note metadata |
| Versions | []Version | List of versions for the current page |
| Page | int | Current page number |
| PerPage | int | Items per page |
| HasMore | bool | Whether more pages exist |

**Errors**:
- 404 if note not found or does not belong to current user.

---

### GET /notes/{slug}/history/{commit}

**Purpose**: Display the full rendered content of a note at a specific historical version.

**Response**: HTML page rendered from `notes/version.html` template.

**Template Data**:

| Field | Type | Description |
| ----- | ---- | ----------- |
| Note | models.Note | The current note metadata |
| Version | Version | The specific version being viewed |
| BodyHTML | template.HTML | Rendered markdown content at this version |
| HasDrawing | bool | Whether a tldraw drawing existed at this version |
| IsHistorical | bool | Always true (for template to show version banner) |

**Errors**:
- 404 if note or commit not found.
- 400 if commit hash is invalid format.

---

### GET /notes/{slug}/history/{commit}/diff

**Purpose**: Display a unified diff between two versions.

**Query Parameters**:

| Param | Type | Default | Description |
| ----- | ---- | ------- | ----------- |
| against | string | parent commit | Commit hash to diff against (defaults to the version's parent) |

**Response**: HTML page rendered from `notes/diff.html` template.

**Template Data**:

| Field | Type | Description |
| ----- | ---- | ----------- |
| Note | models.Note | The note metadata |
| Diff | DiffResult | Parsed diff with lines, metadata |
| OldVersion | Version | The older version |
| NewVersion | Version | The newer version |

**Errors**:
- 404 if note or commit not found.
- 400 if commit hash is invalid format.

---

### GET /notes/{slug}/history/{commit}/drawing

**Purpose**: Serve the raw tldraw JSON for a drawing at a specific historical version. Used by the frontend tldraw renderer.

**Response**: `application/json` — raw tldraw JSON content at the specified commit.

**Errors**:
- 404 if note, commit, or drawing not found at that version.
- 400 if commit hash is invalid format.

---

### POST /notes/{slug}/history/{commit}/revert

**Purpose**: Revert a note to the content at the specified version.

**Behavior**:
1. Retrieve note content at the specified commit.
2. Write it as the current note file.
3. Commit the change with message `revert: {slug} to {shortHash}`.
4. Update SQLite metadata (tags, links, todos, embeddings) from the reverted content.
5. Redirect to the note reader.

**Response**: 302 redirect to `/notes/{slug}` (or `HX-Redirect` header for htmx).

**Errors**:
- 404 if note or commit not found.
- 400 if commit hash is invalid format.
- 409 if the note content at the specified version is identical to the current content (no-op).
