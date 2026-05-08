# Data Model: Note Export as ZIP

**Feature**: 024-note-export-zip  
**Date**: 2026-05-08

## Entities Involved (existing — no new schema)

### Note
- **Title** `string` — sanitised to form ZIP filename
- **Slug** `string` — used to locate markdown and drawing files
- **Body** `string` — written to `note.md` in ZIP

### DrawingFile (in-memory, from `storage.ListDrawingFiles`)
| Field       | Type          | Notes                                     |
|-------------|---------------|-------------------------------------------|
| DrawingID   | string        | Empty string for legacy drawings          |
| Type        | DrawingType   | `tldraw` or `excalidraw`                  |
| IsLegacy    | bool          | true = old single-drawing format          |

### ExportPackage (ephemeral — built in memory, streamed to browser)
| Entry in ZIP            | Source                                              |
|-------------------------|-----------------------------------------------------|
| `note.md`               | `storage.ReadNote(root, userID, slug)`              |
| `sketch-{n}.svg`        | `storage.ReadDrawingSVG(root, userID, slug, drawingID)` |
| `sketch-{n}.tldraw.json`| `storage.ReadDrawingByID(root, userID, slug, drawingID, tldraw)` |
| `sketch-{n}.excalidraw.json` | `storage.ReadDrawingByID(root, userID, slug, drawingID, excalidraw)` |

For **legacy** drawings (IsLegacy=true): source read via `storage.ReadDrawing(root, userID, slug)`. No SVG stored for legacy — SVG entry omitted.

## ZIP Filename Rules

```
Input: note title
→ Replace chars [/\:?*|<>"] with "-"
→ Collapse consecutive dashes to single dash
→ Trim leading/trailing dashes
→ Lowercase
→ Append ".zip"
→ Fallback: "untitled-note.zip" if result is empty
```

## No new DB tables or storage functions required.
