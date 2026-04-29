# Implementation Plan: Product Landing Page with Feature Showcase

**Branch**: `019-landing-page` | **Date**: 2026-04-29 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/019-landing-page/spec.md`

## Summary

Redesign the login page (`login.html`) to serve as a product landing page that showcases all YANT features in a clean, developer-friendly layout. Add SEO meta tags and semantic HTML. Include attribution links for tldraw, Excalidraw, and Mermaid. No new backend logic, dependencies, or data model changes required — this is a template and CSS change only.

## Technical Context

**Language/Version**: Go 1.25 (backend, existing) + HTML/CSS (frontend, existing)
**Primary Dependencies**: No new dependencies. Uses existing Go `html/template`, chi/v5 router, and static CSS.
**Storage**: N/A — no data changes
**Testing**: `go test ./...` (existing handler tests cover login page rendering)
**Target Platform**: Web (all modern browsers, responsive 320px-1920px)
**Project Type**: Web application (monorepo)
**Performance Goals**: Page load under 2 seconds (static HTML, no JS required for feature showcase)
**Constraints**: No JavaScript frameworks for the landing page. Server-rendered HTML only.
**Scale/Scope**: Single template file + CSS additions

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [x] **I. Markdown-first storage** — No storage changes. N/A.
- [x] **II. Simplicity** — Pure HTML/CSS change to an existing template. No new dependencies or abstractions.
- [x] **III. Monorepo** — Changes are within `frontend/templates/` and `frontend/static/css/`. Existing structure preserved.
- [x] **IV. Integration testing** — Existing handler tests cover `LoginGET`. Will verify login page renders correctly with new content.
- [x] **V. Simple web UI** — Static HTML with semantic structure. No JS required for the feature showcase.
- [x] **VI. Commit & test discipline** — Small, incremental changes. Tests must pass before each commit.

## Project Structure

### Documentation (this feature)

```text
specs/019-landing-page/
├── plan.md
├── spec.md
└── checklists/
    └── requirements.md
```

### Source Code (repository root)

```text
frontend/
├── templates/
│   └── login.html          # Redesigned landing/login page
└── static/
    └── css/
        └── app.css          # Additional styles for landing page layout
```

**Structure Decision**: No new files needed. The existing `login.html` template is redesigned in place, and landing page styles are added to the existing `app.css`.
