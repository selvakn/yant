# Tasks: Colorful Tags

**Input**: `/specs/005-colorful-tags/` (spec, plan)

## Phase 1: Backend Model

- [x] T001 Add `tag_colors` table to InitSchema
- [x] T002 Add `Color` field to `TagCount` struct
- [x] T003 Add color palette constant and hash function for auto-assignment
- [x] T004 Add `GetTagColor`, `SetTagColor` functions
- [x] T005 Update `ListTagsForUser` to include colors (from table or auto-assign)
- [x] T006 Add unit tests for color functions

## Phase 2: Backend API

- [x] T007 Update `TagsListGET` to return colors in JSON
- [x] T008 Add `TagColorPUT` handler for `/tags/{name}/color`
- [x] T009 Register route in main.go
- [x] T010 Add integration tests for color endpoint

## Phase 3: Frontend

- [x] T011 Update sidebar.html with colored tag pills
- [x] T012 JS applies colors to all .tag elements (list, reader, editor)
- [x] T013 Ctrl/Cmd+click opens color picker
- [x] T014 CSS for tag colors and picker grid
- [x] T015 Color picker saves via PUT /tags/{name}/color

## Phase 4: Verification

- [x] T017 Run `make test`
- [ ] T018 Manual validation of color display and changes
