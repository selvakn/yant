# Feature Specification: Blog Feature for Tagged Notes

**Feature Branch**: `021-blog-feature`  
**Created**: 2026-05-05  
**Status**: Clarified  
**Input**: User description: "Lets add a blog feature. When loaded with /blog url, we should see the blog. Any note tagged with blog tag should be rendered as blog. Usual blog functionalities like browsing tags, see the latest one first, friendly urls, should be public by default, layout should be like a blog."

## Clarifications

### Session 2026-05-05

- Q: How should blog URLs handle slug collisions across users in a multi-user instance? → A: Include author in URL — `/blog/<username>/<slug>` (e.g., `/blog/selvakn/hello-world`).
- Q: How should wiki-links (`[[private-note]]`) in blog posts be handled for unauthenticated readers? → A: Wiki-links pointing to other blog posts become clickable blog links (`/blog/<username>/<slug>`); all other wiki-links render as styled plain text (non-clickable).
- Q: How should blog index excerpts be generated? → A: Smart-strip — remove frontmatter, headings, drawing markers, and blank lines, then take first 200 chars of remaining plain text.
- Q: Should blog ordering use note `created_at` or a separate publication date? → A: Track a `published_at` timestamp (set when "blog" tag is first added); use that for ordering and display.
- Q: Should blog pages use a separate base template or share the notes app template? → A: Separate base template (`blog_base.html`) with its own header/footer/nav — no notes-app UI.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Browse the Blog (Priority: P1)

A visitor navigates to `/blog` and sees a list of published blog posts, ordered newest first. Each post shows its title, publication date, a short excerpt, and tags. The visitor can click a post to read the full content.

**Why this priority**: This is the core blog experience. Without a browsable blog index, no other blog functionality matters.

**Independent Test**: Can be fully tested by tagging a few notes with "blog", navigating to `/blog`, and verifying they appear in reverse-chronological order with excerpts, titles, and links to full posts.

**Acceptance Scenarios**:

1. **Given** three notes tagged "blog" with different `published_at` dates, **When** a visitor navigates to `/blog`, **Then** all three appear listed newest-published first with title, date, excerpt, and tags displayed.
2. **Given** a note tagged "blog", **When** a visitor clicks on the post title or "Read more" link, **Then** they are taken to the full blog post at `/blog/<username>/<slug>`.
3. **Given** no notes are tagged "blog", **When** a visitor navigates to `/blog`, **Then** they see an empty state message (e.g., "No posts yet").
4. **Given** a note that is NOT tagged "blog", **When** a visitor navigates to `/blog`, **Then** that note does not appear in the blog listing.

---

### User Story 2 - Read a Blog Post (Priority: P1)

A visitor clicks through from the blog index (or arrives via a direct URL) and reads a full blog post. The post is rendered in a blog-appropriate layout with title, date, author, content (markdown rendered to HTML), tags, and navigation to other posts.

**Why this priority**: Reading individual posts is the primary purpose of a blog. Co-equal with the index in forming the MVP.

**Independent Test**: Can be tested by navigating to a blog post URL and verifying the full content renders correctly with blog-specific layout, metadata, and navigation elements.

**Acceptance Scenarios**:

1. **Given** a note tagged "blog" owned by user "selvakn" with slug "my-first-post", **When** a visitor navigates to `/blog/selvakn/my-first-post`, **Then** the full markdown content is rendered in a blog layout with title, date, tags, and author name.
2. **Given** a blog post exists, **When** viewing it, **Then** navigation links or suggestions to other blog posts are visible (e.g., "Previous/Next" or "Related posts").
3. **Given** a note slug that is not tagged "blog", **When** a visitor navigates to `/blog/<username>/<slug>`, **Then** they receive a 404 page.

---

### User Story 3 - Filter Posts by Tag (Priority: P2)

A visitor clicks on a tag from the blog index or a blog post and sees a filtered list of all blog posts that share that tag. The listing uses the same layout as the main blog index.

**Why this priority**: Tag-based browsing is a standard blog feature that helps visitors discover related content. Important but not required for the initial reading experience.

**Independent Test**: Can be tested by tagging multiple notes with "blog" and various other tags, then navigating to `/blog/tag/go` and verifying only matching posts appear.

**Acceptance Scenarios**:

1. **Given** five blog posts where three are tagged "golang", **When** a visitor navigates to `/blog/tag/golang`, **Then** only the three matching posts are displayed, newest first.
2. **Given** a tag with no blog posts, **When** a visitor navigates to `/blog/tag/nonexistent`, **Then** they see an empty state message.
3. **Given** the blog index page, **When** a visitor clicks a tag on any post, **Then** they are taken to the filtered tag page for that tag.

---

### User Story 4 - Blog Posts Are Public by Default (Priority: P2)

When a note author tags a note with "blog", the note becomes publicly accessible via the blog URL without requiring any additional "publish" step. The author does not need to explicitly share or publish the note.

**Why this priority**: Automatic public access is a key differentiator from the existing notes system and removes friction from the blogging workflow.

**Independent Test**: Can be tested by creating a note, adding the "blog" tag, and verifying it is accessible at `/blog/<username>/<slug>` without authentication.

**Acceptance Scenarios**:

1. **Given** an authenticated user "selvakn" creates a note and tags it "blog", **When** an unauthenticated visitor navigates to `/blog/selvakn/<slug>`, **Then** the post is visible and fully readable.
2. **Given** a blog post exists, **When** the author removes the "blog" tag, **Then** the post is no longer accessible via `/blog` URLs.
3. **Given** a blog post with drawings, **When** an unauthenticated visitor views the post, **Then** the drawings (SVG previews) render correctly.

---

### User Story 5 - Blog Pagination (Priority: P3)

When the blog has many posts, the index page is paginated so visitors can browse through older content without overwhelming page loads.

**Why this priority**: Only becomes relevant when the blog has enough content to justify pagination. Not needed for initial launch.

**Independent Test**: Can be tested by creating more than 10 blog posts and verifying the index shows pagination controls and correctly splits content across pages.

**Acceptance Scenarios**:

1. **Given** 15 blog posts and a page size of 10, **When** a visitor navigates to `/blog`, **Then** the first 10 posts are shown with a link to the next page.
2. **Given** a visitor is on page 2 of the blog, **When** they click "Previous" or "Newer", **Then** they are taken back to page 1.
3. **Given** fewer posts than the page size, **When** a visitor views `/blog`, **Then** no pagination controls are shown.

---

### Edge Cases

- What happens when a note is tagged "blog" but has an empty body? It should still appear in the listing with no excerpt.
- What happens when two users have notes with the same slug? No collision — blog URLs include the username: `/blog/<username>/<slug>`.
- What happens when a multi-user instance has multiple users tagging notes as "blog"? All "blog"-tagged notes from all users appear on the blog, attributed to their respective authors.
- What happens if a blog post contains drawings without SVG previews? A placeholder or the drawing title is shown instead.
- What happens when a note tagged "blog" also has a public share link? Both access methods should work independently — `/blog/<username>/<slug>` and `/p/<token>`.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST serve a blog index at `/blog` listing all notes tagged "blog", ordered by `published_at` (newest first).
- **FR-002**: System MUST render individual blog posts at `/blog/<username>/<slug>` using the note's markdown content, rendered to HTML with the existing goldmark pipeline.
- **FR-003**: Blog posts MUST be publicly accessible without authentication.
- **FR-004**: System MUST display post metadata: title, publication date, author name, and tags.
- **FR-005**: Blog index MUST show a smart-stripped excerpt for each post: remove frontmatter, headings, drawing markers (`![[draw:...]]`), and blank lines, then take the first ~200 characters of remaining plain text.
- **FR-006**: System MUST support tag-based filtering at `/blog/tag/<tag-name>`, showing only blog posts with the given tag.
- **FR-007**: Tags displayed on blog posts and index MUST be clickable links to the tag filter page.
- **FR-008**: Notes MUST automatically become blog posts when tagged "blog" and automatically stop being blog posts when the tag is removed.
- **FR-009**: Blog layout MUST be visually distinct from the notes application — clean, reading-focused, with appropriate typography for long-form content.
- **FR-010**: Blog posts MUST render embedded drawings as SVG previews, consistent with the reader view.
- **FR-011**: System MUST paginate the blog index when the number of posts exceeds the page size (default: 10 posts per page).
- **FR-012**: Blog post URLs MUST use the format `/blog/<username>/<slug>` for unique, friendly identification.
- **FR-013**: In a multi-user instance, the blog MUST aggregate posts from all users who tag notes with "blog".
- **FR-014**: Blog pages MUST include appropriate HTML meta tags (title, description, open graph) for SEO and social sharing.
- **FR-015**: System MUST track a `published_at` timestamp per blog post, set when the "blog" tag is first added to a note. Re-adding the tag after removal sets a new `published_at`.
- **FR-016**: Wiki-links in blog posts MUST resolve to clickable blog links (`/blog/<username>/<slug>`) when the target note is also a blog post; otherwise they MUST render as styled plain text (non-clickable).
- **FR-017**: Blog pages MUST use a separate base template (`blog_base.html`) with blog-specific header, footer, and navigation — no notes-app UI elements.

### Key Entities

- **Blog Post**: A note that has been tagged with "blog". Attributes: title, slug, body (markdown), author (username), `published_at` timestamp, tags, drawings. Requires a lightweight `blog_posts` table to track `published_at` (set when "blog" tag is first added). Uniquely identified by the combination of author username and note slug.
- **Blog Tag**: An existing tag in the tag system. The "blog" tag acts as the opt-in flag. Other tags on a blog-tagged note serve as content categorization visible on the blog.
- **Author**: The user who owns the note. Username appears in blog post URLs and is displayed as attribution on posts.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Visitors can discover and read any blog post in under 3 seconds from landing on `/blog`.
- **SC-002**: Adding the "blog" tag to a note makes it publicly visible on the blog within the next page load — no additional steps required.
- **SC-003**: The blog index correctly displays all blog-tagged notes from all users, newest first, with no missing or duplicate entries.
- **SC-004**: Tag filtering returns accurate results — only posts with the selected tag appear.
- **SC-005**: Blog pages render correctly on both desktop and mobile viewports.
- **SC-006**: Blog post pages include valid Open Graph meta tags suitable for sharing on social platforms.

## Assumptions

- The existing tag system supports querying all notes across users by tag name.
- The existing goldmark markdown rendering pipeline (including drawing marker extension, wiki-links, etc.) is reused for blog post rendering.
- The "blog" tag is treated as a reserved/special tag only in that it triggers blog visibility — it otherwise behaves like any normal tag.
- Blog posts use a `published_at` timestamp tracked in a `blog_posts` table, set when the "blog" tag is first added to a note.
- The blog is a read-only public view — there is no commenting, reactions, or subscriber functionality in this version.
- RSS/Atom feed is out of scope for this version.
- The blog shares the same base URL as the notes application (e.g., `example.com/blog`).
- The blog layout and styling will be server-rendered HTML (Go templates + CSS), not a separate frontend application.
