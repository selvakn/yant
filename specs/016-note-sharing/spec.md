# Feature Specification: Share Notes with Specific Users

**Feature Branch**: `016-note-sharing`  
**Created**: 2026-04-22  
**Status**: Draft  
**Input**: User description: "ability to share a note with another user. the creator of the note is the primary owner. and can share with others."

## Clarifications

### Session 2026-04-22

- Q: When a collaborator edits a shared note, how should the version history attribute the edit? → A: Attribute each edit to the user who made it — history entries show the actual editor's username (e.g., "Bob edited at 3pm"). Consistent with Google Docs, GitHub, and Notion's collaboration model; adds accountability without significant complexity.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Owner Shares a Note with Another User (Priority: P1)

Alice creates a note and decides to collaborate on it with Bob, who already has an account. From the note view, Alice opens a "Share" dialog, types Bob's username, and selects a permission level (read or edit). Bob now sees the note in his notes list, can open it, and — if given edit permission — can modify it. Alice remains the owner; Bob's edits are saved to the same underlying note.

**Why this priority**: This is the foundational capability — without it, none of the other scenarios (listing shared notes, revoking access) can function. This is the MVP.

**Independent Test**: Log in as Alice, create a note, share it with bob, log in as bob, verify the note is visible and editable (or read-only, depending on the granted permission).

**Acceptance Scenarios**:

1. **Given** Alice owns a note and Bob has an account, **When** Alice shares the note with Bob (permission: edit), **Then** Bob can see the note in his notes list, open it, and modify it.
2. **Given** Alice shares a note with Bob (permission: read), **When** Bob opens the note, **Then** Bob sees the reader view but does not see an edit button, and attempts to call the edit endpoint are rejected.
3. **Given** Alice tries to share a note with a username that does not exist, **When** she submits the share form, **Then** she sees a clear error message ("User not found") and the share is not created.
4. **Given** Alice tries to share a note with a user who already has access, **When** she re-submits the share form, **Then** the permission is updated in place (no duplicate shares) and a confirmation is shown.
5. **Given** a note has been shared with Bob, **When** Alice opens the note, **Then** she sees Bob listed among the note's collaborators with his permission level.

---

### User Story 2 - Recipient Sees Shared Notes Alongside Their Own (Priority: P1)

Bob logs in. Alongside his personal notes, he can see notes that Alice (or anyone else) has shared with him. Shared notes are visually distinguished (badge, icon, or similar indicator) so Bob always knows which notes he owns vs. has access to. Shared notes are searchable and filterable by tag the same way as owned notes.

**Why this priority**: Sharing delivers no value if the recipient can't discover or access the shared content. This is tied priority with US1.

**Independent Test**: Alice shares multiple notes with Bob. Bob logs in and verifies: the notes appear in his list, they're visually marked as shared, they appear in search results, and tag filters include them.

**Acceptance Scenarios**:

1. **Given** Bob has been granted access to 3 notes by other users, **When** Bob opens his notes list, **Then** he sees his own notes plus the 3 shared notes, with a visual indicator on each shared note.
2. **Given** Bob has read-only access to a shared note, **When** he opens it, **Then** the reader view clearly shows "Shared by Alice (read-only)" and no edit/archive/delete controls appear.
3. **Given** Bob has edit access to a shared note, **When** he opens it, **Then** he sees the owner's name ("Shared by Alice") and he can use the Edit button; but he still cannot delete the note or share it further.
4. **Given** Bob searches for "meeting", **When** the search executes, **Then** results include matches from his own notes and from shared notes he has access to.
5. **Given** Bob edits a shared note and then Alice (the owner) opens its version history, **When** Alice looks at the latest entries, **Then** she sees Bob's username attached to the edits Bob made, distinct from her own edits.

---

### User Story 3 - Owner Manages Collaborators (Priority: P2)

Alice opens a note she owns. In the share dialog, she sees a list of everyone with whom the note is shared, along with their permission level. For any collaborator she can change the permission (read ↔ edit) or revoke access entirely. Revoked users lose access immediately — the note disappears from their list and any open tabs return "not found" on next interaction.

**Why this priority**: Access management is the second essential half of the feature. Without revocation, sharing is one-way and permanent.

**Independent Test**: Share a note with Bob, then revoke Bob's access. Verify Bob can no longer see or open the note.

**Acceptance Scenarios**:

1. **Given** Alice owns a note shared with Bob and Carol, **When** Alice opens the share dialog, **Then** both collaborators are listed with their current permission.
2. **Given** Alice changes Bob's permission from read to edit, **When** Bob refreshes the note, **Then** he sees the Edit button and can modify the note.
3. **Given** Alice revokes Carol's access, **When** Carol refreshes her notes list, **Then** the note no longer appears and opening its URL returns "not found".

---

### User Story 4 - Owner-Only Destructive Actions (Priority: P2)

Only the owner can archive, delete, or re-share a note. Collaborators with edit permission can modify content but cannot change the note's sharing configuration, archive it, or delete it.

**Why this priority**: This is a security boundary. Without this separation, edit-level collaborators could remove access for everyone else.

**Independent Test**: Give Bob edit access, then attempt (as Bob) to archive, delete, or share the note. All three attempts should be rejected.

**Acceptance Scenarios**:

1. **Given** Bob has edit access to a note owned by Alice, **When** Bob tries to archive or delete it, **Then** the action is rejected and the note remains unchanged.
2. **Given** Bob has edit access to a note, **When** Bob looks at the note's UI, **Then** the Archive, Delete, and Share buttons are absent or disabled.
3. **Given** Bob has edit access, **When** he attempts the archive/delete API endpoints directly, **Then** the server returns 403 Forbidden.

---

### Edge Cases

- What if the owner deletes their account? All notes they owned (including shared ones) are deleted; collaborators lose access.
- What if a collaborator deletes their account? The note remains with the owner; the revoked collaborator row is removed.
- What happens to a shared note when the owner archives it? The note is hidden from all collaborators' lists too — access is paused, not revoked.
- Can the owner share a note that's also published publicly? Yes, both mechanisms are independent: public notes are accessible via a capability URL; user-shared notes are visible only to specified authenticated users. A note can be both, neither, or either.
- Can a shared note's wiki-links point to the owner's other notes? Yes, but only if those target notes are also shared with the viewer, otherwise the wiki-link renders as plain text (to prevent information leakage about private notes).
- What if the owner tries to share a note with themselves? The attempt is rejected ("You already own this note").
- Can a user "unshare" a note from their own side (i.e., hide it)? Out of scope for v1 — only the owner can revoke. (Future enhancement.)
- What happens if two collaborators edit simultaneously? Last write wins — the same "last write wins" model used for concurrent note edits in the rest of the app.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST allow a note owner to share their note with another registered user by specifying the recipient's username.
- **FR-002**: When sharing, the owner MUST be able to choose between two permission levels: "read" (view only) or "edit" (view + modify content).
- **FR-003**: The system MUST reject share attempts to usernames that do not exist in the system, with a clear error message.
- **FR-004**: If the owner shares a note with a user who already has access, the existing permission MUST be replaced (not duplicated).
- **FR-005**: Shared notes MUST appear in the recipient's notes list alongside their own notes, with a visual indicator showing they are shared.
- **FR-006**: The reader view of a shared note MUST display the owner's name (e.g., "Shared by Alice").
- **FR-007**: Collaborators with "read" permission MUST NOT be able to modify the note's content. The edit UI and edit endpoints MUST be inaccessible to them.
- **FR-008**: Collaborators with "edit" permission MUST be able to modify the note's body and title, but MUST NOT be able to archive, delete, or change the note's sharing configuration.
- **FR-009**: Only the owner MUST be able to archive, delete, or change share settings on a note.
- **FR-010**: The owner MUST be able to view the full list of collaborators on a note, with their permission levels.
- **FR-011**: The owner MUST be able to change a collaborator's permission level (read ↔ edit).
- **FR-012**: The owner MUST be able to revoke a collaborator's access. Revocation takes effect within 1 second — subsequent requests by the revoked user return "not found".
- **FR-013**: Shared notes MUST be included in the recipient's search and tag filters.
- **FR-014**: When the owner archives a note, it MUST be hidden from all collaborators' lists until restored.
- **FR-015**: Wiki-links inside a shared note that point to other notes not shared with the viewer MUST render as plain text (no information leakage about the owner's private notes).
- **FR-016**: Owners MUST NOT be able to share a note with themselves (shows a clear error).
- **FR-017**: When the owner deletes their account, all notes they owned — including their share grants — MUST be removed.
- **FR-018**: Sharing MUST be independent of public link sharing (feature 015): a note can be public, shared with users, both, or neither.
- **FR-019**: Every edit to a shared note MUST be attributed to the user who made it. The note's version history MUST show each commit with the editor's username and timestamp.

### Key Entities

- **Note Collaborator**: A record of a user having been granted access to a note by the owner. Key attributes: note reference, collaborator user reference, permission level (read/edit), timestamp of grant. Uniqueness: one row per (note, user) pair. Scoped to: a single note.
- **Note Owner**: The user who created the note. Stored on the note itself (existing). Only the owner has destructive/administrative rights over the note and its sharing.
- **Shared-With-Me View**: A list, visible to each user, of notes owned by others but shared with them. Not a separate entity but a derived view.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An owner can share a note with another user by username in under 10 seconds from opening the note.
- **SC-002**: A recipient sees newly shared notes in their notes list on their next page load (no separate refresh required).
- **SC-003**: Revocation takes effect within 1 second — the note disappears from the revoked user's list and URL access returns "not found".
- **SC-004**: 100% of private notes (owned by the user but not shared with a given viewer) remain inaccessible to that viewer — verified by attempting to open every other user's note directly by URL.
- **SC-005**: Permission boundaries are enforced at the API level — direct calls to edit/archive/delete endpoints fail for users without the right role, not just UI hiding.
- **SC-006**: Shared notes appear in the recipient's search results and tag filters with the same relevance as their own notes.

## Assumptions

- Sharing is done by username, not email. The app's existing auth model already uses usernames as the primary identifier.
- Only two permission levels in v1: read and edit. No finer-grained roles (e.g., comment-only, share-only) in v1.
- Collaborators with edit permission can modify content (body, title, tags, todos, drawing) but cannot perform destructive actions (archive, delete) or administrative actions (change sharing). This is the "team member" model.
- Transitive sharing is not supported — collaborators cannot re-share a note they have access to.
- Recipients cannot "leave" a share from their own side in v1. Only the owner can revoke. Future enhancement could add a "hide from my list" option.
- Shared-note wiki-links to private notes render as plain text (consistent with the public notes feature 015).
- Concurrent edits by multiple collaborators use last-write-wins, matching the rest of the app's editing model.
- The UI for managing collaborators lives in the note reader's existing "..." dropdown or a new "Share" button, consistent with the pattern used by the publish toggle.
- Notifications (e.g., "Alice shared a note with you") are out of scope for v1 — recipients discover shared notes by checking their notes list.
- Shared notes are not exported/imported in any special way — they live in the owner's storage and are accessed via queries.
