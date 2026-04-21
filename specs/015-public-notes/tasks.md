# Tasks: Public Note Sharing

**Input**: Design documents from `/specs/015-public-notes/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/api.md

**Tests**: Required by constitution (Principles IV and VI).

## Format: `[ID] [P?] [Story] Description`

---

## Phase 1: Setup

Empty — project already exists.

---

## Phase 2: Foundational (Blocking Prerequisites)

Schema + token generation. All user stories depend on this.

- [ ] T001 Add `public_notes` table schema to `InitSchema()` in `backend/internal/models/models.go`
- [ ] T002 [P] Create `GenerateShareToken()` (16-byte crypto/rand → base64url) in `backend/internal/models/public.go`
- [ ] T003 [P] Create `PublishNote()`, `UnpublishNote()` model functions in `backend/internal/models/public.go`
- [ ] T004 [P] Create `GetNoteByToken()` and `ListPublishedNotes()` query functions in `backend/internal/models/public.go`
- [ ] T005 [P] Create `CountPublishedNotes(userID)` function in `backend/internal/models/public.go`
- [ ] T006 Write model tests (token generation, publish/unpublish, lookup by token, archive side-effect) in `backend/internal/models/public_test.go`

**Checkpoint**: Token generation and DB operations ready.

---

## Phase 3: User Story 1 — Make a Note Public (Priority: P1) MVP

**Goal**: Owner can toggle Publish/Unpublish on a note; the system generates a persistent token; the public URL is shown with a copy button.

**Independent Test**: Open a note, click Publish, verify URL appears and copies correctly. Visit the URL in incognito — renders the note. Click Unpublish — URL returns 404.

### Implementation

- [ ] T007 [US1] Create `PublishPUT` handler (`PUT /notes/{slug}/publish`) returning `{ok, token, public_url}` JSON in `backend/internal/handlers/public.go`
- [ ] T008 [US1] Create `UnpublishPUT` handler (`PUT /notes/{slug}/unpublish`) in `backend/internal/handlers/public.go`
- [ ] T009 [US1] Register `PUT /notes/{slug}/publish` and `PUT /notes/{slug}/unpublish` routes (protected) in `backend/cmd/server/main.go`
- [ ] T010 [US1] Modify `NoteReaderGET` to pass `IsPublic` (bool) and `PublicURL` (string) to the reader template in `backend/internal/handlers/notes.go`
- [ ] T011 [US1] Add "Publish" / "Unpublish" menu item + share URL display with copy button to `frontend/templates/notes/reader.html`
- [ ] T012 [P] [US1] Add share-URL styling (dialog/banner, copy button hover) to `frontend/static/css/app.css`
- [ ] T013 [P] [US1] Add copy-to-clipboard handler for `#copy-public-url` button in `frontend/static/js/app.js`
- [ ] T014 [US1] Write handler tests for PublishPUT/UnpublishPUT (owner, non-owner 404, idempotency) in `backend/internal/handlers/handlers_test.go`

**Checkpoint**: Owner can publish/unpublish from the reader view.

---

## Phase 4: User Story 2 — Read a Public Note Without Signing In (Priority: P1)

**Goal**: Visitors can open `/p/{token}` and read the note without login. No owner UI leaks. Images load. Wiki-links to private notes don't leak existence.

**Independent Test**: Open a published note URL in a fresh incognito browser. Verify: note title + body render, no sidebar, no edit/archive/delete buttons, no tags sidebar. Embedded images load. Wiki-link to a private note shows as plain text.

### Implementation

- [ ] T015 [US2] Create `ResolveWikiLinksPublic()` in `backend/internal/models/public.go` (resolves `[[Title]]` → `/p/{target-token}` only if target is published; otherwise renders as plain text)
- [ ] T016 [US2] Create `PublicNoteGET` handler (`GET /p/{token}`) in `backend/internal/handlers/public.go` — lookup by token, verify published + not archived, read markdown, run public wiki-link resolver, render via goldmark, rewrite `/uploads/{username}/...` image URLs to `/p/{token}/uploads/...`, render checkboxes as inert (no data-slug/data-line)
- [ ] T017 [US2] Register `GET /p/{token}` route (public, outside RequireLogin group) in `backend/cmd/server/main.go`
- [ ] T018 [US2] Create `frontend/templates/public/note.html` — minimal standalone HTML (no base.html), includes `<meta name="robots" content="noindex,nofollow">` and Open Graph tags, renders title + body HTML only
- [ ] T019 [P] [US2] Add public note page styles to `frontend/static/css/app.css` (clean single-column layout, no sidebar/nav)
- [ ] T020 [US2] Write handler tests: published note accessible (200), unpublished 404, archived 404, deleted 404, unknown token 404, no leakage in response body in `backend/internal/handlers/handlers_test.go`
- [ ] T021 [US2] Write test verifying wiki-links to private notes render as plain text (no `<a>` tag, no target slug) in `backend/internal/handlers/handlers_test.go`

**Checkpoint**: Public URL reads work end-to-end.

### Sub-slice 2a: Public Image Serving (dependency of US2)

- [ ] T022 [US2] Create `PublicImageServeGET` handler (`GET /p/{token}/uploads/{filename}`) that verifies token→note relationship and serves image with ownership by note ID in `backend/internal/handlers/public.go`
- [ ] T023 [US2] Register `GET /p/{token}/uploads/{filename}` route (public) in `backend/cmd/server/main.go`
- [ ] T024 [US2] Write test for public image access: image in a public note loads; image not belonging to the note returns 404; image for an unpublished note returns 404 in `backend/internal/handlers/handlers_test.go`

### Sub-slice 2b: Public Drawing Serving (dependency of US2)

- [ ] T025 [US2] Create `PublicDrawingGET` handler (`GET /p/{token}/drawing`) in `backend/internal/handlers/public.go`
- [ ] T026 [US2] Register `GET /p/{token}/drawing` route (public) in `backend/cmd/server/main.go`
- [ ] T027 [US2] Include tldraw read-only canvas in `frontend/templates/public/note.html` when the note has a drawing

---

## Phase 5: Archive/Delete Safety Cascades (P1)

**Goal**: Archiving a public note revokes public access immediately. Deleting removes the share row.

### Implementation

- [ ] T028 [US2] Modify `NotesArchivePUT` to call `UnpublishNote()` on archive in `backend/internal/handlers/notes.go`
- [ ] T029 [US2] Verify `ON DELETE CASCADE` on `public_notes.note_id` via test in `backend/internal/models/public_test.go` (insert public note, delete parent note, confirm public row removed)
- [ ] T030 [US2] Write handler test: archive a public note, verify `/p/{token}` returns 404 in `backend/internal/handlers/handlers_test.go`

---

## Phase 6: User Story 3 — Manage Public Notes from a Central Place (Priority: P3)

**Goal**: Owner can see all published notes in one place and unpublish from there.

**Independent Test**: Publish 2 notes, open `/public`, verify both appear with URLs. Click Unpublish on one, verify it disappears from the list and its URL stops working.

### Implementation

- [ ] T031 [US3] Create `PublicNotesListGET` handler (`GET /public`) in `backend/internal/handlers/public.go`
- [ ] T032 [US3] Register `GET /public` route (protected, authenticated) in `backend/cmd/server/main.go`
- [ ] T033 [US3] Create `frontend/templates/public/list.html` — owner's view (uses base.html) showing title, URL, copy button, unpublish button for each published note
- [ ] T034 [US3] Add "Public notes" link to sidebar with count badge in `frontend/templates/tags/sidebar.html`
- [ ] T035 [US3] Modify `TagsListGET` to include `PublicCount` in sidebar data in `backend/internal/handlers/tags.go`
- [ ] T036 [P] [US3] Add styles for the public notes list (rows with share URL, buttons) in `frontend/static/css/app.css`
- [ ] T037 [US3] Write handler test for `PublicNotesListGET` in `backend/internal/handlers/handlers_test.go`

**Checkpoint**: Full feature complete.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [ ] T038 Run full test suite (`make test`) — all tests pass
- [ ] T039 Run `make lint` — no issues
- [ ] T040 Manually verify quickstart.md flow end-to-end in a browser
- [ ] T041 [P] Add `d` / keyboard shortcut or sidebar link updates to `frontend/templates/base.html` shortcuts modal for public notes

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 2 (Foundational)**: BLOCKS all user stories
- **US1 (Phase 3)**: Depends on Phase 2
- **US2 (Phase 4)**: Depends on Phase 2; independent of US1 in logic, but UI polish from US1 is nice to have before testing US2
- **Archive safety (Phase 5)**: Depends on US1 and US2
- **US3 (Phase 6)**: Depends on US1 (handler for unpublish exists)

### Parallel Opportunities

- T002, T003, T004, T005 — all in same new file but logically independent functions; can be written in one session as "draft public.go"
- T012, T013, T019 (CSS/JS for UI) can run parallel to handler work
- T036 parallel to T031–T035

---

## Implementation Strategy

### MVP scope (US1 + US2)

Complete Phase 2 → US1 → US2 → Archive cascades. At this point the feature is fully functional end-to-end for a single publish/visit flow.

### Incremental delivery

1. Phase 2: Schema + token generation (commit)
2. US1: Publish toggle + owner UI (commit)
3. US2: Public reader + image serving + drawing serving (commit)
4. Phase 5: Archive safety (commit)
5. US3: Public notes list + sidebar (commit)
6. Polish (commit)

---

## Notes

- **Constitution VI**: Run `make test` before every commit. Fix failures before continuing.
- **Constitution IV**: Integration tests required for all new endpoints.
- [P] tasks = different files, no dependencies.
