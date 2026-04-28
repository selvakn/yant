# Data Model: Excalidraw Sketch Choice

## File storage (source of truth)

Drawings are stored as JSON files alongside note markdown. The file extension indicates the drawing tool:

```
notes/<user_id>/<slug>.tldraw.json       # tldraw drawing (existing)
notes/<user_id>/<slug>.excalidraw.json   # excalidraw drawing (new)
```

Only one drawing file can exist per note at any time.

### tldraw content (unchanged)

```json
{
  "document": {
    "store": { "..." : "..." },
    "schema": { "schemaVersion": 2 }
  }
}
```

Output of `getSnapshot(editor.store)` from tldraw SDK.

### Excalidraw content

```json
{
  "type": "excalidraw",
  "version": 2,
  "source": "yant",
  "elements": [ "..." ],
  "appState": { "..." : "..." },
  "files": {}
}
```

Output of `serializeAsJSON({ elements, appState, files })` from Excalidraw SDK.

## Tool type detection

The backend determines the drawing tool by checking which file exists:

1. Check for `<slug>.excalidraw.json` — if present, tool is `excalidraw`.
2. Check for `<slug>.tldraw.json` — if present, tool is `tldraw`.
3. Neither exists → no drawing.

For version history, the git file path at a given commit determines which tool was used at that version.

## SQLite (derived/index)

No new tables or columns required. Drawing existence and tool type are determined by file presence, consistent with the existing approach.

## API contracts

### GET /notes/{slug}/drawing

Returns the drawing data with a `type` field indicating the tool, or 404 if no drawing exists.

**Response** (200 — tldraw):
```json
{
  "type": "tldraw",
  "document": { "..." : "..." }
}
```

**Response** (200 — excalidraw):
```json
{
  "type": "excalidraw",
  "data": { "..." : "..." }
}
```

**Response** (404):
```json
{
  "error": "no drawing"
}
```

### PUT /notes/{slug}/drawing?type={tldraw|excalidraw}

Create or update a drawing. The `type` query parameter specifies the tool. Defaults to `tldraw` if omitted (backward compatibility).

If a drawing of the *other* tool type already exists, the request is rejected with 409 Conflict.

**Request body**: Raw drawing JSON (tool-specific format).

**Response** (200/201):
```json
{
  "ok": true
}
```

**Response** (409):
```json
{
  "error": "drawing exists with different tool type"
}
```

### DELETE /notes/{slug}/drawing

Removes whichever drawing file exists (`.tldraw.json` or `.excalidraw.json`).

**Response** (200):
```json
{
  "ok": true
}
```

### GET /notes/{slug}/history/{commit}/drawing

Returns the drawing data at a specific version. The tool type is determined by which file existed at that commit.

**Response** (200):
```json
{
  "type": "tldraw|excalidraw",
  "document|data": { "..." : "..." }
}
```

### GET /p/{token}/drawing

Public note drawing endpoint. Same response format as the authenticated GET endpoint.

## Cascade behavior

When a note is deleted via `DELETE /notes/{slug}`:
- The markdown file is removed.
- The companion `.tldraw.json` file (if present) is removed.
- The companion `.excalidraw.json` file (if present) is removed.

Existing `noteDelete` handler cascade logic is extended to cover both file extensions.
