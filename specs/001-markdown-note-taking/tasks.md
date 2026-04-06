---
description: "Task list for Markdown Note Taking App — Go backend"
---

# Tasks: Markdown Note Taking App

**Input**: Design documents from `/specs/001-markdown-note-taking/`
**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/api.md ✅

**Tests**: Integration tests are MANDATORY per Constitution Principle IV (≥90% backend coverage).

**Organization**: Tasks grouped by user story for independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story (US1, US2, US3)
- Exact file paths required on all tasks

## Path Conventions

- Backend source: `backend/internal/`
- Backend entry point: `backend/cmd/server/`
- Backend tests: `backend/internal/{package}/{package}_test.go`
- Frontend templates: `frontend/templates/`
- Frontend static: `frontend/static/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and directory structure

- [x] T001 Create directory tree: `backend/cmd/server/`, `backend/internal/auth/`, `backend/internal/handlers/`, `backend/internal/models/`, `backend/internal/storage/`, `backend/notes/`, `backend/uploads/`
- [x] T002 Create directory tree: `frontend/static/vendor/`, `frontend/static/js/`, `frontend/static/css/`, `frontend/templates/notes/`, `frontend/templates/tags/`
- [x] T003 [P] Create `backend/go.mod` with module `github.com/selvakn/yant` and Go 1.22; add dependencies: `github.com/go-chi/chi/v5`, `github.com/yuin/goldmark`, `github.com/alexedwards/scs/v2`, `modernc.org/sqlite`; run `go mod tidy`
- [x] T004 [P] Download and vendor `htmx.min.js` (v2.x) into `frontend/static/vendor/htmx.min.js`
- [x] T005 [P] Download and vendor `easymde.min.js` and `easymde.min.css` (v2.x) into `frontend/static/vendor/`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T006 Create `backend/internal/models/models.go`: define `DB` type wrapping `*sql.DB`; implement `Open(path string) (*DB, error)` using `modernc.org/sqlite`; implement `InitSchema(db *DB) error` creating tables: `users` (id INTEGER PK AUTOINCREMENT, username TEXT UNIQUE NOT NULL, created_at TEXT NOT NULL), `notes` (id INTEGER PK AUTOINCREMENT, user_id INTEGER NOT NULL REFERENCES users(id), slug TEXT NOT NULL, title TEXT NOT NULL DEFAULT 'Untitled Note', created_at TEXT NOT NULL, updated_at TEXT NOT NULL, UNIQUE(user_id,slug)), `note_tags` (note_id INTEGER NOT NULL REFERENCES notes(id), tag_name TEXT NOT NULL, PRIMARY KEY(note_id,tag_name)), `images` (id INTEGER PK AUTOINCREMENT, note_id INTEGER NOT NULL REFERENCES notes(id), filename TEXT NOT NULL, original TEXT NOT NULL, mime_type TEXT NOT NULL, size INTEGER NOT NULL); create indexes: `idx_note_user ON notes(user_id)`, `idx_tag_name_note ON note_tags(tag_name,note_id)`, `idx_image_note ON images(note_id)`
- [x] T007 [P] Create `backend/internal/storage/storage.go`: implement `EnsureUserDir(root string, userID int64) error`; `WriteNote(root string, userID int64, slug, body string) error`; `ReadNote(root string, userID int64, slug string) (string, error)`; `DeleteNoteFile(root string, userID int64, slug string) error`; `EnsureUploadsDir(root string, userID int64) error`
- [x] T008 [P] Create `backend/internal/auth/auth.go`: import `github.com/alexedwards/scs/v2`; define package-level `SessionManager *scs.SessionManager` initialized with `scs.New()` and in-memory store; implement `RequireLogin(next http.Handler) http.Handler` middleware that checks `session.GetString(r.Context(), "username")` and redirects to `/login` if empty; implement `CurrentUsername(r *http.Request) string` helper
- [x] T009 Create `frontend/templates/base.html`: HTML5 layout with `{{define "base"}}`, `<nav>` with app name "YANT" and logout form (`POST /logout`), `<main>{{template "content" .}}</main>`, `<script src="/static/vendor/htmx.min.js">`, `<script src="/static/vendor/easymde.min.js">`, `<link rel="stylesheet" href="/static/vendor/easymde.min.css">`, `<link rel="stylesheet" href="/static/css/app.css">`
- [x] T010 Create `backend/cmd/server/main.go`: parse flags (`-addr`, `-db`, `-notes`, `-uploads`, `-rebuild-db`); call `models.Open` + `models.InitSchema`; configure `scs.SessionManager` from `auth` package; build Chi router `r := chi.NewRouter()`; register middleware: `r.Use(sessionManager.LoadAndSave)`, logger, recoverer; mount static files at `/static/` from `frontend/static/`; parse templates from `frontend/templates/`; register route groups: `/` (auth routes), `/notes` (note routes), `/tags` (tag route), `/uploads` (image serve route); run `http.ListenAndServe`

**Checkpoint**: App starts, serves static files, redirects unauthenticated requests to /login

---

## Phase 3: User Story 1 — Create and Edit a Note (Priority: P1) 🎯 MVP

**Goal**: Users log in with a username, create notes with Markdown body and title, switch between editor and reader mode, see created_at and updated_at timestamps.

**Independent Test**: `POST /login username=alice` → `POST /notes title=Hello body=# Hi` → `GET /notes/hello` verifies rendered HTML + timestamps → reopen → content persists.

### Integration Tests for User Story 1 ⚠️ WRITE FIRST — MUST FAIL BEFORE IMPLEMENTATION

> **Write these tests, run `go test ./internal/handlers/...` — confirm FAIL — then implement**

- [x] T011 [P] [US1] Write `backend/internal/handlers/handlers_test.go` (auth tests): use `httptest.NewServer`; test `POST /login` with `username=alice` returns 302 to `/notes`; test `POST /login` with empty username returns 400; test `POST /login` with new username auto-creates user row in DB; test `POST /logout` clears session and redirects to `/login`; test `GET /notes` without session redirects to `/login`
- [x] T012 [P] [US1] Add to `backend/internal/handlers/handlers_test.go` (notes tests): test `POST /notes` creates SQLite row AND writes `.md` file at `notes/{userID}/{slug}.md`; test empty title defaults to "Untitled Note"; test `GET /notes/{slug}` returns page with rendered Markdown (goldmark output); test `GET /notes/{slug}/edit` returns page with raw Markdown in textarea; test `PUT /notes/{slug}` (via POST + `X-HTTP-Method-Override: PUT`) updates DB `updated_at` AND overwrites `.md` file; test `DELETE /notes/{slug}` removes DB row AND deletes `.md` file; test `GET /notes` lists only the logged-in user's notes
- [x] T013 [P] [US1] Create `backend/internal/storage/storage_test.go`: test `WriteNote` creates file at correct path; test `ReadNote` returns written content; test `DeleteNoteFile` removes file; test `EnsureUserDir` creates directory if missing; use `t.TempDir()` for all paths
- [x] T014 [P] [US1] Create `backend/internal/models/models_test.go` (user + note queries): use in-memory SQLite (`:memory:`); test `GetOrCreateUser` creates new user on first call and returns existing on second; test `CreateNote` inserts row and returns populated `Note` struct; test `GetNote` returns correct note for owner, nil for wrong user; test `ListNotes` returns only notes owned by given user_id; test `UpdateNote` changes title and updated_at; test `DeleteNote` removes row

### Implementation for User Story 1

- [x] T015 [US1] Add user query functions to `backend/internal/models/models.go`: `GetUserByUsername(db *DB, username string) (*User, error)`; `CreateUser(db *DB, username string) (*User, error)`; `GetOrCreateUser(db *DB, username string) (*User, error)`; define `User` struct: `{ID int64, Username string, CreatedAt time.Time}`
- [x] T016 [US1] Add note query functions to `backend/internal/models/models.go`: `GenerateSlug(db *DB, userID int64, title string) (string, error)` (slugify title, append `-2`, `-3` on collision); `CreateNote(db *DB, userID int64, title, slug string) (*Note, error)`; `GetNote(db *DB, userID int64, slug string) (*Note, error)`; `ListNotes(db *DB, userID int64, tag string) ([]*Note, error)`; `UpdateNote(db *DB, userID int64, slug, title string) (*Note, error)`; `DeleteNote(db *DB, userID int64, slug string) error`; define `Note` struct: `{ID int64, UserID int64, Slug, Title string, Tags []string, CreatedAt, UpdatedAt time.Time}`
- [x] T017 [US1] Create `backend/internal/handlers/auth.go`: `LoginGET(w, r)` renders `login.html`; `LoginPOST(w, r)` validates username, calls `models.GetOrCreateUser`, sets `session.Put(ctx, "username", user.Username)`, redirects to `/notes`; `LogoutPOST(w, r)` calls `sessionManager.Destroy(ctx)`, redirects to `/login`
- [x] T018 [US1] Create `backend/internal/handlers/notes.go`: `NotesListGET(w, r)` calls `models.ListNotes` filtered by optional `?tag=` param, renders `notes/list.html`; `NotesCreatePOST(w, r)` calls `models.CreateNote` + `storage.WriteNote`, redirects to `/notes/{slug}/edit`
- [x] T019 [US1] Add to `backend/internal/handlers/notes.go`: `NoteReaderGET(w, r)` calls `storage.ReadNote`, renders Markdown with goldmark (`goldmark.Convert`), renders `notes/reader.html`; `NoteEditorGET(w, r)` calls `storage.ReadNote` for raw body, renders `notes/editor.html`
- [x] T020 [US1] Add to `backend/internal/handlers/notes.go`: `NoteUpdatePUT(w, r)` (handles POST with `X-HTTP-Method-Override: PUT`) calls `models.UpdateNote` + `storage.WriteNote`, returns updated page or `HX-Redirect` header; `NoteDeleteDELETE(w, r)` (handles POST with `X-HTTP-Method-Override: DELETE`) calls `models.DeleteNote` + `storage.DeleteNoteFile`, returns `HX-Redirect: /notes`
- [x] T021 [P] [US1] Create `frontend/templates/login.html`: `{{template "base" .}}` with `{{define "content"}}`, form `action="/login" method="POST"`, username text input, submit button "Sign In"
- [x] T022 [P] [US1] Create `frontend/templates/notes/list.html`: `{{define "content"}}` with "New Note" button (`hx-post="/notes" hx-vals='{"title":"","body":""}' hx-push-url="true"`), `<div id="note-list">` containing range over `.Notes` showing title link to `/notes/{slug}`, comma-separated tags, `updated_at` formatted date, delete button (`hx-post="/notes/{slug}" hx-headers='{"X-HTTP-Method-Override":"DELETE"}' hx-confirm="Delete?" hx-target="closest li"`)
- [x] T023 [P] [US1] Create `frontend/templates/notes/editor.html`: `{{define "content"}}` with title `<input>` (value `{{.Note.Title}}`), `<textarea id="editor">{{.Note.Body}}</textarea>`, EasyMDE init in `<script>` block, save form (`POST /notes/{{.Note.Slug}}` with `X-HTTP-Method-Override: PUT` via htmx), timestamps display (`created_at`, `updated_at`), "View" link to `/notes/{{.Note.Slug}}`
- [x] T024 [P] [US1] Create `frontend/templates/notes/reader.html`: `{{define "content"}}` with `<h1>{{.Note.Title}}</h1>`, `<div class="prose">{{.BodyHTML}}</div>` (rendered goldmark output, use `template.HTML` to avoid escaping), timestamps, tags display, "Edit" link to `/notes/{{.Note.Slug}}/edit`, delete button

**Checkpoint**: Login, create, edit, read, delete notes with timestamps — all working independently

---

## Phase 4: User Story 2 — Drag and Drop Images (Priority: P2)

**Goal**: Users drag image files onto the EasyMDE editor; images stored per-user under `uploads/`; Markdown `![name](url)` inserted at cursor; images render in reader mode.

**Independent Test**: Login → open note editor → `POST /notes/{slug}/images` with PNG file → assert JSON `{"url":"/uploads/..."}` returned → assert file exists at `uploads/{userID}/{filename}` → assert `GET /uploads/{username}/{filename}` returns image bytes for owner, 403 for others.

### Integration Tests for User Story 2 ⚠️ WRITE FIRST — MUST FAIL BEFORE IMPLEMENTATION

- [x] T025 [P] [US2] Add to `backend/internal/handlers/handlers_test.go` (image tests): test `POST /notes/{slug}/images` with valid PNG multipart returns 200 JSON `{"url":"..."}` and file exists on filesystem; test with non-image file (`.txt`) returns 400; test with file >10 MB returns 413; test `GET /uploads/{username}/{file}` returns image bytes for the owner session; test same endpoint returns 403 when session user differs from `{username}` path param; test `DELETE /notes/{slug}` cleans up associated image files from filesystem

### Implementation for User Story 2

- [x] T026 [US2] Add image query functions to `backend/internal/models/models.go`: `CreateImage(db *DB, noteID int64, filename, original, mimeType string, size int64) (*Image, error)`; `GetImagesForNote(db *DB, noteID int64) ([]*Image, error)`; `DeleteImagesForNote(db *DB, noteID int64) ([]string, error)` (returns filenames for filesystem cleanup); define `Image` struct: `{ID, NoteID int64, Filename, Original, MimeType string, Size int64}`
- [x] T027 [US2] Create `backend/internal/handlers/images.go`: `ImageUploadPOST(w, r)` validates `Content-Type` (PNG/JPEG/GIF/WebP only) and file size (≤10 MB, return 413 if exceeded), saves file to `uploads/{userID}/{uuid}.{ext}` via `storage.EnsureUploadsDir`, calls `models.CreateImage`, returns JSON `{"url":"/uploads/{username}/{filename}"}`; `ImageServGET(w, r)` checks session username matches `{username}` chi URL param (403 if not), serves file with correct `Content-Type` header
- [x] T028 [US2] Update `NoteDeleteDELETE` in `backend/internal/handlers/notes.go` to call `models.DeleteImagesForNote` before `models.DeleteNote`, then delete each returned filename from `uploads/{userID}/` filesystem path
- [x] T029 [US2] Update EasyMDE init in `frontend/static/js/app.js`: configure `uploadImage: true` and `imageUploadFunction: function(file, onSuccess, onError)` that POSTs to `/notes/{slug}/images` as `multipart/form-data`, calls `onSuccess(data.url)` on 200, `onError(msg)` on error; read slug from `data-slug` attribute on the editor container element

**Checkpoint**: Drag-and-drop uploads work, images render in reader mode, cleanup on note delete

---

## Phase 5: User Story 3 — Tag Notes for Quick Navigation (Priority: P3)

**Goal**: Hashtag tags (`#word`) in note body are parsed on save, shown in tag sidebar, and used to filter the note list to the current user's notes only.

**Independent Test**: Create notes with `#work` and `#ideas` → `GET /tags` returns `[{"name":"work","count":N}]` for logged-in user → `GET /notes?tag=work` returns only matching notes → remove tag from note body and save → re-check counts updated.

### Integration Tests for User Story 3 ⚠️ WRITE FIRST — MUST FAIL BEFORE IMPLEMENTATION

- [x] T030 [P] [US3] Add to `backend/internal/handlers/handlers_test.go` (tag tests): test `PUT /notes/{slug}` with body `#work #ideas` inserts rows into `note_tags`; test `GET /tags` returns `[{"name":"work","count":N}]` for logged-in user only (not other users' tags); test `GET /notes?tag=work` returns only notes tagged `#work` for current user; test removing `#work` from body on re-save removes row from `note_tags`; test `#Work` and `#work` produce same lowercase `tag_name`
- [x] T031 [P] [US3] Add tag model tests to `backend/internal/models/models_test.go`: test `ParseTags` extracts `#word` patterns and lowercases all; test `SyncTags` deletes old rows and inserts new set for a note; test `ListTagsForUser` returns correct name+count ordered by count desc

### Implementation for User Story 3

- [x] T032 [US3] Add tag functions to `backend/internal/models/models.go`: `ParseTags(body string) []string` uses regexp `#([a-zA-Z0-9_]+)` to extract tags, lowercases, deduplicates; `SyncTags(db *DB, noteID int64, tags []string) error` deletes all `note_tags` rows for `noteID` then bulk-inserts new set; `ListTagsForUser(db *DB, userID int64) ([]TagCount, error)` JOINs `note_tags` with `notes` on `user_id`, returns `[]TagCount{{Name string, Count int}}` ordered by count DESC
- [x] T033 [US3] Update `NoteUpdatePUT` in `backend/internal/handlers/notes.go` to call `models.ParseTags(body)` then `models.SyncTags(db, noteID, tags)` after writing the `.md` file
- [x] T034 [US3] Update `NotesCreatePOST` in `backend/internal/handlers/notes.go` to call `models.ParseTags` + `models.SyncTags` on initial note creation (handles notes created with tags in body)
- [x] T035 [US3] Create `backend/internal/handlers/tags.go`: `TagsListGET(w, r)` calls `models.ListTagsForUser`, returns `tags/sidebar.html` partial for htmx requests (`HX-Request` header present) or JSON `[{"name":"...","count":N}]`
- [x] T036 [P] [US3] Create `frontend/templates/tags/sidebar.html`: `{{define "tags-sidebar"}}` with `<ul>`, range over `.Tags` rendering `<li><a hx-get="/notes?tag={{.Name}}" hx-target="#note-list" hx-push-url="true">#{{.Name}} ({{.Count}})</a></li>`
- [x] T037 [US3] Update `frontend/templates/base.html` to include `<aside id="tag-sidebar" hx-get="/tags" hx-trigger="load" hx-swap="innerHTML">` in the layout; update `frontend/templates/notes/list.html` to add `id="note-list"` on the notes container for htmx swap

**Checkpoint**: Tags parsed on save, sidebar shows per-user tags, note list filters by tag

---

## Phase N: Polish & Cross-Cutting Concerns

**Purpose**: Error handling, styling, DB rebuild, coverage gate

- [x] T038 [P] Create `frontend/static/css/app.css`: body font stack (system-ui), nav flexbox layout, note list card styles (border, padding, hover), reader mode prose styles (max-width 72ch, line-height 1.7, code block background), responsive single-column on narrow screens
- [x] T039 [P] Add HTTP error handlers in `backend/cmd/server/main.go`: custom 404 handler rendering `frontend/templates/404.html`; custom 403 handler; 413 handler returning JSON `{"error":"file too large"}`; 500 recovery middleware already provided by Chi
- [x] T040 [P] Create `frontend/templates/404.html` and `frontend/templates/403.html`: minimal error pages with `{{template "base" .}}` and `{{define "content"}}` blocks
- [x] T041 Implement `RebuildDB(db *DB, notesRoot, uploadsRoot string) error` in `backend/internal/models/models.go`: scan `notesRoot/{userID}/*.md` files, parse title from first `# heading` (fallback: filename), parse `#tags` patterns, read file mtime for timestamps, truncate and re-insert all tables; wire to `--rebuild-db` flag in `main.go`
- [x] T042 [P] Create `frontend/static/js/app.js` finalisation: add `data-slug` attribute reading for EasyMDE imageUploadFunction, add htmx `hx-on::after-request` handler to refresh tag sidebar after note save
- [x] T043 Run full test suite and confirm ≥90% coverage: `cd backend && go test ./... -cover -coverprofile=coverage.out && go tool cover -func=coverage.out`
- [x] T044 Run quickstart.md validation: start server, complete all usage steps (login, create note, add image, add tags, filter by tag, reader mode), verify each user story end-to-end

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 complete — BLOCKS all user stories
- **US1 (Phase 3)**: Depends on Phase 2; no dependency on US2 or US3
- **US2 (Phase 4)**: Depends on Phase 2; integrates with US1 note delete
- **US3 (Phase 5)**: Depends on Phase 2; integrates with US1 note save
- **Polish (Phase N)**: Depends on all desired user stories complete

### Within Each User Story

- Integration tests MUST be written and FAIL before implementation
- Model query functions before handlers
- Handlers before templates
- Core handler before integration hooks (e.g., image cleanup in delete)

### Parallel Opportunities

- T003, T004, T005 (Phase 1) — all parallel
- T007, T008 (Phase 2) — parallel with each other after T006
- T011, T012, T013, T014 (US1 tests) — all parallel with each other
- T021, T022, T023, T024 (US1 templates) — all parallel after routes done
- T038, T039, T040, T042 (Polish) — all parallel

---

## Parallel Example: User Story 1

```bash
# Write all tests in parallel first:
Task: T011 Write handlers_test.go (auth tests)
Task: T012 Write handlers_test.go (notes tests)
Task: T013 Write storage_test.go
Task: T014 Write models_test.go

# Confirm all FAIL, then implement models in parallel:
Task: T015 User query functions in models.go
Task: T016 Note query functions in models.go

# Then implement handlers sequentially (same file):
Task: T017 auth.go handlers
Task: T018 notes.go list + create
Task: T019 notes.go reader + editor
Task: T020 notes.go update + delete

# Then templates in parallel:
Task: T021 login.html
Task: T022 notes/list.html
Task: T023 notes/editor.html
Task: T024 notes/reader.html
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 (T001–T005)
2. Complete Phase 2 (T006–T010) — CRITICAL blocker
3. Complete Phase 3 (T011–T024)
4. **STOP and VALIDATE**: Login, create note, edit, read, delete, check timestamps
5. Run `go test ./... -cover` — must pass ≥90% for handlers package

### Incremental Delivery

1. Setup + Foundational → app skeleton running
2. User Story 1 → full note CRUD with mock login → **MVP**
3. User Story 2 → image drag-and-drop
4. User Story 3 → tag navigation
5. Polish → error pages, CSS, rebuild utility

---

## Notes

- [P] = different files, no dependencies on in-progress tasks
- Integration tests are MANDATORY (Constitution Principle IV) — marked "WRITE FIRST — MUST FAIL"
- `t.TempDir()` provides isolated real filesystem per test — no mocking of storage
- In-memory SQLite (`:memory:`) in model tests — fast and isolated per `TestMain`
- HTTP method override via `X-HTTP-Method-Override` header (htmx `hx-headers`) since HTML forms support GET/POST only
- Each user story is independently completable and testable before moving to next priority
