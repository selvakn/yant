# Feature Specification: Shared Note Authorship & Indicators

**Feature Branch**: `025-shared-note-authorship`  
**Created**: 2026-05-15  
**Status**: Draft  
**Input**: User description: "version history should show the author who edited. this is useful for shared notes. the last updated author also should show in the view page. in the list page as well, show an indicator to express if its a shared note (incoming as well as outgoing)."

## Clarifications

### Session 2026-05-15

- Q: What should the share indicators in the notes list show — icon only, or also contextual label/tooltip? → A: Icon plus contextual label or tooltip: the outgoing indicator shows the number of active collaborators (e.g., "Shared with 2"), and the incoming indicator shows the name of the note owner who shared it (e.g., "Shared by Alice").
- Q: Should users be able to filter the notes list to show only incoming or outgoing shared notes? → A: No filter in v1. Indicators are shown inline per note within the standard list view; filtering by share status is out of scope for this release.
- Q: When a collaborator views a shared note's version history, do they see the full history or only entries since they were granted access? → A: Full history — collaborators see all version entries from note creation, including edits that predate their access grant.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See Who Made Each Edit in Version History (Priority: P1)

Alice owns a note shared with Bob. Bob edits it, then Alice edits it. When either user opens the version history, each entry clearly shows the username of the person who made that specific edit alongside the timestamp. This removes ambiguity in collaborative notes — users can tell at a glance who changed what and when.

**Why this priority**: Without authorship in the history log, shared notes become a black box when multiple people edit them. This is the core accountability feature for collaboration and the most directly actionable piece of information for understanding a note's evolution.

**Independent Test**: Share a note with a second user, have that user make an edit, then open the history view and verify the second user's username appears on their edit entry and the owner's username appears on the owner's edits.

**Acceptance Scenarios**:

1. **Given** a note owned by Alice and edited at different times by Alice and Bob, **When** either user opens the version history, **Then** each history entry shows the username of the editor who made that save.
2. **Given** a note that has only ever been edited by its owner, **When** the owner opens the version history, **Then** every entry shows the owner's username.
3. **Given** a version history entry made before this feature was deployed (no author recorded), **When** a user views the history, **Then** the entry shows "Unknown" or "—" in place of a username, and the system does not error.
4. **Given** a revert action is performed, **When** the resulting version entry appears in history, **Then** it shows the username of the user who performed the revert.

---

### User Story 2 - See Last Updated Author on the Note Reader Page (Priority: P2)

Bob opens a note shared with him. Alongside the note's existing "last updated" timestamp, he sees the name of the person who last edited the note. This gives Bob immediate context — without opening the full version history — about who touched the note most recently.

**Why this priority**: The note reader is the most frequently visited surface. A single "last edited by" label answers the most common question about shared note activity ("did my collaborator make changes?") without requiring a full history review.

**Independent Test**: Have a collaborator edit a shared note. Open the note's reader view and confirm the collaborator's username appears alongside the last-modified timestamp.

**Acceptance Scenarios**:

1. **Given** Alice most recently edited a note, **When** Bob (a collaborator) opens the note's reader view, **Then** he sees "Last updated by Alice" (or equivalent wording) along with the timestamp.
2. **Given** a note has only ever been edited by its owner, **When** the owner opens the reader view, **Then** the "last updated by" shows the owner's own username.
3. **Given** a note whose last-editor information is unavailable (pre-feature data), **When** a user views it, **Then** only the timestamp is shown (the "by" part is omitted gracefully).
4. **Given** Bob edits a shared note, **When** Alice (the owner) views the reader page, **Then** the "last updated by" reflects Bob's username, not Alice's.

---

### User Story 3 - Distinguish Incoming and Outgoing Shared Notes in the List (Priority: P2)

Carol opens her notes list. She owns a note she has shared with others (outgoing) and she also has notes that others have shared with her (incoming). Each shared note carries a small, clear indicator so she knows at a glance which notes are collaborative and in which direction — without having to open each note individually.

**Why this priority**: The list view is the primary navigation surface. Without distinction between incoming and outgoing shares, users mentally treat all notes the same, miss collaboration cues, and cannot quickly find "notes shared with me" vs. "notes I've shared". Incoming and outgoing are distinct workflows (e.g., Carol wants to find what her colleagues added vs. what she delegated).

**Independent Test**: Verify the notes list shows: (a) a note Carol shared with someone displays an outgoing indicator; (b) a note someone shared with Carol displays an incoming indicator; (c) Carol's private unshared notes show no sharing indicator.

**Acceptance Scenarios**:

1. **Given** Carol owns a note and has shared it with at least one other user, **When** Carol views her notes list, **Then** that note shows a distinct outgoing-share indicator (e.g., an icon or badge).
2. **Given** another user has shared a note with Carol, **When** Carol views her notes list, **Then** that note shows a distinct incoming-share indicator, different from the outgoing one.
3. **Given** Carol has a note that is both shared outward (she shared it with someone) and is also one someone shared with her (a different note), **When** Carol views the list, **Then** the indicators are shown on the appropriate separate notes, not combined on a single note.
4. **Given** Carol has unshared private notes, **When** she views her notes list, **Then** those notes show no sharing indicator.
5. **Given** the owner revokes Carol's access to a previously incoming-shared note, **When** Carol refreshes her notes list, **Then** the note disappears and does not show any indicator.

---

### Edge Cases

- What if a note was edited before the authorship feature was deployed? History entries without recorded authors show a placeholder (e.g., "—") instead of a username; the rest of the history functions normally.
- What if the user who made an edit has since been deleted? Show their former username as a static string if it was recorded, or "Deleted User" if it was not.
- What if a note is shared with multiple collaborators and all have edited it? Each version history entry shows the specific editor for that entry — no merging or summarizing across editors.
- What if a collaborator is added after many edits already exist? They see the full history including all prior entries, each attributed to the correct editor.
- What if the same user owns notes they've shared AND has notes shared with them? Both types of indicators appear in their list, on the appropriate notes.
- What if a note is shared outward but all collaborators have had their access revoked? The note no longer has active collaborators, but the outgoing indicator should not appear (or should reflect that sharing is inactive). Assume: indicator only appears when at least one active collaborator exists.
- What if the notes list shows hundreds of notes? Indicators must be rendered efficiently without additional per-note API calls — the list view query should include sharing status.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST display the username of the editor on each version history entry, alongside the existing timestamp.
- **FR-002**: For version history entries created before this feature (where author is unknown), the system MUST show a graceful placeholder (e.g., "—") in place of the username, without causing errors.
- **FR-003**: The note reader (view) page MUST display the username of the user who last modified the note, alongside the last-modified timestamp.
- **FR-004**: If the last-editor is unknown (pre-feature data), the reader page MUST show only the timestamp, omitting the "by [username]" part.
- **FR-005**: The notes list MUST display a distinct visual indicator on notes that the viewing user owns and has actively shared with at least one other user (outgoing share). The indicator MUST include the number of active collaborators (e.g., "Shared with 2").
- **FR-006**: The notes list MUST display a distinct visual indicator on notes that have been shared with the viewing user by someone else (incoming share), visually different from the outgoing indicator. The indicator MUST include the owner's username (e.g., "Shared by Alice").
- **FR-007**: Notes with no active share relationships MUST show no sharing indicator in the notes list.
- **FR-008**: The sharing status — including the collaborator count (for outgoing) and owner username (for incoming) — MUST be determined without additional per-note round trips; the list query MUST include this information.
- **FR-009**: When a collaborator's access is revoked, the corresponding incoming-share indicator MUST disappear from their notes list on the next page load.
- **FR-010**: The outgoing-share indicator MUST only appear when at least one active collaborator currently has access; notes with all shares revoked MUST NOT show the indicator.
- **FR-011**: Collaborators with access to a shared note MUST be able to view the note's complete version history from its creation date, including entries predating their access grant.

### Key Entities

- **Version Entry Author**: The user identity associated with a specific version history record. Recorded at the time of the save. Attributes: username (or null/unknown for legacy entries), timestamp, version reference.
- **Note Last Editor**: The user who most recently saved a change to a note. Derived from the most recent version entry's author. Displayed on the note reader page.
- **Share Indicator State**: A derived value on each note in the list view. For outgoing: the count of active collaborators. For incoming: the owner's username. Computed at query time alongside the note list; no extra round trips.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of new version history entries (post-deployment) display the correct editor's username.
- **SC-002**: The note reader page shows the last-editor's username for all notes where the last edit occurred after deployment, with no additional user action required.
- **SC-003**: The notes list page loads with sharing indicators visible, with no extra interaction required from the user.
- **SC-004**: Sharing indicators (incoming and outgoing) are visually distinguishable and carry contextual detail (collaborator count for outgoing; sharer's name for incoming) — users can tell them apart and understand the relationship without opening the note.
- **SC-005**: Notes with no active shares show no indicator — verified by revoking all shares and confirming the indicator disappears on the next list load.
- **SC-006**: Legacy version entries (pre-deployment) display a graceful fallback instead of an error or blank crash in 100% of cases.

## Assumptions

- The version history (feature 014) and note sharing (feature 016) are already implemented and deployed. This feature builds authorship and list indicators on top of them.
- "Author" means the app-internal username (the same identifier used elsewhere in the app), not an email or display name.
- The author is recorded at save time by the server using the authenticated session — there is no user-provided input for the author field, so it cannot be spoofed via the UI.
- Incoming and outgoing share states are mutually exclusive on any single note: a note is either owned by the viewer (and possibly shared outward) or owned by someone else (and shared inward). A note cannot appear as both incoming and outgoing for the same user.
- For the "last updated by" display on the reader page, only the most recent editor is shown. A full list of contributors is viewable via the version history.
- The outgoing-share indicator is only shown when at least one collaborator currently has active access. Notes that were previously shared but have all shares revoked appear as regular unshared notes.
- Notifications of edits (e.g., "Bob edited your note") are out of scope for this feature. The indicators are passive/informational, not push-based.
- Filtering the notes list by share status (incoming/outgoing) is out of scope for v1. The existing list view shows all notes together with inline indicators.
- Performance: list queries already join on sharing tables (from feature 016); adding the share-state flag is an extension of that join, not a new round trip.
