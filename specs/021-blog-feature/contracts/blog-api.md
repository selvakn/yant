# Blog API Contracts

**Feature**: 021-blog-feature | **Date**: 2026-05-05

All blog routes are public (no authentication required). Blog pages are server-rendered HTML.

## Routes

### GET /blog

**Description**: Blog index page showing all published posts, paginated, newest first.

**Query Parameters**:
| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `page` | int | 1 | Page number (1-indexed) |

**Response**: HTML page rendered from `blog/base.html` + `blog/index.html`

**Template Data**:
- `Posts`: List of `BlogPost` (max 10 per page)
  - Each post includes: title, slug, username, published_at, excerpt, tags (excluding "blog")
- `Tag`: empty string (unfiltered)
- `Page`, `TotalPages`, `HasPrev`, `HasNext`: pagination state
- `AllTags`: all tags used by blog posts (for tag cloud/nav)

**Status Codes**:
| Code | Condition |
|------|-----------|
| 200 | Always (empty state if no posts) |

**Example URL**: `GET /blog?page=2`

---

### GET /blog/tag/{tag}

**Description**: Blog index filtered by a specific tag. Same layout as `/blog`.

**URL Parameters**:
| Param | Type | Description |
|-------|------|-------------|
| `tag` | string | Tag name to filter by (case-insensitive matching) |

**Query Parameters**:
| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `page` | int | 1 | Page number |

**Response**: HTML page rendered from `blog/base.html` + `blog/index.html`

**Template Data**: Same as `/blog` but `Tag` is set and `Posts` are filtered.

**Status Codes**:
| Code | Condition |
|------|-----------|
| 200 | Always (empty state if no posts match tag) |

**Example URL**: `GET /blog/tag/golang?page=1`

---

### GET /blog/{slug}

**Description**: Individual blog post page with full rendered content.

**URL Parameters**:
| Param | Type | Description |
|-------|------|-------------|
| `slug` | string | Note slug |

**Response**: HTML page rendered from `blog/base.html` + `blog/post.html`

**Template Data**:
- `Post`: `BlogPost` with full metadata
- `BodyHTML`: Rendered HTML from Markdown (goldmark + GFM + drawing markers)
  - Wiki-links resolved: blog targets → clickable `/blog/<slug>` links; others → styled plain text
  - Drawing markers → SVG placeholder divs (hydrated by client-side fetch or inline)
  - Todo checkboxes, @due badges rendered (consistent with reader view)
- `Drawings`: List of `NoteDrawing` for this post
- `PrevPost`, `NextPost`: Adjacent posts by `published_at` (for navigation)
- `AllTags`: All blog tags (for sidebar)
- `Title`, `Description`: For HTML `<head>` meta/OG tags

**Status Codes**:
| Code | Condition |
|------|-----------|
| 200 | Post found and is a published blog post |
| 404 | Slug doesn't exist, note is not tagged "blog", or note is archived |

**Example URL**: `GET /blog/my-first-post`

---

### GET /blog/{slug}/drawings/{drawingID}/svg

**Description**: Serve SVG preview for a drawing embedded in a blog post.

**URL Parameters**:
| Param | Type | Description |
|-------|------|-------------|
| `slug` | string | Note slug |
| `drawingID` | string | Drawing ID (8-char alphanumeric) |

**Response**: SVG image

**Headers**:
| Header | Value |
|--------|-------|
| `Content-Type` | `image/svg+xml` |
| `Cache-Control` | `no-cache` |

**Status Codes**:
| Code | Condition |
|------|-----------|
| 200 | Drawing exists and belongs to a published blog post |
| 404 | Post not found, drawing not found, or note is not a blog post |

**Example URL**: `GET /blog/my-first-post/drawings/a1b2c3d4/svg`

## HTML Meta Tags (blog post pages)

```html
<title>{Post.Title} - Blog</title>
<meta name="description" content="{Post.Excerpt}">
<meta property="og:title" content="{Post.Title}">
<meta property="og:description" content="{Post.Excerpt}">
<meta property="og:type" content="article">
<meta property="og:url" content="https://{host}/blog/{slug}">
<meta name="twitter:card" content="summary">
<link rel="canonical" href="https://{host}/blog/{slug}">
```

## HTML Meta Tags (blog index pages)

```html
<title>Blog{" - " + Tag if filtered}</title>
<meta name="description" content="Blog posts{" tagged " + Tag if filtered}">
<meta property="og:title" content="Blog{" - " + Tag if filtered}">
<meta property="og:type" content="website">
<meta property="og:url" content="https://{host}/blog{/tag/Tag if filtered}">
```
