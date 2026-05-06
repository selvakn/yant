# Phase 1 Data Model — Inline Drawing Previews in Edit Mode

This feature introduces **no schema changes**. All persistent entities (notes, drawings, drawing markers, `note_drawings` table) are unchanged from feature 020 (multi-note drawings).

The feature does introduce one **UI-only entity** that lives entirely in the browser:

## Drawing Preview (UI-only)

A read-only visual representation of a drawing inside the editor view. Exactly one Drawing Preview per drawing per note.

### Fields (in-DOM state)

| Field | Type | Source | Notes |
|---|---|---|---|
| `drawingID` | string | `note_drawings.drawing_id` (via `GET /notes/{slug}/drawings`) | Stable identifier; unchanged by rename |
| `displayName` | string | `note_drawings.display_name` | Mutable via inline rename |
| `toolType` | `"excalidraw"` \| `"tldraw"` | `note_drawings.tool_type` | Determines which init bundle and which read-only icon |
| `state` | enum | derived | `preview` \| `editing` \| `placeholder-empty` \| `placeholder-failed` |
| `svgURL` | string | `/notes/{slug}/drawings/{drawingID}/svg` | Refetched after canvas close |

### State transitions

```text
                           click card
       ┌── preview ─────────────────────────► editing ──┐
       │  ▲                                              │
       │  │ refetch SVG                                  │
       │  └────────── click Done / click another card ──┘
       │
       │ initial fetch failed
       ▼
   placeholder-failed
       │ click card
       ▼
       editing
       │ Done after first save
       ▼
   preview
```

Empty placeholder is functionally equivalent to a preview state with no SVG. Treat both via the same DOM container; the only difference is the body content.

### Relationships

- One Drawing Preview ↔ one Drawing (FK by `drawing_id`).
- Drawing Previews are siblings inside a single editor `.drawing-section-seamless` container, ordered by marker order in the markdown body (per FR-005).

### Validation rules

- A Drawing Preview is rendered iff the drawing's `drawing_id` appears in the response of `GET /notes/{slug}/drawings`.
- Display name input on rename: 1–100 characters (mirrors backend validation already enforced by `DrawingByIDRenamePATCH`).
- At most one Drawing Preview can be in `editing` state at any time (FR-007).

### Lifecycle

- Created on initial editor page load and on add-drawing success.
- Updated on rename success (display name only) and on canvas close (SVG refetched).
- Removed on delete success.

There is no persistence — the Drawing Preview's state lives only in the DOM. Refreshing the page reconstructs it from the API.
