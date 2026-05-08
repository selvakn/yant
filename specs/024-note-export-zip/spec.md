# Feature Specification: Note Export as ZIP

**Feature Branch**: `024-note-export-zip`  
**Created**: 2026-05-08  
**Status**: Draft  
**Input**: User description: "feature to export a note. should bundle the markdown, the sketches as svg as well as the sketches as sources, bundle it as zip, name it as the name of the note, and download."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Export a Note as ZIP (Priority: P1)

A user viewing a note wants to export all its content — the note text and any embedded sketches — as a single downloadable ZIP file. The ZIP is named after the note and contains the markdown source, each sketch exported as SVG, and each sketch's raw source file.

**Why this priority**: Core deliverable of the feature. Everything else is secondary to the user being able to trigger and receive the export.

**Independent Test**: Can be fully tested by opening any note that contains at least one sketch, clicking Export, and verifying the downloaded ZIP contains the markdown file, one or more `.svg` files, and the corresponding sketch source files, all correctly named.

**Acceptance Scenarios**:

1. **Given** a note with one or more sketches, **When** the user clicks "Export Note", **Then** a ZIP file downloads named after the note (e.g. `my-meeting-notes.zip`)
2. **Given** the downloaded ZIP, **When** opened, **Then** it contains: `note.md` (the full markdown), one `.svg` per sketch (vector render), and one source file per sketch (raw sketch data)
3. **Given** a note with no sketches, **When** the user exports, **Then** the ZIP downloads and contains only `note.md`
4. **Given** a note with multiple sketches, **When** the ZIP is inspected, **Then** each sketch appears as a separate file pair (SVG + source), disambiguated by index or name

---

### User Story 2 - Export from Note Detail View (Priority: P2)

The export action is accessible directly from the note viewing/editing page without requiring extra navigation.

**Why this priority**: Discoverability and convenience — the user should not need to hunt for the export option.

**Independent Test**: Can be tested by verifying an "Export" button or menu item appears on the note detail page and triggers the download.

**Acceptance Scenarios**:

1. **Given** a user is viewing a note, **When** they look at the note page, **Then** an "Export" action is visible in the note's action area
2. **Given** the user clicks Export, **When** the download begins, **Then** no page navigation occurs; the user remains on the note

---

### Edge Cases

- Note title contains characters invalid in file names (e.g. `/`, `\`, `:`, `?`) — these must be sanitised in the ZIP filename
- A sketch source file is missing or corrupted — the export still succeeds, omitting only the affected sketch
- Note has no title — fallback filename used (e.g. `untitled-note.zip`)
- Note has many sketches — export completes without browser timeout; a loading indicator is shown if the operation takes more than 1 second

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST provide an "Export" action on the note detail page
- **FR-002**: Triggering export MUST produce a ZIP file that downloads in the browser
- **FR-003**: The ZIP filename MUST be derived from the note title, with file-system-unsafe characters replaced
- **FR-004**: The ZIP MUST contain the note's markdown content as `note.md`
- **FR-005**: For each sketch embedded in the note, the ZIP MUST include an SVG export of that sketch
- **FR-006**: For each sketch embedded in the note, the ZIP MUST include the sketch's raw source file (the format used by the sketch tool, to allow re-editing)
- **FR-007**: Sketch files inside the ZIP MUST be clearly named to identify which sketch they belong to (e.g. by index or embedded name)
- **FR-008**: If a note has no sketches, the ZIP MUST still be produced containing only `note.md`
- **FR-009**: The user MUST remain on the note page after the download is triggered

### Key Entities

- **Note**: The document being exported — has a title, markdown body, and zero or more embedded sketches
- **Sketch**: A drawing attached to a note — has a vector SVG representation and a raw source (the editable format)
- **Export Package**: The ZIP file — named after the note, containing `note.md`, SVG files, and source files for all sketches

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can trigger a note export and receive a valid ZIP download in under 3 seconds for notes with up to 10 sketches
- **SC-002**: 100% of sketches present in the note appear in the ZIP (SVG + source), or a clear omission is communicated when a sketch file is unreadable
- **SC-003**: The ZIP can be opened by standard archive tools on macOS, Windows, and Linux without errors
- **SC-004**: The export action is reachable in at most 2 clicks from the note detail page

## Assumptions

- Sketch source files are already stored on the server alongside the note; the export reads these existing files
- SVG export of a sketch is producible from the stored source file without the sketch editor being open
- The export is available to any user who can view the note (no additional permission required)
- ZIP creation happens server-side; the browser only handles the file download
- Sketch files are discoverable by the backend from information already present in the note's markdown
