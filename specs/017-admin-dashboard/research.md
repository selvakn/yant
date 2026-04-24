# Research: Admin Dashboard and User Management

**Feature**: 017-admin-dashboard  
**Date**: 2026-04-24

## Decision 1: Admin Role Storage

**Decision**: Store `is_admin` as a boolean column on the `users` table.

**Rationale**: The spec defines admin as a simple binary designation — no role hierarchy, no fine-grained permissions. A boolean column is the simplest viable approach, avoids a separate roles/permissions table, and allows the existing `User` struct to carry admin status without join queries. The bootstrapping env var (`ADMIN_USER`) seeds this flag at startup.

**Alternatives considered**:
- Separate `user_roles` table with role types — rejected as over-engineered for a two-role system (admin vs. regular user).
- Configuration-only (env var list) — rejected during clarification; the user wants in-app admin management after bootstrap.
- Session-only flag — rejected; admin status must persist across sessions and be queryable for the user list.

## Decision 2: User Account Disabling

**Decision**: Add a `disabled` boolean column to the `users` table. Check this flag on every authenticated request via middleware, not just at login time.

**Rationale**: The spec requires immediate session termination when a user is disabled. Checking the flag per-request (with the session's user ID) ensures a disabled user is blocked even if their session cookie is still valid. Active sessions are also explicitly deleted from the `sessions` table for a clean cutoff.

**Alternatives considered**:
- Check only at login time — rejected; spec requires immediate termination of active sessions.
- Delete session rows only (no disabled column) — rejected; user would be able to re-login via GitHub OAuth immediately. A persistent flag is needed to block future logins.
- Redis-based session invalidation — rejected; project uses SQLite-backed sessions via scs, and session deletion from the `sessions` table achieves the same result without adding a dependency.

## Decision 3: Session Termination on Disable

**Decision**: When disabling a user, delete all rows from the `sessions` table where the session data contains the target user ID, AND set the `disabled` flag. Add middleware to check `disabled` status on each request as a safety net.

**Rationale**: The scs library stores session data as a blob in the `sessions` table. The most reliable approach is a two-layer defense: (1) immediately delete matching session rows to force logout, and (2) check the `disabled` flag on every authenticated request so even if a session survives (race condition), the user is still blocked. Finding sessions by user requires scanning session data or maintaining a user-to-session index.

**Alternatives considered**:
- User-to-session mapping table — possible but adds schema complexity. Since admin disabling is a rare operation, scanning the sessions table is acceptable.
- Destroy sessions via scs API — the scs library only supports destroying the current session (in a request context), not arbitrary sessions by user. Direct DB deletion is necessary.
- Periodic session cleanup — rejected; spec requires "within seconds" termination.

## Decision 4: Admin Middleware Pattern

**Decision**: Create an `AdminOnly` middleware that checks `is_admin` on the current user. Non-admin users receive a 404 (not 403) response to avoid leaking the existence of admin routes.

**Rationale**: Returning 404 instead of 403 is a security best practice for admin endpoints — it prevents enumeration. The middleware loads the user from DB by the session's user ID and checks `is_admin`. This is a per-request DB query but is acceptable given admin pages are low-traffic.

**Alternatives considered**:
- Store `is_admin` in session data — rejected; a user promoted/demoted mid-session would not see the change until re-login. DB check ensures real-time accuracy.
- Return 403 Forbidden — rejected; spec explicitly requires "not found" response to avoid revealing admin section existence.

## Decision 5: Audit Log Design

**Decision**: Create an `admin_audit_log` table with columns: id, admin_username, action, target_type, target_id, details (nullable JSON), created_at. Append-only, no retention limit.

**Rationale**: A simple append-only table is the minimum viable audit trail. Using `admin_username` (text) rather than a foreign key to `users.id` ensures audit entries survive even if the admin user is later deleted. The `details` column allows storing contextual info (e.g., impact summary for deletes) without schema changes. No retention policy per clarification.

**Alternatives considered**:
- Foreign key to admin user ID — rejected; the audit log should be durable even if the admin account is removed.
- Structured JSON log file — rejected; querying and filtering would require file parsing. SQLite table supports SQL filtering by action type and target.
- External logging service — rejected; out of scope per constitution (simplicity principle).

## Decision 6: Admin Bootstrap Flow

**Decision**: Read `ADMIN_USER` env var at startup. If the username exists in the DB, set `is_admin=1`. If the user doesn't exist yet, store the username in memory; when that user first logs in via `GetOrCreateUser`, set `is_admin=1`. On subsequent restarts, additionally grant admin to the env var user without revoking existing admins.

**Rationale**: This handles both cases — the admin user may already have an account, or may sign up later. The env var is additive (never revokes), so existing database-managed admins are preserved across restarts.

**Alternatives considered**:
- Only seed on first startup (use a flag file) — rejected; if the initial admin is misconfigured, there would be no recovery path without DB surgery.
- Override all admin status from env var on each restart — rejected; would break the in-app admin management model.

## Decision 7: Confirmation Dialog Pattern

**Decision**: Use htmx-powered confirmation modals for destructive actions (delete user, delete note). The modal fetches an impact summary from the server before confirming.

**Rationale**: Server-side impact calculation (count of notes, shares, public links) provides accurate numbers. htmx `hx-confirm` is too simple for custom impact summaries, so a two-step pattern is used: (1) button triggers a modal with the impact summary via htmx, (2) the modal's confirm button submits the actual delete.

**Alternatives considered**:
- Browser `confirm()` dialog — rejected; cannot show impact summaries.
- Client-side calculation — rejected; would require exposing counts in the HTML, adding complexity and potential staleness.

## Decision 8: Admin Note Viewing

**Decision**: Admin reads note content from the markdown file on disk (via existing `storage.ReadNote`), renders it with goldmark, and displays in a read-only template. No editor controls are rendered.

**Rationale**: Consistent with the existing note reader pattern. The admin sees the same rendered content as the note owner but without edit, archive, or share controls. Reading from the filesystem maintains the markdown-first principle.

**Alternatives considered**:
- Show raw markdown without rendering — rejected; admins need to see the content as users see it for effective moderation.
- Add a special "admin" role to `GetNoteForViewer` — rejected; admin access is a separate authorization path, not a share-based role. A dedicated handler is cleaner.
