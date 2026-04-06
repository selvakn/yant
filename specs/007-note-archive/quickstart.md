# Quickstart: Note Archive

**Feature**: 007-note-archive
**Date**: 2026-04-05

## Prerequisites

- Go 1.22+
- Existing my-notes development environment
- Notes in the system to archive

## Setup

No new dependencies required.

### Implementation Order

1. **Schema migration** (`backend/internal/models/models.go`)
   - Add `archived` column to InitSchema
   - Update Note struct with Archived field
   - Update scanNote to include archived

2. **Model functions** (`backend/internal/models/models.go`)
   - Update ListNotes with archived filter
   - Add ArchiveNote function
   - Add RestoreNote function
   - Update ListTagsForUser with archived filter

3. **Search update** (`backend/internal/models/search.go`)
   - Add archived parameter to SearchNotes

4. **Handlers** (`backend/internal/handlers/`)
   - Add ArchivePUT handler in notes.go
   - Add RestorePUT handler in notes.go
   - Create archive.go with ArchiveListGET, ArchiveSearchGET

5. **Routes** (`backend/cmd/server/main.go`)
   - Register PUT /notes/{slug}/archive
   - Register PUT /notes/{slug}/restore
   - Register GET /archive
   - Register GET /archive/search

6. **Templates** (`frontend/templates/`)
   - Add Archive link to sidebar in base.html
   - Create archive/list.html
   - Create archive/search-results.html
   - Add Archive button to notes/list.html items
   - Add Restore button to archive/list.html items

7. **Tests** (`backend/internal/handlers/handlers_test.go`)
   - Test archive action
   - Test restore action
   - Test archive list
   - Test archive search
   - Test tag isolation

## Verification

### Manual Testing

1. Start server: `make run`
2. Create a few notes with tags
3. Archive a note from the list
4. Verify it disappears from main list
5. Navigate to Archive section
6. Verify archived note appears
7. Search within archive
8. Filter by tag within archive
9. Restore a note
10. Verify it returns to main list
11. Verify tags update in both sections

### Automated Tests

```bash
cd backend
go test ./... -v -cover
```

Expected coverage: ≥90%

## API Reference

### PUT /notes/{slug}/archive

Archive a note.

**Response**: 200 OK with HX-Redirect to /notes

### PUT /notes/{slug}/restore

Restore an archived note.

**Response**: 200 OK with HX-Redirect to /archive

### GET /archive

List all archived notes.

**Query Parameters**:
| Param | Type | Required | Description |
|-------|------|----------|-------------|
| tag | string | No | Filter by tag |

### GET /archive/search

Search archived notes.

**Query Parameters**:
| Param | Type | Required | Description |
|-------|------|----------|-------------|
| q | string | Yes | Search query |

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Archived notes appear in main list | Check ListNotes archived filter |
| Tags from archived notes in main sidebar | Check ListTagsForUser archived filter |
| Archive section not loading | Check route registration order (/archive before /notes/{slug}) |
| Search not respecting archive context | Check SearchNotes archived parameter |
