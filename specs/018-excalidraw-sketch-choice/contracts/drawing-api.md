# Drawing API Contract (Updated for Multi-Tool Support)

## Endpoints

### GET /notes/{slug}/drawing

Retrieve the drawing for a note. Returns tool type alongside drawing data.

| Field   | Value |
|---------|-------|
| Auth    | Session (authenticated user) |
| Method  | GET |
| Path    | `/notes/{slug}/drawing` |
| Success | 200 with `{"type": "tldraw", "document": {...}}` or `{"type": "excalidraw", "data": {...}}` |
| Not found | 404 with `{"error": "no drawing"}` |

### PUT /notes/{slug}/drawing

Create or update a drawing. Tool type specified via query parameter.

| Field   | Value |
|---------|-------|
| Auth    | Session (authenticated user) |
| Method  | PUT |
| Path    | `/notes/{slug}/drawing?type={tldraw\|excalidraw}` |
| Body    | Raw drawing JSON (tool-specific format) |
| Default | `type=tldraw` if query param omitted |
| Success | 200 with `{"ok": true}` |
| Conflict | 409 with `{"error": "drawing exists with different tool type"}` |
| Size limit | 10MB request body |

### DELETE /notes/{slug}/drawing

Remove the drawing file (either tool type).

| Field   | Value |
|---------|-------|
| Auth    | Session (authenticated user) |
| Method  | DELETE |
| Path    | `/notes/{slug}/drawing` |
| Success | 200 with `{"ok": true}` |

### GET /notes/{slug}/history/{commit}/drawing

Retrieve drawing at a specific version. Tool type determined by git history.

| Field   | Value |
|---------|-------|
| Auth    | Session (authenticated user) |
| Method  | GET |
| Path    | `/notes/{slug}/history/{commit}/drawing` |
| Success | 200 with `{"type": "...", "document\|data": {...}}` |
| Not found | 404 |

### GET /p/{token}/drawing

Public note drawing. Same response format as authenticated GET.

| Field   | Value |
|---------|-------|
| Auth    | None (public token) |
| Method  | GET |
| Path    | `/p/{token}/drawing` |
| Success | 200 with type + data |
| Not found | 404 |

## Frontend Island Contract

Both drawing islands expose the same global function signature:

```typescript
window.initTldrawIsland(container, snapshotUrl, saveUrl, options?) => cleanup()
window.initExcalidrawIsland(container, snapshotUrl, saveUrl, options?) => cleanup()
```

**Parameters**:
- `container`: HTMLElement — DOM element to render into
- `snapshotUrl`: string — GET URL to load drawing data
- `saveUrl`: string — PUT URL to save drawing data
- `options.readOnly`: boolean — render in view-only mode
- `options.initialTool`: string — initial tool selection (e.g., `'draw'`, `'hand'`)

**Returns**: cleanup function that unmounts the React root.

## Template Data Contract

Templates receive a `DrawingType` field alongside the existing `HasDrawing` field:

| Field | Type | Values |
|-------|------|--------|
| `HasDrawing` | bool | Whether any drawing exists |
| `DrawingType` | string | `"tldraw"`, `"excalidraw"`, or `""` (no drawing) |
