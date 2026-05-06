# Phase 0 Research — Inline Drawing Previews in Edit Mode

## R1: Inline-vs-stacked placement decision

**Decision**: Render previews stacked in marker order at the end of the editor area (the existing `.drawing-section-seamless` location). Do not implement CodeMirror line widgets for inline rendering in this iteration.

**Rationale**:
- Spec FR-004 says "when this is technically feasible". Spec FR-005 explicitly accepts the stacked-at-end fallback.
- Constitution Principle II (Simplicity, YAGNI): the simpler implementation that satisfies every FR should win.
- The user's clarification (`if possible, in line, if not at the end (like current)`) makes the fallback first-class.
- The actual user pain point is "I have to click each one to see it" — that pain is fully solved by previews-by-default regardless of placement.
- CodeMirror line widgets are technically feasible but add lifecycle complexity (re-attach on marker move, on content change, on add/delete) that is not justified for a UX delta this small.

**Alternatives considered**:
- *CodeMirror line widgets for true inline rendering*: rejected — meaningful added complexity, no FR strictly requires it, fallback explicitly accepted by user.
- *Side-by-side editor + preview pane*: rejected — major UI rework, not a "small additive change".
- *Render previews above the editor*: rejected — breaks the existing layout convention; placement at end matches today's location and reader-mode flow visually.

## R2: SVG fetching and refresh strategy

**Decision**: Fetch each drawing's SVG via `GET /notes/{slug}/drawings/{drawingID}/svg` on initial page load, in parallel for all drawings in the note. After a canvas closes, re-fetch only the just-edited drawing's SVG and replace it in its preview card. Treat 404/empty responses as the "empty drawing" state and show a placeholder.

**Rationale**:
- This is exactly the existing reader-mode pattern (`reader.html` lines ~127–162). Reusing it gives us SC-007 parity (identical visual size) for free.
- Per-drawing parallel fetches are bounded (≤10 drawings per note per SC-001), each returning a small SVG document — fast even on slow connections.
- The drawing-edit bundle already PUTs the SVG to `{saveUrl}/svg` whenever the user saves (`excalidraw-island.tsx:154`, `tldraw-island.tsx:203`). So after a Close/Done, the latest SVG is already on the server — we only need to refetch.

**Alternatives considered**:
- *One batch endpoint that returns all SVGs in a single response*: rejected — would require a new backend endpoint just for this UI optimization. Per-drawing fetches in parallel are already fast enough at the spec's 10-drawing target.
- *Embed the latest SVG in the page render*: rejected — would inflate page HTML size and complicate edit-mode template; also bypasses the cache benefits of separate SVG GETs.
- *Push-based refresh after save (websocket / SSE)*: rejected — out of scope; refetch on Close is sufficient for the spec.

## R3: Empty / missing / failed-to-load placeholder

**Decision**: Three placeholder states inside a preview card, all sharing one `.drawing-preview-placeholder` style:
1. **Empty** (drawing exists but no SVG yet — newly created): "Empty drawing — click to draw". Card remains clickable; clicking opens the canvas.
2. **Missing** (drawing not in `note_drawings` for the marker — already handled in reader mode): we don't expect this in edit mode (the editor only renders drawings that the API returns), but if a drawing is deleted while open we hide its card on next refresh.
3. **Failed-to-load** (SVG GET errored): "Preview unavailable — click to edit". Clicking still opens the canvas (the JSON load is independent of the SVG).

**Rationale**: Mirrors today's reader-mode messaging ("Preview unavailable — edit to generate") but adapts the action language to edit mode. Single style avoids three separate visual treatments.

**Alternatives considered**:
- *Retry button on failure*: rejected — adds complexity; clicking the card to enter edit mode (then save) regenerates the SVG, which is the same recovery action.

## R4: Canvas Close/Done control

**Decision**: When the canvas opens, append a small `<button class="drawing-canvas-close">Done</button>` overlay inside the canvas card, positioned top-right above the canvas. Clicking it calls the existing `closeDrawingCanvas()` after triggering a final save flush (the bundles autosave on change, but we want a deterministic "no pending writes" signal). Then the card refetches its SVG and returns to preview state.

**Rationale**:
- FR-007a requires an explicit Close/Done control specifically for the single-drawing-on-note case (no other preview to click).
- Top-right corner is a conventional close-button position and won't conflict with the tool's own toolbars (tldraw/excalidraw use bottom-of-canvas tooling).
- Reusing `closeDrawingCanvas()` keeps the cleanup logic consolidated with the existing implementation.

**Alternatives considered**:
- *Place Close in the card header (next to rename/delete)*: viable but visually inconsistent with editing canvases elsewhere; rejected for ergonomics.
- *Esc key to close*: nice-to-have additive shortcut; can be added but not required by the spec — keep out of scope.

## R5: Hover affordance implementation

**Decision**: CSS-only hover state on `.drawing-preview-card`:
- Resting state: subtle 1px border, no shadow.
- Hover state: stronger border (#2563eb at 50% opacity), small shadow, `cursor: pointer`, and a positioned `::after` pseudo-element with the text "Click to edit" fading in at top-right.
- Edit state (canvas open): solid 2px border in primary color, no hover treatment.

**Rationale**: FR-015 requires a hover affordance. Pure CSS is the simplest approach, no JS state needed. Pseudo-element keeps DOM clean.

**Alternatives considered**:
- *JS-driven hover overlay*: rejected — unnecessary; hover is a CSS primitive.
- *Always-visible "Edit" button*: rejected — explicitly contradicts FR-015 ("resting state shows only the SVG and per-preview header").

## R6: Rename UI

**Decision**: Add a small pencil icon (✏️ or unicode pencil character) button next to the display name in the preview card header. Clicking it makes the display name editable inline (a small `<input>` replaces the `<span>`); Enter saves via `PATCH /notes/{slug}/drawings/{drawingID}` with `{"display_name": "..."}`; Esc cancels. The PATCH endpoint already exists (`DrawingByIDRenamePATCH`).

**Rationale**: FR-010b requires rename controls on each preview. Inline-edit is the standard pattern for renaming a label in-place. The PATCH endpoint exists and is fully tested.

**Alternatives considered**:
- *Modal rename dialog*: rejected — over-engineered for a single text input.
- *Defer rename to a future feature*: rejected — FR-010b is in-scope.

## R7: Reader-mode size parity (SC-007)

**Decision**: No CSS change required. The existing `.drawing-svg-preview svg { max-width: 100%; height: auto; }` rule already gives natural aspect ratio with no height cap. We will reuse the same `.drawing-svg-preview` class inside the new edit-mode preview card, ensuring identical visual sizing.

**Rationale**: Verified by reading `frontend/static/css/app.css:479-481`. Reader mode already meets the user's clarification — natural aspect ratio, no cap. The legacy single-drawing reader uses `.drawing-preview-seamless` (live canvas, not SVG) which has heights — but legacy single-drawing notes are not in scope for this feature (this feature applies to multi-drawing notes per FR-001's reference to "every drawing on a note").

**Alternatives considered**:
- *Remove the height caps from `.drawing-canvas` / `.drawing-preview-seamless`*: rejected — those are live editor canvases, not SVG previews; their height behavior is unrelated to SC-007.

## R8: Scope of FR-012 (which editing surfaces are affected)

**Decision**: This feature applies only to the owner's note editor (`frontend/templates/notes/editor.html`). The shared-note editor (`frontend/templates/shared/editor.html`) does not currently implement any drawing UI (verified — no `drawing-list`, no canvas, no add-drawing form), so there is no per-drawing "Edit" toggle there to replace. Adding drawing support to shared editors is a separate, larger initiative.

**Rationale**: FR-012 says "all editing surfaces that today expose the per-drawing 'Edit' toggle". The shared editor doesn't expose it. Scope therefore narrows naturally; documenting here so the assumption is recorded.

**Alternatives considered**:
- *Add full drawing UI to the shared editor*: rejected as out of scope — would re-implement everything from feature 020 in a different template; warrants its own spec.
