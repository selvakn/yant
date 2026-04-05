# Research: Diagram Drawing in Notes

## Decision: Use tldraw SDK

**Rationale**: tldraw is a mature, MIT-licensed infinite canvas for React. Supports shapes, freehand, connectors, text. Provides `getSnapshot`/`loadSnapshot` for JSON persistence. Active development (v4.4.1 as of 2026-04).

**Alternatives considered**:
- Excalidraw — similar quality but larger bundle; tldraw docs are cleaner.
- Fabric.js — lower-level canvas library; more code to build diagramming UX.
- Custom SVG editor — significant effort; reinventing wheel.

## Decision: Island architecture (not full SPA)

**Rationale**: Constitution mandates simplicity; existing Go templates + htmx work well. Converting everything to React is overkill for one feature. Instead, build a self-contained tldraw bundle that exposes a global `initTldrawIsland(container, snapshotUrl, saveUrl)` function. Load it only on pages that need drawings.

**Trade-offs**:
- Pro: Minimal disruption to existing code; bundle loaded on-demand.
- Con: Two "frontend" systems (Go templates + React island); manageable given single feature scope.

## Decision: Vite as build tool

**Rationale**: Vite has first-class React/TypeScript support, fast HMR for dev, produces optimized bundles. Config is minimal (~20 lines). ESBuild alternative considered but Vite handles CSS imports from tldraw more cleanly.

## Decision: Companion JSON file for drawing data

**Rationale**: Markdown-first principle. The `.md` file remains the note source; `.tldraw.json` is an optional companion. If user copies the notes folder, drawings travel with it. If they only copy `.md`, they lose drawings but text is intact—acceptable trade-off.

File layout:
```
notes/<user_id>/
├── my-note.md
└── my-note.tldraw.json   # optional, present only if drawing exists
```

## Decision: One drawing per note

**Rationale**: Simplest model for MVP. Multiple drawings per note would require a list UI and more complex data model. Can extend later if needed.

## tldraw API surface used

- `Tldraw` component with `store` prop for controlled state.
- `createTLStore()`, `loadSnapshot(store, doc)`, `getSnapshot(store)` for persistence.
- `onMount` callback to access editor instance.
- tldraw CSS imported in bundle.

## Bundle size estimate

tldraw core ~400KB minified; with React 18 tree-shaken, total ~450KB gzipped is achievable.
