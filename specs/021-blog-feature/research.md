# Research: Blog Feature for Tagged Notes

**Feature**: 021-blog-feature | **Date**: 2026-05-05

## R-001: Cross-user tag query strategy

**Decision**: Add new query functions that join `notes`, `note_tags`, `users`, and `blog_posts` without a `user_id` filter. The existing `ListNotes` and `ListTagsForUser` are always user-scoped; blog needs global aggregation.

**Rationale**: The existing `note_tags` table already has the data we need. A simple JOIN across `notes`, `note_tags`, `users`, and `blog_posts` gives us everything. No schema changes to existing tables required — only the new `blog_posts` table is added.

**Alternatives considered**:
- Materialized view / denormalized blog table: Rejected — adds write-time complexity for a read pattern that's simple enough with JOINs. Blog traffic is low.
- Query `note_tags` directly without `blog_posts` table: Rejected — we need `published_at` which doesn't exist on any current table, and we need to track publication state separate from tag presence for ordering semantics.

## R-002: published_at lifecycle management

**Decision**: Hook into `SyncTags` to manage `blog_posts` rows. When `SyncTags` adds the "blog" tag, insert a `blog_posts` row with `published_at = now()` if one doesn't exist. When `SyncTags` removes the "blog" tag, delete the `blog_posts` row. Re-adding the tag creates a new row with a fresh `published_at`.

**Rationale**: `SyncTags` is the single point where tags change. Hooking here keeps the logic centralized and ensures `blog_posts` stays in sync regardless of how tags are modified (editor save, API call, etc.).

**Alternatives considered**:
- Separate "publish" API endpoint: Rejected — spec says tagging "blog" is the trigger, no extra step.
- Store `published_at` on the `notes` table: Rejected — pollutes the generic note model with blog-specific concerns.
- Keep `blog_posts` row on tag removal (preserve `published_at`): Rejected — spec says "Re-adding the tag after removal sets a new `published_at`."

## R-003: Wiki-link resolution for blog context

**Decision**: Create a new `ResolveWikiLinksForBlog(db, ownerUserID, body)` function. For each wiki-link `[[title]]`:
1. Resolve the title to a note owned by the same user (same as existing behavior).
2. If the resolved note has a `blog_posts` row (i.e., is a blog post), render as a clickable link to `/blog/<username>/<slug>`.
3. Otherwise, render as `<span class="wikilink-plain">title</span>` (styled, non-clickable).

**Rationale**: Follows the established pattern of context-specific wiki-link resolvers (`ResolveWikiLinks`, `ResolveWikiLinksPublic`, `ResolveWikiLinksForViewer`). Blog links always go to blog URLs, not note URLs. Cross-user wiki-links are not supported (consistent with existing behavior — wiki-links resolve within the same user's notes).

**Alternatives considered**:
- Strip wiki-link syntax entirely: Rejected — user decision was to keep the text visible.
- Resolve cross-user wiki-links: Rejected — existing system doesn't support this; would require title disambiguation UI.

## R-004: Excerpt generation approach

**Decision**: Implement `GenerateExcerpt(body string, maxLen int) string` in a new `markdown/excerpt.go` file. Processing pipeline:
1. Strip drawing markers (`![[draw:...]]`).
2. Strip markdown headings (`# ...` lines).
3. Strip markdown formatting (bold, italic, links, images, code blocks, blockquotes).
4. Collapse multiple whitespace/newlines into single spaces.
5. Trim and take first `maxLen` characters, breaking at a word boundary.
6. Append "..." if truncated.

**Rationale**: A regex-based approach is simpler than parsing the full AST just for excerpt text. The order of operations ensures clean plain text output. Word-boundary truncation avoids mid-word cuts.

**Alternatives considered**:
- Use goldmark AST to extract text nodes: Rejected — over-engineered for excerpt generation; regex strip is sufficient and faster.
- Author-controlled excerpts via frontmatter: Rejected — adds friction; smart-strip provides good defaults. Can be added later if needed.

## R-005: Blog template architecture

**Decision**: Create `frontend/templates/blog/base.html` as a standalone full-document template (similar to `public/note.html` pattern). Blog handlers will use `template.ParseFiles` to compose `base.html` + page template, then `ExecuteTemplate`. Blog pages will have:
- Minimal header with blog title/name and navigation (Home, Tags)
- Clean content area with reading-focused typography
- Footer with basic info
- No notes-app elements (login, sidebar, session bar)

**Rationale**: The spec requires the blog to be "visually distinct" from the notes app. A standalone base template gives complete control and prevents CSS/structural leakage. The `public/note.html` pattern already demonstrates this approach in the codebase.

**Alternatives considered**:
- Share `base.html` with conditional blocks: Rejected — coupling risk; blog design changes could break notes UI and vice versa.

## R-006: Blog route structure and chi routing

**Decision**: Add blog routes to the public (no-auth) section of the router:
- `GET /blog` — index (with `?page=N` query param for pagination)
- `GET /blog/tag/{tag}` — filtered by tag
- `GET /blog/{username}/{slug}` — individual post
- `GET /blog/{username}/{slug}/drawings/{drawingID}/svg` — drawing SVG for blog post

**Rationale**: Blog is fully public (FR-003). Routes follow chi's URL param pattern. Pagination uses query params to keep URLs clean. The `{username}/{slug}` pair uniquely identifies a post.

**Alternatives considered**:
- Pagination via URL path (`/blog/page/2`): Rejected — query params are more standard and avoid route ambiguity with `{username}`.

## R-007: SEO and Open Graph meta tags

**Decision**: Blog templates will include in `<head>`:
- `<title>` with post title or "Blog" for index
- `<meta name="description">` with the excerpt
- `og:title`, `og:description`, `og:type` (article), `og:url`
- `twitter:card` (summary)
- Canonical URL

**Rationale**: Standard SEO/social sharing requirements. No external services needed — all data is available at render time.

**Alternatives considered**:
- JSON-LD structured data: Deferred — can be added later; OG tags cover the immediate need.

## R-008: Rebuild strategy for blog_posts table

**Decision**: `RebuildDB` will scan all notes, check their tags, and insert `blog_posts` rows for any note tagged "blog". Since `published_at` cannot be recovered from Markdown files, use the note's `created_at` as a fallback during rebuild.

**Rationale**: Consistent with constitution Principle I — SQLite is a derived cache. The trade-off (losing original publish time on rebuild) is acceptable since rebuilds are rare administrative operations.

**Alternatives considered**:
- Store `published_at` in Markdown frontmatter: Rejected — adds complexity; Markdown files shouldn't need blog-specific metadata.
