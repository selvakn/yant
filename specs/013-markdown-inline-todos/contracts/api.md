# API Contracts: Inline Markdown Todos

## New Endpoints

### GET /todos

Renders the aggregated pending todos view.

**Query Parameters**:
- `tag` (optional): Filter todos to notes with this tag

**Response**: HTML page (full page with base template)

**Template Data**:
- `Todos` — list of pending todo items with note metadata
- `Tags` — tag list for filtering
- `ActiveTag` — current tag filter (if any)

---

### GET /todos/list

HTMX partial: returns just the todo list items (for tag filter updates).

**Query Parameters**:
- `tag` (optional): Filter todos to notes with this tag

**Response**: HTML partial (todo list items only)

---

### PUT /notes/{slug}/todo

Toggle a todo item's completion status in the note's markdown.

**Request Body** (JSON):
```json
{
  "line": 5,
  "checked": true
}
```

| Field   | Type    | Description                                 |
| ------- | ------- | ------------------------------------------- |
| line    | integer | 1-based line number of the todo in markdown |
| checked | boolean | New completion state                        |

**Success Response** (200):
```json
{
  "ok": true
}
```

**Error Responses**:
- 400: Invalid JSON or line number out of range
- 404: Note not found or line is not a todo line
- 500: File I/O error

---

## Modified Endpoints

### GET /tags (sidebar partial)

**Change**: Response now includes a pending todo count for the sidebar "Todos" link.

**Template Data** (added):
- `TodoCount` — integer count of pending todos across non-archived notes

---

## Existing Endpoints Used

### POST /notes/{slug} (with X-HTTP-Method-Override: PUT)

The existing note update flow triggers todo sync. After saving the markdown body and syncing tags/links, the handler also calls `SyncTodos()` to update the `note_todos` table.

No API contract change — the sync is internal.
