# Research: Note Export as ZIP

**Feature**: 024-note-export-zip  
**Date**: 2026-05-08

## Decision: ZIP Creation Library

- **Decision**: Use Go standard library `archive/zip`
- **Rationale**: Already available, no new dependency, supports streaming to `http.ResponseWriter`
- **Alternatives considered**: `klauspost/compress/zip` (faster but unnecessary here), `mholt/archiver` (overkill)

## Decision: Drawing Discovery

- **Decision**: Use `storage.ListDrawingFiles(root, userID, slug)` which returns both legacy and new-format drawings
- **Rationale**: Already handles both file formats; returns `DrawingFile{DrawingID, Type, IsLegacy}`
- **For new-format**: `ReadDrawingByID(root, userID, slug, drawingID, dt)` + `ReadDrawingSVG(root, userID, slug, drawingID)`
- **For legacy**: `ReadDrawing(root, userID, slug)` — SVG path does not exist for legacy (no drawingID)

## Decision: File Naming Inside ZIP

- **Decision**: `note.md`, `sketch-{n}.svg`, `sketch-{n}.{tool}.json` (1-indexed)
- **Rationale**: Matches spec FR-007 (clearly named), is unambiguous, avoids special characters
- **Alternatives considered**: Use drawing display name — rejected (may contain spaces/special chars, requires sanitisation)

## Decision: SVG Missing Handling

- **Decision**: Omit SVG silently when not found (per spec edge case: "omitting only the affected sketch")
- **Rationale**: SVG is a rendered preview that may not exist if sketch was never opened in editor; source JSON is more important for re-editing
- **Implication**: Source JSON is always included when it exists; SVG included only when present

## Decision: ZIP Filename Sanitisation

- **Decision**: Replace `/ \ : ? * | < > "` with `-`, collapse repeated dashes, trim leading/trailing dashes; fallback to `untitled-note.zip`
- **Rationale**: Covers all chars invalid on Windows + Unix + URL context; simple and predictable

## Decision: Handler Location

- **Decision**: New file `backend/internal/handlers/export.go`
- **Rationale**: Keeps handler clean and discoverable; no new package needed

## Decision: Streaming vs Buffer

- **Decision**: Write ZIP into `bytes.Buffer` then write to response
- **Rationale**: Allows setting `Content-Length` header before streaming; ZIP must be finalised before size is known. Notes with ≤10 sketches are small; buffering is safe.

## Architecture Summary

```
GET /notes/{slug}/export
  → NoteExportZIP handler
  → Verify note ownership (existing pattern)
  → ReadNote markdown
  → ListDrawingFiles → for each file, ReadDrawingByID + ReadDrawingSVG
  → Build zip in bytes.Buffer
  → Set Content-Disposition: attachment; filename="<sanitized>.zip"
  → Write buffer to response
```

No new DB tables. No new models. No new storage functions.
