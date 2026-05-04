# Quickstart: Blog Feature

**Feature**: 021-blog-feature | **Date**: 2026-05-05

## Prerequisites

- Go 1.25+ installed
- Existing YANT application running locally (`make run`)
- At least one user account with some notes

## How to Test the Blog

### 1. Tag a note as "blog"

Open any note in the editor and add the `#blog` tag anywhere in the body. Save the note. The note is now a blog post.

### 2. Visit the blog

Navigate to `http://localhost:8080/blog`. The tagged note should appear in the blog index with its title, excerpt, publication date, and tags.

### 3. Read a blog post

Click on the post title to navigate to `http://localhost:8080/blog/<your-username>/<note-slug>`. The full note content renders in the blog layout.

### 4. Filter by tag

If the blog post has additional tags (e.g., `#golang`, `#tutorial`), click a tag on the blog index or post page to filter: `http://localhost:8080/blog/tag/golang`.

### 5. Remove from blog

Remove the `#blog` tag from the note and save. The post disappears from the blog. Navigating to the old URL returns 404.

## Key Files

| File | Purpose |
|------|---------|
| `backend/internal/models/models.go` | `blog_posts` table, `ListBlogPosts`, `GetBlogPost`, `SyncTags` integration |
| `backend/internal/handlers/blog.go` | Blog HTTP handlers |
| `backend/internal/handlers/blog_test.go` | Blog handler integration tests |
| `backend/internal/markdown/excerpt.go` | Smart-strip excerpt generation |
| `backend/cmd/server/main.go` | Blog route registration |
| `frontend/templates/blog/base.html` | Blog base template |
| `frontend/templates/blog/index.html` | Blog index page |
| `frontend/templates/blog/post.html` | Blog post page |
| `frontend/static/css/blog.css` | Blog-specific styles |

## Running Tests

```bash
make test      # all tests including blog
make coverage  # verify >=90% coverage
```

## Architecture Notes

- Blog is a **read-only public view** — no blog-specific write APIs.
- The "blog" tag is the only trigger. No publish button, no draft state.
- `blog_posts` table is a derived cache. `RebuildDB` can reconstruct it (with `created_at` as fallback for `published_at`).
- Wiki-links in blog posts resolve to blog URLs when the target is also a blog post; otherwise styled plain text.
- Drawing SVGs are served via `/blog/{username}/{slug}/drawings/{drawingID}/svg`.
