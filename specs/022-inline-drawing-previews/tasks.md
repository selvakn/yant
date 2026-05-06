---
description: "Task list for feature 022 — inline drawing previews in edit mode"
---

# Tasks: Inline Drawing Previews in Edit Mode

**Input**: Design documents from `/specs/022-inline-drawing-previews/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md

**Tests**: Backend has no behavioural change — existing backend tests must stay green. No new backend tests are needed (per plan.md). No frontend test infrastructure exists in this project; verification is via the steps in `quickstart.md`.

**Constitution gate**: Principle VI mandates `make test && make lint` green before every commit; commit in small atomic steps; fix failures before progressing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Independent of other in-flight tasks; safe to do in any order
- **[Story]**: User story tag (US1 / US2 / US3) or **CC** (cross-cutting)

---

## Phase 1: Setup

No setup needed — change is isolated to existing files.

---

## Phase 2: Foundational

**Purpose**: CSS scaffolding shared by every preview card (US1 and US2 both depend on it).

- [x] T001 [CC] Add `.drawing-preview-card`, `.drawing-preview-header`, `.drawing-preview-name`, `.drawing-preview-tool`, `.drawing-preview-body`, `.drawing-preview-placeholder` rules to `frontend/static/css/app.css`. Reuse `.drawing-svg-preview` (already in app.css line 479) for the SVG container so SC-007 parity is automatic. Include the resting/edit/hover/empty styles per `research.md` R5.
- [x] T002 [CC] Add `.drawing-canvas-close` style (top-right "Done" button overlay) to `frontend/static/css/app.css`.

**Checkpoint**: CSS is in place but inert until JS in later tasks generates the markup.

---

## Phase 3: User Story 1 — Read-only previews by default (P1) 🎯 MVP

**Goal**: Every drawing on a note renders as a read-only SVG preview on edit-mode page load with no clicks.

**Independent Test**: Open a note with multiple drawings in edit mode → all previews visible immediately (per `quickstart.md` US1 steps).

- [x] T003 [US1] In `frontend/templates/notes/editor.html`, rewrite `refreshDrawingList()` to render `.drawing-preview-card` elements instead of `.drawing-list-item` rows. Each card contains a header (`.drawing-preview-header` with `.drawing-preview-name`, `.drawing-preview-tool`, rename pencil button, delete button) and a body (`.drawing-preview-body` initially empty).
- [x] T004 [US1] In the same file, add `loadPreviewSVG(card, drawingID)` helper that fetches `GET /notes/{slug}/drawings/{drawingID}/svg` and inserts the SVG into `.drawing-preview-body` wrapped in `.drawing-svg-preview`. Handle 404 / empty / network errors by inserting a `.drawing-preview-placeholder` element with appropriate copy ("Empty drawing — click to draw" vs "Preview unavailable — click to edit") per `research.md` R3.
- [x] T005 [US1] After T003 builds the cards, kick off SVG fetches in parallel (one fetch per card) so the markdown editor stays interactive while images load. Do **not** await in series.
- [x] T006 [US1] Wire the existing add-drawing flow (`createDrawing`) to refresh the list after creation. The new drawing's card appears with the empty placeholder; if the existing flow auto-opens the canvas (it does today via `refreshDrawingList(data.drawing_id, toolType)`), keep that behaviour.

**Checkpoint**: With T001–T006 done, `quickstart.md` US1 steps pass. Run `make test && make lint` — both must be green. Commit.

---

## Phase 4: User Story 2 — Click to edit, with explicit Close (P1)

**Goal**: Clicking a preview opens its live canvas; an explicit Done button (or click-on-another-preview) closes it and refreshes the preview.

**Independent Test**: Per `quickstart.md` US2 steps.

- [x] T007 [US2] In `frontend/templates/notes/editor.html`, attach a click handler on `.drawing-preview-body` (and on `.drawing-preview-placeholder`) that calls `openDrawingCanvas(drawingID, toolType, card)`. Clicking the header (rename / delete buttons) must NOT trigger edit mode — stop propagation on those.
- [x] T008 [US2] Modify `openDrawingCanvas` to: (a) close any currently-open canvas via `closeDrawingCanvas()` (which auto-saves per existing bundle behaviour); (b) replace the `.drawing-preview-body` contents in the clicked card with a new `.drawing-canvas` div (instead of inserting a sibling); (c) load the bundle and call the existing `initFn` against that div; (d) append a `.drawing-canvas-close` button labelled "Done" to the card.
- [x] T009 [US2] Modify `closeDrawingCanvas` to: (a) run the existing `drawingCleanup()`; (b) restore the `.drawing-preview-body` to its preview state by calling `loadPreviewSVG(card, drawingID)` (refetching the latest SVG that was just persisted by the bundle); (c) remove the Done button; (d) clear `activeDrawingID`.
- [x] T010 [US2] Wire the Done button's click handler to call `closeDrawingCanvas()`. Add a 100ms-or-so delay before refetching the SVG to give the bundle's PUT-to-`/svg` time to land — or, better, await a `Promise` that the bundle exposes if available (verify by reading `excalidraw-island.tsx` / `tldraw-island.tsx`; if no promise hook, fall back to a small delay and document it).
- [x] T011 [US2] Update the click-on-another-preview path: when the user clicks preview B while preview A's canvas is open, the existing `openDrawingCanvas` logic naturally calls `closeDrawingCanvas` first (T008). Verify A returns to preview state and B opens its canvas — no special code needed beyond T008/T009 if implemented correctly.

**Checkpoint**: With T007–T011 done, `quickstart.md` US2 steps pass. Run `make test && make lint` — both must be green. Commit.

---

## Phase 5: User Story 3 — Placement at end of editor (P2)

**Goal**: Previews are stacked in marker order at the end of the editor area (the user-accepted fallback per FR-005).

**Independent Test**: Per `quickstart.md` US3 steps.

- [x] T012 [US3] In `refreshDrawingList()`, ensure the order in which preview cards are appended matches the order of marker occurrences in the markdown body (`easyMDE.value()`). For each drawing, find the index of its `![[draw:<id>]]` marker; sort the rendered cards by that index. Drawings whose markers are missing from the markdown go after the ones with markers, in API order (existing `appendOrphanedMarkers` logic already handles inserting markers for orphans).
- [x] T013 [US3] After T012, verify the existing markup wrapper `.drawing-section-seamless` already lives below the EasyMDE container in `editor.html` (it does — line 41). No structural change required. Confirm visually via the quickstart.

**Checkpoint**: With T012–T013 done, `quickstart.md` US3 steps pass.

---

## Phase 6: Cross-cutting — Rename, delete, hover, tool tag

**Goal**: Make every preview card own its rename + delete + tool indication, replacing the removed drawing-list. Surface the hover affordance from FR-015.

- [x] T014 [CC] Add inline rename UI inside `.drawing-preview-header` of the new card markup (T003): a small pencil button next to `.drawing-preview-name`. Click → swap the name span for an `<input>`; Enter → `PATCH /notes/{slug}/drawings/{drawingID}` with `{"display_name": value}`; Esc → cancel. Use the existing `escapeHTML()` helper for safety.
- [x] T015 [CC] Add the delete handler on the card-header delete button: same `DELETE /notes/{slug}/drawings/{drawingID}` call as today's drawing-list-delete. On success, remove the card from the DOM (or re-run `refreshDrawingList()`).
- [x] T016 [CC] Show the tool type ("Excalidraw" / "tldraw") as a small `.drawing-preview-tool` badge in each card header (port from existing `.drawing-list-tool`).
- [x] T017 [CC] Verify hover affordance from T001 renders as designed: hover gives border + cursor-pointer + "Click to edit" hint; resting state stays clean. Tweak CSS as needed.
- [x] T018 [CC] Remove the now-orphan `.drawing-list`, `.drawing-list-item`, `.drawing-list-name`, `.drawing-list-tool`, `.drawing-list-edit`, `.drawing-list-delete` rules from `frontend/static/css/app.css` once nothing references them. (Safe to remove only after T003 has fully replaced the old markup.)

**Checkpoint**: With T014–T018 done, every per-drawing action lives on its preview card. Old list-row CSS is gone. Run `make test && make lint`.

---

## Phase 7: Verification & Polish

- [x] T019 [CC] Walk through every step in `quickstart.md` (US1, US2, US3, placeholders, rename, delete, reader parity, negative cases). Capture failures and fix them as discrete sub-tasks.
- [x] T020 [CC] `make test && make lint` — both green.
- [x] T021 [CC] Final commit summarizing the feature on branch `022-inline-drawing-previews`.

---

## Dependencies & Execution Order

- **T001, T002 (Foundational CSS)** must complete before T003 (which uses the classes).
- **T003** must complete before T004–T006 (US1 SVG fetching), T007 (US2 click handlers), T012 (US3 ordering), T014–T016 (header controls).
- **T008** must complete before T009/T010 (close path depends on the new open path).
- **T018** must run last among CSS deletions to avoid breaking interim state.

### Within a user story

- Tests aren't generated for this feature (no project-level frontend tests). Constitution Principle VI still applies: `make test && make lint` green before each commit.

### Parallel opportunities

- T001 and T002 can be done together in one CSS edit pass — naturally serial since they touch the same file but trivially small.
- Within Phase 6, T014, T015, T016 touch different parts of the same template; do them sequentially to avoid merge conflicts but they are conceptually independent.

---

## Notes

- **No backend changes**, no schema changes, no new dependencies.
- Existing endpoints used: `GET /notes/{slug}/drawings`, `GET /notes/{slug}/drawings/{drawingID}/svg`, `POST /notes/{slug}/drawings`, `PATCH /notes/{slug}/drawings/{drawingID}`, `DELETE /notes/{slug}/drawings/{drawingID}`. All already covered by existing tests.
- After every task or logical group: `make test && make lint`. Fix failures before progressing (Constitution Principle VI).
- Commit small. Avoid one giant "feature done" commit.
