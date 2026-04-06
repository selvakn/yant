# Data Model: Note Archive

**Feature**: 007-note-archive
**Date**: 2026-04-05

## Overview

Extend the existing `notes` table with an `archived` column. No new tables required.

## Schema Changes

### notes table (modified)

| Column | Type | Default | Notes |
|--------|------|---------|-------|
| id | INTEGER | AUTO | Primary key (existing) |
| user_id | INTEGER | - | Foreign key to users (existing) |
| slug | TEXT | - | URL-safe identifier (existing) |
| title | TEXT | 'Untitled Note' | Note title (existing) |
| created_at | TEXT | - | RFC3339 timestamp (existing) |
| updated_at | TEXT | - | RFC3339 timestamp (existing) |
| **archived** | INTEGER | 0 | **NEW**: 0=active, 1=archived |

### New Index

```sql
CREATE INDEX IF NOT EXISTS idx_note_archived ON notes(user_id, archived);
```

## Entity Changes

### Note struct (Go)

```go
type Note struct {
    ID        int64
    UserID    int64
    Slug      string
    Title     string
    Tags      []string
    Archived  bool      // NEW
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

## Function Signatures

### Modified Functions

```go
// ListNotes - add archived filter
func ListNotes(db *DB, userID int64, tag string, archived bool) ([]*Note, error)

// SearchNotes - add archived filter
func SearchNotes(db *DB, notesDir string, userID int64, query string, archived bool) ([]SearchResult, error)

// ListTagsForUser - add archived filter
func ListTagsForUser(db *DB, userID int64, archived bool) ([]TagCount, error)
```

### New Functions

```go
// ArchiveNote - set archived=1
func ArchiveNote(db *DB, userID int64, slug string) error

// RestoreNote - set archived=0
func RestoreNote(db *DB, userID int64, slug string) error
```

## State Transitions

```
┌─────────────┐                    ┌─────────────┐
│   Active    │ ─── Archive ────▶  │  Archived   │
│ (archived=0)│ ◀─── Restore ───── │ (archived=1)│
└─────────────┘                    └─────────────┘
```

## Data Invariants

1. A note can only be in one state: active OR archived (never both)
2. Archiving preserves all note content (title, body, tags, drawings)
3. Restoring returns note to exact previous state
4. Archived notes remain accessible via direct URL (for editing)

## Migration Strategy

1. Check if `archived` column exists
2. If not, run: `ALTER TABLE notes ADD COLUMN archived INTEGER NOT NULL DEFAULT 0`
3. Create index if not exists

No data migration needed — all existing notes default to archived=0 (active).
