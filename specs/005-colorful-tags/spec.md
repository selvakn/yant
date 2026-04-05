# Feature Specification: Colorful Tags

**Feature Branch**: `005-colorful-tags`  
**Created**: 2026-04-05  
**Status**: Draft  
**Input**: User description: "Make tags colorful with auto-assigned backgrounds from a curated palette, user-changeable"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See colorful tags automatically (Priority: P1)

When a user creates or views notes with tags, each tag displays with a distinct background color drawn from a curated palette. The color assignment is automatic and consistent—the same tag always gets the same color across all views.

**Why this priority**: Core visual improvement; no user action required to benefit.

**Independent Test**: Create a note with `#work` and `#ideas`; both tags appear with colored backgrounds; refresh and the colors remain the same.

**Acceptance Scenarios**:

1. **Given** a note with tags, **When** the user views the note list, editor, or reader, **Then** each tag displays with a background color from the palette.
2. **Given** the same tag appears on multiple notes, **When** the user views those notes, **Then** the tag has the same color everywhere.
3. **Given** a new tag is created, **When** no color has been assigned yet, **Then** the system auto-assigns a color from the palette.

---

### User Story 2 - Change a tag's color (Priority: P2)

The user can change a tag's assigned color by selecting from the available palette. The change applies everywhere that tag appears.

**Why this priority**: Personalization after auto-assignment; secondary to having colors at all.

**Independent Test**: Click on a tag's color indicator, select a different color from the palette, confirm the tag updates everywhere.

**Acceptance Scenarios**:

1. **Given** a tag with an assigned color, **When** the user initiates a color change, **Then** a picker shows the available palette colors.
2. **Given** the user selects a new color, **When** they confirm, **Then** the tag's color updates in all views (list, editor, reader, sidebar).

---

### Edge Cases

- Two tags may end up with the same auto-assigned color if the palette is smaller than the tag count; this is acceptable.
- Invalid or missing color data falls back to the first palette color.
- Tag colors persist across sessions; if storage fails, fall back to auto-assignment.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Every tag MUST display with a background color from the defined 10-color palette.
- **FR-002**: The system MUST auto-assign a color when a tag is first created, using a deterministic method so the same tag name yields the same default color.
- **FR-003**: Users MUST be able to change a tag's color by choosing from the palette.
- **FR-004**: Color assignments MUST persist across page reloads and sessions.
- **FR-005**: Color changes MUST be reflected everywhere the tag appears without requiring a page refresh (or with minimal delay after save).
- **FR-006**: Text on colored backgrounds MUST remain readable (appropriate contrast).

### Key Entities

- **Tag**: Existing entity; gains a `color` attribute (hex code from palette).
- **Color Palette**: Fixed set of 10 colors: Ink Black (#001219), Dark Teal (#005f73), Dark Cyan (#0a9396), Pearl Aqua (#94d2bd), Vanilla Custard (#e9d8a6), Golden Orange (#ee9b00), Burnt Caramel (#ca6702), Rusty Spice (#bb3e03), Oxidized Iron (#ae2012), Brown Red (#9b2226).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of tags display with a background color on page load.
- **SC-002**: Users can change a tag's color in under 3 clicks/taps.
- **SC-003**: Color changes persist and appear correctly after page reload (0 regressions in 10 test scenarios).
- **SC-004**: Text remains readable on all 10 palette colors (contrast ratio ≥ 4.5:1 per WCAG AA).

## Assumptions

- The 10-color palette is fixed for this feature; adding/removing palette colors is out of scope.
- Color is stored per tag name (not per tag instance on a note).
- Auto-assignment uses a hash of the tag name to pick a palette index, ensuring consistency without storage until user overrides.
- Only authenticated users can change tag colors (same access as editing notes).
