# Feature Specification: Diagram Drawing in Notes

**Feature Branch**: `004-tldraw-diagrams`  
**Created**: 2026-04-05  
**Status**: Draft  
**Input**: User description: "As part of the notes, need the ability to add a diagram using tldraw. Research about tldraw.dev and add the ability to draw in the notes. Not every note will have a drawing though. The content of the drawing should be stored as source, so that it can be further edited."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Create a drawing for a note (Priority: P1)

While viewing or editing a note, the user can open a drawing canvas and sketch diagrams, flowcharts, or freeform illustrations. The drawing is associated with that note and persists across sessions.

**Why this priority**: Core value—users need to create drawings before they can view or edit them.

**Independent Test**: Open a note, create a new drawing, add shapes and lines, close the canvas, return to the note later—drawing is intact.

**Acceptance Scenarios**:

1. **Given** a note with no drawing, **When** the user initiates "Add drawing", **Then** a blank canvas opens where they can draw.
2. **Given** the user has drawn content on the canvas, **When** they save or close the canvas, **Then** the drawing content is persisted and associated with the note.
3. **Given** a note already has a drawing, **When** the user returns to the note, **Then** they can view or continue editing the same drawing.

---

### User Story 2 - Edit an existing drawing (Priority: P2)

The user can reopen a note's drawing and make changes—move shapes, add new elements, delete parts. The updated drawing replaces the previous version.

**Why this priority**: Without editing, drawings become static images; edit capability preserves the "source" nature.

**Independent Test**: Open a note's existing drawing, modify it (e.g., add a shape), save; reopen and confirm modification persisted.

**Acceptance Scenarios**:

1. **Given** a note has an existing drawing, **When** the user opens the drawing canvas, **Then** all previously saved content appears and is editable.
2. **Given** the user modifies the drawing, **When** they save, **Then** the updated drawing replaces the prior version.

---

### User Story 3 - View drawing in note reader (Priority: P3)

When reading a note that has a drawing, the user sees a rendered preview or thumbnail of the drawing without entering edit mode. They can click to open the full editor if they want to modify it.

**Why this priority**: Reader experience should show the drawing; pure editing mode is limiting.

**Independent Test**: Open a note with a drawing in reader view; drawing preview displays; clicking opens editor.

**Acceptance Scenarios**:

1. **Given** a note with a drawing, **When** the user views the note in reader mode, **Then** a visual representation of the drawing is displayed inline.
2. **Given** the preview is displayed, **When** the user clicks on it, **Then** the drawing opens in the editor for modification.

---

### Edge Cases

- Creating a drawing but closing without saving: prompt user or auto-save draft.
- Very large or complex drawings: graceful handling without freezing the interface.
- Note without a drawing: no empty placeholder clutter; drawing is opt-in via explicit action.
- Deleting a note: associated drawing data is also removed.
- Concurrent edits (same user, multiple tabs): last save wins; no real-time sync requirement for MVP.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a way to attach exactly one drawing to a note (one-to-one relationship for simplicity).
- **FR-002**: The drawing canvas MUST support basic diagramming: shapes (rectangles, circles, arrows), freehand drawing, text labels, and connectors.
- **FR-003**: Drawing content MUST be stored as editable source data (not flattened images) so users can resume editing.
- **FR-004**: System MUST persist drawing content across browser sessions and page reloads.
- **FR-005**: When a note is deleted, its associated drawing data MUST also be deleted.
- **FR-006**: Reader view MUST display the drawing visually (static render or embedded preview) without requiring the user to open the editor.
- **FR-007**: System MUST allow a user to remove/delete a drawing from a note without deleting the note itself.

### Key Entities

- **Drawing**: Editable diagram data associated with a note; stored as structured source (not a raster image); one drawing per note maximum.
- **Note**: Existing entity; gains an optional association to a Drawing.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can create a drawing and have it appear on page reload within 2 seconds of opening the note.
- **SC-002**: Drawing edits (add shape, move element) are saved and visible upon reopening in under 3 seconds total round-trip.
- **SC-003**: Reader preview of a drawing renders within 1 second of page load for drawings with up to 100 elements.
- **SC-004**: At least 80% of test users report the drawing experience as "intuitive" or "easy to use" in informal feedback.

## Assumptions

- One drawing per note is sufficient for MVP; multiple drawings per note is out of scope.
- Real-time collaboration (multiple users editing simultaneously) is out of scope.
- The drawing tool provides standard whiteboard-like primitives; highly specialized diagramming (UML, network diagrams) is not required but shapes should support general-purpose use.
- Drawing source data is stored alongside or linked to the note file in a way that preserves the markdown-first principle—the markdown file itself remains portable, and drawing data is a companion artifact.
- A frontend build system will be introduced to support the interactive canvas; this is an infrastructure change accepted as part of this feature.
- Only authenticated users can create or edit drawings (same access rules as notes).
