# Implementation Plan: Inline Drawing Previews in Edit Mode

**Branch**: `022-inline-drawing-previews` | **Date**: 2026-05-06 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/022-inline-drawing-previews/spec.md`

## Summary

Replace today's per-drawing list-row UI in the note editor with read-only SVG previews that are visible by default — one card per drawing, each showing the drawing's display name, tool tag, rename and delete controls, and the rendered SVG. Click a card to swap that single card's preview for the existing live editing canvas; an explicit Close/Done button (and click-on-another-card) returns it to a refreshed preview. Reuse the existing `/notes/{slug}/drawings/{id}/svg` endpoint and the existing per-tool initFn pipeline — this is a frontend-only change to `frontend/templates/notes/editor.html` and `frontend/static/css/app.css`. Previews stack in marker order at the end of the editor area (the user-accepted fallback in spec FR-005). No backend changes.

## Technical Context

**Language/Version**: Go 1.25+ (backend, untouched), vanilla JS + EasyMDE/CodeMirror (frontend)
**Primary Dependencies**: chi/v5, goldmark, scs/v2, modernc.org/sqlite (untouched); EasyMDE, htmx, vendored excalidraw + tldraw bundles (untouched)
**Storage**: Markdown files + SQLite `note_drawings` table (untouched — this is a UI-only change)
**Testing**: Go `testing` (`make test`); no JS test infrastructure in this project — verification is via `make build && make run` plus manual browser testing per project convention
**Target Platform**: Linux server, modern web browsers
**Project Type**: Web application (backend + frontend in same repo)
**Performance Goals**: Same as feature 020 — up to 10 drawings per note rendered as previews on page load within 5 seconds; markdown text remains responsive while previews load (typing latency unchanged)
**Constraints**: No backend API changes. No new dependencies. Reuse existing SVG and drawing endpoints. Reuse the existing reader-mode SVG-rendering CSS (`.drawing-svg-preview`).
**Scale/Scope**: Frontend-only change limited to the owner's note editor template (`frontend/templates/notes/editor.html`) and `app.css`. Estimated ~150 LOC JS net (replacing the existing `drawing-list` rendering) + small CSS additions.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [x] **I. Markdown-first storage** — No storage changes. Markdown files and the `note_drawings` table remain authoritative; this feature only changes how the editor renders existing drawings.
- [x] **II. Simplicity** — No new dependencies. No new abstractions. The simplest path that satisfies the spec: reuse existing endpoints (`/drawings`, `/drawings/{id}/svg`), reuse existing init bundles, reuse existing CSS class `.drawing-svg-preview` from reader mode. Stacked-at-end placement (the user-accepted fallback) avoids the complexity of CodeMirror line widgets while still satisfying every FR — and no FR requires inline-with-text rendering.
- [x] **III. Monorepo** — Change is isolated to `frontend/templates/notes/editor.html` and `frontend/static/css/app.css`. No shared-contract changes.
- [x] **IV. Integration testing** — Backend has no behavioural change, so existing backend coverage (≥90%) remains intact. No new backend code; no new backend tests required. Frontend manual test plan documented in `quickstart.md`.
- [x] **V. Simple web UI** — Reuses existing vanilla-JS pattern. No framework. Lightweight. The only DOM additions are `.drawing-preview-card` elements that wrap an SVG the server already produces.
- [x] **VI. Commit & test discipline** — Implementation tasks group into small atomic commits (see `tasks.md`). `make test && make lint` run green before each commit. Failure response: fix before progressing.

No violations to record in Complexity Tracking.

## Project Structure

### Documentation (this feature)

```text
specs/022-inline-drawing-previews/
├── plan.md              # This file
├── research.md          # Phase 0 — placement-strategy decision and SVG-refresh strategy
├── data-model.md        # Phase 1 — Drawing Preview as a UI-only entity (no schema changes)
├── quickstart.md        # Phase 1 — manual verification steps
├── contracts/           # No external interface changes; intentionally empty
└── tasks.md             # Phase 2 (/speckit-tasks output)
```

### Source Code (repository root)

Files actually edited by this feature:

```text
frontend/
├── templates/
│   └── notes/
│       └── editor.html          # Replace drawing-list-item with drawing-preview-card; rewrite refreshDrawingList; add Close button to canvas; add rename UI
└── static/
    └── css/
        └── app.css              # Add .drawing-preview-card, .drawing-preview-header, .drawing-preview-body, hover affordance, edit-state styling
```

Files inspected but not modified (verified unchanged behaviour):

```text
backend/internal/handlers/drawings.go  # SVG endpoints already in place; rename PATCH already in place
backend/cmd/server/main.go             # Routes already in place
frontend-build/src/excalidraw-island.tsx  # Bundle save behaviour already PUTs SVG on save
frontend-build/src/tldraw-island.tsx       # Same
frontend/templates/notes/reader.html       # Reader-mode SVG rendering already uses natural aspect ratio (.drawing-svg-preview); no changes needed for SC-007 parity
```

**Structure Decision**: Frontend-only delta. No new files. The owner's note editor template and shared `app.css` are the only edited files. Backend, schema, and frontend-build (Vite/Node) all remain untouched.

## Complexity Tracking

> No constitutional violations to justify.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| _none_ | _n/a_ | _n/a_ |
