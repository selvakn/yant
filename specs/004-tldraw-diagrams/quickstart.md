# Quickstart: Diagram Drawing in Notes

## Prerequisites

- Node.js 18+ and npm (for building tldraw bundle)
- Go 1.22+ (existing)

## Build the tldraw bundle

```bash
cd frontend-build
npm install
npm run build
# Output: ../frontend/static/vendor/tldraw-bundle.js, .css
```

## Run the server

```bash
make run
```

## Test the feature

1. Log in and open any note in **Edit** mode.
2. Click **Add Drawing** (or similar button) below the editor.
3. A canvas opens—draw shapes, lines, add text.
4. Click **Save Drawing** or close the canvas; data is persisted.
5. Refresh the page; click to reopen the drawing—content is intact.
6. Open the note in **Reader** mode; drawing preview appears.
7. Click the preview to edit the drawing.
8. Delete the drawing via a "Remove Drawing" action; confirm it's gone on reload.

## Verify file storage

```bash
ls notes/1/
# Should show <slug>.md and optionally <slug>.tldraw.json
```

## Verify API

```bash
# Get drawing (replace slug)
curl -b cookies.txt http://localhost:8080/notes/my-note/drawing

# Save drawing
curl -b cookies.txt -X PUT http://localhost:8080/notes/my-note/drawing \
  -H "Content-Type: application/json" \
  -d '{"document":{"store":{}}}'

# Delete drawing
curl -b cookies.txt -X DELETE http://localhost:8080/notes/my-note/drawing
```
