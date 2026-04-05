# Data Model: Diagram Drawing in Notes

## File storage (source of truth)

Drawings are stored as JSON files alongside note markdown:

```
notes/<user_id>/<slug>.tldraw.json
```

**Content**: tldraw document snapshot (shapes, pages, bindings). Example structure:

```json
{
  "document": {
    "store": { ... },
    "schema": { "schemaVersion": 2, ... }
  }
}
```

This is the output of `getSnapshot(editor.store).document` from tldraw SDK.

## SQLite (derived/index)

No new tables required for MVP. The existence of a drawing is determined by file presence. If we later need to query "notes with drawings," we can add a boolean column or join table.

**Future consideration**: `note_drawings` table with `note_id`, `created_at`, `updated_at` for indexing.

## API contracts

### GET /notes/{slug}/drawing

Returns the drawing JSON or 404 if no drawing exists.

**Response** (200):
```json
{
  "document": { ... }
}
```

**Response** (404):
```json
{
  "error": "no drawing"
}
```

### PUT /notes/{slug}/drawing

Create or update the drawing. Body is the tldraw document JSON.

**Request**:
```json
{
  "document": { ... }
}
```

**Response** (200/201):
```json
{
  "ok": true
}
```

### DELETE /notes/{slug}/drawing

Remove the drawing file.

**Response** (200):
```json
{
  "ok": true
}
```

## Cascade behavior

When a note is deleted via `DELETE /notes/{slug}`:
- The markdown file is removed.
- The companion `.tldraw.json` file (if present) is also removed.

Existing `noteDelete` handler in `notes.go` will be extended.
