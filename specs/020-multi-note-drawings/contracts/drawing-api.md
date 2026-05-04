# API Contracts: Multi-Drawing Endpoints

All endpoints require authentication. Responses use `application/json`.

## List Drawings for a Note

```
GET /notes/{slug}/drawings
```

**Response 200**:
```json
{
  "drawings": [
    {
      "drawing_id": "abc12345",
      "display_name": "Architecture Diagram",
      "tool_type": "excalidraw",
      "has_marker": true,
      "created_at": "2026-05-04T12:00:00Z",
      "updated_at": "2026-05-04T13:00:00Z"
    }
  ]
}
```

`has_marker` indicates whether a `![[draw:<id>]]` marker exists in the markdown body.

## Create a New Drawing

```
POST /notes/{slug}/drawings
Content-Type: application/json

{
  "display_name": "Architecture Diagram",
  "tool_type": "excalidraw"
}
```

**Response 201**:
```json
{
  "drawing_id": "abc12345",
  "display_name": "Architecture Diagram",
  "tool_type": "excalidraw",
  "marker": "![[draw:abc12345]]"
}
```

The client inserts the returned `marker` at the cursor position in the editor.

**Response 400**: empty display name or name exceeds 100 chars.

## Get Drawing Content

```
GET /notes/{slug}/drawings/{drawingID}
```

**Response 200** (same wrapped format as existing single-drawing API):
```json
{
  "type": "excalidraw",
  "data": { ... }
}
```

or for tldraw:
```json
{
  "type": "tldraw",
  "document": { ... }
}
```

**Response 404**: drawing not found.

## Save Drawing Content

```
PUT /notes/{slug}/drawings/{drawingID}
Content-Type: application/json

{ ... raw drawing JSON ... }
```

**Response 200**: `{"ok": true}`  
**Response 404**: drawing not found.  
**Response 413**: body exceeds 10 MB limit.

Triggers version control commit for the drawing file.

## Rename Drawing

```
PATCH /notes/{slug}/drawings/{drawingID}
Content-Type: application/json

{
  "display_name": "New Name"
}
```

**Response 200**: `{"ok": true, "display_name": "New Name"}`  
**Response 400**: empty name or exceeds 100 chars.

## Delete Drawing

```
DELETE /notes/{slug}/drawings/{drawingID}
```

**Response 200**: `{"ok": true}`

Deletes the file from disk and removes the `note_drawings` row. The marker in markdown becomes an orphan (rendered as placeholder in reader mode). Triggers version control commit.

## Backward Compatibility

The existing single-drawing endpoints remain functional during the transition:

- `GET /notes/{slug}/drawing` — returns the first (or only) drawing, supporting legacy clients.
- `PUT /notes/{slug}/drawing` — saves to the first drawing; triggers lazy migration if needed.
- `DELETE /notes/{slug}/drawing` — deletes all drawings (legacy behavior).

These will be deprecated after migration is complete.
