# Implementation Plan: Seamless note tags

**Branch**: `003-note-tags` | **Date**: 2026-04-05 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/003-note-tags/spec.md`

## Summary

Add an in-editor **tag summary bar** (chips + quick-add field) synced with the Markdown body. Tags remain embedded as `#tokens` in the file (constitution: Markdown-first). The server already parses tags and syncs `note_tags` on save; work is **frontend** (editor template + CSS + small client script), plus **aligning the hashtag grammar** in `ParseTags` with hyphens so `#my-tag` is one tag. Suggestions use existing `GET /tags` JSON. Integration tests cover regex and save behavior; editor UI is verified manually or via existing handler tests if extended.

## Technical Context

**Language/Version**: Go 1.22+ (backend), vanilla JS in templates (frontend)  
**Primary Dependencies**: chi, goldmark, EasyMDE (existing), htmx (existing)  
**Storage**: Markdown files + SQLite `note_tags` (unchanged)  
**Testing**: `go test ./...`, `make test` / coverage per constitution  
**Target Platform**: Linux server, modern browsers  
**Project Type**: Web application (`backend/`, `frontend/`)  
**Performance Goals**: Debounced chip refresh on editor change (~150ms)  
**Constraints**: No new heavy JS frameworks; progressive enhancement  
**Scale/Scope**: Single-user-per-note editor; suggestions from `/tags` list

## Constitution Check

- [x] **I. Markdown-first** — Tags stay in `.md` body as `#tag` tokens.
- [x] **II. Simplicity** — No new services; reuse `/tags` JSON.
- [x] **III. Monorepo** — Changes under `frontend/templates`, `frontend/static`, `backend/internal/models` only.
- [x] **IV. Integration testing** — Extend `handlers_test` / `models_test` for tag grammar if needed; maintain coverage.
- [x] **V. Simple web UI** — Chips + input; matches existing slate/blue styles.
- [x] **VI. Commit & test discipline** — Run full test suite before commits.

## Project Structure

### Documentation (this feature)

```text
specs/003-note-tags/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── spec.md
└── tasks.md
```

### Source Code (repository root)

```text
frontend/templates/notes/editor.html   # Tag bar + JS
frontend/static/css/app.css              # Chip / tag bar styles
backend/internal/models/models.go        # tagRe hyphen support
backend/internal/models/models_test.go   # ParseTags cases
```

**Structure Decision**: Editor-only UI; no new routes required.

## Complexity Tracking

> No constitution violations; table not used.
