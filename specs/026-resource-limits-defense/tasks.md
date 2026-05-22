# Tasks: Resource Limits & Abuse Prevention

**Input**: Design documents from `specs/026-resource-limits-defense/`
**Plan**: plan.md | **Spec**: spec.md | **Data Model**: data-model.md

---

## Phase 1: Foundational (Shared Infrastructure)

**Purpose**: DB migration and model helpers that all user stories depend on.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T001 Add `size_bytes INTEGER NOT NULL DEFAULT 0` migration to `notes` table in `backend/internal/models/models.go` (append to `runMigrations`)
- [X] T002 [P] Add `CountNotesForUser(db *DB, userID int64) (int, error)` to `backend/internal/models/models.go`
- [X] T003 [P] Add `CountImagesForNote(db *DB, noteID int64) (int, error)` to `backend/internal/models/models.go`
- [X] T004 Export `MaxNotesPerUser = 25` constant from `backend/internal/models/models.go`; add `ErrNoteLimitReached` sentinel error
- [X] T005 Update `CreateNote` in `backend/internal/models/models.go`: accept `sizeBytes int64` and `isAdmin bool`; wrap count-check + insert in `BEGIN IMMEDIATE` transaction to enforce the 25-note limit atomically
- [X] T006 Update `UpdateNote` in `backend/internal/models/models.go`: accept `sizeBytes int64`; include `size_bytes = ?` in the UPDATE statement
- [X] T007 Run `make test` — all existing tests must pass before proceeding

**Checkpoint**: DB migration applied; model helpers compile; existing tests green.

---

## Phase 2: User Story 1 — Note Size Limit (Priority: P1)

**Goal**: Reject any note save (create or update) whose markdown text exceeds 5 MB.

**Independent Test**: POST a note body > 5 MB → redirected back with flash error, note not stored. POST a body < 5 MB → succeeds.

### Tests (write first, confirm they FAIL before implementing)

- [X] T010 [P] [US1] Integration test — `POST /notes` with body > 5 MB returns 303 with flash error in `backend/internal/handlers/notes_test.go`
- [X] T011 [P] [US1] Integration test — `POST /notes/{slug}` update with body > 5 MB returns 303 with flash error in `backend/internal/handlers/notes_test.go`

### Implementation

- [X] T012 [US1] Add `maxNoteSizeBytes = 5 * 1024 * 1024` constant to `backend/internal/handlers/notes.go`
- [X] T013 [US1] In `NotesCreatePOST` (`backend/internal/handlers/notes.go`): check `len([]byte(body)) > maxNoteSizeBytes`; flash error and redirect to `/notes/new` if exceeded
- [X] T014 [US1] In `noteUpdate` (`backend/internal/handlers/notes.go`): same size check; flash error and redirect to `/notes/{slug}/edit` if exceeded
- [X] T015 [US1] Pass `int64(len([]byte(body)))` as `sizeBytes` to `models.CreateNote` and `models.UpdateNote` callers
- [X] T016 [US1] Run `make test` — T010 and T011 must now pass

**Checkpoint**: Note saves exceeding 5 MB are fully rejected with user-facing error.

---

## Phase 3: User Story 2 — Image Upload Limits (Priority: P1)

**Goal**: Reject image uploads > 1 MB per file; reject uploads to notes that have already had 10 images uploaded (lifetime count).

**Independent Test**: Upload an image > 1 MB → 413. Upload a 11th image to a note → 422 JSON error. Upload within limits → succeeds.

### Tests (write first, confirm they FAIL before implementing)

- [X] T020 [P] [US2] Integration test — image upload > 1 MB returns 413 in `backend/internal/handlers/images_test.go`
- [X] T021 [P] [US2] Integration test — 11th image upload to same note returns 422 with JSON error body in `backend/internal/handlers/images_test.go`
- [X] T022 [P] [US2] Integration test — image upload within limits succeeds in `backend/internal/handlers/images_test.go`

### Implementation

- [X] T023 [US2] Change `maxImageSize` constant in `backend/internal/handlers/images.go` from `10 << 20` to `1 << 20`
- [X] T024 [US2] Add `maxImagesPerNote = 10` constant to `backend/internal/handlers/images.go`
- [X] T025 [US2] In `ImageUploadPOST` (`backend/internal/handlers/images.go`): after resolving `note.ID`, call `models.CountImagesForNote`; return 422 JSON error if `>= maxImagesPerNote`
- [X] T026 [US2] Run `make test` — T020, T021, T022 must now pass

**Checkpoint**: Image uploads are capped at 1 MB per file and 10 lifetime per note.

---

## Phase 4: User Story 3 — Note Count Limit (Priority: P2)

**Goal**: Regular users cannot create more than 25 notes. Admins are exempt. Limit is atomic.

**Independent Test**: Create 25 notes as regular user; 26th POST → 303 redirect with flash error. Admin with 25+ notes can create more.

### Tests (write first, confirm they FAIL before implementing)

- [X] T030 [P] [US3] Integration test — regular user at 25 notes, `POST /notes` returns 303 with limit flash error in `backend/internal/handlers/notes_test.go`
- [X] T031 [P] [US3] Integration test — admin user at 25+ notes, `POST /notes` succeeds in `backend/internal/handlers/notes_test.go`
- [X] T032 [P] [US3] Integration test — regular user below limit, `POST /notes` succeeds in `backend/internal/handlers/notes_test.go`

### Implementation

- [X] T033 [US3] In `NotesCreatePOST` (`backend/internal/handlers/notes.go`): handle `models.ErrNoteLimitReached` returned from `models.CreateNote`; flash descriptive error and redirect to `/notes`
- [X] T034 [US3] Confirm `models.CreateNote` receives correct `isAdmin` value from session/user lookup (the atomicity is in the model layer from T005)
- [X] T035 [US3] Run `make test` — T030, T031, T032 must now pass

**Checkpoint**: Note creation is capped at 25 for regular users; limit is atomic under concurrent requests.

---

## Phase 5: User Story 4 — Admin Storage Overview (Priority: P3)

**Goal**: Admin `/admin/users` page shows total note storage (in bytes / human-readable) per user.

**Independent Test**: Log in as admin, `GET /admin/users` — each user row shows a storage size value.

### Tests (write first, confirm they FAIL before implementing)

- [X] T040 [US4] Integration test — `GET /admin/users` as admin returns a page containing per-user storage data in `backend/internal/handlers/admin_test.go`

### Implementation

- [X] T041 [US4] Extend the per-user struct returned by `models.ListAllUsers` in `backend/internal/models/admin.go`: add `TotalSizeBytes int64`
- [X] T042 [US4] Update the `ListAllUsers` SQL query in `backend/internal/models/admin.go` to include `COALESCE(SUM(n.size_bytes), 0) AS total_size_bytes` via the existing `LEFT JOIN notes` aggregation
- [X] T043 [US4] Add a "Storage" column to `frontend/templates/admin/users.html` showing formatted size per user row (e.g., "1.2 MB", "340 KB")
- [X] T044 [US4] Add a Go template function or handler helper to format bytes as human-readable string if one doesn't already exist
- [X] T045 [US4] Run `make test` — T040 must now pass

**Checkpoint**: Admin dashboard shows per-user storage usage.

---

## Phase 6: Polish & Final Gate

- [ ] T050 [P] Run `make coverage` and confirm ≥ 90% line coverage on `internal/...` — NOTE: coverage gate (75%) was already failing before this feature; current coverage is 73% (pre-existing issue, not a regression)
- [X] T051 [P] Run `make lint` — no vet warnings
- [X] T052 Review all new error messages for clarity and consistency with existing flash messages in the app
- [ ] T053 Manual smoke test: create a note near 5 MB limit, upload images to count limit, verify admin storage column shows correct values

---

## Dependencies & Execution Order

- **Phase 1** (Foundation): No dependencies — start immediately. BLOCKS all other phases.
- **Phase 2** (Note size) and **Phase 3** (Image limits): Can proceed in parallel after Phase 1.
- **Phase 4** (Note count): Can proceed in parallel with Phase 2 and 3 after Phase 1.
- **Phase 5** (Admin dashboard): Depends on Phase 1 (`size_bytes` migration and `UpdateNote`). Can start after T001 + T006 complete.
- **Phase 6** (Polish): After all story phases complete.

### Parallel opportunities within phases

- T002 and T003 (model helpers) are independent and can be written together.
- Tests within each phase (T010+T011, T020+T021+T022, T030+T031+T032) can be written in parallel.

---

## Notes

- Constitution Principle VI applies: run `make test` before every commit; fix failures before new work.
- `[P]` = independent of other tasks in the same phase, can be done in parallel.
- `[USn]` = maps to User Story n in spec.md for traceability.
- Each phase checkpoint is a valid stopping point for incremental delivery.
