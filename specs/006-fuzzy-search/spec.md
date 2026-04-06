# Feature Specification: Fuzzy Search for Notes

**Feature Branch**: `006-fuzzy-search`  
**Created**: 2026-04-05  
**Status**: Draft  
**Input**: User description: "Add a search functionality. It should filter the notes based on the title, tags as well as the body contents. And it should filter the notes on keystroke. Implement a fuzzy search so that the search is more useful."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Search notes as you type (Priority: P1)

The user types in a search box and sees the note list filter in real-time. Matching notes appear instantly without pressing Enter or clicking a button.

**Why this priority**: Core interaction—real-time feedback is the main UX differentiator.

**Independent Test**: Type "meet" in the search box; notes containing "meeting", "Meet", or "meetings" appear immediately; notes without matches disappear.

**Acceptance Scenarios**:

1. **Given** the notes list page, **When** the user types in the search box, **Then** the list updates after each keystroke (with a small debounce for performance).
2. **Given** a search term, **When** results appear, **Then** matching portions are visually highlighted in the results.
3. **Given** the user clears the search box, **When** the input is empty, **Then** all notes are shown again.

---

### User Story 2 - Fuzzy matching across title, tags, and body (Priority: P2)

The search tolerates typos and partial matches. Searching "mtng" should find "meeting". Results are ranked by relevance—title matches rank higher than body matches.

**Why this priority**: Fuzzy matching makes search more forgiving and useful.

**Independent Test**: Type "recpie" (misspelled); note titled "Recipe Ideas" appears in results.

**Acceptance Scenarios**:

1. **Given** a search term with a typo, **When** results are shown, **Then** notes with similar words still appear.
2. **Given** multiple matching notes, **When** results are displayed, **Then** title matches appear before tag matches, which appear before body-only matches.
3. **Given** a search term, **When** it matches a tag, **Then** notes with that tag appear even if the term isn't in the title or body.

---

### User Story 3 - Navigate and open results (Priority: P3)

The user can use keyboard arrows to navigate results and press Enter to open the selected note.

**Why this priority**: Power-user efficiency after basic search works.

**Independent Test**: Type a query, press Down Arrow twice, press Enter; the third result opens.

**Acceptance Scenarios**:

1. **Given** search results are displayed, **When** the user presses Down/Up arrow, **Then** the selection moves through results.
2. **Given** a result is selected, **When** the user presses Enter, **Then** that note opens in reader or editor view.

---

### Edge Cases

- Empty search input: show all notes (no filter).
- No matches: display a friendly "No notes found" message.
- Very long search queries: handle gracefully without performance degradation.
- Special characters in search: treat as literal text, no regex injection.
- Notes with no body: still searchable by title and tags.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The notes list page MUST include a search input field.
- **FR-002**: As the user types, the note list MUST filter to show only matching notes (debounced by ~150-300ms).
- **FR-003**: Search MUST match against note title, tags, and body content.
- **FR-004**: Search MUST use fuzzy matching to tolerate typos and partial matches.
- **FR-005**: Results MUST be ranked by relevance: title matches > tag matches > body matches.
- **FR-006**: Matching text MUST be highlighted in the displayed results.
- **FR-007**: Keyboard navigation (Up/Down arrows, Enter to open) MUST be supported.
- **FR-008**: Clearing the search input MUST restore the full note list.
- **FR-009**: Search MUST be case-insensitive.

### Key Entities

- **Search Query**: User-entered text used to filter notes.
- **Search Result**: A note that matches the query, with relevance score and match highlights.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Search results update within 300ms of the user stopping typing.
- **SC-002**: Fuzzy search finds relevant notes for at least 80% of single-typo queries in testing.
- **SC-003**: Users can find a specific note in under 5 seconds using search (vs. scrolling).
- **SC-004**: Keyboard navigation works without requiring mouse interaction.

## Assumptions

- Search is client-side for MVP if note count is small (<500 notes per user); server-side index can be added later for scale.
- Fuzzy matching uses a standard algorithm (e.g., Levenshtein distance, trigram similarity, or similar).
- Search only applies to the logged-in user's notes (same access rules as note list).
- Search does not persist across page reloads; each visit starts with no filter.
