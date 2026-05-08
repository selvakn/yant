# Contract: Note Export Endpoint

## GET /notes/{slug}/export

**Purpose**: Download a ZIP archive containing the note's markdown and all sketch files.

### Authentication
- Requires active session (same as all `/notes/` routes)
- 401 redirect to login if unauthenticated

### Path Parameters
| Parameter | Type   | Description         |
|-----------|--------|---------------------|
| slug      | string | Note slug identifier|

### Success Response — 200 OK
| Header                | Value                                                    |
|-----------------------|----------------------------------------------------------|
| Content-Type          | `application/zip`                                        |
| Content-Disposition   | `attachment; filename="<sanitized-title>.zip"`           |
| Content-Length        | byte count of ZIP body                                   |

**ZIP contents** (for a note with 2 sketches):
```
note.md
sketch-1.svg
sketch-1.tldraw.json
sketch-2.svg
sketch-2.excalidraw.json
```

- `note.md` always present
- Sketch entries appear only when the corresponding file exists on disk
- SVG omitted silently when not stored (sketch never rendered)
- Index (1, 2, ...) is determined by the order `ListDrawingFiles` returns them

### Error Responses
| Status | Condition                              |
|--------|----------------------------------------|
| 404    | Note not found or not owned by user    |
| 500    | Unexpected read/write error            |

### Behaviour
- User remains on current page (browser handles file download natively)
- No page navigation occurs
- For notes with no sketches: ZIP contains only `note.md`
