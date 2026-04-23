# Tasks: Note Sharing

**Input**: Design docs in `/specs/016-note-sharing/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/api.md

## Format: `[ID] [P?] [Story] Description`

---

## Phase 1: Setup

Empty.

---

## Phase 2: Foundational

- [ ] T001 Add `note_shares` table to `InitSchema()` in `backend/internal/models/models.go`
- [ ] T002 Create `backend/internal/models/shares.go` with types `NoteShare`, `NoteCollaborator`, `SharedNoteSummary` and functions: `GrantShare`, `RevokeShare`, `ListSharesForNote`, `ListSharedNotesForUser`, `CountSharedNotesForUser`, `GetShareByViewerAndNote`
- [ ] T003 Add `GetNoteForViewer(db, viewerID int64, ownerUsername, slug) (*Note, string, error)` in `backend/internal/models/shares.go` — returns note, role (`owner|editor|reader`), error
- [ ] T004 Add `UpdateNoteByID(db, noteID int64, title string)` helper in `backend/internal/models/models.go` (current `UpdateNote` takes owner userID; we need a variant that skips the owner check since the handler already authorized via share)
- [ ] T005 [P] Write tests for grant/revoke/list/count and `GetNoteForViewer` in `backend/internal/models/shares_test.go`
- [ ] T006 Add `CommitFileAs(notesDir, relPath, message, authorName, authorEmail string) error` in `backend/internal/versioning/git.go`

**Checkpoint**: Schema and model layer ready; git attribution override available.

---

## Phase 3: US1 — Owner shares a note (P1)

- [ ] T007 [US1] Create `backend/internal/handlers/shares.go` with `ShareCreatePUT` handler (`PUT /notes/{slug}/share`, body `{username, permission}`) — rejects self-share, unknown usernames, invalid permission; upserts via `GrantShare`
- [ ] T008 [US1] Add `ShareDeletePUT` handler (`DELETE /notes/{slug}/share/{username}`) in `backend/internal/handlers/shares.go`
- [ ] T009 [US1] Add `ShareListGET` handler (`GET /notes/{slug}/shares`) in `backend/internal/handlers/shares.go`
- [ ] T010 [US1] Register the three routes (protected) in `backend/cmd/server/main.go`
- [ ] T011 [US1] Modify `NoteReaderGET` in `backend/internal/handlers/notes.go` to pass `Collaborators []NoteCollaborator` to the reader template
- [ ] T012 [US1] Add **Share** button + share dialog to `frontend/templates/notes/reader.html` — form for username+permission, list of current collaborators, change/revoke per row
- [ ] T013 [P] [US1] Add share dialog CSS (modal, per-row controls) in `frontend/static/css/app.css`
- [ ] T014 [P] [US1] Add share dialog JS handlers (open, submit, update, revoke) in `frontend/static/js/app.js`
- [ ] T015 [US1] Handler tests for ShareCreatePUT (valid grant, non-owner 404, self-share rejected, unknown username rejected, upsert on re-grant), ShareDeletePUT, ShareListGET in `backend/internal/handlers/handlers_test.go`

**Checkpoint**: Owner can grant/revoke/manage shares from the reader.

---

## Phase 4: US2 — Recipient sees shared notes (P1)

- [ ] T016 [US2] Create `SharedNotesListGET` handler (`GET /shared`) in `backend/internal/handlers/shares.go` — calls `ListSharedNotesForUser`
- [ ] T017 [US2] Create `frontend/templates/shared/list.html` — list grouped by owner, with title, permission badge, owner username, date
- [ ] T018 [US2] Register `GET /shared` (protected) in `backend/cmd/server/main.go`
- [ ] T019 [US2] Create `SharedNoteReaderGET` handler (`GET /shared/{username}/{slug}`) — uses `GetNoteForViewer`; passes `OwnerUsername`, `Role`, `Permission` to template
- [ ] T020 [US2] Create `ResolveWikiLinksForViewer(db, viewerID, ownerID, body)` in `backend/internal/models/shares.go` — only links to targets also shared with viewer; all others plain text
- [ ] T021 [US2] Use `ResolveWikiLinksForViewer` inside `SharedNoteReaderGET` (and rewrite checkbox rendering to be inert since shared-note checkbox toggle can come later; for now render as disabled)
- [ ] T022 [US2] Create `frontend/templates/shared/reader.html` (separate from owner reader) — "Shared by {owner}" banner, no Archive/Delete/Share/Publish, no todo-toggle (read-only for both roles initially; edit flow covered in US4)
- [ ] T023 [US2] Register `GET /shared/{username}/{slug}` in `backend/cmd/server/main.go`
- [ ] T024 [P] [US2] Add CSS for the shared-by banner, permission badge, shared list rows in `frontend/static/css/app.css`
- [ ] T025 [US2] Modify `backend/internal/handlers/tags.go` — include `SharedCount` from `CountSharedNotesForUser` in sidebar data
- [ ] T026 [US2] Add "Shared with me" link with count badge to `frontend/templates/tags/sidebar.html`
- [ ] T027 [US2] Handler tests: `SharedNotesListGET` (shows only shared, excludes own), `SharedNoteReaderGET` (grant → 200, no grant → 404, archived → 404), wiki-link leakage test (private target → plain text) in `backend/internal/handlers/handlers_test.go`

**Checkpoint**: Recipients can see and read shared notes; private wiki-links don't leak.

---

## Phase 5: US4 — Owner-only destructive actions (P2) + shared-note editing

- [ ] T028 [US4] Create `SharedNoteEditorGET` handler (`GET /shared/{username}/{slug}/edit`) in `backend/internal/handlers/shares.go` — requires edit permission, else 403
- [ ] T029 [US4] Create `SharedNoteUpdate` handler (`POST /shared/{username}/{slug}`) — requires edit permission; writes to owner's file via `storage.WriteNote(h.notesDir, ownerID, slug, body)`; syncs tags/links/todos/embedding; uses `versioning.CommitFileAs` with viewer's username as author
- [ ] T030 [US4] Register `GET /shared/{username}/{slug}/edit` and `POST /shared/{username}/{slug}` in `backend/cmd/server/main.go`
- [ ] T031 [US4] Create `frontend/templates/shared/editor.html` — same structure as `notes/editor.html` but with "Shared by {owner}" banner and no Archive/Delete/Share in the more-actions menu
- [ ] T032 [US4] Handler tests: edit with edit permission (200 + markdown written to owner's file + git author is collaborator), edit with read permission (403), edit without any share (404), attempted archive/delete on shared note endpoint (confirm `/notes/{slug}/archive` still returns 404 for non-owners)
- [ ] T033 [US4] Integration test: owner edits → commit authored as owner; collaborator edits → commit authored as collaborator; verify via `git log` output parsing in `backend/internal/handlers/handlers_test.go`

**Checkpoint**: Edit-permission collaborators can modify shared notes; attribution flows through version history.

---

## Phase 6: US3 — Owner manages collaborators (P2)

US3 is largely delivered by US1's dialog. This phase just verifies the management flows end-to-end.

- [ ] T034 [US3] Handler test: grant Bob read → change to edit (upsert) → verify permission updated; revoke → verify subsequent access returns 404 in `backend/internal/handlers/handlers_test.go`
- [ ] T035 [US3] UI verification via browser (manual, documented in quickstart.md): dialog lists current collaborators, permission dropdown change is immediate, revoke removes the row

**Checkpoint**: All four user stories complete.

---

## Phase 7: Polish

- [ ] T036 Run full test suite (`make test`) — all pass
- [ ] T037 Run `make lint` — clean
- [ ] T038 Manually verify quickstart.md end-to-end in a browser
- [ ] T039 Update CLAUDE.md note about `note_shares` table if auto-updater missed anything

---

## Dependencies & Execution Order

- Phase 2 (foundational) blocks all user stories
- US1 (P1) — grant/revoke endpoints + owner UI
- US2 (P1) — depends on Phase 2; independent of US1 logic (can be developed in parallel) but tested after to exercise shares granted via US1
- US4 (P2) — depends on US2 (reader template) for the banner pattern; adds the editor flow and attribution
- US3 (P2) — verified after US1/US4 exist; mostly testing
- Polish — last

---

## Notes

- **Constitution VI**: full test suite green before every commit.
- **Constitution IV**: integration-level handler tests required for every new endpoint.
- [P] marks tasks that touch different files and have no intra-story ordering dependency.
