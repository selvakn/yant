# Implementation Plan: Blog Feature for Tagged Notes

**Branch**: `021-blog-feature` | **Date**: 2026-05-05 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/021-blog-feature/spec.md`

## Summary

Add a public blog view to the notes application. Any note tagged "blog" becomes a publicly accessible blog post at `/blog/<username>/<slug>`. The blog has its own base template with clean, reading-focused layout, separate from the notes-app UI. A `blog_posts` table tracks `published_at` timestamps (set when the "blog" tag is first added). Blog posts are ordered by `published_at` (newest first), support tag-based filtering, pagination, and render embedded drawings as SVG previews. Wiki-links in blog posts resolve to clickable blog links when the target is also a blog post; otherwise they render as styled plain text.

## Technical Context

**Language/Version**: Go 1.25  
**Primary Dependencies**: chi/v5 (routing), goldmark + GFM extension (markdown), scs/v2 (sessions), modernc.org/sqlite (database), html/template (server-rendered templates)  
**Storage**: Markdown files (source of truth) + SQLite `blog_posts` table (derived publication metadata) + existing `notes`, `note_tags`, `users` tables  
**Testing**: Go `testing` package + `net/http/httptest` for handler tests; existing test helpers in `handlers_test.go`  
**Target Platform**: Linux server (Docker: Debian bookworm-slim)  
**Project Type**: Web service (monolith)  
**Performance Goals**: Blog index renders in <500ms; individual posts in <200ms  
**Constraints**: No new external dependencies; reuse existing goldmark pipeline and storage patterns  
**Scale/Scope**: Hundreds of blog posts across a handful of users

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [x] **I. Markdown-first storage** — Notes remain Markdown files on disk. The `blog_posts` table is a derived index tracking only `published_at`; it can be rebuilt from `note_tags` (with loss of original publish time, acceptable trade-off documented).
- [x] **II. Simplicity** — No new external dependencies. Blog is a read-only view over existing notes filtered by tag. Excerpt generation uses simple string processing. No JS framework for blog pages.
- [x] **III. Monorepo** — Blog templates live in `frontend/templates/blog/`, handlers in `backend/internal/handlers/blog.go`, model queries in existing `models.go`. Same repo, same deploy.
- [x] **IV. Integration testing** — Plan includes handler-level integration tests for all blog endpoints, covering index, post, tag filter, pagination, 404s, and drawing SVG serving. Coverage target >=90%.
- [x] **V. Simple web UI** — Blog uses server-rendered Go templates with static CSS. No JavaScript required for blog pages (SVG previews are inline `<img>` tags fetched server-side or embedded). Clean typography for reading.
- [x] **VI. Commit & test discipline** — Implementation will commit after each logical unit (data model, queries, handlers, templates, CSS). Full test suite green before each commit.

## Project Structure

### Documentation (this feature)

```text
specs/021-blog-feature/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```text
backend/
├── cmd/server/main.go          # New /blog routes added to public group
├── internal/
│   ├── models/
│   │   └── models.go           # blog_posts table, ListBlogPosts, GetBlogPost, blog publication hooks
│   ├── handlers/
│   │   ├── blog.go             # NEW: BlogIndexGET, BlogPostGET, BlogTagGET, BlogDrawingSVGGET
│   │   └── blog_test.go        # NEW: Integration tests for all blog handlers
│   └── markdown/
│       └── excerpt.go          # NEW: GenerateExcerpt function for smart-stripping

frontend/
├── templates/
│   └── blog/
│       ├── base.html           # NEW: Standalone blog base template (no notes-app UI)
│       ├── index.html          # NEW: Blog index listing
│       └── post.html           # NEW: Individual blog post
├── static/
│   └── css/
│       └── blog.css            # NEW: Blog-specific stylesheet
```

**Structure Decision**: Blog is a new presentation layer over existing data. No new domain packages needed — blog queries go into existing `models.go`, handlers into a new `blog.go` file following the established pattern. Templates use a separate `blog/base.html` to isolate blog layout from the notes app.

## Constitution Re-Check (Post Phase 1 Design)

- [x] **I. Markdown-first storage** — Confirmed: `blog_posts` table stores only `note_id` + `published_at`. All content remains in Markdown files. Table is rebuildable via `RebuildDB` (with `created_at` fallback for `published_at`).
- [x] **II. Simplicity** — Confirmed: No new dependencies. One new table with 2 columns. Excerpt generation is regex-based string processing. Blog is a read-only view.
- [x] **III. Monorepo** — Confirmed: Blog templates in `frontend/templates/blog/`, handlers in `backend/internal/handlers/blog.go`, queries in existing `models.go`.
- [x] **IV. Integration testing** — Confirmed: `blog_test.go` will cover all 4 routes (index, tag, post, drawing SVG) plus edge cases (404, pagination, empty state, archived notes excluded). Target >=90% coverage.
- [x] **V. Simple web UI** — Confirmed: Server-rendered HTML + CSS. No JavaScript on blog pages. SVG previews served as `<img>` tags.
- [x] **VI. Commit & test discipline** — Confirmed: Implementation tasks are structured for incremental commits: data model → queries → handlers → templates → CSS. Each commit will pass the full test suite.
