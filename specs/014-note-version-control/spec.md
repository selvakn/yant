# Feature Specification: Note Version Control

**Feature Branch**: `014-note-version-control`  
**Created**: 2026-04-14  
**Status**: Draft  
**Input**: User description: "Version control the changes made to notes using git, with history, diff, date views, version viewing, and revert capabilities in the UI"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - View Note Change History (Priority: P1)

A user opens a note and wants to see what changes have been made over time. They navigate to a history view for that note, which displays a chronological list of all saved versions. Each entry shows the date and time of the change. This gives the user full visibility into how a note has evolved.

**Why this priority**: History is the foundational capability — without it, none of the other version control features (diff, view, revert) are meaningful. It also provides immediate value by giving users confidence that their changes are tracked.

**Independent Test**: Can be fully tested by creating a note, editing it several times, and verifying the history view shows each version with correct timestamps.

**Acceptance Scenarios**:

1. **Given** a note that has been edited 5 times, **When** the user opens the history view for that note, **Then** they see a list of 5 version entries sorted from newest to oldest, each showing the date and time of the save.
2. **Given** a note that has never been edited after creation, **When** the user opens the history view, **Then** they see a single entry representing the initial creation.
3. **Given** the history view is displayed, **When** the user looks at any version entry, **Then** they see a human-readable date/time (e.g., "Apr 14, 2026 at 3:42 PM") and a short summary or indicator of what changed (e.g., lines added/removed).

---

### User Story 2 - View a Note at a Specific Version (Priority: P2)

A user browsing the history of a note wants to see exactly what the note looked like at a particular point in time. They select a version from the history list and the system displays the full rendered note content as it existed at that version — in the same read-only format as the normal note reader.

**Why this priority**: Being able to view past versions is the next most valuable capability after seeing the history. It lets users recover lost content, verify what was written, and understand the evolution of a note.

**Independent Test**: Can be fully tested by editing a note to change its content, then viewing a previous version and confirming the displayed content matches what was saved at that time.

**Acceptance Scenarios**:

1. **Given** a note with 3 versions in history, **When** the user selects version 2, **Then** the system displays the full note content as it was at version 2, rendered as formatted markdown (read-only).
2. **Given** the user is viewing a historical version of a note, **When** they look at the page, **Then** there is a clear indicator showing which version they are viewing and the date it was saved, distinguishing it from the current version.
3. **Given** the user is viewing a historical version, **When** they want to return to the current version, **Then** they can navigate back to the current note with a single click.

---

### User Story 3 - Compare Versions (Diff View) (Priority: P3)

A user wants to understand exactly what changed between two versions of a note. From the history view, they can see the differences between consecutive versions (or between any selected version and the current version). The diff highlights additions and deletions clearly so the user can quickly understand the scope of changes.

**Why this priority**: Diff is essential for understanding changes at a granular level. While viewing a version (P2) shows what existed, diff shows what specifically changed — critical for reviewing edits and understanding intent.

**Independent Test**: Can be fully tested by editing a note to add and remove specific lines, then viewing the diff and confirming the additions and deletions are correctly highlighted.

**Acceptance Scenarios**:

1. **Given** a note where version 2 added two paragraphs and removed one sentence compared to version 1, **When** the user views the diff between version 1 and version 2, **Then** the added paragraphs are highlighted as additions and the removed sentence is highlighted as a deletion.
2. **Given** the history view with multiple versions, **When** the user selects a version entry, **Then** they can see the diff of that version compared to its previous version (what changed in that save).
3. **Given** the diff view is displayed, **When** the user reads it, **Then** additions and deletions are shown as a source-level unified diff of the raw markdown, with added lines colored distinctly from removed lines, and surrounding unchanged context lines shown for readability.

---

### User Story 4 - Revert a Note to a Previous Version (Priority: P4)

A user realizes that recent changes to a note were undesirable and wants to restore the note to how it was at an earlier version. From the history view or while viewing a specific historical version, they can choose to revert. The revert replaces the current note content with the content from the selected version, and this revert itself is recorded as a new version in the history.

**Why this priority**: Revert is the recovery action. While history and viewing give visibility, revert allows the user to act on that information. It's lower priority because it's used less frequently, but it's critical when needed.

**Independent Test**: Can be fully tested by editing a note, reverting to version 1, and verifying that the current note content matches version 1's content and that a new version entry appears in the history.

**Acceptance Scenarios**:

1. **Given** a note at version 3, **When** the user reverts to version 1, **Then** the current note content becomes identical to what it was at version 1, and the note remains accessible at its usual URL.
2. **Given** the user initiates a revert, **When** the revert completes, **Then** a new version entry is created in the history (the revert does not erase subsequent history), and the entry indicates it was a revert.
3. **Given** the user is viewing a historical version, **When** they click a revert action, **Then** the system asks for confirmation before performing the revert.
4. **Given** a revert has been performed, **When** the user views the history, **Then** all previous versions (including post-revert ones) remain visible and accessible.

---

### Edge Cases

- What happens when a note is created but never edited? The history should show a single initial version.
- What happens when a note is deleted? Version history is no longer accessible (consistent with current delete behavior).
- What happens when a note is archived and later restored? The version history should be preserved across archive/restore cycles.
- What happens when a user reverts to the current version (latest)? The system should indicate no changes are needed or treat it as a no-op.
- What happens when the note content is identical to a previous save (e.g., user opens editor and saves without changes)? No new version should be created if the content has not changed.
- What happens when viewing history for a note with a very large number of versions (e.g., 500+)? The history view should paginate or lazy-load entries to remain responsive.
- What happens when a note is renamed? Version history follows the note across renames and remains fully accessible under the new slug.
- What happens to existing notes when the feature is first deployed? All pre-existing notes are committed as an initial version on first run, ensuring every note has history from day one.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST automatically create a new version record every time a note's content is saved, provided the content has actually changed.
- **FR-002**: System MUST track and display the date and time of each version for every note.
- **FR-003**: System MUST provide a history view for each note, listing all versions in reverse chronological order (newest first).
- **FR-004**: System MUST allow users to view the full rendered content of any historical version of a note in a read-only format.
- **FR-005**: System MUST provide a diff view showing additions and deletions between consecutive versions for markdown content (source-level unified diff).
- **FR-006**: System MUST allow users to view the diff between any historical version and the current version.
- **FR-015**: System MUST display drawing changes in the diff view as a side-by-side rendering of the tldraw canvas at each version, both in read-only mode.
- **FR-007**: System MUST allow users to revert a note to any previous version.
- **FR-008**: System MUST record a revert action as a new version in the history (non-destructive revert), preserving all prior history.
- **FR-009**: System MUST require user confirmation before performing a revert.
- **FR-010**: System MUST clearly indicate when a user is viewing a historical version rather than the current version.
- **FR-011**: System MUST NOT create a new version if the saved content is identical to the current content.
- **FR-012**: System MUST preserve version history when a note is archived and restored.
- **FR-013**: System MUST provide navigation from the note reader to the history view and back.
- **FR-014**: System MUST handle notes with large version histories (500+ versions) without degraded usability (pagination or lazy loading).
- **FR-016**: System MUST seed all pre-existing notes with an initial version on first run, so that every note has at least one entry in its version history.

### Key Entities

- **Note Version**: A snapshot of a note's content at a specific point in time. Key attributes: version identifier, timestamp of save, reference to the note, content snapshot, change summary (lines added/removed).
- **Version History**: An ordered collection of all versions for a given note. Provides chronological navigation and serves as the entry point for diff, view, and revert operations.
- **Diff**: A computed comparison between two versions of a note, showing line-level additions, deletions, and unchanged context.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can access the full version history of any note within 2 seconds.
- **SC-002**: Users can view any historical version of a note within 2 seconds.
- **SC-003**: Users can see a clear, color-coded diff between any two versions within 3 seconds.
- **SC-004**: Users can revert a note to any previous version in under 3 clicks from the note reader.
- **SC-005**: 100% of note saves that change content result in a new tracked version.
- **SC-006**: Revert operations preserve full history — no versions are lost or overwritten.
- **SC-007**: Version history remains intact across note archive and restore operations.

## Clarifications

### Session 2026-04-14

- Q: Should there be one git repository per user or a single shared repository for all users' notes? → A: Single git repository covering all users' notes.
- Q: When a note is renamed (slug changes), does version history carry over to the new slug? → A: History follows the note across renames (preserved).
- Q: What format should the diff view use? → A: Source-level unified diff (raw markdown, colored +/- lines).
- Q: How should drawing (tldraw) changes appear in history and diff? → A: Drawings appear in history; diff shows side-by-side rendered tldraw views in read-only mode (no raw JSON).
- Q: When the feature is deployed, should existing notes be seeded with an initial version? → A: Yes, commit all existing notes as an initial version on first run.

## Assumptions

- Version control applies to note markdown content and associated drawing files (tldraw JSON). Uploaded images are not version-controlled since they are referenced by URL and not directly edited. Drawing diffs are presented as side-by-side read-only tldraw canvas renderings (not raw JSON).
- The feature is scoped to individual note versioning — there is no cross-note "global history" or "undo" across multiple notes.
- Concurrent editing by multiple users on the same note is out of scope; the system assumes single-writer semantics.
- The existing note save flow (create/update via the editor) is the trigger for version creation — no separate "commit" action is required from the user.
- Version history is per-user and per-note, consistent with the existing per-user note storage model. A single git repository covers all users' notes; history queries are scoped by user file path.
- The history UI is accessed from the note reader page via a dedicated control (e.g., a history button/link), not from the notes list.
- Performance expectations are based on typical personal note-taking usage (hundreds of notes, dozens of versions per note on average).
- Git is used as the underlying version control mechanism for tracking note file changes on the backend.
