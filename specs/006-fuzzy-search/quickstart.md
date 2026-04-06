# Quickstart: Fuzzy Search for Notes

**Feature**: 006-fuzzy-search
**Date**: 2026-04-05

## Prerequisites

- Go 1.22+
- Existing my-notes development environment
- Notes in the system to search

## Setup

### 1. Add dependency

```bash
cd backend
go get github.com/lithammer/fuzzysearch/fuzzy
```

### 2. Implementation order

1. **Add search model functions** (`backend/internal/models/`)
   - `SearchResult` struct
   - `SearchNotes()` function
   - `ScoreNote()` helper
   - `HighlightMatch()` helper

2. **Add search handler** (`backend/internal/handlers/notes.go`)
   - `NotesSearchGET` handler
   - Register route in `main.go`

3. **Add search UI** (`frontend/templates/notes/`)
   - Update `list.html` with search input
   - Create `search-results.html` partial

4. **Add keyboard navigation** (`frontend/static/js/`)
   - Arrow key navigation
   - Enter to open selection

5. **Add styles** (`frontend/static/css/app.css`)
   - Search input styling
   - `<mark>` highlight styling
   - Selected result styling

## Verification

### Manual testing

1. Start server: `make run`
2. Log in and navigate to notes list
3. Type in search box
4. Verify:
   - Results filter as you type
   - Typos still find relevant notes
   - Matches are highlighted
   - Title matches rank above body matches
   - Arrow keys navigate results
   - Enter opens selected note
   - Clearing input shows all notes

### Automated tests

```bash
cd backend
go test ./... -v -cover
```

Expected test coverage: ≥90%

## API Reference

### GET /notes/search

Search notes for the logged-in user.

**Query Parameters**:
| Param | Type | Required | Description |
|-------|------|----------|-------------|
| q | string | Yes | Search query (1-200 chars) |

**Response**: HTML partial with matching notes (for htmx swap)

**Example**:
```bash
curl -b cookies.txt "http://localhost:8080/notes/search?q=meeting"
```

Returns:
```html
<li class="note-item search-result selected">
  <a href="/notes/team-meeting" class="note-title">
    Team <mark>Meeting</mark> Notes
  </a>
  <div class="note-meta">
    <span class="tag">#work</span>
    <span class="note-date">Apr 5, 2026</span>
  </div>
  <div class="note-snippet">
    ...discussed the quarterly <mark>meeting</mark> schedule...
  </div>
</li>
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| No results appear | Check browser console for htmx errors; verify user has notes |
| Results lag | Increase debounce delay in `hx-trigger` |
| Highlighting breaks layout | Ensure `<mark>` styles don't affect sizing |
| Keyboard nav not working | Check JS console for errors; verify event listeners |
