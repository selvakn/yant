# Feature Specification: Note Archive

**Feature Branch**: `007-note-archive`  
**Created**: 2026-04-05  
**Status**: Draft  
**Input**: User description: "Add archive functionality, to archive the notes and it should be accessible (with search and tag filter) in a separate section. We should be able to bring back the archived notes."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Archive a note (Priority: P1)

The user archives a note to remove it from the active notes list without permanently deleting it. This helps declutter the main view while preserving content for future reference.

**Why this priority**: Core feature—users need to be able to archive before anything else works.

**Independent Test**: Click "Archive" on a note; the note disappears from the main list.

**Acceptance Scenarios**:

1. **Given** a note in the active notes list, **When** the user clicks the archive action, **Then** the note is removed from the active list.
2. **Given** an archived note, **When** the user views the active notes list, **Then** the archived note is not visible.
3. **Given** the user archives a note, **When** the action completes, **Then** a confirmation or undo option is shown briefly.

---

### User Story 2 - View archived notes (Priority: P1)

The user accesses a separate "Archive" section to browse all archived notes. This section functions like the main notes list with search and tag filtering.

**Why this priority**: Users need to access archived content to restore or review it.

**Independent Test**: Navigate to Archive section; see all previously archived notes.

**Acceptance Scenarios**:

1. **Given** the user has archived notes, **When** they navigate to the Archive section, **Then** all archived notes are displayed.
2. **Given** the Archive section is open, **When** the user searches for text, **Then** only matching archived notes appear.
3. **Given** the Archive section is open, **When** the user filters by tag, **Then** only archived notes with that tag appear.

---

### User Story 3 - Restore an archived note (Priority: P1)

The user restores an archived note to bring it back to the active notes list. The note returns exactly as it was before archiving.

**Why this priority**: Restoration completes the archive workflow and ensures archiving is reversible.

**Independent Test**: Click "Restore" on an archived note; the note appears in the active list and disappears from archive.

**Acceptance Scenarios**:

1. **Given** an archived note, **When** the user clicks "Restore", **Then** the note moves back to the active notes list.
2. **Given** a restored note, **When** the user views the active notes list, **Then** the note appears with all its original content, tags, and drawings intact.
3. **Given** a restored note, **When** the user views the Archive section, **Then** the restored note is no longer visible there.

---

### Edge Cases

- Archive section with no archived notes: display a friendly "No archived notes" message.
- Searching in archive with no matches: display "No notes found" message.
- Archiving a note that has a drawing: the drawing is preserved and restored along with the note.
- Tag sidebar behavior: archived notes' tags should not appear in the main tags sidebar; archived tags appear in the archive section's sidebar.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Each note MUST have an "Archive" action visible in the notes list and reader/editor views.
- **FR-002**: Archiving a note MUST immediately remove it from the active notes list.
- **FR-003**: A dedicated "Archive" section MUST be accessible from the main navigation.
- **FR-004**: The Archive section MUST display all archived notes belonging to the user.
- **FR-005**: The Archive section MUST support search filtering (same as main notes list).
- **FR-006**: The Archive section MUST support tag filtering (same as main notes list).
- **FR-007**: Each archived note MUST have a "Restore" action.
- **FR-008**: Restoring a note MUST return it to the active notes list with all content intact.
- **FR-009**: Archived notes MUST NOT appear in the main notes list or main search results.
- **FR-010**: Archived notes' tags MUST NOT appear in the main sidebar tag list.

### Key Entities

- **Note**: Extended with an "archived" status (true/false).
- **Archive Section**: A view/page dedicated to archived notes.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can archive a note in under 2 seconds (single click/action).
- **SC-002**: Users can access the Archive section in under 2 clicks from anywhere.
- **SC-003**: Users can restore an archived note in under 2 seconds (single click/action).
- **SC-004**: Search and filter in Archive section returns results in under 300ms.
- **SC-005**: 100% of note content, tags, and drawings are preserved through archive/restore cycle.

## Assumptions

- Archive is a soft state on the note; no data is lost when archiving.
- Only the note owner can archive/restore their notes (existing authentication applies).
- Archived notes remain editable if accessed directly (e.g., via direct URL).
- The "Archive" navigation item appears in the sidebar, similar to tag filters.
- No automatic archiving rules; archiving is always a manual user action.
- Archived notes count toward any storage limits (if applicable in future).
