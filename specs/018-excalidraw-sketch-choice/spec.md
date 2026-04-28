# Feature Specification: Excalidraw Sketch Choice

**Feature Branch**: `018-excalidraw-sketch-choice`  
**Created**: 2026-04-28  
**Status**: Draft  
**Input**: User description: "add a feature to use excalidraw instead of tldraw for the notes sketch. In the note, when a sketch is created, user should be able to choose one of these two. Keep all the other features (like readonly in the view mode, version control, save, etc) same as what we have how. If needed go through the existing specs, plans and tasks to identify them."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Choose a Drawing Tool When Creating a New Sketch (Priority: P1)

When a user opens a note that does not yet have a drawing and clicks "Add drawing," they are presented with a choice between two sketch tools: Excalidraw and tldraw. The user selects one and the corresponding drawing canvas opens immediately, ready for use. The chosen tool is remembered for that note's drawing going forward.

**Why this priority**: The core value of this feature is giving users the ability to pick their preferred drawing tool. Without this choice flow, the feature has no differentiation from the existing single-tool behavior.

**Independent Test**: Open a note with no drawing, click "Add drawing," verify both tool options are presented, select one, and confirm the correct canvas loads.

**Acceptance Scenarios**:

1. **Given** a note with no existing drawing, **When** the user clicks "Add drawing," **Then** a prompt or selection UI appears offering two options: Excalidraw and tldraw.
2. **Given** the selection UI is displayed, **When** the user selects Excalidraw, **Then** an Excalidraw canvas opens in the drawing area.
3. **Given** the selection UI is displayed, **When** the user selects tldraw, **Then** a tldraw canvas opens in the drawing area (same as current behavior).
4. **Given** the user has selected a tool and begins drawing, **When** they save and return later, **Then** the drawing reopens with the same tool that was originally chosen — no re-selection is needed.

---

### User Story 2 - Create and Save an Excalidraw Drawing (Priority: P2)

After choosing Excalidraw as the drawing tool, the user can create diagrams using Excalidraw's full drawing capabilities — shapes, freehand, text, arrows, and connectors. The drawing content is saved as editable source data (not a flattened image), preserving the ability to resume editing later.

**Why this priority**: Once the user can choose Excalidraw, they need to actually use it effectively. This story ensures Excalidraw has feature parity with tldraw for the core create-and-save workflow.

**Independent Test**: Select Excalidraw for a note, draw shapes and text, save, reload the page, and confirm all drawing elements are intact and editable.

**Acceptance Scenarios**:

1. **Given** the user has opened an Excalidraw canvas on a note, **When** they draw shapes, add text, and create connectors, **Then** all elements are rendered on the canvas as expected.
2. **Given** the user has drawn content on the Excalidraw canvas, **When** they save (explicitly or via auto-save), **Then** the drawing data is persisted as editable source data associated with the note.
3. **Given** a note with a saved Excalidraw drawing, **When** the user returns to the note and opens the drawing, **Then** the Excalidraw canvas loads with all previously saved elements intact and fully editable.

---

### User Story 3 - View Excalidraw Drawing in Read-Only Mode (Priority: P3)

When a user views a note that has an Excalidraw drawing in reader mode, the drawing is displayed as a static, non-editable rendering — consistent with how tldraw drawings are currently displayed in reader mode. The user can see the full drawing without accidentally modifying it.

**Why this priority**: Reader mode is essential for the note-viewing experience. Drawings must be visible without requiring the user to enter edit mode, and accidental edits must be prevented.

**Independent Test**: Create a note with an Excalidraw drawing, navigate to the note's reader view, and verify the drawing is displayed read-only without edit controls.

**Acceptance Scenarios**:

1. **Given** a note with an Excalidraw drawing, **When** the user views the note in reader mode, **Then** the drawing is rendered visually inline without any editing controls.
2. **Given** the drawing is displayed in reader mode, **When** the user interacts with the drawing area, **Then** no modifications can be made to the drawing content.
3. **Given** the reader mode shows the drawing, **When** the user switches to edit mode, **Then** the full Excalidraw editor opens with the drawing content editable.

---

### User Story 4 - Version Control for Excalidraw Drawings (Priority: P4)

When a note with an Excalidraw drawing is saved and the drawing content has changed, the version control system tracks the change — identical to how tldraw drawing changes are currently tracked (see spec 014-note-version-control). History entries, version viewing, diff rendering (side-by-side read-only canvases), and revert all work for Excalidraw drawings the same way they work for tldraw drawings.

**Why this priority**: Version control is an existing capability that must extend to the new drawing tool. Without it, Excalidraw drawings would be a regression compared to tldraw.

**Independent Test**: Create an Excalidraw drawing, edit it multiple times, view the version history, and confirm each change appears as a separate version. View a diff and confirm side-by-side rendering works.

**Acceptance Scenarios**:

1. **Given** a note with an Excalidraw drawing, **When** the user edits the drawing and saves, **Then** a new version is recorded in the note's version history.
2. **Given** the version history shows multiple drawing changes, **When** the user views the diff between two versions, **Then** the system displays side-by-side read-only renderings of the Excalidraw drawing at each version.
3. **Given** the user is viewing a historical version of a note with an Excalidraw drawing, **When** they view that version, **Then** the Excalidraw drawing is rendered in read-only mode showing its state at that point in time.
4. **Given** the user wants to revert, **When** they revert to a previous version that had an Excalidraw drawing, **Then** the drawing is restored to its state at that version.

---

### User Story 5 - Backward Compatibility with Existing tldraw Drawings (Priority: P5)

All notes that already have tldraw drawings continue to work exactly as before — viewing, editing, saving, version history, and diff rendering remain unchanged. No migration or manual action is required from the user.

**Why this priority**: Existing drawings must not break. Backward compatibility is non-negotiable but does not require new development effort beyond ensuring the refactored system correctly detects and uses tldraw for existing drawings.

**Independent Test**: Open an existing note with a tldraw drawing, verify it renders in reader and editor mode, edit and save, confirm version history still works.

**Acceptance Scenarios**:

1. **Given** a note with an existing tldraw drawing (created before this feature), **When** the user opens the note, **Then** the drawing loads and displays using tldraw exactly as before.
2. **Given** an existing tldraw drawing, **When** the user edits and saves it, **Then** the drawing continues to be saved in tldraw format and opens with tldraw on subsequent visits.
3. **Given** an existing tldraw drawing, **When** the user views its version history and diffs, **Then** all history entries are intact and diff rendering works as before.

---

### Edge Cases

- What happens when a user deletes a drawing and then adds a new one? They should be presented with the tool choice again, regardless of which tool was previously used.
- What happens to the drawing when the note is deleted? The companion drawing file (regardless of tool type) is deleted along with the note, consistent with existing cascade-delete behavior.
- What happens if the Excalidraw bundle fails to load? The system should display an error message in the drawing area and not corrupt existing drawing data.
- What happens when a user views a historical version created with tldraw, but the current drawing uses Excalidraw (or vice versa)? The version viewer renders using the tool that matches the drawing data at that version.
- What happens when a note is archived and restored? The drawing and its tool association are preserved across archive/restore cycles.
- What happens if a user has a very large Excalidraw drawing? The system handles it gracefully within the same performance bounds as tldraw drawings.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST present the user with a choice between Excalidraw and tldraw when creating a new drawing on a note that has no existing drawing.
- **FR-002**: System MUST persist the chosen drawing tool type alongside the drawing data, so the correct editor and viewer are used when the drawing is reopened.
- **FR-003**: System MUST support creating, saving, and editing drawings using Excalidraw with the same workflow as tldraw (canvas open, draw, save, reopen).
- **FR-004**: Excalidraw drawings MUST be stored as editable source data (not flattened images), enabling future editing after save.
- **FR-005**: System MUST display Excalidraw drawings in read-only mode when the note is viewed in reader mode, with no editing controls visible.
- **FR-006**: System MUST display tldraw drawings in read-only mode when the note is viewed in reader mode, preserving existing behavior.
- **FR-007**: System MUST track version history for Excalidraw drawings identically to how tldraw drawings are tracked — new versions are created on content change, diffs show side-by-side read-only renderings, and revert restores the drawing state.
- **FR-008**: System MUST render historical versions of drawings using the correct tool (Excalidraw or tldraw) that matches the drawing data at that version.
- **FR-009**: System MUST support deleting an Excalidraw drawing from a note without deleting the note itself, consistent with existing tldraw drawing deletion.
- **FR-010**: System MUST cascade-delete the drawing file (regardless of tool type) when the parent note is deleted.
- **FR-011**: System MUST continue to support all existing tldraw drawings without requiring any migration or manual action.
- **FR-012**: System MUST re-present the tool choice when a user creates a new drawing after having deleted the previous one, regardless of which tool was previously used.
- **FR-013**: System MUST preserve drawing data and tool association across note archive and restore cycles.

### Key Entities

- **Drawing**: Editable diagram data associated with a note. Extended with a tool type indicator (Excalidraw or tldraw). Stored as structured source data. One drawing per note maximum.
- **Drawing Tool Type**: An attribute of a drawing indicating which editor/viewer to use — Excalidraw or tldraw. Determined at creation time and persisted with the drawing.
- **Note**: Existing entity with an optional association to a Drawing (unchanged relationship, but the Drawing now carries tool type information).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can select a drawing tool and begin drawing within 3 seconds of clicking "Add drawing."
- **SC-002**: Excalidraw drawings save and reload with all elements intact, with the same round-trip performance as tldraw drawings (under 3 seconds).
- **SC-003**: Read-only rendering of Excalidraw drawings appears within 1 second of page load for drawings with up to 100 elements.
- **SC-004**: 100% of existing tldraw drawings continue to function without any user action after the feature is deployed.
- **SC-005**: Version history, diff viewing, and revert work for Excalidraw drawings with the same responsiveness as tldraw drawings (history loads within 2 seconds, diffs render within 3 seconds).
- **SC-006**: Users can delete an Excalidraw drawing and create a new one (with either tool) without encountering errors or stale state.

## Assumptions

- Excalidraw provides a persistence model similar to tldraw — structured JSON data that can be saved and restored to reproduce the drawing state. This is a well-documented capability of the Excalidraw library.
- The Excalidraw bundle will be built using the same island architecture as tldraw — a separate bundle loaded on demand, not a full SPA conversion.
- One drawing per note remains sufficient. This feature does not change the one-to-one relationship between notes and drawings.
- The tool choice is permanent for a given drawing. Users cannot switch a drawing from Excalidraw to tldraw (or vice versa) without deleting and recreating it.
- Real-time collaboration on drawings remains out of scope, consistent with existing constraints.
- The existing drawing API endpoints will be extended to accommodate tool type metadata rather than being duplicated for each tool.
- Both drawing tools are available to all authenticated users — there is no per-user tool preference or admin-level tool restriction.
- The Excalidraw library is MIT-licensed and compatible with the project's licensing requirements.
