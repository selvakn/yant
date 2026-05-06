# Feature Specification: Inline Drawing Previews in Edit Mode

**Feature Branch**: `022-inline-drawing-previews`
**Created**: 2026-05-06
**Status**: Draft
**Input**: User description: "in edit mode as well, i want to see the drawings. today its editable, but one at a time. instead i want to see the svg (readonly) by default, if possible, in line, if not at the end (like current), and once clicked, it can go into edit mode."

## Clarifications

### Session 2026-05-06

- Q: How does the user exit edit mode for a drawing back to its read-only preview? → A: Explicit Close/Done button on the canvas; clicking another preview also closes the current one (per existing FR-007).
- Q: What happens to today's drawing-list management UI under the new previews-by-default layout? → A: Remove the drawing-list. Each preview owns its own header (display name, tool-type tag, rename/delete actions). The add-drawing form remains as today.
- Q: How should each preview be sized inside the editor? → A: Render at the drawing's natural aspect ratio at a generous default width, with no height cap (drawing-heavy notes may get tall). The same sizing rule MUST also be applied to reader/view-mode rendering so edit and read views look identical.
- Q: How should a preview signal that it's clickable to enter edit mode? → A: Hover-state affordance — highlight/border + cursor-pointer + subtle "Edit" hint appears on hover; resting state is just the SVG + header.
- Q: How should keyboard-only users interact with previews to enter edit mode? → A: Mouse-only for now; keyboard accessibility (focus, Enter/Space activation, screen-reader labelling) is explicitly deferred to a future feature.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See All Drawings as Read-Only Previews by Default in Edit Mode (Priority: P1)

When a user opens a note that contains one or more drawings in edit mode, every drawing on the note is rendered as a read-only visual preview right away — without the user having to click any "Edit" button. The previews show the drawings exactly as they appear in reader mode (the same SVG rendering), so the author can scan the entire note — text and visuals together — without losing context. A user can keep working on the markdown text while glancing at all drawings simultaneously, instead of opening one canvas at a time.

**Why this priority**: This is the core ask. Today, drawings in edit mode are hidden behind per-drawing "Edit" buttons, which means the author can only see one drawing at a time and must toggle through them to remember what each contains. Showing all previews by default removes that friction and aligns edit-mode context with reader mode.

**Independent Test**: Open a note with 3 drawings (mixed Excalidraw and tldraw) in edit mode. Verify all 3 drawings render as visible read-only previews on initial page load with no clicks required, each labeled with its display name and using the correct tool's renderer.

**Acceptance Scenarios**:

1. **Given** a note with multiple drawings in edit mode, **When** the page first loads, **Then** every drawing renders as a read-only preview without requiring any clicks.
2. **Given** drawings of mixed types (Excalidraw and tldraw) on the same note, **When** the user opens edit mode, **Then** each preview is rendered using the corresponding tool's read-only renderer and matches what the user would see in reader mode.
3. **Given** a note with no drawings, **When** the user opens edit mode, **Then** no preview area is shown and the existing "Add drawing" controls remain visible and usable.
4. **Given** a drawing whose source file is missing, **When** the user opens the note in edit mode, **Then** a placeholder is shown in place of that drawing's preview indicating it cannot be displayed, while the other drawings still preview normally.

---

### User Story 2 - Click a Preview to Edit That Drawing (Priority: P1)

When the user wants to actually modify a drawing, they click on its read-only preview. The preview transitions to an interactive editing canvas for that single drawing, while the other drawings on the note remain in their read-only preview state. When the user finishes editing (saves and/or closes the canvas), the drawing returns to its read-only preview state and reflects the latest content.

**Why this priority**: Without this, the previews are decorative — the user must still be able to edit. Pairing the previews with click-to-edit is what makes the new layout a complete replacement for today's "Edit/Close" toggle list.

**Independent Test**: In a note with 3 drawings, click the second preview, edit it, save, and close. Verify the second drawing returns to a read-only preview reflecting the new content while the first and third drawings stayed visible as previews the entire time.

**Acceptance Scenarios**:

1. **Given** a note in edit mode with multiple drawing previews, **When** the user clicks one preview, **Then** that single preview is replaced with the interactive editing canvas for that drawing, and the other drawings remain rendered as read-only previews.
2. **Given** a drawing is being edited, **When** the user clicks the canvas's Close/Done button, **Then** the canvas closes (applying existing save behavior) and that drawing returns to a read-only preview reflecting the latest saved content.
3. **Given** a drawing is being edited, **When** the user clicks another drawing's preview, **Then** the previously open canvas is closed (saving any pending changes per existing save behavior) and the newly clicked drawing's canvas opens — keeping the "one canvas open at a time" rule.
4. **Given** the user is editing a drawing, **When** they navigate away or close the page, **Then** existing save behavior applies unchanged — no regression compared to today.

---

### User Story 3 - Previews Render Inline With Markdown When Possible (Priority: P2)

The previews should appear at the position of each drawing's syntax marker within the markdown editing surface, so that the author sees the same drawing-text layout while editing as a reader would in reader mode. If rendering previews truly inline within the markdown editing surface is not technically feasible, the system falls back to rendering all previews stacked at the end of the editor area (the same location used today), but still as read-only previews shown by default rather than collapsed behind buttons.

**Why this priority**: Inline placement gives the strongest WYSIWYG experience and helps the author understand layout while writing. However, the user explicitly accepted an end-of-editor fallback if inline rendering proves infeasible, so this is high-value but not a blocker for shipping the rest of the feature.

**Independent Test**: Place drawing markers at the top, middle, and end of a note's markdown. Open edit mode. If inline rendering is delivered: verify each preview appears immediately adjacent to its marker line. If end-of-editor fallback is delivered: verify all previews appear in marker order at the end of the editor area, each labeled, with no preview hidden behind a button.

**Acceptance Scenarios**:

1. **Given** drawing markers placed at multiple positions in the markdown, **When** edit mode is rendered using inline previews, **Then** each preview appears aligned with its marker's position in the text and the marker text remains visible/editable as plain text.
2. **Given** the system falls back to end-of-editor rendering, **When** edit mode is rendered, **Then** all previews appear stacked in marker order at the end of the editing surface, each preview labeled with its display name, and all visible by default (no clicks required to reveal).
3. **Given** the user reorders markers within the markdown, **When** the user saves and re-opens edit mode, **Then** the previews appear in the new marker order (inline at the new positions, or stacked in the new order at the end depending on the chosen placement).

---

### Edge Cases

- A note has many drawings (e.g., 10+): all previews render without blocking the editor's responsiveness; loading may be progressive but does not delay text-editing interactivity.
- A drawing has no saved content yet (e.g., the user just inserted a marker but has not opened the canvas): the preview area shows a placeholder labeled with the display name and a clear indication that the drawing is empty; clicking it opens the canvas to start drawing.
- A drawing fails to load (network error, corrupted file): the preview area shows an error placeholder with a retry affordance; the rest of the previews are unaffected.
- The same drawing marker appears more than once in the markdown: the preview renders once for the first occurrence (consistent with today's reader-mode behavior); duplicate markers do not produce duplicate previews.
- The user removes a drawing's marker from the markdown but does not delete the drawing: the drawing has no preview surface in the markdown view but remains accessible via the existing drawing list / management UI for re-insertion or deletion.
- A drawing is currently being edited (canvas open) when an autosave or external action triggers a re-render of previews: the open canvas is preserved and only the other drawings' previews refresh.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST render every drawing on a note as a read-only visual preview in edit mode by default, on initial page load, without requiring any user click to reveal it.
- **FR-002**: Each preview rendered in edit mode MUST be visually consistent with the same drawing's appearance in reader mode (same renderer per tool type, same content, same labelling with the drawing's display name).
- **FR-003**: System MUST render previews for both Excalidraw and tldraw drawings on the same note, each using its respective read-only renderer.
- **FR-004**: System MUST render previews inline within the markdown editing surface — aligned to the position of each drawing's syntax marker — when this is technically feasible with the current editor.
- **FR-005**: When inline rendering is not feasible, System MUST fall back to rendering all previews stacked in marker order at the end of the editor area, with all previews visible by default (not collapsed behind per-drawing buttons).
- **FR-006**: System MUST allow the user to click a preview to switch that drawing into interactive edit mode, replacing only that preview with the editing canvas while leaving other drawings' previews in their read-only state.
- **FR-007**: System MUST keep the "one canvas open at a time" rule — opening a second drawing's canvas closes the previously open canvas (applying existing save/close behavior) and the previously open drawing returns to a read-only preview reflecting saved content.
- **FR-007a**: System MUST provide an explicit Close/Done control on the open editing canvas so the user can exit edit mode without having to click another drawing's preview. Clicking this control applies existing save behavior and returns the drawing to its read-only preview state. This is required for the case where the note has only one drawing (no other preview to click).
- **FR-008**: When a drawing is saved and the canvas is closed (via the explicit Close/Done control or by opening another drawing), System MUST return that drawing to its read-only preview reflecting the latest saved content, without the user needing to refresh the page.
- **FR-009**: System MUST display a clear placeholder in place of a preview when a drawing's source content is missing, empty (newly created with no drawing yet), or fails to load. The placeholder MUST remain clickable to open the canvas (or, for missing/failed drawings, remain visible with an error state) and MUST NOT prevent other previews from rendering.
- **FR-010**: System MUST preserve existing add-drawing, rename, delete, and version-control behavior. The new preview-by-default behavior is additive and MUST NOT regress any existing drawing-management workflow.
- **FR-010a**: System MUST remove the existing per-drawing list-row UI (the `drawing-list` rows that today show name + tool type + Edit button). The previews replace it as the primary surface for viewing and entering edit mode.
- **FR-010b**: Each preview MUST display, alongside the visual, a header showing the drawing's display name and tool-type indicator, plus controls to rename and delete that drawing. These controls MUST trigger the same backend actions as today's drawing-list row controls (no behavioural regression).
- **FR-010c**: The existing add-drawing form (name input + per-tool "+ Excalidraw" / "+ tldraw" buttons) MUST remain available in edit mode in its current form and behaviour.
- **FR-013**: System MUST render each preview at the drawing's natural aspect ratio, scaled to a generous default width within the editor's content column, with no height cap. Drawings retain their natural proportions, even if this makes the editing surface taller on drawing-heavy notes.
- **FR-014**: System MUST apply the same natural-aspect-ratio, no-height-cap sizing rule (FR-013) to drawings rendered in reader/view mode, so the same drawing on the same note appears at the same size in edit mode and reader mode. If today's reader-mode rendering differs from this rule, it MUST be updated for consistency.
- **FR-015**: Each preview's resting state in edit mode MUST show only the SVG and per-preview header (FR-010b) — no always-visible Edit affordance overlaying the drawing. On hover, System MUST surface a clickability affordance: a highlight or border, a pointer cursor, and a subtle "Edit" hint. The whole preview area remains clickable to enter edit mode.
- **FR-011**: System MUST render previews progressively when many drawings are present on a note, so that the markdown text remains immediately interactive even before all previews finish loading.
- **FR-012**: System MUST apply the new preview-by-default behavior in all editing surfaces that today expose the per-drawing "Edit" toggle (the owner's note editor, and any shared-note editor with edit access).

### Key Entities

- **Drawing Preview**: A read-only visual rendering of a drawing inside the editor view. One per drawing per note. Reuses the existing reader-mode renderer for the drawing's tool type. Has three visual states — *preview* (default), *editing* (when its canvas is open), and *placeholder* (when source is missing, empty, or failed to load). Each preview also owns a small header (display name + tool-type tag) and rename/delete controls — replacing the previous drawing-list row as the per-drawing management surface.
- **Drawing**, **Drawing Marker**, **Note**: Existing entities from feature 020 (multi-note drawings). Unchanged in identity or storage; this feature only changes how each drawing is displayed in edit mode.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: When opening a note with up to 10 drawings in edit mode, all drawing previews are visible without any user clicks within 5 seconds of page load (consistent with reader-mode performance for the same note).
- **SC-002**: 100% of drawings on a note in edit mode are visible by default — zero drawings are hidden behind a button or toggle on initial page load.
- **SC-003**: Switching a single drawing from preview to editing canvas (and back to preview after save) takes the same time as today's per-drawing "Edit" / "Close" interaction, with no measurable regression.
- **SC-004**: Markdown text remains responsive (typing latency unchanged) while previews are loading, even on notes with the maximum supported number of drawings.
- **SC-005**: When the user finishes editing a drawing, the returned read-only preview reflects the just-saved content with no manual refresh needed in 100% of cases.
- **SC-006**: Existing drawing add / rename / delete / version-history flows continue to work with zero functional regressions verified against existing acceptance scenarios from feature 020.
- **SC-007**: For any given drawing, the rendered visual size in edit-mode preview and in reader/view mode is identical (same width, same scaling, same proportions) — verifiable by comparing the two views side-by-side.

## Assumptions

- The existing reader-mode rendering pipeline for drawings (used to render Excalidraw/tldraw drawings as visual previews when viewing a note) is the same pipeline reused to render previews in edit mode. No new rendering technology is introduced.
- Whether previews can render truly inline (interleaved with the markdown editing surface at marker positions) versus stacked at the end of the editor is a technical implementation question to be resolved in the plan phase. The user has explicitly accepted "stacked at the end (like today's location)" as an acceptable fallback if inline placement is infeasible with the current markdown editor.
- "One canvas open at a time" remains the editing rule — this feature does not introduce simultaneous editing of multiple drawings; it only removes the requirement to click before *seeing* a drawing.
- Existing save behavior (when, how, and what triggers persistence) is unchanged. Closing a canvas back to preview does not introduce new save semantics beyond what happens today when the canvas closes.
- The target user is the note's editor (owner or a collaborator with edit access); read-only viewers continue to use reader mode and are unaffected.
- The feature applies to notes with any combination of Excalidraw and tldraw drawings; no new tool types are added.
- Performance targets reuse the same upper bound of 10 drawings per note used in feature 020 — beyond that, previews continue to function but performance may degrade gracefully.
- Keyboard accessibility for previews (focus, Enter/Space activation, screen-reader labelling) is explicitly out of scope for this feature and deferred to a future, dedicated accessibility pass. The interaction model in this feature is mouse-driven.
