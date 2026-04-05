# Feature Specification: Markdown Note Taking App

**Feature Branch**: `001-markdown-note-taking`
**Created**: 2026-04-05
**Status**: Draft
**Input**: User description: "note taking app, user should be able to author in markdown, with reader mode showing the render. Should be able to drag and drop images to the markdown. Should have a title, and date/time (created and updated) timestamps captured. Should be able to add tags (ex: #important) and the tags will be used for quick navigations."

## Clarifications

### Session 2026-04-05

- Q: Note visibility between users? → A: Notes are private per user (each user sees only their own).
- Q: Mock login — new username behavior? → A: Auto-create account on first login (no separate registration).
- Q: Tag scope — per-user or global? → A: Per-user (tag list derived only from that user's notes).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Create and Edit a Note (Priority: P1)

A user opens the app and creates a new note. They type a title, then
compose the body using Markdown syntax in an editor pane. As they write,
they can switch to a reader mode to see the rendered Markdown. The note
automatically records a created timestamp. Each subsequent edit updates
the last-modified timestamp. The user can return to the note later to
continue editing.

**Why this priority**: This is the fundamental capability of the app.
Without note creation and editing, no other feature has value.

**Independent Test**: Create a note with a title and Markdown body,
save it, reopen it, verify content persists and timestamps are correct.

**Acceptance Scenarios**:

1. **Given** the user is on the home screen, **When** they choose to
   create a new note, **Then** an editor opens with an empty title field
   and Markdown body area, and a created-at timestamp is recorded.
2. **Given** the user has typed Markdown content, **When** they switch
   to reader mode, **Then** the rendered HTML is displayed faithfully
   (headings, bold, italic, lists, code blocks, links).
3. **Given** the user edits an existing note, **When** they save,
   **Then** the updated-at timestamp refreshes to the current date/time
   while the created-at timestamp remains unchanged.
4. **Given** the user reopens a previously saved note, **When** the
   editor loads, **Then** the title, body, and both timestamps are
   displayed correctly.

---

### User Story 2 - Drag and Drop Images (Priority: P2)

While editing a note, the user drags an image file from their desktop
and drops it onto the editor area. The image is stored locally and a
Markdown image reference is inserted at the cursor position. In reader
mode, the image renders inline.

**Why this priority**: Image support greatly enriches notes (diagrams,
screenshots, photos) and is a core differentiator from plain text
editors.

**Independent Test**: Open a note, drag an image onto the editor,
verify the Markdown image syntax appears and the image renders in
reader mode.

**Acceptance Scenarios**:

1. **Given** the user is editing a note, **When** they drag and drop an
   image file onto the editor, **Then** the image is saved and a
   Markdown image reference (`![alt](path)`) is inserted at the drop
   position.
2. **Given** a note contains an image reference, **When** the user
   switches to reader mode, **Then** the image renders inline at the
   correct location.
3. **Given** the user drops a non-image file, **When** the drop event
   fires, **Then** the system ignores the drop or shows a brief
   notification that only image files are accepted.
4. **Given** the user drops multiple images at once, **When** the drop
   completes, **Then** each image is stored and a separate Markdown
   image reference is inserted for each.

---

### User Story 3 - Tag Notes for Quick Navigation (Priority: P3)

The user adds tags to a note using hashtag syntax (e.g., `#important`,
`#work`, `#ideas`). Tags can be added anywhere in the note body or in
a dedicated tags field. The app provides a navigation view that lists
all tags and allows the user to click a tag to see all notes with that
tag.

**Why this priority**: Tags are the primary organizational mechanism,
enabling users to find and group notes without a rigid folder hierarchy.

**Independent Test**: Create several notes with different tags, navigate
to the tag list, click a tag, and verify only matching notes appear.

**Acceptance Scenarios**:

1. **Given** the user types `#important` in a note, **When** the note
   is saved, **Then** the tag "important" is recognized and associated
   with the note.
2. **Given** multiple notes have the tag `#work`, **When** the user
   clicks the `#work` tag in the navigation panel, **Then** a filtered
   list of all notes tagged with `#work` is displayed.
3. **Given** the user removes a tag from a note, **When** the note is
   saved, **Then** the tag association is removed and the note no longer
   appears under that tag in navigation.
4. **Given** no notes have a particular tag, **When** the user views
   the tag list, **Then** that tag does not appear in the navigation.

---

### Edge Cases

- What happens when the user creates a note with no title? The system
  MUST assign a default title (e.g., "Untitled Note") and allow the
  user to rename it later.
- What happens when the user drags an extremely large image (>10 MB)?
  The system MUST either accept it with a warning about storage impact
  or reject it with a clear size-limit message.
- What happens when two tags differ only by case (e.g., `#Work` vs
  `#work`)? Tags MUST be treated as case-insensitive to avoid
  duplication.
- What happens when the user deletes a note that contains images?
  Associated image files MUST be cleaned up if no other note references
  them.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST allow users to create, edit, and delete notes
  composed of a title and a Markdown body.
- **FR-002**: System MUST provide an editor mode for authoring Markdown
  and a reader mode for viewing the rendered output.
- **FR-003**: System MUST automatically record a created-at timestamp
  when a note is first saved and update the updated-at timestamp on
  every subsequent save.
- **FR-004**: System MUST support drag-and-drop of image files into the
  editor, storing images locally and inserting Markdown image references.
- **FR-005**: System MUST accept common image formats (PNG, JPEG, GIF,
  WebP) via drag-and-drop.
- **FR-006**: System MUST parse hashtag-style tags (e.g., `#important`)
  from note content and make them available for navigation.
- **FR-007**: System MUST provide a tag-based navigation view that
  lists only the logged-in user's tags and filters their notes by
  selected tag.
- **FR-008**: System MUST treat tags as case-insensitive (e.g., `#Work`
  and `#work` resolve to the same tag).
- **FR-009**: System MUST display a list of the logged-in user's notes
  on the home screen, showing title, tags, and last-updated timestamp.
- **FR-010**: System MUST persist all notes and images across sessions
  (data survives app restart).
- **FR-011**: System MUST support multiple users. Each user logs in by
  entering a username (mock authentication; no password required).
- **FR-012**: System MUST isolate notes per user. A user MUST only see,
  edit, and delete their own notes and images.
- **FR-013**: System MUST auto-create a new user account when an
  unrecognized username is entered at login (no separate registration
  step).

### Key Entities *(include if feature involves data)*

- **User**: A person who uses the app. Attributes: username (unique
  identifier). Authentication is mocked (username-only login); real
  authentication will be added in a future iteration.
- **Note**: The primary content object. Attributes: title, Markdown
  body, created-at timestamp, updated-at timestamp, associated tags,
  referenced images, owner (User).
- **Tag**: A label derived from hashtag syntax in note content. Used
  for grouping and filtering notes. Case-insensitive, unique by
  normalized name.
- **Image**: A binary file (PNG, JPEG, GIF, WebP) attached to one or
  more notes. Stored locally with a reference path embedded in the
  note's Markdown body.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can create a new note and see it in the note list
  within 2 seconds.
- **SC-002**: Switching between editor and reader mode completes in
  under 1 second with no content loss.
- **SC-003**: Dragging and dropping an image results in a visible
  image reference in the editor within 2 seconds.
- **SC-004**: Tag navigation filters and displays matching notes in
  under 1 second for a collection of up to 1,000 notes.
- **SC-005**: All note data (title, body, timestamps, tags, images)
  persists correctly across application restarts with zero data loss.
- **SC-006**: 90% of first-time users can create a tagged note with
  an image without external instructions.

## Assumptions

- The app supports multiple users with private, isolated notes.
- The app runs in a modern web browser (Chrome, Firefox, Safari,
  Edge — latest two major versions).
- Notes are stored locally; cloud sync is out of scope for this
  feature.
- There is no upper limit on number of notes, but performance targets
  (SC-004) are defined for up to 1,000 notes.
- Authentication is mocked (username-only login, no password). Real
  authentication will be added in a future iteration.
- Image storage is on the local filesystem; no external CDN or object
  storage is involved.
