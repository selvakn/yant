# Feature Specification: Resource Limits & Abuse Prevention

**Feature Branch**: `026-resource-limits-defense`  
**Created**: 2026-05-21  
**Status**: Draft  
**Input**: User description: "security defense features. Apply a max size for the notes, includes the max size for the images and the number of images per notes. A note should nt be larger than 5 MB for reason. Limit the max number of notes to 25 for all users except admin. The goal is prevent and shield the app from any user from absuing and attempt to load the app and the data storage"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Note Size Limit Enforced on Save (Priority: P1)

A regular user writes a very large note or pastes a large amount of content. When they try to save, the system rejects the note if its total content exceeds 5 MB, displaying a clear error message without losing what they had before the limit was hit.

**Why this priority**: The note size limit is the most direct protection against storage abuse; a single oversized note can exhaust disk space and destabilize the server.

**Independent Test**: Create a note with content exceeding 5 MB, attempt to save, and confirm the save is rejected with an error. Existing notes below the limit remain unaffected.

**Acceptance Scenarios**:

1. **Given** a regular user has a note with content just under 5 MB, **When** they save the note, **Then** the save succeeds normally.
2. **Given** a regular user attempts to save a note with content at or over 5 MB, **When** the save request is submitted, **Then** the system rejects it with an error message indicating the note is too large and the note is not stored.
3. **Given** an admin user saves a note exceeding 5 MB, **When** the request is submitted, **Then** the system applies the same 5 MB limit to admin users (limit is universal for note content).

---

### User Story 2 - Image Upload Size and Count Limits (Priority: P1)

A regular user tries to embed multiple large images into a note. The system rejects any single image that exceeds a defined maximum size, and also rejects an image upload if the note already contains the maximum allowed number of embedded images.

**Why this priority**: Images are the primary vector for storage abuse; a user could embed dozens of large images to consume gigabytes of storage.

**Independent Test**: Attempt to upload an image over the size limit and confirm rejection. Upload images up to the count limit and confirm the next upload is refused.

**Acceptance Scenarios**:

1. **Given** a user uploads an image within the allowed size limit, **When** the upload is processed, **Then** the image is accepted and stored.
2. **Given** a user attempts to upload an image exceeding the per-image size limit, **When** the upload request is received, **Then** the system rejects it with a clear error message and the image is not stored.
3. **Given** a note has had 10 or more images uploaded to it over its lifetime (including deleted ones still in version history), **When** a user attempts to upload another image to that note, **Then** the system rejects the upload with an error indicating the lifetime image limit has been reached.
4. **Given** a note has fewer images than the maximum, **When** a user uploads an additional image within the size limit, **Then** the upload succeeds.

---

### User Story 3 - Note Count Limit for Regular Users (Priority: P2)

A regular user who already has 25 notes tries to create a new one. The system prevents creation beyond the 25-note limit and shows a clear message explaining why. Admin users are exempt from this limit and can create unlimited notes.

**Why this priority**: Capping note count bounds total storage per non-admin user, preventing individual users from accumulating unbounded data.

**Independent Test**: Create 25 notes as a regular user, then attempt to create a 26th and confirm the system refuses. Verify an admin user can create beyond 25 notes.

**Acceptance Scenarios**:

1. **Given** a regular user has 24 notes, **When** they create a new note, **Then** the note is created (they are still under the limit).
2. **Given** a regular user already has 25 notes, **When** they attempt to create a new note, **Then** the system rejects the request with an error message and no new note is created.
3. **Given** an admin user has 25 or more notes, **When** they create a new note, **Then** the note is created successfully (admins are exempt).
4. **Given** a regular user is at the 25-note limit and deletes one note, **When** they attempt to create a new note, **Then** creation succeeds (the limit is based on current note count, not lifetime count).

---

### User Story 4 - Admin Storage Usage Overview (Priority: P3)

An admin visits the admin dashboard and can see, for each user, how much total note storage they are consuming. This helps identify users who are approaching or misusing their allocation without requiring the admin to inspect individual notes.

**Why this priority**: Operational visibility is important but does not block the core abuse-prevention enforcement; it enhances the admin's ability to act on observed patterns.

**Independent Test**: Log in as admin, navigate to the admin dashboard, and confirm each user has a total note storage size displayed.

**Acceptance Scenarios**:

1. **Given** an admin views the admin dashboard, **When** the page loads, **Then** each user row displays the total size of their note text content in a human-readable format (e.g., KB or MB).
2. **Given** a user creates or updates notes, **When** the admin next views the dashboard, **Then** the displayed storage size reflects the updated total.

---

### Edge Cases

- Concurrent create requests at the note count limit: the system MUST enforce the limit atomically so that no user can exceed 25 notes regardless of request timing (see FR-011).
- How does the system handle a note that was created before limits were introduced and already exceeds 5 MB — can it still be read and edited (but not saved with over-limit content)?
- What happens if an image upload fails mid-way — is partial data cleaned up and not counted toward the limit?
- Version history reverts are not blocked by the image count limit — restoring a note to a historical state is always allowed. However, those historically-uploaded images still count toward the lifetime upload limit, so a future new upload may be rejected if the cumulative count reaches 10.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST reject any note save (create or update) where the markdown text content size exceeds 5 MB, returning an error to the user. Image files are bounded separately by FR-002 and FR-003 and are not included in this measurement.
- **FR-002**: The system MUST reject any image upload where the image file size exceeds 1 MB per image.
- **FR-003**: The system MUST reject any new image upload to a note if the note has accumulated 10 or more uploaded images over its lifetime. Once an image is uploaded to a note it counts permanently toward the limit, even if it is later removed from the note text (it remains in version history). The count is cumulative, not a snapshot of currently-embedded images.
- **FR-004**: The system MUST prevent regular (non-admin) users from creating more than 25 notes; the 26th creation attempt MUST be refused with a descriptive error.
- **FR-005**: Admin users MUST be exempt from the note count limit (FR-004) and MUST be able to create notes without restriction.
- **FR-006**: All limit violations MUST result in a user-facing error message that clearly explains which limit was exceeded and what the limit is.
- **FR-007**: The system MUST NOT silently truncate or partially save content when a limit is exceeded — the operation MUST fully fail.
- **FR-008**: Existing notes that were created before limits were enforced MUST remain readable, even if they exceed current size limits. Saving changes to an over-limit note MUST be rejected until the content is reduced below the limit.
- **FR-009**: The note count limit MUST reflect the user's current note count (not lifetime); deleting a note MUST free up one slot toward the 25-note limit.
- **FR-010**: All limit-enforcement checks MUST be performed server-side; client-side validation is optional and supplementary only.
- **FR-011**: The note count limit check MUST be enforced atomically — concurrent create requests from the same user MUST NOT result in more than 25 notes being created, even if requests arrive simultaneously.
- **FR-012**: The admin dashboard MUST display the total size of all note content (markdown text) per user, so admins can identify users consuming disproportionate storage.

### Key Entities

- **Note**: A user-owned document with text content and embedded images. Bounded by a 5 MB text content size limit and a per-note lifetime image upload count limit.
- **Image**: A file embedded within a note. Bounded by per-file size (1 MB) and per-note count (10 images).
- **User**: An account holder who owns notes. Regular users are subject to the 25-note count cap; admin users are exempt.
- **Limit Configuration**: The set of enforced thresholds (note size, image size, image count, note count). Assumed to be hardcoded constants initially; not user-configurable.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Any note save request exceeding 5 MB is rejected 100% of the time with an error response — no oversized notes are stored.
- **SC-002**: Any image upload exceeding 1 MB or that would push a note's lifetime upload count over 10 is rejected 100% of the time.
- **SC-003**: A regular user cannot hold more than 25 notes at any point in time; the 26th creation attempt is always refused.
- **SC-004**: Admin users can create and manage notes without encountering note count rejections, regardless of how many notes they own.
- **SC-005**: Limit enforcement adds no perceptible delay to normal (within-limit) note save and image upload operations — operations within limits complete in the same time as before the feature was introduced.
- **SC-006**: All error responses include a human-readable message that allows users to understand what limit was hit without needing to consult documentation.
- **SC-007**: The admin dashboard displays per-user total note storage size, allowing admins to identify storage-heavy accounts at a glance.

## Assumptions

- The 1 MB per-image limit and 10 images per-note limit are confirmed thresholds (see Clarifications).
- "Images" refers to files uploaded through the note editor's image insertion mechanism (not drawings or other binary attachments uploaded via other flows).
- The 5 MB note limit applies to the markdown text content only. Images are stored as separate files and are not counted toward this limit; they are bounded by FR-002 and FR-003.
- Admin role is already established in the system (feature 017-admin-dashboard); the `is_admin` flag or equivalent is already available on user records.
- Limits are applied uniformly to all regular users; there is no per-user configuration.
- The limits are hardcoded constants in the first implementation; a future feature could make them configurable.
- Notes created before these limits were introduced are grandfathered for reading but not for saving over-limit content.

## Clarifications

### Session 2026-05-22

- Q: What should the per-image size limit and per-note image count limit be? → A: 1 MB per image, 10 images per note (confirmed defaults)
- Q: Does the 5 MB note limit apply to text content only or text + image files? → A: Text content (markdown) only — images are bounded separately by FR-002/FR-003
- Q: Should the note count limit be enforced atomically under concurrent requests? → A: Yes — limit must never be exceeded even under simultaneous requests (FR-011)
- Q: Should version history reverts be blocked if they restore images beyond the count limit? → A: No — reverts are always allowed; however, once uploaded, images count permanently toward the lifetime limit for that note (even deleted images still in history), so new uploads may be blocked when the cumulative count reaches 10
- Q: Should the admin dashboard show observability into limit violations? → A: Yes — display total note storage size per user in the admin dashboard (FR-012)
