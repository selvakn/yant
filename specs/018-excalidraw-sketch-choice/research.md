# Research: Excalidraw Sketch Choice

## Decision: Add Excalidraw as a second drawing tool alongside tldraw

**Rationale**: Users benefit from having a choice between two well-established diagramming tools. Excalidraw provides a hand-drawn aesthetic that some users prefer, while tldraw offers a more polished, precise feel. Both are MIT-licensed, React-based, and support JSON-based persistence.

**Alternatives considered**:
- Replace tldraw entirely with Excalidraw — breaks backward compatibility, removes user choice.
- Fabric.js or custom SVG — lower-level, much more implementation effort for equivalent UX.
- Embed via iframe (excalidraw.com) — breaks seamless auth/CSP, adds external dependency.

## Decision: Use `@excalidraw/excalidraw` npm package

**Rationale**: The `@excalidraw/excalidraw` package (latest stable: ~0.18.x) is the official React component. It supports React 18 (peer dependency `^17.0.2 || ^18.2.0 || ^19.0.0`), which matches the existing tldraw island setup. MIT license — compatible with existing project licensing.

**Key APIs**:
- `<Excalidraw>` component with `initialData`, `onChange`, `excalidrawAPI` ref callback.
- `viewModeEnabled={true}` for read-only rendering (pan/zoom enabled, editing disabled).
- `serializeAsJSON({ elements, appState })` for persistence (stable format, strips deleted elements).
- `restore(parsed)` / `restoreElements` for reloading saved data.
- CSS import: `@excalidraw/excalidraw/dist/excalidraw.css` (or via package exports).

**Bundle size**: Comparable to tldraw (~300-500KB gzipped depending on tree-shaking). React 18 is shared between both islands via Vite config, so it is not duplicated.

## Decision: Separate island bundle for Excalidraw

**Rationale**: Follow the same island architecture as tldraw. Create `excalidraw-island.tsx` that exposes `window.initExcalidrawIsland(container, snapshotUrl, saveUrl, options)` — same contract as `initTldrawIsland`. Build as a separate Vite entry point producing `excalidraw-bundle.js` + `excalidraw-bundle.css`.

**Trade-offs**:
- Pro: Each bundle loaded only when needed; no bundle bloat for notes without drawings.
- Pro: React shared via Vite `external` / dedup — both islands reuse the same React instance.
- Con: Two separate bundles to build and maintain. Acceptable given clear separation and identical contract.

**Alternative rejected**: Single unified bundle with both tools — larger download for every drawing, more complex lazy-loading logic inside the bundle.

## Decision: Tool type stored in drawing file name convention

**Rationale**: Existing drawings use `<slug>.tldraw.json`. Excalidraw drawings will use `<slug>.excalidraw.json`. This file-extension convention:
- Requires zero migration for existing tldraw drawings.
- Makes tool type detection trivial at the filesystem level.
- Works naturally with git version control (different file paths, clear in diffs).
- The backend storage layer already uses `drawingPath()` — extend to accept a tool type parameter.

**Alternatives considered**:
- Tool type field inside the JSON — requires reading/parsing the file to detect type; existing `.tldraw.json` files would need migration or special-casing.
- Separate metadata file — over-engineering for a single boolean attribute.
- SQLite column — unnecessary; file presence is the detection mechanism.

## Decision: Drawing API extended with `type` query parameter

**Rationale**: The existing API (`GET/PUT/DELETE /notes/{slug}/drawing`) will accept an optional `?type=excalidraw` or `?type=tldraw` query parameter on PUT to indicate the drawing tool. For GET and DELETE, the backend checks which file exists (`.tldraw.json` or `.excalidraw.json`). The GET response will include a `type` field in the JSON wrapper so the frontend knows which island to initialize.

**API changes**:
- `GET /notes/{slug}/drawing` — returns `{"type": "tldraw", "document": {...}}` or `{"type": "excalidraw", "data": {...}}`. 404 if no drawing.
- `PUT /notes/{slug}/drawing?type=excalidraw` — saves as `.excalidraw.json`. Default type is `tldraw` for backward compatibility.
- `DELETE /notes/{slug}/drawing` — deletes whichever drawing file exists.
- Version history endpoint unchanged — file path in git determines type.

## Decision: Tool selection UI as inline buttons (not modal)

**Rationale**: When no drawing exists, the "Add a sketch" button is replaced by two side-by-side buttons: "Excalidraw" and "tldraw". Clicking one immediately creates the drawing with that tool. This is simpler than a modal dialog and consistent with the existing lightweight UI patterns (no modals elsewhere in the drawing flow).

## Decision: Vite multi-entry build

**Rationale**: Extend `vite.config.ts` to produce two separate bundles from two entry points: `tldraw-island.tsx` and `excalidraw-island.tsx`. Each produces its own JS and CSS file. React is shared via Vite's dependency deduplication. The `copy-vendor.js` script remains unchanged (it handles non-Vite vendor assets).

**Alternative rejected**: Two separate Vite configs — unnecessary complexity; Vite supports multiple `lib.entry` configurations or can be configured with `rollupOptions.input` for multi-entry.
