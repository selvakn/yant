# Feature Specification: Public Note Sharing

**Feature Branch**: `015-public-notes`  
**Created**: 2026-04-14  
**Status**: Draft  
**Input**: User description: "add a functionlity to make a note public, so that it can be accessed without login"

## Clarifications

### Session 2026-04-14

- Q: Should the system track view counts / last-viewed timestamps on public notes? → A: No view tracking in v1 — keep the feature simple. Can be added later without migration pain.
- Q: What access control does v1 support beyond the unguessable URL? → A: Unguessable URL only — no password, no expiry. Matches MVP direction and industry defaults (Google Docs, Notion).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Make a Note Public (Priority: P1)

The note owner wants to share one of their notes with someone who doesn't have an account on the app. From the note's view, they toggle a "Public" switch. The app generates a unique, unguessable public URL for the note. The owner copies the URL and shares it (via chat, email, etc.). Anyone with the URL can read the note without logging in.

**Why this priority**: This is the foundational capability — without the ability to publish a note, no read-side feature (accessing via URL) can exist. This is the MVP.

**Independent Test**: Can be fully tested by toggling a note's public status and visiting the generated URL in an incognito browser session to confirm the note is readable without authentication.

**Acceptance Scenarios**:

1. **Given** the owner is viewing one of their notes, **When** they toggle the "Public" option on, **Then** the system displays a public URL they can copy, and the note is accessible at that URL without login.
2. **Given** a note is public and the URL has been shared, **When** an unauthenticated visitor opens the URL, **Then** they see the rendered note (title and body) with no sign-in prompt.
3. **Given** a note is public, **When** the owner toggles "Public" off, **Then** the public URL returns a "not found" response for unauthenticated visitors.
4. **Given** a note is public, **When** the owner toggles it off and on again, **Then** the system reuses the same public URL (the link does not change across toggles) so previously shared links continue to work.

---

### User Story 2 - Read a Public Note Without Signing In (Priority: P1)

A visitor receives a public note URL from a friend. They open it in any browser without an account. The note renders with its title, body, rendered markdown, and drawing (if any). The visitor cannot see the owner's other notes, the sidebar navigation, or any edit controls.

**Why this priority**: The read-side is the other half of the MVP — without a clean public reading experience, the share feature delivers no value. Tied priority with Story 1.

**Independent Test**: Visit a public note URL in a fresh browser (no cookies, no session) and verify the note renders correctly, no private UI elements appear, and no redirect to login occurs.

**Acceptance Scenarios**:

1. **Given** a visitor has a public note URL, **When** they open it in an unauthenticated browser, **Then** they see the note title, rendered markdown body, and the note's drawing (if any) in read-only mode.
2. **Given** a visitor is viewing a public note, **When** they look at the page, **Then** they do not see the sidebar, tags list, search box, edit button, archive button, or any other owner-only controls.
3. **Given** a visitor is viewing a public note, **When** that note contains a `[[wiki-link]]` to another note, **Then** the link is either non-clickable or only clickable if that other note is also public. Links to private notes MUST NOT navigate anywhere or reveal the note's existence.
4. **Given** a visitor is viewing a public note with interactive todos, **When** they see the checkboxes, **Then** the checkboxes render in read-only mode (visible state, not clickable/togglable).
5. **Given** a note URL that was public but has since been made private, **When** a visitor opens it, **Then** they see a "Note not found" page with no information about the note.

---

### User Story 3 - Manage Public Notes from a Central Place (Priority: P3)

The owner wants to see at a glance which of their notes are currently public. A section in the sidebar (or a dedicated view) lists all their public notes. From there, they can quickly visit the public URL, copy the link, or revoke public access.

**Why this priority**: Nice-to-have once the core feature is working — it gives users visibility and control over their shared content.

**Independent Test**: Mark two notes as public, navigate to the "Public notes" view, verify both appear with their public URLs, and verify the "Make private" action works from this view.

**Acceptance Scenarios**:

1. **Given** the owner has made some notes public, **When** they open the "Public notes" view, **Then** they see a list of all their currently public notes with titles and copyable URLs.
2. **Given** the owner is viewing the public notes list, **When** they click "Make private" on a note, **Then** that note is removed from the list and its public URL stops working.

---

### Edge Cases

- What happens if the owner archives a public note? Archived notes become private automatically — the public URL stops working until the note is both restored and re-made public.
- What happens if the owner deletes a public note? The public URL returns "not found."
- What if the note contains an image uploaded by the owner? The image MUST be accessible to unauthenticated visitors viewing the public note (the access grant flows with the public note).
- What if a public note links to another note via `[[wiki-link]]` that is private? The link MUST NOT reveal the existence, title, or any content of the private note. The safest behavior is to render the link text as-is without a hyperlink, or render it as a broken link.
- What if the owner's account is deleted? All their public URLs stop working.
- What happens if a visitor tries to guess URLs? The public slug/token MUST be long and unguessable (sufficient entropy to prevent enumeration).
- How does a public note look on social sharing platforms? The page MUST include basic Open Graph / meta tags so that links render with the note title and a short excerpt when shared.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST provide a toggle in the note owner's view to make a note public or private.
- **FR-002**: When a note is made public, the system MUST generate a unique, unguessable public identifier (token/slug with sufficient entropy) that is used in the public URL.
- **FR-003**: The system MUST expose a public URL at a distinct path (e.g., `/p/{public-id}`) that serves the note without requiring authentication.
- **FR-004**: The public note page MUST render the note's title, markdown body, and drawing (if any) in read-only mode.
- **FR-005**: The public note page MUST NOT expose any owner-only UI: sidebar, tags list, search, edit button, archive button, todos toggling, or navigation to other notes that are not public.
- **FR-006**: When a public note is made private again, the public URL MUST return "not found" for unauthenticated visitors.
- **FR-007**: When a public note is archived or deleted, the public URL MUST return "not found" for unauthenticated visitors.
- **FR-008**: Toggling a note between public and private MUST preserve the same public identifier so previously shared links continue to work when the note is re-published.
- **FR-009**: Images embedded in a public note MUST be accessible to unauthenticated visitors when loaded as part of that public note.
- **FR-010**: Wiki-links to notes that are not public MUST NOT reveal the existence or any content of those notes (render as non-clickable text or broken link).
- **FR-011**: Todo checkboxes on a public note MUST render in read-only mode for unauthenticated visitors — they cannot be toggled.
- **FR-012**: The public note page MUST include basic HTML meta tags (title, description) for link previews on social platforms.
- **FR-013**: The system MUST provide a view listing all of the owner's currently public notes with their titles and public URLs.
- **FR-014**: The owner MUST be able to copy a public URL and revoke public access from both the note view and the public notes list.

### Key Entities

- **Public Note Share**: A derived/optional attribute of a Note. Key fields: a boolean "is public" flag (on the Note), a persistent unique public identifier (generated once on first publish, retained on toggle), and the timestamp of first publish. Identified by its parent Note.
- **Public Note Page**: The unauthenticated rendering of a public note — displays only the owner-chosen content (title, body, drawing) and nothing else about the owner or their private data.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An owner can publish a note and copy its public URL in under 10 seconds.
- **SC-002**: A visitor with a public URL can read the note in under 2 seconds of page load on a standard broadband connection, with no login prompt.
- **SC-003**: 100% of private notes remain inaccessible to unauthenticated visitors — no URL enumeration, error message, or hint reveals their existence.
- **SC-004**: Revoking public access takes effect within 1 second — subsequent visits to the URL return "not found."
- **SC-005**: Public URLs survive toggling public → private → public: the same URL works after re-publishing, so shared links do not rot.
- **SC-006**: Visitors see no owner-only UI elements on public pages — verified by a checklist of private UI (sidebar, search, edit, archive, etc.) being absent.

## Assumptions

- The existing per-note slug is already part of the private URL (`/notes/{slug}`) and must remain private. A new, separate public identifier is introduced for public URLs.
- The public URL is an "unguessable URL" share model (capability URL) — anyone with the URL can read. This is consistent with how link-sharing works in tools like Google Docs, Notion, and Dropbox. No passwords, expiry, or per-recipient access control in v1.
- Public notes are read-only for visitors. Interactive elements (todos, drawings) render in read-only mode. No commenting, reactions, or edits.
- Public notes are indexed by search engines only if explicitly desired by the owner in a future enhancement — v1 defaults to `noindex` to prevent accidental SEO exposure.
- Embedded images are served from the same domain and become accessible when referenced by a public note.
- Revocation is immediate (no caching delay) — when an owner toggles a note to private, the next request returns "not found."
- The feature reuses the existing markdown rendering, storage, and authentication infrastructure.
- Wiki-links to private notes are the trickiest edge case. The safest v1 behavior is to render them as plain text (not hyperlinks) in the public view, avoiding any information leakage about private notes.
- No view tracking, analytics, or last-viewed timestamps in v1. The owner sees which notes are public but not how often they've been accessed.
