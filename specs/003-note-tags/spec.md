# Feature Specification: Seamless note tags

**Feature Branch**: `003-note-tags`  
**Created**: 2026-04-05  
**Status**: Ready for review  
**Input**: User description: "should be able to add tags in the notes. think of best and simplest experience to add the tags, it should be seemless during note taking (to add tags and remove tags). once its specified, plan and breakdown the tasks and go ahead and implement them as well"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See and manage tags while writing (Priority: P1)

While editing a note, the user sees which tags apply to that note without scanning the full text. They can remove a tag in one action (for example dismissing a tag next to the editor) and the note content updates to match. Tags stay consistent with what is stored for search and browsing elsewhere.

**Why this priority**: Core “seamless” value—tagging must not interrupt writing flow.

**Independent Test**: Open any note with tags in the body; confirm tags appear in a dedicated area; remove one tag and save; confirm the tag no longer appears in that area or in the note’s tag list after save.

**Acceptance Scenarios**:

1. **Given** a note whose content includes at least one valid tag token, **When** the user opens the note for editing, **Then** each distinct tag is shown in a compact summary area near the editor (not only buried in the text).
2. **Given** the summary area shows a tag, **When** the user removes that tag using the summary control, **Then** the tag is removed from the note content such that it would no longer be counted as attached to the note after save.
3. **Given** the user is editing, **When** they save the note, **Then** browsing and filtering by tags reflects the tags implied by the saved content.

---

### User Story 2 - Add tags without breaking flow (Priority: P2)

The user can add a new tag from the editing screen using a small, obvious control (for example typing a name and confirming), without leaving the page or opening a separate screen. Optionally, they can still type tags directly in the note text as before.

**Why this priority**: Completes “add” parity with “remove” and supports users who do not want to manage hashtags manually in prose.

**Independent Test**: From the editor, add a tag only via the quick-add control; save; confirm the tag appears wherever tags are listed for that note and for filtering.

**Acceptance Scenarios**:

1. **Given** the user is editing a note, **When** they enter a valid tag name via the quick-add path and confirm, **Then** the note content gains that tag in a predictable way and the summary area updates to include it.
2. **Given** duplicate or empty input, **When** the user tries to add a tag, **Then** the system avoids creating duplicate tags or blank tags (clear, minimal friction).

---

### User Story 3 - Reuse existing tags quickly (Priority: P3)

When adding a tag, the user can see suggestions from tags they have used before (on other notes), so spelling stays consistent and entry is faster.

**Why this priority**: Improves speed and consistency; secondary to basic add/remove.

**Independent Test**: With at least two existing tags in the account, start adding a tag; suggestions include existing names; selecting or completing one adds it correctly.

**Acceptance Scenarios**:

1. **Given** the user has created tags before, **When** they use the quick-add control, **Then** they can choose or complete from prior tag names without typing the full name from memory.

---

### Edge Cases

- Tag names that are invalid (empty, unsupported characters, or excessively long) are rejected or normalized with a clear, low-friction response—no silent data loss.
- Removing a tag when the same token appears multiple times removes the tag association without leaving stray invalid markup.
- Very long notes: updating the summary remains responsive during typing (no disruptive lag).
- Concurrent sessions: last save wins for note content; no new cross-session merge requirement in scope.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: While editing a note, the system MUST show a concise summary of all tags that apply to that note according to the same rules used elsewhere for tag extraction from note content.
- **FR-002**: The user MUST be able to remove a listed tag from the summary without manually finding every occurrence in the text, and the underlying content MUST be updated so the tag is no longer associated after save.
- **FR-003**: The user MUST be able to add a new tag from the editing screen with at most two steps after focusing the add control (enter name and confirm), without navigating away.
- **FR-004**: Adding or removing tags via the summary MUST update the editable note content so that saving persists tags consistently with list and filter views.
- **FR-005**: The system MUST ignore or normalize invalid tag input and MUST NOT create duplicate entries for the same logical tag on the same note.
- **FR-006**: Where prior tags exist for the user, the quick-add path MUST offer a way to reuse those names (suggestions or autocomplete), without requiring the user to remember exact spelling.

### Key Entities

- **Tag**: A short label attached to a note, derived from note content and used for filtering and discovery; uniqueness rules match the product’s existing tag model.
- **Note (editing session)**: The draft content shown in the editor, including any tags represented in text or updated via the summary controls until saved.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can add a new tag using only the quick-add path and confirm it appears in the tag summary within 5 seconds of completion under typical conditions.
- **SC-002**: A user can remove a tag from the summary and, after save, that tag no longer appears in the note’s tag summary or in tag-based filtering for that note (100% consistency in manual test scenarios).
- **SC-003**: At least 90% of participants in informal testing report that tagging feels “part of writing” rather than a separate task (survey or hallway test).
- **SC-004**: With 50 notes and 20 distinct tags, opening the editor and typing for 30 seconds does not produce noticeable input stalls attributable to tag UI updates.

## Assumptions

- Tags continue to be represented in note content using the product’s established token rules so that files remain portable and the database index stays aligned with markdown source.
- Users are already authenticated; tagging applies per user as today.
- Scope is the note editor experience; changing global tag management or sharing tags across users is out of scope.
- “Valid tag” character rules follow a single documented pattern shared between display and storage (letters, digits, underscores, and hyphenated words unless restricted for simplicity).
