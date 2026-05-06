# Quickstart — Inline Drawing Previews in Edit Mode

This feature is a frontend-only delta. No new build steps, no new dependencies, no schema migrations.

## Build & run

```bash
make test          # backend tests must stay green (no behavioural change expected)
make lint          # go vet should pass
make build         # produces ./bin/server
make run           # serves on :8080
```

Open `http://localhost:8080`, log in, and navigate to a note that has multiple drawings.

## Manual verification (US1 — preview-by-default)

1. Pick or create a note with at least 2 drawings (one Excalidraw, one tldraw).
2. Open the note in edit mode.
3. **Expect**: each drawing renders as a read-only SVG preview on the page on initial load. No clicks required.
4. **Expect**: each preview shows the drawing's display name and tool tag in a small header.
5. **Expect**: the markdown editor remains responsive — you can type immediately, even before all SVGs finish loading.

## Manual verification (US2 — click to edit)

1. Click anywhere on the body of the second preview.
2. **Expect**: the SVG is replaced with the live drawing canvas; the other preview remains a static SVG.
3. Make a change inside the canvas (draw a line, add a shape).
4. Click the **Done** button at the top-right of the canvas.
5. **Expect**: canvas closes, the preview re-renders with the updated SVG showing your change.
6. Click the first preview, then click the second preview without using Done.
7. **Expect**: the first canvas closes (changes auto-saved), preview refreshes; the second preview's canvas opens.

## Manual verification (US3 — placement)

1. In a note with multiple drawings, scroll to the bottom of the editor.
2. **Expect**: all previews are stacked there (in marker order), with the add-drawing form below them — same location as today's drawing-list, just with previews instead of list rows.

## Manual verification — placeholder states

1. Add a new drawing to a note (+ Excalidraw or + tldraw).
2. **Expect**: a new preview card appears at the end immediately, showing an "empty drawing — click to draw" placeholder (no SVG yet) — or the canvas is opened directly per existing flow. Either way, after closing without drawing anything, the placeholder shows.
3. Click the empty placeholder.
4. **Expect**: canvas opens, you can draw, save, close → preview now shows the SVG.

## Manual verification — rename

1. Hover the header of a preview.
2. Click the rename (pencil) button next to the display name.
3. **Expect**: the name becomes an inline input.
4. Type a new name; press Enter.
5. **Expect**: the name updates in place (PATCH succeeds; the marker text in the markdown is unchanged because rename only updates metadata).
6. Hover another preview, click pencil, press Esc.
7. **Expect**: rename cancelled, original name retained.

## Manual verification — delete

1. Click the delete button in a preview's header.
2. Confirm the prompt.
3. **Expect**: the preview disappears; backend marker handling unchanged from today.

## Manual verification — reader-mode parity (SC-007)

1. View the same note in reader mode (the `Edit → Notes/slug` view).
2. Compare the SVG rendering of each drawing side-by-side with the edit-mode preview.
3. **Expect**: same width, same proportions, same scaling.

## Negative checks (regression guard)

- Open a note that has the legacy single-drawing format. **Expect**: legacy live canvas still loads at the bottom (unchanged behaviour).
- Open a note with **no** drawings. **Expect**: no preview area; only the "Add drawing" form is visible.
- View a note in reader mode after this feature ships. **Expect**: reader mode looks unchanged (no regressions).
- Run `make test && make lint` after every commit. Both must be green.
