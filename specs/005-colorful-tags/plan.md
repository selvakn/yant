# Implementation Plan: Colorful Tags

**Branch**: `005-colorful-tags` | **Date**: 2026-04-05 | **Spec**: [spec.md](./spec.md)

## Summary

Add background colors to tags from a fixed 10-color palette. Auto-assign via hash of tag name; allow user override. Store overrides in `tag_colors` table; return color with tag data in API; style tags in all views.

## Technical Context

**Language/Version**: Go 1.22+ (backend), vanilla JS (frontend)  
**Primary Dependencies**: chi, existing templates  
**Storage**: SQLite `tag_colors(user_id, tag_name, color)`  
**Testing**: `go test ./...`  
**Target Platform**: Linux server, modern browsers

## Constitution Check

- [x] **I. Markdown-first** — Tag colors are metadata, not in markdown files.
- [x] **II. Simplicity** — Hash-based default; simple picker UI.
- [x] **III. Monorepo** — Changes in backend/internal, frontend/templates, frontend/static.
- [x] **IV. Integration testing** — Extend handler tests for color endpoints.
- [x] **V. Simple web UI** — CSS-only color pills; minimal JS for picker.
- [x] **VI. Commit & test discipline** — Tests before commits.

## Color Palette

```
Ink Black:       #001219
Dark Teal:       #005f73
Dark Cyan:       #0a9396
Pearl Aqua:      #94d2bd
Vanilla Custard: #e9d8a6
Golden Orange:   #ee9b00
Burnt Caramel:   #ca6702
Rusty Spice:     #bb3e03
Oxidized Iron:   #ae2012
Brown Red:       #9b2226
```

## Project Structure

```text
backend/internal/models/models.go      # tag_colors table, TagCount.Color, hash function
backend/internal/models/models_test.go # tests for color functions
backend/internal/handlers/tags.go      # extend TagsListGET, add TagColorPUT
backend/cmd/server/main.go             # register PUT /tags/{name}/color
frontend/templates/tags/sidebar.html   # colored tag pills
frontend/templates/notes/list.html     # colored tags in note list
frontend/templates/notes/editor.html   # colored tag chips
frontend/templates/notes/reader.html   # colored tags in meta
frontend/static/css/app.css            # tag color styles, picker
```
