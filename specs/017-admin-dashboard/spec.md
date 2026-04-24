# Feature Specification: Admin Dashboard and User Management

**Feature Branch**: `017-admin-dashboard`  
**Created**: 2026-04-24  
**Status**: Draft  
**Input**: User description: "admin functionality, to manage all users, and their notes and the note sharing, and manage public notes, and overall metrics, etc"

## Clarifications

### Session 2026-04-24

- Q: When an admin disables a user who is currently logged in, what happens to their active session? → A: Active sessions are terminated immediately — the disabled user is logged out within seconds, not just blocked on next login.
- Q: Should destructive admin actions (delete user, delete note) require confirmation before executing? → A: Yes, destructive actions only (delete user, delete note) require a confirmation dialog showing an impact summary (e.g., number of notes, shares, and public links that will be removed). Non-destructive actions (disable/enable, unpublish, revoke share) execute immediately.
- Q: Should the audit log have a retention policy or size limit? → A: No retention limit in v1. Admin action volume is inherently low, so unbounded growth is not a practical concern. A retention policy can be added later without schema changes.
- Q: How should admin users be designated beyond the initial seed? → A: The env var seeds the first admin, who can then promote/demote other users to admin via the admin UI. Admin status is stored in the database; the env var is only used for bootstrapping.
- Q: Should admin promote/demote actions require a confirmation dialog? → A: No — promote and demote execute immediately without confirmation, since they are fully reversible. Consistent with FR-023 (confirmation only for irreversible destructive actions).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Admin Dashboard with Platform Metrics (Priority: P1)

An administrator opens the admin section and immediately sees a dashboard summarizing the health and activity of the platform. The dashboard displays key metrics: total registered users, recently active users, total notes, notes created recently, number of public notes, and number of active sharing relationships. This gives the admin a quick overview without navigating into individual management views.

**Why this priority**: The dashboard is the landing page for all admin activity. It provides the context admins need before taking any action, and is the most immediately useful deliverable.

**Independent Test**: Log in as an admin user, navigate to the admin section, and verify that all metric counts are displayed and match the actual data in the system.

**Acceptance Scenarios**:

1. **Given** a designated admin user is logged in, **When** they navigate to the admin section, **Then** they see a dashboard showing: total users, users active in the last 30 days, total notes, notes created in the last 7 days, total public notes, and total active shares.
2. **Given** an admin is viewing the dashboard, **When** a new user signs up or a note is created, **Then** the dashboard reflects the updated counts on the next page load.
3. **Given** a non-admin user is logged in, **When** they attempt to access the admin section URL, **Then** they receive a "not found" response with no indication that an admin section exists.

---

### User Story 2 - User Management (Priority: P1)

An admin needs to review and manage user accounts. They open the user management view, which lists all registered users with key details (username, note count, last active date, account creation date, admin status). The admin can search for a specific user. They can view a user's profile showing their notes, shared notes, and public notes. The admin can disable a user (preventing login) or re-enable a previously disabled user. The admin can promote a regular user to admin or demote another admin back to a regular user. In extreme cases, the admin can delete a user account entirely, which removes all their notes, shares, and public links.

**Why this priority**: User management is the most critical admin capability. Without it, there is no way to handle abusive accounts, inactive users, or account-related support requests.

**Independent Test**: Log in as admin, navigate to user management, find a specific user by search, disable their account, verify they cannot log in, re-enable the account, verify they can log in again.

**Acceptance Scenarios**:

1. **Given** an admin opens the user management view, **When** the page loads, **Then** they see a list of all registered users showing username, note count, last active date, and account creation date.
2. **Given** an admin searches for a user by username, **When** they type a partial username, **Then** the list filters to show matching users.
3. **Given** an admin clicks on a user, **When** the user detail view opens, **Then** it shows the user's notes (count and titles), their active shares (notes shared by and with them), and their public notes.
4. **Given** an admin disables a user account, **When** that user is currently logged in, **Then** their active session is terminated immediately and subsequent requests redirect to the login page with a message that their account has been disabled.
5. **Given** an admin disables a user account, **When** that user attempts to log in later, **Then** they are shown a message that their account has been disabled and cannot access the application.
6. **Given** an admin re-enables a previously disabled user, **When** that user logs in again, **Then** they can access their notes and data as before.
7. **Given** an admin initiates a user deletion, **When** the confirmation dialog appears, **Then** it shows an impact summary (number of notes, shares, and public links that will be removed) and requires explicit confirmation before proceeding.
8. **Given** an admin confirms user deletion, **When** the deletion completes, **Then** all notes owned by that user are deleted, all shares they granted or received are revoked, and all their public note URLs return "not found".
9. **Given** an admin views the user list, **When** a user has admin status, **Then** they are visually indicated as an admin in the list.
10. **Given** an admin promotes a regular user to admin, **When** that user next loads any page, **Then** they see the admin navigation link and can access the admin section.
11. **Given** an admin demotes another admin, **When** the demoted user next loads any page, **Then** they no longer see admin navigation and cannot access the admin section.
12. **Given** only one admin remains, **When** that admin attempts to demote themselves or be demoted, **Then** the action is rejected with a message that at least one admin must exist.

---

### User Story 3 - Note Administration and Moderation (Priority: P2)

An admin needs to review notes across the platform for moderation purposes. They can browse all notes with filters (by user, by date, by public status, by shared status). The admin can view any note's content in read-only mode. If a note violates platform guidelines, the admin can delete it. The admin cannot edit note content — only view and delete.

**Why this priority**: Content moderation is essential for any multi-user platform, but is secondary to user management since user-level actions (disable/delete) can address most urgent issues.

**Independent Test**: Log in as admin, browse notes, filter by a specific user, open a note to read its content, delete a test note, and verify it is removed from the owner's list and any public/shared access stops working.

**Acceptance Scenarios**:

1. **Given** an admin opens the note administration view, **When** the page loads, **Then** they see a paginated list of all notes across all users showing title, owner, created date, and status indicators (public, shared, archived).
2. **Given** an admin filters notes by a specific user, **When** the filter is applied, **Then** only that user's notes are displayed.
3. **Given** an admin clicks on a note, **When** the detail view opens, **Then** they see the full rendered content in read-only mode, along with metadata (owner, dates, tags, public status, list of collaborators).
4. **Given** an admin initiates a note deletion, **When** the confirmation dialog appears, **Then** it shows the note title, owner, and impact summary (number of shares and public link status that will be affected) and requires explicit confirmation.
5. **Given** an admin confirms note deletion, **When** the deletion completes, **Then** the note is removed from the owner's list, its public URL (if any) returns "not found", and all share grants for that note are revoked.
6. **Given** an admin is viewing a note, **When** they look at the interface, **Then** there is no edit button or mechanism to modify the note's content.

---

### User Story 4 - Public Notes Management (Priority: P2)

An admin wants to review all notes that are currently published publicly. They open a public notes management view that lists every public note across all users, showing the note title, owner, public URL, and when it was published. The admin can unpublish any note (equivalent to the owner toggling it private), which immediately makes the public URL return "not found".

**Why this priority**: Public notes are externally visible, making them the highest-risk content for abuse. However, this is a subset of note administration and can be addressed via the general note moderation view in the short term.

**Independent Test**: Create a public note as a regular user, log in as admin, find the note in the public notes view, unpublish it, and verify the public URL now returns "not found".

**Acceptance Scenarios**:

1. **Given** an admin opens the public notes management view, **When** the page loads, **Then** they see a list of all currently public notes across all users with title, owner username, public URL, and published date.
2. **Given** an admin unpublishes a public note, **When** the action completes, **Then** the note's public URL immediately returns "not found" for unauthenticated visitors, and the note remains in the owner's private collection.
3. **Given** an admin unpublishes a note, **When** the owner views their note, **Then** the public toggle shows the note as private, and the owner can re-publish it if they choose.

---

### User Story 5 - Sharing Oversight (Priority: P3)

An admin wants to see all active note-sharing relationships on the platform. They open a sharing overview that lists all active shares — showing the note title, owner, collaborator, permission level, and when the share was granted. The admin can revoke any share if needed (e.g., if a user reports unwanted sharing).

**Why this priority**: Sharing oversight is a supporting admin capability. Most sharing issues can be resolved through user management or note moderation, making this a lower priority.

**Independent Test**: As admin, view the sharing overview, find a specific share between two users, revoke it, and verify the collaborator no longer has access to the note.

**Acceptance Scenarios**:

1. **Given** an admin opens the sharing overview, **When** the page loads, **Then** they see a list of all active shares showing note title, owner, collaborator, permission level (read/edit), and grant date.
2. **Given** an admin revokes a share, **When** the action completes, **Then** the collaborator immediately loses access to the note — it disappears from their list and URL access returns "not found".
3. **Given** an admin filters shares by a specific user, **When** the filter is applied, **Then** only shares involving that user (as owner or collaborator) are displayed.

---

### User Story 6 - Admin Audit Trail (Priority: P3)

All actions taken by administrators are logged for accountability. The audit log records who performed the action, what action was taken, which entity was affected, and when it happened. Admins can view the audit log from the admin section.

**Why this priority**: Audit logging is important for accountability and debugging but does not directly affect the admin's ability to manage the platform.

**Independent Test**: Perform several admin actions (disable a user, delete a note, unpublish a public note), then check the audit log to verify each action is recorded with correct details.

**Acceptance Scenarios**:

1. **Given** an admin performs any management action (disable user, delete note, unpublish note, revoke share), **When** the action completes, **Then** an audit log entry is created with: admin username, action type, target entity (user/note/share), target identifier, and timestamp.
2. **Given** an admin opens the audit log view, **When** the page loads, **Then** they see a chronological list of all admin actions, most recent first.
3. **Given** an admin filters the audit log by action type or target user, **When** the filter is applied, **Then** only matching entries are displayed.

---

### Edge Cases

- What happens if an admin disables another admin? The action succeeds — a disabled admin cannot log in. The system prevents disabling or demoting the last remaining active admin.
- What happens if an admin deletes their own account? The action is rejected — admins cannot delete their own account through the admin interface.
- What happens if an admin demotes themselves? The action succeeds if at least one other active admin exists. If they are the last admin, the demotion is rejected.
- What happens if the bootstrap env var is empty or the listed user doesn't exist yet? If empty, no admin is seeded at startup — the system operates without an admin section until an admin is created. If the user doesn't exist yet, admin status is granted when they first log in via GitHub OAuth.
- What happens on restart if the env var points to a different user than the existing admins? The env var user is additionally granted admin status. Existing admins in the database retain their status — the env var does not revoke existing admins.
- What happens to shared notes when an admin disables a user? Notes owned by the disabled user remain in the system but become inaccessible to collaborators. Public URLs for the disabled user's notes return "not found". When the user is re-enabled, access is restored.
- What if the admin tries to access a note that was just deleted by another admin? The system shows "Note not found" gracefully.
- Can an admin see the content of notes shared with specific users? Yes, admins can view any note content in read-only mode for moderation purposes.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST support bootstrapping the first administrator via an environment variable (a single GitHub username) provided at application startup. On startup, if the specified user exists, they are granted admin status. If they do not yet exist, admin status is granted when they first log in.
- **FR-001a**: Admins MUST be able to promote any registered user to admin and demote any other admin via the admin UI. Admin status is persisted in the database.
- **FR-001b**: The system MUST prevent the last remaining admin from being demoted or deleted, ensuring at least one admin always exists.
- **FR-002**: The system MUST expose an admin section accessible only to designated admin users. Non-admin users attempting to access admin URLs MUST receive a "not found" response.
- **FR-003**: The admin dashboard MUST display platform-wide metrics: total registered users, users active in the last 30 days, total notes, notes created in the last 7 days, total public notes, and total active shares.
- **FR-004**: The admin section MUST provide a user management view listing all registered users with username, note count, last active date, and account creation date.
- **FR-005**: The admin section MUST allow searching and filtering users by username.
- **FR-006**: The admin section MUST provide a user detail view showing a user's notes, shares (given and received), and public notes.
- **FR-007**: Admins MUST be able to disable a user account, which immediately terminates any active sessions for that user, prevents future logins, and hides their public notes from unauthenticated visitors.
- **FR-008**: Admins MUST be able to re-enable a previously disabled user account, restoring their login ability and public note visibility.
- **FR-009**: Admins MUST be able to delete a user account, which cascades: deletes all their notes, revokes all shares they granted or received, and removes all public note URLs.
- **FR-010**: Admins MUST NOT be able to delete their own account through the admin interface.
- **FR-011**: The admin section MUST provide a note administration view with a paginated list of all notes across all users, showing title, owner, created date, and status indicators (public, shared, archived).
- **FR-012**: Admins MUST be able to view any note's full content in read-only mode.
- **FR-013**: Admins MUST be able to delete any note, which removes it from the owner's collection, revokes all its shares, and makes its public URL (if any) return "not found".
- **FR-014**: Admins MUST NOT be able to edit any note's content — admin note access is strictly read-only plus delete.
- **FR-015**: The admin section MUST provide a public notes management view listing all currently public notes with title, owner, public URL, and published date.
- **FR-016**: Admins MUST be able to unpublish any public note, which immediately makes the public URL return "not found" while preserving the note in the owner's private collection.
- **FR-017**: The admin section MUST provide a sharing overview listing all active shares with note title, owner, collaborator, permission level, and grant date.
- **FR-018**: Admins MUST be able to revoke any share, immediately removing the collaborator's access.
- **FR-019**: All admin actions (disable/enable user, delete user, delete note, unpublish note, revoke share, promote/demote admin) MUST be recorded in an audit log with: admin username, action type, target entity, target identifier, and timestamp.
- **FR-020**: The admin section MUST provide an audit log view showing all recorded admin actions in reverse chronological order, with filtering by action type and target user.
- **FR-021**: When a user account is disabled, notes shared with that user's collaborators MUST become inaccessible until the account is re-enabled.
- **FR-022**: The note administration view MUST support filtering by owner, by public status, and by shared status.
- **FR-023**: Destructive admin actions (delete user, delete note) MUST require a confirmation dialog that displays an impact summary (e.g., number of notes, shares, and public links affected) before execution. Non-destructive actions (disable/enable user, unpublish note, revoke share) MUST NOT require confirmation.

### Key Entities

- **Admin Role**: A designation stored as a boolean flag on the User entity in the database. The first admin is bootstrapped via an environment variable at startup; subsequent admins are promoted/demoted by existing admins through the admin UI. At least one admin must exist at all times.
- **Admin Audit Log Entry**: A record of an administrative action. Key attributes: admin username, action type (disable-user, enable-user, delete-user, delete-note, unpublish-note, revoke-share, promote-admin, demote-admin), target entity type, target identifier, timestamp. Append-only — entries are never modified or deleted.
- **User Account Status**: An extension of the User entity to track whether an account is active or disabled. Key attribute: a boolean active/disabled flag on the user record.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An admin can view the platform dashboard and understand the current state of the system within 5 seconds of navigating to the admin section.
- **SC-002**: An admin can find a specific user by searching their username within 5 seconds.
- **SC-003**: Disabling a user account takes effect within 1 second — the user's next login attempt is rejected and their public notes become inaccessible.
- **SC-004**: 100% of admin actions are recorded in the audit log with actor, action, target, and timestamp.
- **SC-005**: Non-admin users have zero visibility into the admin section — no navigation links, no accessible URLs, no error messages that reveal its existence.
- **SC-006**: An admin can find, review, and moderate (delete or unpublish) any note within 30 seconds.
- **SC-007**: Deleting a user account cascades correctly — within 1 second, all the user's notes, shares, and public URLs are removed.

## Assumptions

- The first admin is bootstrapped via an environment variable (a single GitHub username), consistent with the app's existing configuration pattern for GitHub OAuth credentials. After bootstrapping, admin status is stored in the database and managed through the admin UI by existing admins. The env var is only consulted at startup for initial seeding — it does not override database-stored admin status on subsequent restarts.
- Admins have read-only access to note content. They can view and delete but never edit user content. This follows the principle of least privilege for moderation.
- Disabling a user preserves their data (notes, shares) but makes everything inaccessible. Re-enabling restores access. This is a reversible action, unlike deletion.
- Deleting a user is irreversible and cascades to all their data, consistent with existing behavior described in spec 016 (note-sharing edge cases).
- The admin section is a separate area within the existing application, not a standalone service. It reuses the existing authentication, session management, and UI patterns.
- There is no concept of "super admin" vs. "regular admin" — all admins have the same capabilities regardless of whether they were bootstrapped via env var or promoted through the UI.
- Admin users can also use the application as regular users (create notes, share, etc.) alongside their admin capabilities.
- The audit log is append-only and internal. It is not exposed to non-admin users. No external SIEM integration is planned for v1. No retention limit or automatic pruning in v1 — admin action volume is low enough that unbounded growth is not a concern.
- The metrics on the dashboard are computed on page load (no real-time updates or WebSocket push). Refreshing the page updates the numbers.
- Pagination and search are essential for the user and note management views to handle growth, but no specific page size is mandated — reasonable defaults (e.g., 25 items per page) are acceptable.
