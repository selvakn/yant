# Research: Note Archive

**Feature**: 007-note-archive
**Date**: 2026-04-05

## Research Questions

### 1. Archive State Storage

**Decision**: Add `archived` boolean column to `notes` table (SQLite)

**Rationale**:
- Archive is metadata, not content — belongs in SQLite index
- Markdown files remain unchanged (portable per Principle I)
- Simple boolean: archived = 1 or 0
- Default to 0 (not archived) for existing notes

**Alternatives Considered**:

| Approach | Pros | Cons | Rejected Because |
|----------|------|------|------------------|
| Move files to archive/ folder | Visual separation | Breaks URLs, complex | Violates Markdown portability |
| Rename files with .archived | No DB change | File system clutter, search issues | Complicates file handling |
| Soft delete timestamp | More info (when archived) | Overkill for simple on/off | YAGNI - Principle II |

### 2. Query Strategy

**Decision**: Filter by `archived` column in ListNotes; separate functions for archive list

**Rationale**:
- ListNotes already takes filters (tag); add archived parameter
- Default behavior: exclude archived notes (archived=0)
- Archive section: query archived=1 only
- Search respects archive context (main vs archive section)

**Implementation**:
```go
// ListNotes gains an optional archived filter
func ListNotes(db *DB, userID int64, tag string, archived bool) ([]*Note, error)

// Archive-specific list
func ListArchivedNotes(db *DB, userID int64, tag string) ([]*Note, error)
```

### 3. UI Integration

**Decision**: Archive/Restore buttons in note actions; Archive link in sidebar

**Rationale**:
- Archive button: visible on note list items, reader, editor
- Restore button: visible only in archive section
- Archive nav: sidebar link below tags, above "All Notes"
- Consistent with existing htmx patterns

**UI Flow**:
```
Main Notes List:
  [Note] → Archive button → POST /notes/{slug}/archive
           → htmx removes note from list

Archive Section:
  [Note] → Restore button → POST /notes/{slug}/restore
           → htmx removes note from archive list
```

### 4. Tag Sidebar Behavior

**Decision**: Two separate tag lists based on context

**Rationale**:
- Main sidebar: tags from active (non-archived) notes only
- Archive sidebar: tags from archived notes only
- Prevents confusion about which notes are filtered

**Implementation**:
- Existing `ListTagsForUser` gets `archived` parameter
- Sidebar template conditionally shows archive-specific tags

### 5. Search Integration

**Decision**: Search respects current context (active vs archive)

**Rationale**:
- Main search: searches active notes only
- Archive search: searches archived notes only
- Existing SearchNotes gets `archived` parameter
- No cross-section search needed

### 6. Schema Migration

**Decision**: Add column with ALTER TABLE; default to 0

**Rationale**:
- SQLite ALTER TABLE ADD COLUMN is simple
- Default 0 means existing notes remain active
- No data migration needed
- RebuildDB updated to preserve archived status if present in filename metadata

**Migration**:
```sql
ALTER TABLE notes ADD COLUMN archived INTEGER NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_note_archived ON notes(user_id, archived);
```

## Summary

| Decision | Choice |
|----------|--------|
| Storage | Boolean `archived` column in SQLite |
| Query | Filter in ListNotes, separate archive list |
| UI | Archive/Restore buttons, sidebar nav link |
| Tags | Context-specific tag lists |
| Search | Respects current context |
| Migration | ALTER TABLE with default 0 |
