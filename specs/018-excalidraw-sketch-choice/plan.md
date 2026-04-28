# Implementation Plan: Excalidraw Sketch Choice

**Branch**: `018-excalidraw-sketch-choice` | **Date**: 2026-04-28 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/018-excalidraw-sketch-choice/spec.md`

## Summary

Add **Excalidraw** as a second drawing tool alongside tldraw. When creating a new sketch on a note, users choose between Excalidraw and tldraw. The chosen tool persists for that drawing. All existing features (read-only viewer, version control with side-by-side diff, save/auto-save, cascade delete) work identically for both tools. Existing tldraw drawings require zero migration.

The drawing tool type is identified by file extension: `.tldraw.json` (existing) vs `.excalidraw.json` (new). A separate Vite-built Excalidraw island bundle mirrors the tldraw island architecture.

## Technical Context

**Language/Version**: Go 1.25 (backend), TypeScript/React 18 (frontend bundles)
**Primary Dependencies**: `@excalidraw/excalidraw` ~0.18.x (npm, MIT), tldraw 4.x (existing), Vite 6 (build), chi/v5, goldmark (existing)
**Storage**: `<slug>.excalidraw.json` files alongside `<slug>.md`; `<slug>.tldraw.json` unchanged; SQLite unaffected
**Testing**: `go test ./...` with ≥90% coverage; `npm run build` for bundle verification; manual E2E for canvas interaction
**Target Platform**: Linux server, modern browsers (Chrome, Firefox, Safari, Edge)
**Project Type**: Web application (backend/, frontend/, frontend-build/)
**Performance Goals**: Canvas load <1s; save round-trip <500ms; bundle loaded on-demand
**Constraints**: No full SPA conversion; island architecture maintained; backward-compatible API
**Scale/Scope**: Single-user per drawing; multiplayer out of scope

## Constitution Check

- [x] **I. Markdown-first** — Drawing JSON stored as companion file `<slug>.excalidraw.json`; markdown remains portable; JSON is optional/companion. Identical pattern to existing tldraw.
- [x] **II. Simplicity** — Reuses existing island architecture and API contract pattern. One new npm dependency (`@excalidraw/excalidraw`). No new abstractions beyond extending the storage layer with tool type awareness.
- [x] **III. Monorepo** — New `excalidraw-island.tsx` in existing `frontend-build/`; backend handlers in existing `backend/internal/`. Shared API contract documented in `contracts/`.
- [x] **IV. Integration testing** — Drawing handler tests extended to cover both tool types; ≥90% coverage maintained. Version history tests cover cross-tool scenarios.
- [x] **V. Simple web UI** — Tool selection as inline buttons; Excalidraw canvas embedded same as tldraw; bundles loaded on-demand per page.
- [x] **VI. Commit & test discipline** — `make test` before each commit; `npm run build` verified; failures fixed before proceeding.

## Project Structure

### Documentation (this feature)

```text
specs/018-excalidraw-sketch-choice/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── spec.md
├── tasks.md
├── contracts/
│   └── drawing-api.md
└── checklists/
    └── requirements.md
```

### Source Code (repository root)

```text
frontend-build/
├── package.json                        # Add @excalidraw/excalidraw dependency
├── vite.config.ts                      # Multi-entry: tldraw + excalidraw bundles
└── src/
    ├── tldraw-island.tsx               # Existing (unchanged)
    └── excalidraw-island.tsx           # NEW: Excalidraw island with same contract

frontend/static/vendor/
├── tldraw-bundle.js                    # Existing (rebuilt, unchanged output)
├── tldraw-bundle.css                   # Existing
├── excalidraw-bundle.js               # NEW: built output
└── excalidraw-bundle.css              # NEW: built output

frontend/templates/notes/
├── editor.html                         # Updated: tool selection UI, conditional bundle loading
├── reader.html                         # Updated: conditional bundle by tool type
├── version.html                        # Updated: conditional bundle by tool type
└── diff.html                           # Updated: conditional bundle by tool type

frontend/templates/shared/
└── reader.html                         # Updated: conditional bundle by tool type

frontend/templates/public/
└── note.html                           # Updated: conditional bundle by tool type

backend/internal/storage/
└── drawings.go                         # Extended: tool type parameter, dual-file detection

backend/internal/handlers/
├── drawings.go                         # Extended: type query param, wrapped response
├── history.go                          # Extended: detect drawing type at commit
├── notes.go                            # Extended: pass DrawingType to templates
├── public.go                           # Extended: DrawingType support
└── shares.go                           # Extended: DrawingType support

backend/internal/versioning/
└── git.go                              # Minor: DrawingType field in DiffResult
```

**Structure Decision**: Extend existing island architecture — Excalidraw bundle is a parallel island to tldraw, loaded on-demand based on the drawing's tool type.

## Complexity Tracking

| Decision | Why Needed | Simpler Alternative Rejected Because |
|----------|------------|--------------------------------------|
| Two separate bundles | Each drawing tool has distinct React components, CSS, and APIs | Single bundle would force downloading ~800KB+ for every drawing page regardless of which tool is used |
| File extension for tool type | Zero-migration, filesystem-detectable, git-friendly | JSON field inside data requires parsing; SQLite column adds unnecessary coupling |
