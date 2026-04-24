# Data Model: Admin Dashboard and User Management

**Feature**: 017-admin-dashboard  
**Date**: 2026-04-24

## Schema Changes

### Modified Table: `users`

Two new columns added via migration:

| Column | Type | Default | Description |
|--------|------|---------|-------------|
| is_admin | INTEGER NOT NULL | 0 | 1 = admin, 0 = regular user |
| disabled | INTEGER NOT NULL | 0 | 1 = account disabled, 0 = active |

**Migration SQL** (applied in `migrateSchema`):

```sql
-- Add is_admin column if missing
ALTER TABLE users ADD COLUMN is_admin INTEGER NOT NULL DEFAULT 0;

-- Add disabled column if missing
ALTER TABLE users ADD COLUMN disabled INTEGER NOT NULL DEFAULT 0;

-- Index for admin lookups
CREATE INDEX IF NOT EXISTS idx_users_admin ON users(is_admin);
```

### New Table: `admin_audit_log`

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | INTEGER | PRIMARY KEY AUTOINCREMENT | Unique entry ID |
| admin_username | TEXT | NOT NULL | Username of the admin who performed the action |
| action | TEXT | NOT NULL | Action type (see enum below) |
| target_type | TEXT | NOT NULL | Entity type: "user", "note", "share", "public_note" |
| target_id | TEXT | NOT NULL | Identifier of the target (username, note slug, etc.) |
| details | TEXT | | Optional JSON with contextual info (impact summary) |
| created_at | TEXT | NOT NULL | RFC3339 timestamp |

**Action types** (string enum, not enforced by CHECK — extensible):
- `disable-user`
- `enable-user`
- `delete-user`
- `promote-admin`
- `demote-admin`
- `delete-note`
- `unpublish-note`
- `revoke-share`

**Create SQL**:

```sql
CREATE TABLE IF NOT EXISTS admin_audit_log (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    admin_username  TEXT    NOT NULL,
    action          TEXT    NOT NULL,
    target_type     TEXT    NOT NULL,
    target_id       TEXT    NOT NULL,
    details         TEXT,
    created_at      TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_log_created ON admin_audit_log(created_at);
CREATE INDEX IF NOT EXISTS idx_audit_log_action  ON admin_audit_log(action);
```

## Go Structs

### Extended User (admin view)

```go
type AdminUserView struct {
    ID        int64
    Username  string
    IsAdmin   bool
    Disabled  bool
    CreatedAt time.Time
    NoteCount int
    LastActive time.Time  // derived from most recent note updated_at or session activity
}
```

### Audit Log Entry

```go
type AuditLogEntry struct {
    ID            int64
    AdminUsername  string
    Action        string
    TargetType    string
    TargetID      string
    Details       string    // JSON or empty
    CreatedAt     time.Time
}
```

### Dashboard Metrics

```go
type DashboardMetrics struct {
    TotalUsers       int
    ActiveUsers30d   int
    TotalNotes       int
    NotesCreated7d   int
    TotalPublicNotes int
    TotalActiveShares int
}
```

## Entity Relationships

```text
users (modified)
  ├── is_admin (bool) ─── determines admin section access
  ├── disabled (bool) ─── blocks login + hides public notes
  ├── 1:N → notes (existing)
  ├── 1:N → note_shares (existing, as grantor or collaborator)
  └── 1:N → admin_audit_log (as admin_username, text reference)

admin_audit_log (new)
  └── admin_username (text) ─── intentionally NOT a foreign key
      (survives admin account deletion)
```

## State Transitions

### User Account Lifecycle

```text
                     ┌─────────────┐
    Sign-up via  ──→ │   Active    │ ←── re-enable
    GitHub OAuth     │ disabled=0  │
                     └──────┬──────┘
                            │
                    admin disables
                            │
                     ┌──────▼──────┐
                     │  Disabled   │
                     │ disabled=1  │
                     └──────┬──────┘
                            │
                    admin deletes
                            │
                     ┌──────▼──────┐
                     │  Deleted    │  (CASCADE: notes, shares,
                     │  (removed)  │   public_notes, images, etc.)
                     └─────────────┘
```

### Admin Role Lifecycle

```text
                     ┌─────────────┐
    Bootstrap via ──→│    Admin    │ ←── promoted by admin
    ADMIN_USER env   │ is_admin=1  │
                     └──────┬──────┘
                            │
                    demoted by admin
                    (if not last admin)
                            │
                     ┌──────▼──────┐
                     │  Regular    │
                     │ is_admin=0  │
                     └─────────────┘
```

## Cascade Behavior

### User Deletion Cascade

When an admin deletes a user account, the following are removed (via SQL CASCADE + explicit cleanup):

1. All notes owned by the user (CASCADE removes: note_tags, images, note_links, note_todos, note_embeddings, vec_note_embeddings, public_notes, note_shares)
2. All share grants where the user is a collaborator (note_shares.user_id)
3. Markdown files and uploaded images from filesystem
4. Session rows for the user

### Note Deletion Cascade (admin)

When an admin deletes a note:

1. SQL CASCADE removes: note_tags, images, note_links, note_todos, note_embeddings, vec_note_embeddings, public_notes, note_shares
2. Markdown file removed from filesystem
3. Drawing file removed if present
4. Upload files removed from filesystem

## Validation Rules

- `is_admin`: Only modifiable by existing admins. Cannot demote the last remaining admin (COUNT query guard).
- `disabled`: Only modifiable by admins. Disabling triggers immediate session deletion + sets flag. Cannot disable self (prevented in handler, not DB constraint).
- `admin_audit_log`: Append-only. No UPDATE or DELETE operations exposed.
- `ADMIN_USER` env var: Single username. Applied additively on startup — never revokes existing admins.

## Indexes

| Index | Table | Columns | Purpose |
|-------|-------|---------|---------|
| idx_users_admin | users | is_admin | Fast admin list lookup |
| idx_audit_log_created | admin_audit_log | created_at | Reverse-chronological listing |
| idx_audit_log_action | admin_audit_log | action | Filter by action type |
