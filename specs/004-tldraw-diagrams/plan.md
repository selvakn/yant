# Implementation Plan: Diagram Drawing in Notes

**Branch**: `004-tldraw-diagrams` | **Date**: 2026-04-05 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/004-tldraw-diagrams/spec.md`

## Summary

Add diagram drawing capability using **tldraw** (React-based infinite canvas SDK). Because tldraw requires React 18+, we introduce a **minimal frontend build system** that compiles a self-contained bundle loaded only on pages with drawings. Drawing data (JSON) is stored as a **companion file** next to each note's markdown, preserving markdown-first portability.

## Technical Context

**Language/Version**: Go 1.22+ (backend), TypeScript/React 18 (tldraw bundle)  
**Primary Dependencies**: tldraw 4.x (npm), Vite (build), chi, goldmark (existing)  
**Storage**: `<slug>.tldraw.json` files alongside `<slug>.md`; SQLite index unchanged  
**Testing**: `go test ./...`, manual E2E for canvas; Vite build verified by `npm run build`  
**Target Platform**: Linux server, modern browsers (Chrome, Firefox, Safari, Edge)  
**Project Type**: Web application (backend/, frontend/, new frontend-build/)  
**Performance Goals**: Canvas load <1s; save round-trip <500ms  
**Constraints**: Bundle size target <500KB gzipped; no full SPA conversion  
**Scale/Scope**: Single-user per drawing; multiplayer out of scope

## Constitution Check

- [x] **I. Markdown-first** — Drawing JSON stored as companion file `<slug>.tldraw.json`; markdown remains portable; JSON is optional/companion.
- [x] **II. Simplicity** — Minimal "island" bundle for tldraw only; no full React SPA; Vite chosen for fast builds and small config.
- [x] **III. Monorepo** — New `frontend-build/` directory for tldraw source; built output goes to `frontend/static/vendor/`.
- [x] **IV. Integration testing** — Backend drawing endpoints covered by handler tests; canvas interaction tested manually.
- [x] **V. Simple web UI** — Canvas embedded in existing editor/reader templates; styling matches current design.
- [x] **VI. Commit & test discipline** — `make test` before each commit; add `npm run build` to CI/pre-commit if needed.

## Project Structure

### Documentation (this feature)

```text
specs/004-tldraw-diagrams/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── spec.md
├── tasks.md
└── checklists/requirements.md
```

### Source Code (repository root)

```text
frontend-build/                   # NEW: tldraw bundle source
├── package.json
├── vite.config.ts
├── tsconfig.json
└── src/
    └── tldraw-island.tsx         # React entry point exposing global init

frontend/static/vendor/
├── tldraw-bundle.js              # Built output (gitignored source maps)
└── tldraw-bundle.css

frontend/templates/notes/
├── editor.html                   # Add drawing section
└── reader.html                   # Add drawing preview

backend/internal/storage/
└── drawings.go                   # Read/write/delete <slug>.tldraw.json

backend/internal/handlers/
└── drawings.go                   # GET/PUT/DELETE /notes/{slug}/drawing

backend/internal/handlers/
└── notes.go                      # Cascade delete drawing on note delete
```

**Structure Decision**: Island architecture—tldraw is bundled separately and loaded on-demand; existing templates and Go backend remain primary.

## Complexity Tracking

| Decision | Why Needed | Simpler Alternative Rejected Because |
|----------|------------|--------------------------------------|
| Vite build system | tldraw requires React 18 with JSX transform | Vendoring pre-built tldraw would lock version and bloat; CDN adds external dependency |
| Companion JSON file | Keeps markdown portable; drawing data is structured | Embedding JSON in markdown (code fence) complicates parsing and bloats text |
