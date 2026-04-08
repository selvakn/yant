# Feature Specification: Mermaid Diagram Support

**Feature Branch**: `012-mermaid-diagrams`  
**Created**: 2026-04-08  
**Status**: Draft  
**Input**: User description: "add support for adding mermaid-js/mermaid diagrams in the notes, inline in the markdown. Plan and Task all the activities and go ahead and implement all of them."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Render Mermaid Diagrams in Notes (Priority: P1)

A user writes a note in Markdown and includes a mermaid code block (using the standard triple-backtick syntax with `mermaid` as the language identifier). When they view the note in the reader, the mermaid code block is rendered as a visual diagram (flowchart, sequence diagram, etc.) instead of displaying raw text.

**Why this priority**: This is the core feature. Without rendering, there is no value in supporting mermaid syntax.

**Independent Test**: Create a note with a mermaid flowchart code block, save it, and view it in the reader. The diagram should appear as a rendered SVG.

**Acceptance Scenarios**:

1. **Given** a note containing a ` ```mermaid ` code block with a valid flowchart definition, **When** the user views the note in the reader, **Then** the mermaid code is rendered as an interactive SVG diagram.
2. **Given** a note containing multiple mermaid code blocks, **When** the user views the note, **Then** all diagrams are rendered independently.
3. **Given** a note containing both regular Markdown content and mermaid diagrams, **When** the user views the note, **Then** regular Markdown renders as usual and mermaid blocks render as diagrams, in the correct document order.

---

### User Story 2 - Edit Mermaid Source in the Editor (Priority: P2)

A user opens a note for editing that contains mermaid code blocks. The mermaid source code is displayed as plain text in the Markdown editor, allowing the user to modify the diagram definition. The user writes or edits mermaid syntax just like any other code block.

**Why this priority**: Users need to be able to author and modify diagrams. The editor already supports code blocks natively via EasyMDE, so mermaid source is editable by default.

**Independent Test**: Open a note with a mermaid code block in the editor, modify the diagram source, save, and confirm the change is reflected in the reader view.

**Acceptance Scenarios**:

1. **Given** a note with an existing mermaid code block, **When** the user opens it in the editor, **Then** the raw mermaid source is visible and editable as plain text.
2. **Given** a user editing a note, **When** they type a new ` ```mermaid ` code block with valid syntax and save, **Then** the diagram renders correctly in the reader view.

---

### User Story 3 - Graceful Handling of Invalid Mermaid Syntax (Priority: P3)

A user writes a mermaid code block with invalid syntax. When viewing the note, the application handles the error gracefully rather than breaking the page or showing a blank area.

**Why this priority**: Error resilience prevents a poor user experience but is not the core value proposition.

**Independent Test**: Create a note with a malformed mermaid code block, view it, and confirm a clear error indication is shown without affecting the rest of the page.

**Acceptance Scenarios**:

1. **Given** a note with an invalid mermaid code block, **When** the user views it in the reader, **Then** an error message or the raw source is displayed in place of the diagram, and the rest of the page renders normally.
2. **Given** a note with one valid and one invalid mermaid block, **When** the user views it, **Then** the valid diagram renders correctly and the invalid one shows an error indicator.

---

### Edge Cases

- What happens when the mermaid library fails to load (e.g., no internet for CDN, or script error)? The raw code block text should remain visible.
- What happens with very large or complex diagrams? They should render without freezing the page, with reasonable size limits handled by the browser viewport.
- What happens with mermaid code blocks in archived notes? They should render the same as in active notes.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST render mermaid code blocks (` ```mermaid `) as SVG diagrams when displaying notes in the reader view.
- **FR-002**: System MUST support all standard mermaid diagram types (flowcharts, sequence diagrams, class diagrams, state diagrams, ER diagrams, Gantt charts, pie charts, etc.).
- **FR-003**: System MUST render mermaid diagrams client-side in the browser, without server-side processing.
- **FR-004**: System MUST display the raw mermaid source code as editable text in the Markdown editor.
- **FR-005**: System MUST gracefully handle invalid mermaid syntax by showing an error indicator without breaking the rest of the page.
- **FR-006**: System MUST support multiple mermaid code blocks within a single note.
- **FR-007**: System MUST render mermaid diagrams in both active and archived note views.
- **FR-008**: Mermaid diagrams MUST render correctly alongside other Markdown content (headings, lists, images, tldraw drawings, backlinks).
- **FR-009**: System MUST load the mermaid library only when a note contains mermaid code blocks, to avoid unnecessary overhead.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Mermaid diagrams render within 2 seconds of the note page loading for diagrams with up to 50 nodes.
- **SC-002**: All standard mermaid diagram types (flowchart, sequence, class, state, ER, Gantt, pie) render correctly.
- **SC-003**: Notes with invalid mermaid syntax display without page errors or blank sections.
- **SC-004**: No increase in page load time for notes that do not contain mermaid code blocks.

## Assumptions

- The mermaid library will be vendored or loaded from a CDN; no build step changes needed for mermaid itself.
- The existing Markdown rendering pipeline (goldmark) can be extended or post-processed to identify mermaid code blocks.
- Users are familiar with mermaid syntax from GitHub, GitLab, or other Markdown tools.
- No server-side rendering or export-to-image is needed for the initial implementation.
- The editor does not need a live preview of mermaid diagrams; raw source editing is sufficient.
