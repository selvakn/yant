# Tasks: Colorful Tags

**Input**: `/specs/005-colorful-tags/` (spec, plan)

## Phase 1: Backend Model

- [ ] T001 Add `tag_colors` table to InitSchema
- [ ] T002 Add `Color` field to `TagCount` struct
- [ ] T003 Add color palette constant and hash function for auto-assignment
- [ ] T004 Add `GetTagColor`, `SetTagColor` functions
- [ ] T005 Update `ListTagsForUser` to include colors (from table or auto-assign)
- [ ] T006 Add unit tests for color functions

## Phase 2: Backend API

- [ ] T007 Update `TagsListGET` to return colors in JSON
- [ ] T008 Add `TagColorPUT` handler for `/tags/{name}/color`
- [ ] T009 Register route in main.go
- [ ] T010 Add integration tests for color endpoint

## Phase 3: Frontend

- [ ] T011 Update sidebar.html with colored tag pills
- [ ] T012 Update notes/list.html with colored tags
- [ ] T013 Update notes/editor.html tag chips with colors
- [ ] T014 Update notes/reader.html with colored tags
- [ ] T015 Add CSS for tag colors and color picker
- [ ] T016 Add JS for color picker (click tag → show palette → save)

## Phase 4: Verification

- [ ] T017 Run `make test`
- [ ] T018 Manual validation of color display and changes
