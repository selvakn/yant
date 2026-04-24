# Admin Routes Contract

**Feature**: 017-admin-dashboard  
**Date**: 2026-04-24

## Middleware

All admin routes are protected by two middleware layers:

1. **`auth.RequireLogin`** — existing middleware, redirects to `/login` if no session
2. **`AdminOnly`** — new middleware, returns 404 if `is_admin=0` or user not found

Additionally, a **disabled-user check** is added to the existing `RequireLogin` middleware (or as a separate middleware in the auth chain). If `disabled=1`, the session is destroyed and the user is redirected to `/login?error=disabled`.

## Route Group: `/admin`

All routes use server-rendered HTML (Go templates + htmx). JSON responses are used only for htmx partial updates and confirmation dialogs.

### Dashboard

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/admin` | `AdminDashboardGET` | Dashboard with platform metrics |

**Response**: HTML page with metrics cards (total users, active 30d, total notes, notes 7d, public notes, active shares).

---

### User Management

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/admin/users` | `AdminUsersListGET` | Paginated user list with search |
| GET | `/admin/users/{username}` | `AdminUserDetailGET` | User detail view |
| POST | `/admin/users/{username}/disable` | `AdminUserDisablePOST` | Disable user account |
| POST | `/admin/users/{username}/enable` | `AdminUserEnablePOST` | Enable user account |
| POST | `/admin/users/{username}/promote` | `AdminUserPromotePOST` | Promote to admin |
| POST | `/admin/users/{username}/demote` | `AdminUserDemotePOST` | Demote from admin |
| GET | `/admin/users/{username}/delete-confirm` | `AdminUserDeleteConfirmGET` | Fetch impact summary for confirmation dialog |
| DELETE | `/admin/users/{username}` | `AdminUserDeleteDELETE` | Delete user account (cascade) |

**Query parameters** (user list):
- `q` — search filter (partial username match)
- `page` — page number (default: 1)

**Disable/Enable behavior**:
- Executes immediately (no confirmation dialog)
- Disable: sets `disabled=1`, deletes user's session rows, returns htmx partial update
- Enable: sets `disabled=0`, returns htmx partial update

**Promote/Demote behavior**:
- Executes immediately (no confirmation dialog)
- Demote guard: returns 400 if target is the last remaining admin
- Cannot demote/promote self (returns 400)

**Delete behavior**:
- Two-step: GET `/delete-confirm` returns an impact summary (htmx modal), then DELETE executes
- Impact summary includes: note count, share count, public note count
- Cannot delete self (returns 400)

---

### Note Administration

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/admin/notes` | `AdminNotesListGET` | Paginated note list with filters |
| GET | `/admin/notes/{id}` | `AdminNoteDetailGET` | Read-only note view |
| GET | `/admin/notes/{id}/delete-confirm` | `AdminNoteDeleteConfirmGET` | Fetch impact summary |
| DELETE | `/admin/notes/{id}` | `AdminNoteDeleteDELETE` | Delete note (cascade) |

**Query parameters** (note list):
- `owner` — filter by owner username
- `public` — filter: "yes" or "no"
- `shared` — filter: "yes" or "no"
- `page` — page number (default: 1)

**Note detail**: renders markdown body in read-only mode (same as note reader but without edit/archive/share controls). Shows metadata: owner, created/updated dates, tags, public status, collaborators list.

**Delete behavior**: two-step confirmation, same pattern as user delete.

---

### Public Notes Management

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/admin/public-notes` | `AdminPublicNotesListGET` | List all public notes |
| POST | `/admin/public-notes/{id}/unpublish` | `AdminPublicNoteUnpublishPOST` | Unpublish a note |

**Unpublish behavior**: executes immediately (non-destructive, reversible by owner). Returns htmx partial update removing the row from the list.

---

### Sharing Oversight

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/admin/shares` | `AdminSharesListGET` | List all active shares |
| DELETE | `/admin/shares/{noteID}/{username}` | `AdminShareRevokeDELETE` | Revoke a share |

**Query parameters** (shares list):
- `user` — filter by username (as owner or collaborator)
- `page` — page number (default: 1)

**Revoke behavior**: executes immediately (non-destructive). Returns htmx partial update.

---

### Audit Log

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/admin/audit-log` | `AdminAuditLogGET` | View audit log |

**Query parameters**:
- `action` — filter by action type
- `user` — filter by target user
- `page` — page number (default: 1)

## Error Responses

| Scenario | Response |
|----------|----------|
| Non-admin user accesses /admin/* | 404 Not Found (HTML error page) |
| Disabled user accesses any protected route | Redirect to `/login?error=disabled` |
| Admin tries to delete self | 400 Bad Request with error message |
| Admin tries to demote last admin | 400 Bad Request with error message |
| Target user not found | 404 Not Found |
| Target note not found | 404 Not Found |

## Navigation

Admin users see an "Admin" link in the top navigation bar (in `base.html`). The link is conditionally rendered based on the `IsAdmin` template variable, which is set in `baseData()` by checking the user's `is_admin` flag.

Non-admin users never see this link.
