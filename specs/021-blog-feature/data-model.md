# Data Model: Blog Feature for Tagged Notes

**Feature**: 021-blog-feature | **Date**: 2026-05-05

## New Table: `blog_posts`

Tracks publication metadata for notes tagged "blog". This is a derived index — the source of truth is the "blog" tag on the Markdown file (synced via `note_tags`).

### Schema

```sql
CREATE TABLE IF NOT EXISTS blog_posts (
    note_id      INTEGER PRIMARY KEY REFERENCES notes(id) ON DELETE CASCADE,
    published_at TEXT    NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_blog_posts_published ON blog_posts(published_at DESC);
```

### Fields

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `note_id` | INTEGER | PK, FK → notes(id) ON DELETE CASCADE | The note that is a blog post |
| `published_at` | TEXT | NOT NULL | RFC3339 timestamp of when the "blog" tag was first added |

### Lifecycle

- **Created**: When `SyncTags` adds the "blog" tag to a note, insert into `blog_posts` with `published_at = time.Now().UTC().Format(time.RFC3339)`.
- **Deleted**: When `SyncTags` removes the "blog" tag from a note, delete from `blog_posts`.
- **Re-publish**: If a note is un-tagged then re-tagged "blog", a new row is created with a fresh `published_at`.

### Relationships

```text
users (1) ──< notes (1) ──< note_tags (many)
                │
                └── blog_posts (0..1)
```

- A note has at most one `blog_posts` row (1:1 optional).
- A blog post is always a note; a note is a blog post only when the "blog" tag is present.

## Existing Tables (no changes)

### `notes`

Used as-is. Blog queries join on `notes.id`, `notes.slug`, `notes.title`, `notes.user_id`, `notes.created_at`.

### `users`

Used as-is. Blog queries join on `users.id` and `users.username` to resolve the `{username}` in blog URLs.

### `note_tags`

Used as-is. The "blog" tag in `note_tags` is the opt-in flag. Other tags on blog-tagged notes are displayed as content categories on blog pages.

### `note_drawings`

Used as-is. Blog post pages query drawings by `note_id` and serve SVGs via the blog drawing route.

## Query Patterns

### List blog posts (index, paginated)

```sql
SELECT n.id, n.slug, n.title, n.user_id, n.created_at, n.updated_at,
       u.username, bp.published_at
FROM blog_posts bp
JOIN notes n ON n.id = bp.note_id
JOIN users u ON u.id = n.user_id
WHERE n.archived = 0
ORDER BY bp.published_at DESC
LIMIT ? OFFSET ?
```

### List blog posts by tag (filtered, paginated)

```sql
SELECT n.id, n.slug, n.title, n.user_id, n.created_at, n.updated_at,
       u.username, bp.published_at
FROM blog_posts bp
JOIN notes n ON n.id = bp.note_id
JOIN users u ON u.id = n.user_id
JOIN note_tags t ON t.note_id = n.id
WHERE n.archived = 0 AND t.tag_name = ?
ORDER BY bp.published_at DESC
LIMIT ? OFFSET ?
```

### Get single blog post

```sql
SELECT n.id, n.slug, n.title, n.user_id, n.created_at, n.updated_at,
       u.username, bp.published_at
FROM blog_posts bp
JOIN notes n ON n.id = bp.note_id
JOIN users u ON u.id = n.user_id
WHERE u.username = ? AND n.slug = ? AND n.archived = 0
```

### Count blog posts (for pagination)

```sql
SELECT COUNT(*) FROM blog_posts bp
JOIN notes n ON n.id = bp.note_id
WHERE n.archived = 0
```

### Count blog posts by tag

```sql
SELECT COUNT(*) FROM blog_posts bp
JOIN notes n ON n.id = bp.note_id
JOIN note_tags t ON t.note_id = n.id
WHERE n.archived = 0 AND t.tag_name = ?
```

### Check if note is a blog post (for wiki-link resolution)

```sql
SELECT 1 FROM blog_posts WHERE note_id = ?
```

### List all tags used by blog posts (for tag navigation)

```sql
SELECT t.tag_name, COUNT(*) as count
FROM note_tags t
JOIN blog_posts bp ON bp.note_id = t.note_id
JOIN notes n ON n.id = t.note_id
WHERE n.archived = 0 AND t.tag_name != 'blog'
GROUP BY t.tag_name
ORDER BY count DESC, t.tag_name ASC
```

## Go Structs

### BlogPost (new)

```go
type BlogPost struct {
    Note        *Note
    Username    string
    PublishedAt time.Time
    Excerpt     string    // computed at query time, not stored
    Tags        []string  // all tags except "blog"
}
```

### BlogPageData (template data for index)

```go
type BlogIndexData struct {
    Posts       []*BlogPost
    Tag         string      // empty for unfiltered index
    Page        int
    TotalPages  int
    HasPrev     bool
    HasNext     bool
}
```

### BlogPostData (template data for individual post)

```go
type BlogPostData struct {
    Post        *BlogPost
    BodyHTML    template.HTML
    Drawings    []*NoteDrawing
    PrevPost    *BlogPost   // nil if first
    NextPost    *BlogPost   // nil if last
    AllTags     []TagCount  // for sidebar/navigation
}
```

## Validation Rules

- `published_at` is always set server-side; never user-supplied.
- Blog queries always filter `n.archived = 0` — archived notes are never visible on the blog.
- The "blog" tag itself is excluded from the displayed tag list on blog posts (it's implicit).
- Pagination: page must be >= 1; page size defaults to 10.
