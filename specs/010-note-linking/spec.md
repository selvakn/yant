# Feature Specification: Note Linking with Backlinks

**Feature Branch**: `010-note-linking`  
**Created**: 2026-04-06  
**Status**: Draft  
**Input**: User description: "Should be able to link one note from another. Come up with easy way to do that. Should be able to do that while typing, in the markdown itself. In every note, show the notes which links to it, as a list, at the bottom, with their title."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Link to Another Note While Typing (Priority: P1)

A user is writing a note and wants to reference another note. They type `[[` which triggers an autocomplete dropdown showing matching note titles. They select a note, and a link in the format `[[note title]]` is inserted into the markdown. When the note is saved and viewed, the link renders as a clickable link to the referenced note.

**Why this priority**: This is the core interaction -- without inline linking, the entire feature has no value.

**Independent Test**: Create two notes. In the second note, type `[[` followed by a few characters of the first note's title. Select from autocomplete. Save. View the note and verify the link navigates to the first note.

**Acceptance Scenarios**:

1. **Given** a user is editing a note, **When** they type `[[`, **Then** an autocomplete dropdown appears showing note titles matching subsequent keystrokes.
2. **Given** the autocomplete is showing results, **When** the user selects a note, **Then** the text `[[selected note title]]` is inserted at the cursor position.
3. **Given** a saved note contains `[[other note title]]`, **When** the note is viewed in the reader, **Then** it renders as a clickable link that navigates to the referenced note.
4. **Given** a saved note contains `[[nonexistent title]]`, **When** the note is viewed, **Then** the link text is still displayed but not as a clickable link (shown as plain text or visually distinct as a broken link).

---

### User Story 2 - View Backlinks on a Note (Priority: P1)

When viewing any note, the user sees a "Linked from" section at the bottom listing all other notes that link to this note. Each backlink shows the linking note's title as a clickable link.

**Why this priority**: Backlink discovery is the second half of the feature -- linking is only useful if you can see what links to a note.

**Independent Test**: Create note A and note B. In note B, add a link to note A using `[[A's title]]`. View note A and verify note B appears in the backlinks section at the bottom.

**Acceptance Scenarios**:

1. **Given** note B contains a link `[[note A title]]`, **When** the user views note A, **Then** a "Linked from" section at the bottom lists note B with its title as a clickable link.
2. **Given** multiple notes link to note A, **When** the user views note A, **Then** all linking notes are listed in the backlinks section.
3. **Given** no notes link to note A, **When** the user views note A, **Then** no backlinks section is shown.
4. **Given** note B links to note A and note B is later deleted, **When** the user views note A, **Then** note B no longer appears in the backlinks.

---

### User Story 3 - Backlinks Update on Save (Priority: P2)

When a user adds or removes a `[[link]]` in a note and saves it, the backlinks on the referenced notes update accordingly. This happens automatically without any manual refresh or rebuild.

**Why this priority**: Ensures data consistency, but the system works for basic usage even with slightly stale backlinks.

**Independent Test**: Create note A and note B. Edit note B to add `[[note A title]]`, save. View note A and confirm backlink. Edit note B again, remove the link, save. View note A and confirm the backlink is gone.

**Acceptance Scenarios**:

1. **Given** a user adds `[[note A title]]` to note B, **When** note B is saved, **Then** note A's backlinks include note B immediately.
2. **Given** a user removes a `[[note A title]]` link from note B, **When** note B is saved, **Then** note A's backlinks no longer include note B.

---

### Edge Cases

- What happens when a linked note's title changes? The link uses the title at the time of writing. The system resolves links by matching the title text, so a renamed note may cause existing links to become broken.
- What happens when the user types `[[` but there are no matching notes? The autocomplete shows an empty state or a "no matches" message.
- What happens when a note links to itself? Self-links are allowed but the note does not appear in its own backlinks.
- What happens with archived notes? Links to archived notes still render as clickable links. Archived notes can show backlinks.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST support a `[[note title]]` syntax in note markdown for linking to other notes.
- **FR-002**: The editor MUST provide autocomplete suggestions when the user types `[[`, showing matching note titles filtered by subsequent keystrokes.
- **FR-003**: The reader MUST render `[[note title]]` as a clickable link navigating to the referenced note.
- **FR-004**: The reader MUST display unresolvable links (where no note matches the title) as visually distinct non-clickable text.
- **FR-005**: The reader MUST display a "Linked from" section at the bottom of each note listing all notes that contain a link to the current note.
- **FR-006**: Each backlink entry MUST show the linking note's title as a clickable link.
- **FR-007**: The system MUST update backlink data when a note is saved (links added or removed).
- **FR-008**: The system MUST remove backlink references when a linking note is deleted.
- **FR-009**: The autocomplete MUST only show notes belonging to the current user.
- **FR-010**: The backlinks section MUST NOT appear when no notes link to the current note.

### Key Entities

- **Note Link**: A directional reference from one note to another, identified by the linked note's title text as written in the `[[...]]` syntax. Resolved to a target note at render time.
- **Backlink**: The inverse of a note link -- a record that note X is referenced by note Y, used to populate the "Linked from" section.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can insert a note link in under 5 seconds using the `[[` autocomplete.
- **SC-002**: 100% of valid `[[note title]]` references render as working clickable links in the reader.
- **SC-003**: Backlinks appear on the referenced note within 1 second of saving the linking note.
- **SC-004**: The backlinks section accurately reflects all current incoming links with zero stale entries.

## Assumptions

- The `[[note title]]` syntax is a well-known convention (used by Obsidian, Notion, etc.) and will be intuitive to users.
- Link resolution is case-insensitive to match the existing case-insensitive tag behavior.
- Links are scoped to the current user's notes -- a user cannot link to another user's notes.
- The existing markdown rendering pipeline can be extended to handle the new `[[...]]` syntax.
- The database rebuild command (`--rebuild-db`) will also rebuild link/backlink data from the markdown files.
