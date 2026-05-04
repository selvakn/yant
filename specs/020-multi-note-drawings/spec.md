# Feature Specification: Multiple Drawings Per Note

**Feature Branch**: `020-multi-note-drawings`  
**Created**: 2026-05-04  
**Status**: Draft  
**Input**: User description: "lets add the ability to attach multiple drawings with the notes."

## Clarifications

### Session 2026-05-04

- Q: Where do drawings appear relative to the note content in reader mode? → A: Drawings are embedded at specific positions within the markdown via a syntax marker (e.g., `![[draw:name]]`). By default, when a new drawing is created, the marker is appended at the end of the document. Users can move the marker to any position in the markdown to control where the drawing renders.
- Q: Is the drawing name the stable identifier (used in files/markers), or is there a separate ID? → A: Use an auto-generated stable ID (e.g., short slug or random token) for file names and syntax markers. The display name is stored as separate metadata. Renaming a drawing is a cheap metadata update — no file renames, no marker changes, no version history disruption.
- Q: Where should drawing metadata (display name, tool type) be stored? → A: SQLite `note_drawings` table (drawing_id, note_id, display_name, tool_type, created_at, updated_at), consistent with existing patterns. Markdown files remain source of truth for marker positions; drawing JSON files remain source of truth for drawing content.
- Q: How should existing single-drawing notes (legacy file format) be handled? → A: Lazy/transparent migration. The system detects legacy files (`<slug>.tldraw.json` / `<slug>.excalidraw.json`) at read time and serves them as-is. Conversion to the new naming scheme and `note_drawings` row insertion happens only on first edit or when a second drawing is added to the note.
- Q: How does the user interact with drawing markers in the editor? → A: The "Add drawing" button inserts a marker at the current cursor position (or end of document if no cursor). Clicking an existing marker line in the editor opens the drawing canvas for that drawing. This keeps the editor consistent with EasyMDE's plain-text model.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Add Multiple Drawings to a Note (Priority: P1)

A user is writing a note and wants to include several visual diagrams at different points in the text. Instead of being limited to a single drawing, the user can add a new drawing at any time, each with its own name/label and independent canvas. When a drawing is created, a syntax marker (e.g., `![[draw:name]]`) is inserted into the markdown — by default at the end of the document. The user can then move this marker to any position in the markdown to control where the drawing renders. Each drawing can be opened, edited, and saved independently.

**Why this priority**: The core value of this feature is removing the one-drawing limit. Without the ability to create and manage multiple drawings, the feature has no purpose.

**Independent Test**: Open a note, add a first drawing, save it, then add a second drawing with a different name. Verify both drawings appear in the markdown as syntax markers and each opens its own independent canvas.

**Acceptance Scenarios**:

1. **Given** a note with no existing drawings, **When** the user clicks "Add drawing," **Then** the tool selection UI appears (Excalidraw or tldraw), followed by a prompt to name the drawing, and a syntax marker is inserted at the current cursor position (or end of document if no cursor).
2. **Given** a note that already has one drawing, **When** the user clicks "Add drawing" again, **Then** a new drawing can be created with a different name and tool choice, and a new marker is inserted at the current cursor position (or end of document).
3. **Given** a note with multiple drawing markers in the markdown, **When** the user views the note in edit mode, **Then** each marker is visible as plain text in the editor and the user can click on a marker line to open that drawing's canvas for editing.
4. **Given** a note with multiple drawings, **When** the user saves the note, **Then** all drawings retain their content and marker positions, and are independently accessible on subsequent visits.
5. **Given** a drawing marker in the markdown, **When** the user moves the marker to a different position in the text, **Then** the drawing renders at the new position in reader mode.

---

### User Story 2 - View Multiple Drawings in Reader Mode (Priority: P2)

When a user views a note in reader mode, each drawing renders at the position of its syntax marker within the markdown content. Each drawing displays its name/label as a heading and renders in read-only mode using the appropriate tool (Excalidraw or tldraw). The user can scroll through the note and see drawings interspersed with text at the positions the author chose.

**Why this priority**: Reader mode is the primary way notes are consumed. Drawings must render at their intended positions within the note content for the feature to deliver value.

**Independent Test**: Create a note with two drawings (one Excalidraw, one tldraw) with markers placed at different positions in the markdown. Navigate to reader mode and verify each renders at its marker position, labeled, and read-only.

**Acceptance Scenarios**:

1. **Given** a note with drawing markers at specific positions in the markdown, **When** the user views the note in reader mode, **Then** each drawing renders at its marker position with its name as a heading.
2. **Given** a note with drawings of mixed types (Excalidraw and tldraw), **When** viewed in reader mode, **Then** each drawing renders using its respective tool's read-only viewer at the correct position.
3. **Given** a note with multiple drawings in reader mode, **When** the user interacts with any drawing area, **Then** no modifications can be made.
4. **Given** a drawing marker exists in the markdown but the referenced drawing has been deleted, **When** viewed in reader mode, **Then** the marker position shows a placeholder indicating the drawing is missing.

---

### User Story 3 - Edit and Delete Individual Drawings (Priority: P3)

A user can select any individual drawing from a note to edit, rename, or delete it — without affecting the other drawings. Deleting a drawing removes it from the note entirely. Renaming changes only the label displayed in the list and reader mode.

**Why this priority**: Managing individual drawings (edit, rename, delete) is essential for a usable multi-drawing experience. Without granular control, users cannot maintain their notes effectively.

**Independent Test**: Create a note with two drawings. Edit the first drawing's content, rename the second, delete the first, and verify the second remains intact and correctly named.

**Acceptance Scenarios**:

1. **Given** a note with multiple drawings, **When** the user selects one drawing to edit, **Then** only that drawing's canvas opens and the others remain unchanged.
2. **Given** a note with multiple drawings, **When** the user deletes one drawing, **Then** it is removed from the list and disk, and the remaining drawings are unaffected.
3. **Given** a note with a drawing, **When** the user renames its display name, **Then** the new name appears in the drawing list and in reader mode, while the syntax marker in the markdown remains unchanged (it uses the stable ID).

---

### User Story 4 - Version Control for Multiple Drawings (Priority: P4)

Each drawing change is tracked independently in the note's version history. When a user modifies one drawing, only that drawing's change appears in the version diff. History, diff viewing, and revert all work per-drawing, consistent with the existing version control behavior.

**Why this priority**: Version control is an existing capability that must extend to multiple drawings. Without it, the multi-drawing feature would be a regression from the current single-drawing version control.

**Independent Test**: Create a note with two drawings. Edit drawing A, save. Edit drawing B, save. View history and verify two separate version entries appear. Revert the note to before drawing B was edited and verify drawing A still has its latest state while drawing B is restored.

**Acceptance Scenarios**:

1. **Given** a note with multiple drawings, **When** the user edits and saves one drawing, **Then** a new version entry appears in the note's history referencing that specific drawing change.
2. **Given** multiple versions involving different drawings, **When** the user views a diff, **Then** the diff shows the correct drawing's before-and-after state.
3. **Given** a user wants to revert, **When** they revert to a previous version, **Then** all drawings are restored to their state at that version (including any that were added or removed since).

---

### User Story 5 - Multiple Drawings in Shared and Public Notes (Priority: P5)

When a note with multiple drawings is shared with another user or published as a public note, all drawings are visible to the viewer. Shared notes with edit access allow the collaborator to add, edit, and delete drawings. Public notes display all drawings in read-only mode.

**Why this priority**: Sharing and public notes are existing features that must work with multiple drawings for consistency.

**Independent Test**: Share a note with two drawings. Verify the recipient sees both drawings. Publish the note publicly and verify the public URL renders all drawings.

**Acceptance Scenarios**:

1. **Given** a note with multiple drawings shared with read access, **When** the recipient views the note, **Then** all drawings render in read-only mode.
2. **Given** a note with multiple drawings shared with edit access, **When** the collaborator opens the note, **Then** they can add new drawings and edit existing ones.
3. **Given** a note with multiple drawings published publicly, **When** an anonymous user visits the public URL, **Then** all drawings are rendered inline in read-only mode.

---

### Edge Cases

- What happens when a user adds two drawings with the same display name? The system shows a validation warning but allows it since drawings are identified by stable ID, not name. Unique names are recommended but not enforced as a hard constraint.
- What happens when a drawing is deleted and a new one is created with the same name? The system allows it — the old drawing's history remains in version control, and the new one starts fresh.
- What happens when all drawings are deleted from a note? The note reverts to having no drawings, and "Add drawing" shows the tool selection UI as if it were a new note.
- What happens if the user wants to reorder drawings? The user moves the syntax markers to different positions in the markdown. No separate reorder UI is needed.
- What happens if a user duplicates a drawing marker in the markdown (two `![[draw:diagram-1]]` lines)? The system renders the drawing at the first occurrence only; subsequent duplicates are ignored or shown as a warning.
- What happens if the user removes the marker from the markdown but does not delete the drawing? The drawing file persists on disk but is not rendered in reader mode. The drawing list in edit mode still shows it, allowing the user to re-insert the marker or delete the drawing.
- What happens with very many drawings (e.g., 20+) on a single note? The system should render them progressively and not block page load.
- What happens to the existing single-drawing notes after this feature is deployed? They continue to work via lazy migration — legacy files are detected and served at read time. The file is renamed and a `note_drawings` metadata row is created only when the user first edits the drawing or adds a second drawing to the note.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST allow users to create multiple drawings per note, each identified by a stable auto-generated ID and a user-provided display name.
- **FR-002**: System MUST present the tool choice (Excalidraw or tldraw) independently for each new drawing — different drawings on the same note can use different tools.
- **FR-003**: System MUST require a user-provided display name for each drawing at creation time. The system generates a stable ID used in file names and syntax markers. Display names should be unique within a note (enforced as a validation warning, not a hard block).
- **FR-004**: System MUST store each drawing as an independent file with editable source data, preserving the ability to edit each drawing separately.
- **FR-005**: System MUST render each drawing at the position of its syntax marker within the markdown content in reader mode, with its name as a heading.
- **FR-013**: System MUST support a drawing syntax marker using the stable ID (e.g., `![[draw:abc123]]`) in the markdown that controls where a drawing renders. When a new drawing is created, the marker is appended at the end of the document by default. The display name is resolved from metadata at render time.
- **FR-014**: System MUST display a placeholder in reader mode when a drawing marker references a drawing that does not exist or has been deleted.
- **FR-016**: System MUST insert the drawing syntax marker at the current cursor position in the editor when "Add drawing" is clicked. If no cursor position is active, the marker is appended at the end of the document.
- **FR-017**: System MUST allow the user to open a drawing's canvas for editing by clicking on its marker line in the editor.
- **FR-006**: System MUST allow individual drawings to be edited, renamed (display name only — no file or marker changes), and deleted without affecting other drawings on the same note.
- **FR-007**: System MUST track version history for each drawing file independently, supporting history, diff, and revert operations.
- **FR-008**: System MUST render historical versions of each drawing using the correct tool type (Excalidraw or tldraw).
- **FR-009**: System MUST cascade-delete all drawing files when the parent note is deleted.
- **FR-010**: System MUST display all drawings in shared notes (respecting read/edit permissions) and public notes (read-only).
- **FR-011**: System MUST maintain backward compatibility with existing single-drawing notes using lazy migration — legacy files are detected and served at read time without upfront changes. Conversion to the new naming scheme and metadata row creation occurs only on first edit or when a second drawing is added.
- **FR-012**: System MUST enforce a maximum drawing display name length (reasonable limit, e.g., 100 characters) and reject empty names.
- **FR-015**: System MUST support rebuilding the `note_drawings` table from drawing files on disk (via `--rebuild-db`), inferring tool type from file extension and defaulting display name to the drawing ID when metadata is unavailable.

### Key Entities

- **Drawing**: A diagram attached to a note. Each drawing has a stable auto-generated ID, a user-provided display name, a tool type (Excalidraw or tldraw), and structured source data. Metadata is stored in the `note_drawings` SQLite table; content is stored as a JSON file on disk. Multiple drawings can belong to one note.
- **Drawing ID**: A stable, auto-generated identifier (short slug or random token) assigned at creation time. Used in file names and syntax markers. Never changes, even if the display name is updated.
- **Drawing Display Name**: A user-provided label for a drawing. Mutable. Shown as a heading in reader mode and in the drawing list in edit mode. Stored in SQLite.
- **Note**: Existing entity, now with a one-to-many relationship to drawings (previously one-to-one).
- **Drawing Marker**: A syntax token embedded in the note's markdown using the drawing ID (e.g., `![[draw:abc123]]`) that determines where a drawing renders in reader mode. Each drawing has exactly one marker. Markers can be repositioned by the user in the editor.
- **note_drawings table**: Stores drawing metadata — drawing_id (PK), note_id (FK), display_name, tool_type, created_at, updated_at. Source of truth for metadata; rebuildable from drawing files on disk (tool type inferred from file extension, display name defaults to drawing ID if metadata is missing).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can create up to 10 drawings per note without performance degradation in the editor or reader view.
- **SC-002**: Each individual drawing saves and reloads with all elements intact in under 3 seconds, consistent with existing single-drawing performance.
- **SC-003**: Reader mode renders all drawings (up to 10) on a note within 5 seconds of page load.
- **SC-004**: 100% of existing single-drawing notes continue to function without any user action after deployment.
- **SC-005**: Version history correctly tracks and displays changes to individual drawings within a multi-drawing note.
- **SC-006**: All drawings on a note are visible when the note is shared or published, with no missing or broken drawings.

## Assumptions

- Drawing files are named using the stable auto-generated ID (e.g., `<slug>--<drawingID>.excalidraw.json`). Drawing metadata (display name, tool type, created_at, updated_at) is stored in the SQLite `note_drawings` table, consistent with how the project uses SQLite for derived/index data.
- A reasonable upper bound on drawings per note is not enforced in the system — the success criteria target 10 drawings, but the system does not hard-limit. Performance may degrade gracefully beyond that.
- Drawing display names are plain text labels (alphanumeric, spaces, hyphens, underscores) — no special characters or Markdown formatting. Drawing IDs are system-generated alphanumeric tokens safe for use in file names and markdown syntax.
- Existing single-drawing notes (files named `<slug>.tldraw.json` or `<slug>.excalidraw.json`) are detected at read time and served transparently. On first edit or when a second drawing is added, the legacy file is renamed to the new `<slug>--<drawingID>.<type>.json` format, a `note_drawings` row is created with display name "Drawing 1", and a syntax marker is inserted into the markdown.
- Drawing render order in reader mode is determined by marker position in the markdown, not creation order. Users control order by moving markers in the editor.
- Each drawing on a note is fully independent — there is no linking, layering, or cross-referencing between drawings on the same note.
