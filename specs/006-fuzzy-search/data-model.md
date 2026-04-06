# Data Model: Fuzzy Search for Notes

**Feature**: 006-fuzzy-search
**Date**: 2026-04-05

## Overview

No schema changes required. Search uses existing `notes` and `note_tags` tables plus on-disk Markdown files.

## Existing Entities (Unchanged)

### Note (SQLite + Markdown file)

| Field | Type | Source | Notes |
|-------|------|--------|-------|
| ID | int64 | SQLite `notes.id` | Primary key |
| UserID | int64 | SQLite `notes.user_id` | Foreign key to users |
| Slug | string | SQLite `notes.slug` | URL-safe identifier |
| Title | string | SQLite `notes.title` | Searchable field |
| Tags | []string | SQLite `note_tags.tag_name` | Searchable field |
| Body | string | Markdown file on disk | Searchable field (loaded on search) |
| CreatedAt | time.Time | SQLite | For display |
| UpdatedAt | time.Time | SQLite | For sorting |

## New Structures (In-Memory Only)

### SearchResult

Represents a note that matched a search query, with scoring and highlight info.

| Field | Type | Description |
|-------|------|-------------|
| Note | *Note | The matched note |
| Body | string | Note body (for highlighting) |
| Score | int | Relevance score (higher = more relevant) |
| TitleHighlight | template.HTML | Title with `<mark>` tags |
| TagsHighlight | []template.HTML | Tags with `<mark>` tags |
| BodySnippet | template.HTML | Excerpt of body with `<mark>` tags |

### SearchRequest

| Field | Type | Validation |
|-------|------|------------|
| Query | string | Required, trimmed, max 200 chars |
| UserID | int64 | From session (implicit) |

## Functions

### SearchNotes(db *DB, notesDir string, userID int64, query string) ([]SearchResult, error)

1. Load all notes for user from SQLite
2. Load each note's body from disk
3. Score each note against query using fuzzy matching
4. Filter notes with score > 0
5. Sort by score descending
6. Generate highlights for top N results
7. Return sorted results

### HighlightMatch(text, query string) template.HTML

1. Find fuzzy match positions in text
2. Wrap matched characters in `<mark>` tags
3. HTML-escape non-matched portions
4. Return safe HTML

### BodySnippet(body, query string, maxLen int) template.HTML

1. Find first match position in body
2. Extract surrounding context (±50 chars)
3. Apply highlighting
4. Add ellipsis if truncated
5. Return safe HTML

## Data Flow

```
User types query
       │
       ▼
GET /notes/search?q=...
       │
       ▼
Handler: NotesSearchGET
       │
       ├─► models.SearchNotes(db, notesDir, userID, query)
       │         │
       │         ├─► ListNotes (get metadata)
       │         │
       │         ├─► storage.ReadNote (get each body)
       │         │
       │         └─► Score + Filter + Sort + Highlight
       │
       ▼
Render search-results.html partial
       │
       ▼
htmx swaps into #note-list
```

## Indexes

No new indexes required. Existing indexes sufficient:
- `idx_note_user` on `notes(user_id)` — fast note listing per user
