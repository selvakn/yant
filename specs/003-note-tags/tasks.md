# Tasks: Seamless note tags

**Input**: `/specs/003-note-tags/` (spec, plan, data-model)
**Prerequisites**: plan.md, spec.md

## Phase 1: Model alignment

- [x] T001 Extend `ParseTags` regex to allow hyphens in tag names; update unit tests in `models_test.go`
- [x] T002 Run `go test ./...` and fix any regressions

## Phase 2: Editor UI

- [x] T003 Add tag bar markup and client script to `frontend/templates/notes/editor.html` (chips, remove, add, debounced sync, `datalist` from `/tags`)
- [x] T004 Add styles for tag bar and chips in `frontend/static/css/app.css`

## Phase 3: Verification

- [x] T005 Manual quickstart validation; ensure htmx save still sends updated body
