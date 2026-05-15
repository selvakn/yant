# Tasks: Shared Note Authorship & Indicators

**Input**: Design documents from `specs/025-shared-note-authorship/`  
**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓, contracts/ ✓

**Organization**: Tasks grouped by user story for independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no shared dependencies)
- **[Story]**: User story label (US1, US2, US3)
- Exact file paths included in all task descriptions

---

## Phase 1: Setup (No new infrastructure needed)

**Purpose**: This feature extends existing infrastructure — no new project setup required.

- [x] T001 Verify test suite is green before starting: run `make test` from repo root

---

## Phase 2: Foundational (Blocking Prerequisite for All Stories)

**Purpose**: Extend the `Version` struct and `parseGitLog()` with author information — required by all three user stories.

⚠️ **CRITICAL**: All user story phases depend on this phase completing first.

- [x] T002 Add `AuthorName string` field to `Version` struct in `backend/internal/versioning/git.go`
- [x] T003 Update `Log()` format string from `%H|%h|%aI|%s` to `%H|%h|%aI|%s|%an` in `backend/internal/versioning/git.go`
- [x] T004 Update `GetVersion()` format string from `%H|%h|%aI|%s` to `%H|%h|%aI|%s|%an` in `backend/internal/versioning/git.go`
- [x] T005 Update `parseGitLog()` to parse 5 fields with `SplitN(line, "|", 5)` and set `AuthorName = parts[4]`; leave `AuthorName` empty when fewer than 5 parts in `backend/internal/versioning/git.go`
- [x] T006 Add test `TestLog_IncludesAuthorName` verifying `AuthorName` is populated from git commits in `backend/internal/versioning/git_test.go`
- [x] T007 Run `make test` — all existing tests must remain green before continuing

**Checkpoint**: `Version.AuthorName` is populated from git log. `make test` is green.

---

## Phase 3: User Story 1 — Version History Shows Author (Priority: P1) 🎯 MVP

**Goal**: Each entry in the version history list shows the username of the editor who made that save, for both owner and shared note history.

**Independent Test**: Create two user accounts. Share a note; have the second user make an edit. Open the history page as the owner — each row should show the editor's username in an "Author" column.

### Implementation for User Story 1

- [ ] T008 In `noteUpdate()` in `backend/internal/handlers/notes.go`: replace `versioning.CommitFile(...)` with `versioning.CommitFileAs(h.notesDir, relPath, "update: "+slug, username, username+"@yant.local")` where `username := usernameFromSession(r)`
- [ ] T009 In `NotesCreatePOST()` in `backend/internal/handlers/notes.go`: replace `versioning.CommitFile(...)` with `versioning.CommitFileAs(h.notesDir, relPath, "create: "+slug, username, username+"@yant.local")` where `username := usernameFromSession(r)`
- [ ] T010 In `DrawingByIDPUT()` in `backend/internal/handlers/drawings.go`: replace `versioning.CommitFile(...)` with `versioning.CommitFileAs(h.notesDir, relPath, "update drawing: "+slug+"/"+drawingID, username, username+"@yant.local")` where `username := usernameFromSession(r)`
- [ ] T011 Add "Author" column header `<th>Author</th>` after the Date header in `frontend/templates/notes/history.html`
- [ ] T012 Add author cell `<td class="history-author">{{if .AuthorName}}{{.AuthorName}}{{else}}—{{end}}</td>` in the `{{range .Versions}}` row in `frontend/templates/notes/history.html`
- [ ] T013 [US1] Add new handler `SharedNoteHistoryGET` to `backend/internal/handlers/shares.go`: verify viewer access via `models.GetNoteForViewer`, use `note.UserID` to construct `relPath`, call `versioning.Log()` with pagination, render `shared/history.html`
- [ ] T014 [US1] Register route `r.Get("/shared/{username}/{slug}/history", h.SharedNoteHistoryGET)` in `backend/cmd/server/main.go` after the existing shared routes
- [ ] T015 [US1] Create `frontend/templates/shared/history.html`: topbar with back link to `/shared/{{.OwnerUsername}}/{{.Note.Slug}}`, same Author-column table as notes/history.html, pagination, no Revert button
- [ ] T016 [US1] Add link to shared note history in `frontend/templates/shared/reader.html`: topbar actions, `<a href="/shared/{{.OwnerUsername}}/{{.Note.Slug}}/history" class="topbar-history" title="Version history">History</a>`
- [ ] T017 [US1] Add test `TestNoteUpdate_CommitsWithAuthorName` in `backend/internal/handlers/handlers_test.go`: create note as user, make edit, verify most recent `versioning.Log()` entry has non-empty `AuthorName` matching the session user
- [ ] T018 [US1] Add test `TestSharedNoteHistoryGET_ShowsVersions` in `backend/internal/handlers/history_test.go` (or a new `backend/internal/handlers/shares_history_test.go`): create note owner + collaborator, have collaborator edit the note, GET `/shared/{username}/{slug}/history` as collaborator, verify 200 response and `Versions` in template data
- [ ] T019 [US1] Run `make test` — all tests must pass before committing

**Checkpoint**: Owner's history page shows Author column. Shared history route works. `make test` is green. Commit.

---

## Phase 4: User Story 2 — Last Updated Author on Reader Pages (Priority: P2)

**Goal**: The note reader page (both owner and shared) shows "by [username]" next to the last-modified timestamp, using the most recent git commit's author.

**Independent Test**: Have a collaborator edit a shared note. Open the note reader as the owner — the topbar should display "by [collaborator-username]" after the timestamp.

### Implementation for User Story 2

- [ ] T020 [P] [US2] In `NoteReaderGET()` in `backend/internal/handlers/notes.go`: after note is fetched, call `versioning.Log(h.notesDir, relPath, 1, 0)` and set `lastEditor = versions[0].AuthorName` (empty if AuthorName is "" or "yant"); add `"LastEditor": lastEditor` to the template data map
- [ ] T021 [P] [US2] In `SharedNoteReaderGET()` in `backend/internal/handlers/shares.go`: same pattern — use `note.UserID` for `relPath`, get last version, set `lastEditor`, add `"LastEditor": lastEditor` to template data
- [ ] T022 [P] [US2] In `frontend/templates/notes/reader.html`: after the `<time>` element in the topbar, add `{{if .LastEditor}}<span class="last-editor">by {{.LastEditor}}</span>{{end}}`
- [ ] T023 [P] [US2] In `frontend/templates/shared/reader.html`: after the `<time>` element in the topbar, add `{{if .LastEditor}}<span class="last-editor">by {{.LastEditor}}</span>{{end}}`
- [ ] T024 [US2] Add test `TestNoteReaderGET_ShowsLastEditor` in `backend/internal/handlers/handlers_test.go`: create note, save edit as named user, GET `/notes/{slug}`, verify response body or template data contains the editor's username
- [ ] T025 [US2] Run `make test` — all tests must pass before committing

**Checkpoint**: Note reader topbar shows "by [username]" for both owner and shared views. `make test` is green. Commit.

---

## Phase 5: User Story 3 — Share Indicators on Notes Lists (Priority: P2)

**Goal**: The `/notes` list shows an outgoing badge ("↑ Shared with N") on notes the owner has shared. The `/shared` list shows an incoming badge ("↓ [OwnerName]") on each shared note.

**Independent Test**: Share a note with another user. Open `/notes` as the owner — the shared note shows "↑ Shared with 1". Revoke access — badge disappears on next load.

### Implementation for User Story 3

- [ ] T026 [US3] Add `ListShareCountsForOwner(db *DB, userID int64) (map[int64]int, error)` to `backend/internal/models/shares.go`: query `SELECT ns.note_id, COUNT(*) FROM note_shares ns JOIN notes n ON n.id=ns.note_id WHERE n.user_id=? AND n.archived=0 GROUP BY ns.note_id`; return `map[int64]int` (note ID → collaborator count)
- [ ] T027 [US3] In `NotesListGET()` in `backend/internal/handlers/notes.go`: after `models.ListNotes(...)`, call `models.ListShareCountsForOwner(h.db, userID)` and add `"ShareStates": shareStates` to the template data map
- [ ] T028 [US3] In `frontend/templates/notes/list.html`: inside `{{range .Notes}}`, before the `<time>` element, add `{{$count := index $.ShareStates .ID}}{{if gt $count 0}}<span class="share-badge-out" title="Shared with {{$count}} user{{if gt $count 1}}s{{end}}">↑ {{$count}}</span>{{end}}`
- [ ] T029 [US3] In `frontend/templates/shared/list.html`: replace the existing `<span class="shared-by-label">` with `<span class="share-badge-in" title="Shared with you by {{.OwnerUsername}}">↓ {{.OwnerUsername}}</span>` and remove the now-redundant separate "Shared by" text
- [ ] T030 [US3] Add test `TestListShareCountsForOwner` in `backend/internal/models/shares_test.go` (create if missing) or `backend/internal/handlers/handlers_test.go`: insert shares in DB, call `ListShareCountsForOwner`, verify correct counts; verify archived notes excluded; verify notes with no shares absent from map
- [ ] T031 [US3] Run `make test` — all tests must pass before committing

**Checkpoint**: Outgoing badge visible on `/notes` list for shared notes. Incoming badge visible on `/shared` list. Badge disappears after revoking. `make test` is green. Commit.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: CSS styling for new indicators, final validation.

- [ ] T032 [P] Add CSS rules for `.share-badge-out`, `.share-badge-in`, `.last-editor` to `frontend/static/css/main.css` (or the project's primary stylesheet): muted color, small font-size, no-wrap
- [ ] T033 [P] Add CSS rule for `.history-author` column in `frontend/static/css/main.css`: consistent with existing `.history-message` and `.history-stats` column widths
- [ ] T034 Run `make coverage` and verify ≥90% line coverage on `backend/internal/...` is maintained
- [ ] T035 Manual walkthrough of quickstart.md scenarios to verify all three user stories work end-to-end

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies — start immediately
- **Phase 2 (Foundational)**: Depends on Phase 1 — **BLOCKS all user stories**
- **Phase 3 (US1)**: Depends on Phase 2 — all T002–T007 must be complete
- **Phase 4 (US2)**: Depends on Phase 2 — can start in parallel with Phase 3 after T007
- **Phase 5 (US3)**: Depends on Phase 2 — can start in parallel with Phase 3 and 4 after T007
- **Phase 6 (Polish)**: Depends on Phases 3, 4, 5 all complete

### User Story Dependencies

- **US1 (P1)**: Must complete Phase 2 first — no dependency on US2 or US3
- **US2 (P2)**: Must complete Phase 2 first — no dependency on US1 or US3
- **US3 (P2)**: Must complete Phase 2 first — no dependency on US1 or US2

### Within Each Phase

- T008, T009, T010 (owner commit attribution) can run in parallel — different files
- T011, T012 (history template) must be sequential (same file)
- T013, T014, T015, T016 can run in parallel after T011/T012 (different files)
- T020, T021, T022, T023 are all independent — different files, run in parallel
- T026, T027, T028, T029 sequential per logical dependency (model → handler → template)

---

## Parallel Example: US2 (Last Editor)

```
# All four tasks are in different files — launch together:
T020: backend/internal/handlers/notes.go     (NoteReaderGET last-editor lookup)
T021: backend/internal/handlers/shares.go    (SharedNoteReaderGET last-editor lookup)
T022: frontend/templates/notes/reader.html   (owner reader template)
T023: frontend/templates/shared/reader.html  (shared reader template)
```

---

## Implementation Strategy

### MVP First (US1 — Version History Authorship)

1. Complete Phase 1: Run green baseline (`make test`)
2. Complete Phase 2: Extend Version struct + parseGitLog (T002–T007)
3. Complete Phase 3: US1 tasks T008–T019
4. **STOP and VALIDATE**: Open history page, verify Author column shows usernames
5. Commit, then continue to US2 and US3

### Incremental Delivery

1. Phase 2 → Foundation
2. Phase 3 → US1: History authorship (MVP, highest user value)
3. Phase 4 → US2: Last-editor on reader pages (quick follow-on)
4. Phase 5 → US3: Share indicators on lists (completes the feature)
5. Phase 6 → Polish (CSS, coverage check)

---

## Notes

- All commits must pass `make test` (Constitution Principle VI)
- `make coverage` gate: ≥90% on `backend/internal/...` (Constitution Principle IV)
- `AuthorName == "yant"` is treated as legacy/unknown — rendered as "—" in templates
- The `CommitFileAs` signature already exists; no new versioning API needed
- The `SharedNoteHistoryGET` handler reuses the same template pattern as `NoteHistoryGET`
- No new DB migrations required; no new tables or columns
