# Quickstart: Excalidraw Sketch Choice

## Prerequisites

- Go 1.25+
- Node.js 24+ (for building frontend bundles)
- Make

## Build

```bash
# Install Excalidraw dependency and build both bundles
make build-frontend

# Build Go backend
make build
```

## Run

```bash
make run
```

## Test the feature

1. Navigate to any note in the editor view.
2. Click "+ Add a sketch" — two options should appear: "Excalidraw" and "tldraw".
3. Select "Excalidraw" — the Excalidraw canvas should load.
4. Draw some shapes, wait for auto-save indicator.
5. Navigate away and return — the drawing should reload with Excalidraw.
6. View the note in reader mode — the drawing should render read-only.
7. Delete the drawing, add a new one — tool choice should reappear.
8. Select "tldraw" — verify tldraw canvas loads (existing behavior preserved).

## Verify version control

1. Create an Excalidraw drawing and save.
2. Edit the drawing and save again.
3. Go to note history — both versions should appear.
4. View the diff — side-by-side Excalidraw canvases should render.
5. Revert to the first version — drawing should restore.

## Verify backward compatibility

1. Open a note that already has a tldraw drawing.
2. Verify it loads and edits correctly in tldraw.
3. Check version history still works for existing tldraw drawings.

## Run tests

```bash
make test
make coverage
```
