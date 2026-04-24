# Tasks: Admin Dashboard and User Management

**Input**: Design documents from `/specs/017-admin-dashboard/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/admin-routes.md, quickstart.md

**Tests**: Required by constitution (Principle IV: integration tests ≥90% coverage, Principle VI: test-before-commit).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup

**Purpose**: Schema migration, admin model layer, and middleware infrastructure

- [ ] T001 Add `is_admin` and `disabled` column migrations to `migrateSchema` in `backend/internal/models/models.go`
- [ ] T002 Add `admin_audit_log` table creation to `migrateSchema` in `backend/internal/models/models.go`
- [ ] T003 [P] Create admin model structs (`AdminUserView`, `AuditLogEntry`, `DashboardMetrics`) and audit log write function in `backend/internal/models/admin.go`
- [ ] T004 [P] Add `ADMIN_USER` env var parsing and bootstrap logic in `backend/cmd/server/main.go`
- [ ] T005 Update `User` struct to include `IsAdmin` and `Disabled` fields and update `scanUser`/`GetOrCreateUser` in `backend/internal/models/models.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Middleware and navigation that ALL user stories depend on

**CRITICAL**: No user story work can begin until this phase is complete

- [ ] T006 Add disabled-user check middleware (`RequireActive`) that destroys session and redirects to `/login?error=disabled` in `backend/internal/auth/auth.go`
- [ ] T007 Implement `AdminOnly` middleware returning 404 for non-admin users in `backend/internal/handlers/admin.go`
- [ ] T008 Add `IsAdmin` flag to `baseData()` template context in `backend/internal/handlers/handlers.go`
- [ ] T009 Add conditional "Admin" nav link in `frontend/templates/base.html`
- [ ] T010 Register `/admin` route group with `RequireLogin`, `RequireActive`, and `AdminOnly` middleware in `backend/cmd/server/main.go`
- [ ] T011 Add login error message for disabled accounts (`error=disabled`) in `backend/internal/handlers/auth.go` and `frontend/templates/login.html`
- [ ] T012 Write tests for `RequireActive` middleware (disabled user redirected, active user passes) in `backend/internal/handlers/admin_test.go`
- [ ] T013 Write tests for `AdminOnly` middleware (admin passes, non-admin gets 404) in `backend/internal/handlers/admin_test.go`
- [ ] T014 Write tests for admin bootstrap logic (env var seeds admin on startup, additive on restart) in `backend/internal/models/admin_test.go`

**Checkpoint**: Foundation ready — admin middleware protects routes, disabled users blocked, nav link visible to admins only

---

## Phase 3: User Story 1 - Admin Dashboard with Platform Metrics (Priority: P1)

**Goal**: Admin sees a dashboard with platform-wide metrics (total users, active users 30d, total notes, notes 7d, public notes, active shares)

**Independent Test**: Log in as admin, navigate to `/admin`, verify all metric counts match actual data

### Tests for User Story 1

- [ ] T015 [P] [US1] Write test for `GetDashboardMetrics` model function returning correct counts in `backend/internal/models/admin_test.go`
- [ ] T016 [P] [US1] Write test for `AdminDashboardGET` handler rendering metrics page for admin and 404 for non-admin in `backend/internal/handlers/admin_test.go`

### Implementation for User Story 1

- [ ] T017 [US1] Implement `GetDashboardMetrics` query function in `backend/internal/models/admin.go`
- [ ] T018 [US1] Implement `AdminDashboardGET` handler in `backend/internal/handlers/admin.go`
- [ ] T019 [US1] Create dashboard template with metrics cards in `frontend/templates/admin/dashboard.html`
- [ ] T020 [US1] Register `GET /admin` route in `backend/cmd/server/main.go`

**Checkpoint**: Admin dashboard functional — navigate to `/admin` and see live platform metrics

---

## Phase 4: User Story 2 - User Management (Priority: P1)

**Goal**: Admin can list, search, view detail, disable/enable, promote/demote, and delete users

**Independent Test**: Log in as admin, search for a user, disable them, verify they cannot log in, re-enable, verify they can log in again

### Tests for User Story 2

- [ ] T021 [P] [US2] Write tests for `ListAllUsers`, `GetAdminUserDetail`, `DisableUser`, `EnableUser`, `DeleteUserCascade`, `PromoteAdmin`, `DemoteAdmin`, `CountAdminUsers` model functions in `backend/internal/models/admin_test.go`
- [ ] T022 [P] [US2] Write tests for user management handlers (list, detail, disable, enable, promote, demote, delete-confirm, delete) including authorization and edge cases (self-delete, last-admin demote) in `backend/internal/handlers/admin_test.go`

### Implementation for User Story 2

- [ ] T023 [US2] Implement `ListAllUsers` (paginated, searchable) and `GetAdminUserDetail` model functions in `backend/internal/models/admin.go`
- [ ] T024 [US2] Implement `DisableUser` (set flag + delete sessions) and `EnableUser` model functions in `backend/internal/models/admin.go`
- [ ] T025 [US2] Implement `PromoteAdmin`, `DemoteAdmin`, and `CountAdminUsers` (last-admin guard) model functions in `backend/internal/models/admin.go`
- [ ] T026 [US2] Implement `DeleteUserCascade` model function (DB cascade + filesystem cleanup) in `backend/internal/models/admin.go`
- [ ] T027 [US2] Implement `GetUserImpactSummary` model function (note/share/public counts for confirmation dialog) in `backend/internal/models/admin.go`
- [ ] T028 [US2] Implement user list handler `AdminUsersListGET` with search and pagination in `backend/internal/handlers/admin.go`
- [ ] T029 [US2] Implement user detail handler `AdminUserDetailGET` in `backend/internal/handlers/admin.go`
- [ ] T030 [US2] Implement `AdminUserDisablePOST` and `AdminUserEnablePOST` handlers in `backend/internal/handlers/admin.go`
- [ ] T031 [US2] Implement `AdminUserPromotePOST` and `AdminUserDemotePOST` handlers with last-admin guard in `backend/internal/handlers/admin.go`
- [ ] T032 [US2] Implement `AdminUserDeleteConfirmGET` and `AdminUserDeleteDELETE` handlers in `backend/internal/handlers/admin.go`
- [ ] T033 [P] [US2] Create user list template with search bar, pagination, and admin/disabled badges in `frontend/templates/admin/users.html`
- [ ] T034 [P] [US2] Create user detail template showing notes, shares, public notes in `frontend/templates/admin/user-detail.html`
- [ ] T035 [US2] Register all user management routes (`/admin/users/*`) in `backend/cmd/server/main.go`

**Checkpoint**: Full user management functional — list, search, disable/enable, promote/demote, delete with cascade

---

## Phase 5: User Story 3 - Note Administration and Moderation (Priority: P2)

**Goal**: Admin can browse all notes with filters, view any note read-only, and delete notes

**Independent Test**: Log in as admin, browse notes, filter by user, open a note, delete it, verify it's removed from owner's list

### Tests for User Story 3

- [ ] T036 [P] [US3] Write tests for `ListAllNotes` (with filters), `GetNoteForAdmin`, `AdminDeleteNote` model functions in `backend/internal/models/admin_test.go`
- [ ] T037 [P] [US3] Write tests for note admin handlers (list with filters, detail read-only, delete-confirm, delete cascade) in `backend/internal/handlers/admin_test.go`

### Implementation for User Story 3

- [ ] T038 [US3] Implement `ListAllNotes` (paginated, filterable by owner/public/shared) model function in `backend/internal/models/admin.go`
- [ ] T039 [US3] Implement `GetNoteForAdmin` (fetch note by ID with owner info) and `GetNoteImpactSummary` model functions in `backend/internal/models/admin.go`
- [ ] T040 [US3] Implement `AdminDeleteNote` model function (DB cascade + filesystem cleanup) in `backend/internal/models/admin.go`
- [ ] T041 [US3] Implement `AdminNotesListGET` handler with filter query params in `backend/internal/handlers/admin.go`
- [ ] T042 [US3] Implement `AdminNoteDetailGET` handler with read-only markdown rendering in `backend/internal/handlers/admin.go`
- [ ] T043 [US3] Implement `AdminNoteDeleteConfirmGET` and `AdminNoteDeleteDELETE` handlers in `backend/internal/handlers/admin.go`
- [ ] T044 [P] [US3] Create note list template with owner/public/shared filters and pagination in `frontend/templates/admin/notes.html`
- [ ] T045 [P] [US3] Create read-only note detail template (rendered markdown, metadata, no edit controls) in `frontend/templates/admin/note-detail.html`
- [ ] T046 [US3] Register all note admin routes (`/admin/notes/*`) in `backend/cmd/server/main.go`

**Checkpoint**: Note moderation functional — browse, filter, view, and delete any note

---

## Phase 6: User Story 4 - Public Notes Management (Priority: P2)

**Goal**: Admin can list all public notes and unpublish any note

**Independent Test**: Create a public note as regular user, log in as admin, find it in public notes view, unpublish, verify public URL returns 404

### Tests for User Story 4

- [ ] T047 [P] [US4] Write tests for `ListAllPublicNotes` and `AdminUnpublishNote` model functions in `backend/internal/models/admin_test.go`
- [ ] T048 [P] [US4] Write tests for public notes handlers (list, unpublish) in `backend/internal/handlers/admin_test.go`

### Implementation for User Story 4

- [ ] T049 [US4] Implement `ListAllPublicNotes` (all users, paginated) model function in `backend/internal/models/admin.go`
- [ ] T050 [US4] Implement `AdminUnpublishNote` model function (reuses existing `UnpublishNote`) in `backend/internal/models/admin.go`
- [ ] T051 [US4] Implement `AdminPublicNotesListGET` and `AdminPublicNoteUnpublishPOST` handlers in `backend/internal/handlers/admin.go`
- [ ] T052 [US4] Create public notes list template with unpublish buttons in `frontend/templates/admin/public-notes.html`
- [ ] T053 [US4] Register public notes admin routes (`/admin/public-notes/*`) in `backend/cmd/server/main.go`

**Checkpoint**: Public notes management functional — list and unpublish any public note

---

## Phase 7: User Story 5 - Sharing Oversight (Priority: P3)

**Goal**: Admin can see all active shares and revoke any share

**Independent Test**: View sharing overview, find a share, revoke it, verify collaborator lost access

### Tests for User Story 5

- [ ] T054 [P] [US5] Write tests for `ListAllShares` and `AdminRevokeShare` model functions in `backend/internal/models/admin_test.go`
- [ ] T055 [P] [US5] Write tests for shares handlers (list with user filter, revoke) in `backend/internal/handlers/admin_test.go`

### Implementation for User Story 5

- [ ] T056 [US5] Implement `ListAllShares` (paginated, filterable by user) model function in `backend/internal/models/admin.go`
- [ ] T057 [US5] Implement `AdminRevokeShare` model function in `backend/internal/models/admin.go`
- [ ] T058 [US5] Implement `AdminSharesListGET` and `AdminShareRevokeDELETE` handlers in `backend/internal/handlers/admin.go`
- [ ] T059 [US5] Create shares overview template with user filter and revoke buttons in `frontend/templates/admin/shares.html`
- [ ] T060 [US5] Register shares admin routes (`/admin/shares/*`) in `backend/cmd/server/main.go`

**Checkpoint**: Sharing oversight functional — list and revoke any share

---

## Phase 8: User Story 6 - Admin Audit Trail (Priority: P3)

**Goal**: All admin actions logged, viewable audit log with filtering

**Independent Test**: Perform admin actions, open audit log, verify each action recorded with correct details

### Tests for User Story 6

- [ ] T061 [P] [US6] Write tests for `WriteAuditLog` and `ListAuditLog` (with filters) model functions in `backend/internal/models/admin_test.go`
- [ ] T062 [P] [US6] Write tests for audit log handler (list, filter by action/user) in `backend/internal/handlers/admin_test.go`

### Implementation for User Story 6

- [ ] T063 [US6] Implement `ListAuditLog` (paginated, filterable by action type and target user) model function in `backend/internal/models/admin.go`
- [ ] T064 [US6] Add `WriteAuditLog` calls to all admin action handlers (disable, enable, delete user, promote, demote, delete note, unpublish, revoke share) in `backend/internal/handlers/admin.go`
- [ ] T065 [US6] Implement `AdminAuditLogGET` handler in `backend/internal/handlers/admin.go`
- [ ] T066 [US6] Create audit log template with action/user filters and pagination in `frontend/templates/admin/audit-log.html`
- [ ] T067 [US6] Register audit log route (`/admin/audit-log`) in `backend/cmd/server/main.go`

**Checkpoint**: Audit trail complete — every admin action logged and viewable

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Integration validation, edge cases, and cleanup

- [ ] T068 Write integration test: disabled user's public notes return 404 for unauthenticated visitors in `backend/internal/handlers/admin_test.go`
- [ ] T069 Write integration test: user deletion cascades to filesystem (markdown + uploads removed) in `backend/internal/handlers/admin_test.go`
- [ ] T070 Write integration test: non-admin user gets 404 on all `/admin/*` routes in `backend/internal/handlers/admin_test.go`
- [ ] T071 Write integration test: last-admin guard prevents demotion and deletion in `backend/internal/handlers/admin_test.go`
- [ ] T072 Verify `make test` and `make coverage` pass with ≥75% threshold
- [ ] T073 Run `make lint` and fix any vet warnings
- [ ] T074 Validate quickstart.md flow end-to-end (build, run with ADMIN_USER, verify admin access)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion — BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational (Phase 2)
- **User Story 2 (Phase 4)**: Depends on Foundational (Phase 2)
- **User Story 3 (Phase 5)**: Depends on Foundational (Phase 2)
- **User Story 4 (Phase 6)**: Depends on Foundational (Phase 2)
- **User Story 5 (Phase 7)**: Depends on Foundational (Phase 2)
- **User Story 6 (Phase 8)**: Depends on Foundational (Phase 2) — audit log writes are added to handlers from US2-US5
- **Polish (Phase 9)**: Depends on all user stories being complete

### User Story Dependencies

- **US1 (Dashboard)**: Independent — no dependency on other stories
- **US2 (User Mgmt)**: Independent — no dependency on other stories
- **US3 (Note Admin)**: Independent — no dependency on other stories
- **US4 (Public Notes)**: Independent — no dependency on other stories
- **US5 (Sharing)**: Independent — no dependency on other stories
- **US6 (Audit Trail)**: Depends on US1-US5 handlers existing (adds audit log calls to them)

### Within Each User Story

- Tests written FIRST, must FAIL before implementation
- Model functions before handler functions
- Handlers before templates
- Route registration last
- All tests must PASS before committing (Principle VI)

### Parallel Opportunities

- T003 and T004 (setup) can run in parallel
- T015 and T016 (US1 tests) can run in parallel
- T021 and T022 (US2 tests) can run in parallel
- T033 and T034 (US2 templates) can run in parallel
- T036 and T037 (US3 tests) can run in parallel
- T044 and T045 (US3 templates) can run in parallel
- T047 and T048 (US4 tests) can run in parallel
- T054 and T055 (US5 tests) can run in parallel
- T061 and T062 (US6 tests) can run in parallel
- All user stories (Phase 3-8) can run in parallel once Phase 2 is complete

---

## Parallel Example: User Story 2

```bash
# Launch tests for User Story 2 together:
Task: "Tests for ListAllUsers, DisableUser, etc. in backend/internal/models/admin_test.go"
Task: "Tests for user management handlers in backend/internal/handlers/admin_test.go"

# Launch templates for User Story 2 together:
Task: "Create user list template in frontend/templates/admin/users.html"
Task: "Create user detail template in frontend/templates/admin/user-detail.html"
```

---

## Implementation Strategy

### MVP First (User Stories 1 + 2 Only)

1. Complete Phase 1: Setup (schema + models + bootstrap)
2. Complete Phase 2: Foundational (middleware + nav + routes)
3. Complete Phase 3: US1 Dashboard (metrics visible)
4. Complete Phase 4: US2 User Management (the core admin capability)
5. **STOP and VALIDATE**: Test admin dashboard + user management independently
6. Deploy if ready — this is a viable admin MVP

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. Add US1 (Dashboard) → Deploy (admin can see metrics)
3. Add US2 (User Mgmt) → Deploy (admin can manage users — MVP!)
4. Add US3 (Note Admin) → Deploy (admin can moderate content)
5. Add US4 (Public Notes) → Deploy (admin can manage public exposure)
6. Add US5 (Sharing) → Deploy (admin can oversee sharing)
7. Add US6 (Audit Trail) → Deploy (all actions logged — feature complete)
8. Polish → Final validation and release

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- **Constitution**: Run `make test` before every commit. Fix failures before new work (Principle VI)
- Commit after each task or logical group (only when tests pass)
- Stop at any checkpoint to validate story independently
- US6 (Audit Trail) is best implemented last since it adds logging calls to all other handlers
