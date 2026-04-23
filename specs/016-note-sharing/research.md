# Research: Note Sharing

## Decision 1: URL Scheme for Shared Notes

**Decision**: Use a separate namespace `/shared/{owner-username}/{slug}` for notes shared with the viewer. Keep `/notes/{slug}` strictly owner-scoped.

**Rationale**: Slugs are unique per user, not globally. If Alice shares her "home" note with Bob, and Bob also has his own "home" note, a single `/notes/home` URL is ambiguous. Using a different prefix with the owner's username embedded makes the URL unambiguous and makes authorization obvious: `/notes/*` = own only; `/shared/*` = explicit share grant required.

**Alternatives considered**:
- Keep `/notes/{slug}` and disambiguate by checking ownership first, then shares: requires owner-username in URL anyway for collisions; more coupling in handlers. Rejected.
- `/notes/{owner-username}/{slug}` for both owned and shared: breaks all existing `/notes/{slug}` URLs and bookmarks. Rejected.

## Decision 2: Disk Storage Layout

**Decision**: Keep the existing layout `{notesDir}/{ownerID}/{slug}.md`. Collaborator edits write to the owner's file, not a copy.

**Rationale**: Spec says last-write-wins is acceptable. A single canonical copy avoids sync/merge complexity. The owner's disk path is already the source of truth. Git versioning tracks the history; per-commit author override provides attribution.

**Alternatives considered**:
- Per-editor shadow copies: requires merge logic, conflict resolution. Massive increase in scope. Rejected.
- Symlinks in the collaborator's directory: filesystem-specific, fragile, gives no benefit. Rejected.

## Decision 3: Git Attribution via Per-Commit Author Override

**Decision**: Add `versioning.CommitFileAs(notesDir, relPath, message, authorName, authorEmail)` that shells `git -c user.name=... -c user.email=... commit ...` for a single commit. Owner edits continue to use the repo's default identity; collaborator edits pass the collaborator's username.

**Rationale**: Native git author tracking, browseable via existing version history UI. No separate audit table to maintain. The `-c` flag is the standard way to set per-commit identity in git.

**Alternatives considered**:
- Embed `edited by X` in the commit message: works but the author field is the canonical place; mixing metadata into the message complicates display. Rejected.
- Separate `note_edits` audit table: duplicates information already in git; adds write path complexity. Rejected.

## Decision 4: Permission Levels

**Decision**: Two levels: `read` and `edit`. Enforce at the API layer (not just UI).

**Rationale**: Matches the minimal viable collaboration model. Finer-grained roles (comment-only, share-only) add UX complexity without clear user value for a personal app.

**Alternatives considered**:
- Three levels (read/edit/admin): admins could re-share; rejected due to transitive-sharing scope exclusion.
- Custom ACLs per-field (e.g., can-edit-todos-only): massive scope increase. Rejected.

## Decision 5: Scope of "Shared Notes" Integration

**Decision**: Shared notes get their own dedicated `/shared` list view. They do NOT appear mixed into `/notes`, and they do NOT appear in the `/todos` aggregator. Search stays per-user.

**Rationale**: Separating owned and shared into distinct views keeps authorization obvious (no risk of leaking shared/private across boundaries in a combined query). Matches the mental model of "my stuff vs. theirs". The `/todos` aggregator is explicitly documented as personal-todo-oriented in v1.

**Alternatives considered**:
- Unified `/notes` list with a badge: requires restructured list query and complicates ownership UI. Deferrable to v2.
- Shared todos in the aggregator: could be useful but adds questions (whose view? whose clicks update?). Deferred.

## Decision 6: Owner Identity for URL Lookup

**Decision**: Use the owner's **username** (not userID) in the shared URL. `/shared/{username}/{slug}`.

**Rationale**: Human-readable, matches the rest of the URL style (notes use slugs, not IDs). Usernames are already unique.

**Alternatives considered**:
- Use ownerID: exposes internal IDs; harder to share verbally. Rejected.

## Decision 7: Wiki-Link Behavior in Shared Notes

**Decision**: When rendering a shared note, resolve `[[Title]]` against the **owner's** notes. If the target is also shared with the viewer, hyperlink to `/shared/{owner}/{target-slug}`. Otherwise, render as plain text.

**Rationale**: Matches the existing public-notes leakage-prevention pattern. Shared-note readers should never discover titles/slugs of the owner's other notes by browsing wiki-links.

**Alternatives considered**:
- Resolve against viewer's own notes: incorrect — the viewer is reading the owner's content. Rejected.
- Allow any link, 404 on private targets: reveals existence via the link shape. Rejected.

## Decision 8: Revocation Semantics

**Decision**: Revocation deletes the `note_shares` row. Subsequent access by the revoked user returns 404.

**Rationale**: Simple, clear, immediate. No caching, no soft-delete complexity.

**Alternatives considered**:
- Soft-delete with `revoked_at` timestamp: adds query complexity with no user-facing benefit in v1. Rejected.
