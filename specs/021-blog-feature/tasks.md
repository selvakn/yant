# Tasks: Blog Feature for Tagged Notes

**Input**: Design documents from `/specs/021-blog-feature/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/blog-api.md

**Constitution**: Principle VI requires test-before-commit — full test suite green before every commit. Principle IV requires integration tests with >=90% coverage.

## Phase 1: Foundation (Shared Infrastructure)

**Purpose**: Data model, queries, and shared utilities that all blog user stories depend on.

- [ ] T001 [US1] Create `blog_posts` table schema in `backend/internal/models/models.go` (add to `initDB`, schema as specified in data-model.md)
- [ ] T002 [US1] Add blog model functions in `backend/internal/models/models.go`: `PublishBlogPost(db, noteID)`, `UnpublishBlogPost(db, noteID)`, `IsBlogPost(db, noteID)`, `GetBlogPost(db, username, slug)`, `ListBlogPosts(db, page, pageSize)`, `CountBlogPosts(db)`, `ListBlogTags(db)`
- [ ] T003 [US1] Integrate blog publication into `SyncTags` in `backend/internal/models/models.go` — when "blog" tag is added, call `PublishBlogPost`; when removed, call `UnpublishBlogPost`
- [ ] T004 [P] [US1] Create excerpt generator in `backend/internal/markdown/excerpt.go` with `GenerateExcerpt(body string, maxLen int) string`
- [ ] T005 [P] [US1] Write unit tests for excerpt generator in `backend/internal/markdown/excerpt_test.go`
- [ ] T006 [US1] Write unit tests for blog model functions in `backend/internal/models/models_test.go` (publish, unpublish, list, count, SyncTags integration)
- [ ] T007 [US1] Update `RebuildDB` in `backend/internal/models/rebuild.go` to populate `blog_posts` from `note_tags` (fallback `published_at` = `created_at`)
- [ ] T008 [US1] Write tests for `RebuildDB` blog post population in `backend/internal/models/rebuild_test.go`

**Checkpoint**: All blog data model and queries are in place with tests passing.

---

## Phase 2: User Story 1 — Browse the Blog (Priority: P1)

**Goal**: Public blog index at `/blog` showing all blog-tagged notes, newest first, with title, date, excerpt, tags.

**Independent Test**: Tag notes with "blog", navigate to `/blog`, verify listing.

### Tests

- [ ] T009 [US1] Write handler integration tests in `backend/internal/handlers/blog_test.go`: `TestBlogIndexGET_empty`, `TestBlogIndexGET_with_posts`, `TestBlogIndexGET_excludes_archived`, `TestBlogIndexGET_excludes_non_blog`

### Implementation

- [ ] T010 [US1] Create blog handler file `backend/internal/handlers/blog.go` with `BlogIndexGET` handler
- [ ] T011 [P] [US1] Create blog base template `frontend/templates/blog/base.html` (standalone, no notes-app UI, clean reading layout, CSS link to blog.css)
- [ ] T012 [P] [US1] Create blog index template `frontend/templates/blog/index.html` (post listing with title, date, excerpt, tags, empty state)
- [ ] T013 [P] [US1] Create blog stylesheet `frontend/static/css/blog.css` (reading-focused typography, responsive, clean layout)
- [ ] T014 [US1] Register `GET /blog` route in `backend/cmd/server/main.go` (public group, no auth)
- [ ] T015 [US1] Run full test suite, commit: "feat(blog): add blog index with listing, excerpts, and blog-specific layout"

**Checkpoint**: `/blog` shows blog-tagged notes with clean layout.

---

## Phase 3: User Story 2 — Read a Blog Post (Priority: P1)

**Goal**: Individual blog post at `/blog/{username}/{slug}` with full rendered content, drawings, navigation.

**Independent Test**: Navigate to `/blog/selvakn/test-post`, verify full content renders in blog layout.

### Tests

- [ ] T016 [US2] Write handler integration tests in `backend/internal/handlers/blog_test.go`: `TestBlogPostGET_success`, `TestBlogPostGET_not_found`, `TestBlogPostGET_not_blog_returns_404`, `TestBlogPostGET_archived_returns_404`

### Implementation

- [ ] T017 [US2] Add `BlogPostGET` handler in `backend/internal/handlers/blog.go` (markdown rendering, wiki-link resolution, drawing data, prev/next navigation, OG meta tags)
- [ ] T018 [US2] Create `ResolveWikiLinksForBlog(db, userID, body)` in `backend/internal/models/models.go` — blog targets → `/blog/<username>/<slug>` links; others → `<span class="wikilink-plain">title</span>`
- [ ] T019 [P] [US2] Create blog post template `frontend/templates/blog/post.html` (full content, metadata, tags, prev/next, OG meta tags in head)
- [ ] T020 [US2] Add `BlogDrawingSVGGET` handler in `backend/internal/handlers/blog.go` (serve SVG for blog post drawings)
- [ ] T021 [US2] Register `GET /blog/{username}/{slug}` and `GET /blog/{username}/{slug}/drawings/{drawingID}/svg` routes in `backend/cmd/server/main.go`
- [ ] T022 [US2] Add SVG drawing hydration script to blog post template (fetch and inject SVG previews, same pattern as reader.html)
- [ ] T023 [US2] Write test for `ResolveWikiLinksForBlog` in `backend/internal/models/models_test.go`
- [ ] T024 [US2] Write test for `BlogDrawingSVGGET` in `backend/internal/handlers/blog_test.go`
- [ ] T025 [US2] Run full test suite, commit: "feat(blog): add individual blog post page with drawings, wiki-links, and navigation"

**Checkpoint**: Full blog post reading experience works with drawings and navigation.

---

## Phase 4: User Story 3 — Filter Posts by Tag (Priority: P2)

**Goal**: Tag-filtered blog listing at `/blog/tag/{tag}`.

**Independent Test**: Navigate to `/blog/tag/golang`, verify only matching posts appear.

### Tests

- [ ] T026 [US3] Write handler integration tests in `backend/internal/handlers/blog_test.go`: `TestBlogTagGET_with_matching_posts`, `TestBlogTagGET_empty_tag`, `TestBlogTagGET_excludes_non_matching`

### Implementation

- [ ] T027 [US3] Add `ListBlogPostsByTag(db, tag, page, pageSize)` and `CountBlogPostsByTag(db, tag)` to `backend/internal/models/models.go`
- [ ] T028 [US3] Add `BlogTagGET` handler in `backend/internal/handlers/blog.go` (reuses index template with tag filter context)
- [ ] T029 [US3] Register `GET /blog/tag/{tag}` route in `backend/cmd/server/main.go`
- [ ] T030 [US3] Write tests for `ListBlogPostsByTag` and `CountBlogPostsByTag` in `backend/internal/models/models_test.go`
- [ ] T031 [US3] Run full test suite, commit: "feat(blog): add tag-based filtering for blog posts"

**Checkpoint**: Tag filtering works; clickable tags on index and post pages navigate to filtered view.

---

## Phase 5: User Story 4 — Blog Posts Are Public by Default (Priority: P2)

**Goal**: Tagging a note "blog" makes it publicly accessible without extra steps.

**Independent Test**: Tag a note, visit `/blog/<username>/<slug>` unauthenticated — post is visible.

*Note*: This is largely already handled by the public blog routes (no auth middleware). This phase adds explicit tests and edge case handling.

### Tests

- [ ] T032 [US4] Write integration test in `backend/internal/handlers/blog_test.go`: `TestBlogPost_accessible_without_auth`, `TestBlogPost_removed_tag_returns_404`

### Implementation

- [ ] T033 [US4] Verify and test that removing the "blog" tag (via note save) triggers `UnpublishBlogPost` and the post returns 404 on blog routes
- [ ] T034 [US4] Run full test suite, commit: "feat(blog): verify public-by-default blog post access"

**Checkpoint**: Blog posts are automatically public when tagged; automatically removed when untagged.

---

## Phase 6: User Story 5 — Blog Pagination (Priority: P3)

**Goal**: Blog index paginates when posts exceed 10 per page.

**Independent Test**: Create >10 blog posts, verify pagination controls on `/blog`.

### Tests

- [ ] T035 [US5] Write integration tests in `backend/internal/handlers/blog_test.go`: `TestBlogIndexGET_pagination_first_page`, `TestBlogIndexGET_pagination_second_page`, `TestBlogIndexGET_no_pagination_when_few_posts`

### Implementation

- [ ] T036 [US5] Add pagination logic to `BlogIndexGET` and `BlogTagGET` handlers (parse `?page=N`, compute total pages, prev/next flags)
- [ ] T037 [US5] Add pagination controls to `frontend/templates/blog/index.html` (Previous/Next links, page indicator)
- [ ] T038 [US5] Run full test suite, commit: "feat(blog): add pagination to blog index"

**Checkpoint**: Blog index paginates correctly with navigation controls.

---

## Phase 7: Polish & Cross-Cutting

**Purpose**: Final refinements across all stories.

- [ ] T039 [P] Add tag cloud/navigation sidebar to blog base template (list all blog tags with counts)
- [ ] T040 [P] Responsive CSS review — verify blog renders well on mobile viewports
- [ ] T041 Update `specs/021-blog-feature/spec.md` status to "Implemented"
- [ ] T042 Run full test suite and coverage check (`make test && make coverage`), commit: "feat(blog): polish and finalize blog feature"

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Foundation)**: No dependencies — start here
- **Phase 2 (US1 Browse)**: Depends on Phase 1
- **Phase 3 (US2 Read)**: Depends on Phase 2 (needs blog templates and index)
- **Phase 4 (US3 Tags)**: Depends on Phase 1 (can run parallel to Phase 3)
- **Phase 5 (US4 Public)**: Depends on Phase 2 (tests existing behavior)
- **Phase 6 (US5 Pagination)**: Depends on Phase 2
- **Phase 7 (Polish)**: Depends on all previous phases

### Within Each Phase

- Tests FIRST, ensure they FAIL before implementation (TDD)
- Models/queries before handlers
- Handlers before templates
- Full test suite green before each commit (Constitution Principle VI)

### Parallel Opportunities

- T004 + T005 (excerpt) can run parallel with T001-T003 (data model)
- T011 + T012 + T013 (templates + CSS) can run parallel with T010 (handler)
- Phase 4 can start as soon as Phase 1 is complete (parallel with Phase 3)
