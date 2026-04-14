# Feature Specification: Inline Markdown Todos with Cross-Note Pending View

**Feature Branch**: `013-markdown-inline-todos`  
**Created**: 2026-04-13  
**Status**: Draft  
**Input**: User description: "feature to tag individual line in a markdown as todo, with target date to complete and status (pending or complete, marked with a checkbox), and add a view to list all the pending todos across notes, along with their tags, as a list and that list should be navigatable to the notes on clicking. It should be easy to mark a todo as complete. while the content should be stored as plain markdowns, the view should be sophisticated to make it easy to add, mark as complete, navigate, etc"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Add a Todo to a Note (Priority: P1)

A user is writing a note and wants to capture an actionable task inline. They type a checkbox line using standard markdown task list syntax with an optional due date. For example:

```
- [ ] Review the quarterly report @due(2026-04-20)
```

The editor treats this as a normal markdown line (no special UI required to create it). When the note is saved and viewed in reader mode, the line renders as a checkbox with the due date displayed visually.

**Why this priority**: Without the ability to write and parse todos from markdown, no other feature (aggregation, completion) can function. This is the foundational capability.

**Independent Test**: Can be fully tested by writing a note with `- [ ]` and `- [x]` lines, saving, and confirming they render as interactive checkboxes with due date badges in reader view.

**Acceptance Scenarios**:

1. **Given** a user is editing a note, **When** they type `- [ ] Task text @due(2026-04-20)` and save, **Then** the reader view renders the line as an unchecked checkbox with "Task text" and a visible due date of "Apr 20, 2026".
2. **Given** a note contains `- [x] Done task @due(2026-04-10)`, **When** the user views the note, **Then** the line renders as a checked (struck-through) checkbox with the due date.
3. **Given** a user writes `- [ ] Task without a date`, **When** the note is viewed, **Then** the line renders as an unchecked checkbox with no due date shown.
4. **Given** a note contains a `- [ ]` todo line, **When** the user clicks the rendered checkbox in reader view, **Then** the markdown source is updated to `- [x]` and the note is saved automatically.

---

### User Story 2 - View All Pending Todos Across Notes (Priority: P2)

A user wants to see everything they still need to do, across all their notes, in one place. They navigate to a dedicated "Todos" view that lists all pending (unchecked) todo items aggregated from every note.

Each todo item in the list shows: the task text, the due date (if any), the tags from the parent note, and the note title — clickable to navigate to the source note.

**Why this priority**: The cross-note aggregation view is the primary value proposition — turning scattered inline tasks into a unified, actionable list.

**Independent Test**: Can be tested by creating multiple notes with `- [ ]` lines, navigating to the todos view, and verifying all pending items appear with correct metadata and navigation links.

**Acceptance Scenarios**:

1. **Given** three notes each contain pending todo items, **When** the user navigates to the todos view, **Then** all pending items from all three notes are listed.
2. **Given** the todos view is displayed, **When** the user looks at a todo item, **Then** they see the task text, due date (if present), note title, and tags from the parent note.
3. **Given** the todos view is displayed, **When** the user clicks on a todo's note title, **Then** they are navigated to the reader view of that note.
4. **Given** the todos view is displayed, **Then** items are sorted by due date ascending (earliest due first), with undated items listed after dated items.
5. **Given** the todos view contains items, **When** a todo's due date is in the past, **Then** the item is visually highlighted as overdue.
6. **Given** the todos view is displayed, **When** the user clicks a tag, **Then** the list is filtered to show only todos from notes that have that tag.

---

### User Story 3 - Mark a Todo as Complete from the Todos View (Priority: P3)

A user is reviewing their pending todos list and wants to check off a completed task without navigating to the source note. They click the checkbox next to a todo item in the aggregated view, which marks it complete and updates the underlying markdown in the source note.

**Why this priority**: This is the key usability feature that makes the todos view actionable — not just a read-only dashboard, but a place to manage tasks efficiently.

**Independent Test**: Can be tested by marking a todo complete in the aggregated view, then opening the source note to verify the markdown changed from `- [ ]` to `- [x]`.

**Acceptance Scenarios**:

1. **Given** the todos view shows a pending item, **When** the user clicks the checkbox, **Then** the item is marked complete, the checkbox becomes checked, and the item fades out or moves to a completed section.
2. **Given** the user marks a todo complete in the todos view, **When** they open the source note, **Then** the corresponding markdown line reads `- [x]` instead of `- [ ]`.
3. **Given** the user marks a todo complete in the todos view, **When** the update fails (e.g., network error), **Then** the checkbox reverts to unchecked and an error message is shown.

---

### User Story 4 - Navigate to Todos View (Priority: P4)

A user wants quick access to their todos. The todos view is accessible from the main navigation (top nav or sidebar) and via a keyboard shortcut.

**Why this priority**: Discoverability — users need a clear path to the todos feature.

**Independent Test**: Can be tested by verifying the navigation link and keyboard shortcut both lead to the todos view.

**Acceptance Scenarios**:

1. **Given** the user is on any page, **When** they look at the navigation, **Then** there is a visible link to the todos view.
2. **Given** the user is on any page, **When** they open the sidebar, **Then** there is a "Todos" link with a count of pending items.

---

### Edge Cases

- What happens when a note with pending todos is archived? The todos from archived notes should not appear in the active todos view.
- What happens when a note with pending todos is deleted? The corresponding todos disappear from the todos view.
- What happens when two `- [ ]` lines in the same note have identical text? Each is treated as a distinct todo, identified by its line position in the markdown.
- What happens when a due date is malformed (e.g., `@due(not-a-date)`)? The `@due(...)` text is shown as-is without date formatting; the todo still appears but with no parsed date for sorting.
- What happens when a todo line contains other markdown formatting (bold, links, inline code)? The formatting is preserved in both the reader view and the todos view.
- What happens when a user edits a note and changes a `- [x]` back to `- [ ]`? The todo re-appears in the pending todos view after saving.
- What happens when a todo is toggled from the todos view or reader while the note is open in the editor? Last write wins — no conflict detection or merging. This is acceptable for a single-user personal app.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST parse `- [ ]` and `- [x]` markdown checkbox lines as todo items.
- **FR-002**: System MUST recognize the `@due(YYYY-MM-DD)` annotation on todo lines as a target completion date.
- **FR-003**: System MUST render todo items as interactive checkboxes in the note reader view.
- **FR-004**: Clicking a checkbox in reader view MUST toggle the todo status (`- [ ]` ↔ `- [x]`) in the markdown source and save the note.
- **FR-005**: System MUST provide a dedicated todos view that aggregates all pending (`- [ ]`) todo items across all of a user's non-archived notes.
- **FR-006**: Each todo item in the aggregated view MUST display: task text, due date (if any), parent note title (as a clickable link), and the parent note's tags.
- **FR-007**: The todos view MUST sort items by due date ascending, with overdue items first, then upcoming, then undated items last.
- **FR-008**: Users MUST be able to mark a todo as complete directly from the aggregated todos view by clicking a checkbox, which updates the markdown source in the corresponding note.
- **FR-009**: The todos view MUST be accessible via a navigation link (in the sidebar) showing the count of pending items.
- **FR-010**: Todo items from archived notes MUST NOT appear in the todos view.
- **FR-014**: The todos view MUST support filtering by tag — clicking a tag filters the list to todos from notes with that tag, consistent with notes list behavior.
- **FR-011**: All todo data MUST be stored as plain markdown in the note files — no separate todo storage.
- **FR-012**: Overdue todos (past due date) MUST be visually distinguished in the todos view.
- **FR-013**: System MUST support todo lines without a due date — these function as undated tasks.

### Key Entities

- **Todo Item**: A parsed representation of a `- [ ]` or `- [x]` line from a note. Key attributes: task text, status (pending/complete), due date (optional), source note (slug, title), line position within the note, and tags (inherited from the parent note).
- **Todos View**: An aggregated, sorted, interactive list of all pending todo items across the user's non-archived notes.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can add a todo to a note in under 5 seconds by typing a `- [ ]` line — no special UI or mode required.
- **SC-002**: Users can mark a todo as complete with a single click, from either the note reader or the aggregated todos view.
- **SC-003**: The todos view loads and displays all pending items across all notes within 2 seconds.
- **SC-004**: When a todo is marked complete in the todos view, the underlying note's markdown is updated within 1 second.
- **SC-005**: 100% of pending todos from non-archived notes appear in the aggregated view — no items are lost or duplicated.
- **SC-006**: Users can navigate from any todo in the aggregated view to its source note in a single click.

## Clarifications

### Session 2026-04-13

- Q: Should the todos view support filtering? → A: Filter by tag only, consistent with how the notes list works.
- Q: What happens when a todo is toggled while the note is open in the editor? → A: Last write wins — no conflict detection. Acceptable for a single-user personal app.

## Assumptions

- Users are familiar with markdown checkbox syntax (`- [ ]` / `- [x]`) or will learn it quickly from examples.
- The `@due(YYYY-MM-DD)` syntax is chosen as the date annotation format because it is human-readable, unambiguous, and does not conflict with other markdown constructs. The system does not support other date formats (e.g., "next Friday").
- The todos view only shows the current user's own todos — there is no multi-user/shared todo functionality.
- Completed (`- [x]`) items are not shown in the aggregated todos view. Users can see completed items by visiting the source note.
- Todo parsing happens at read time (when a note is viewed or the todos view is loaded) based on the markdown content — the markdown files remain the single source of truth.
- The feature re-uses the existing authentication, session management, and note storage infrastructure.
- A keyboard shortcut for the todos view will follow the existing shortcut conventions in the app.
