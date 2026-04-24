# Implementation Plan: Admin Dashboard and User Management

**Branch**: `017-admin-dashboard` | **Date**: 2026-04-24 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/017-admin-dashboard/spec.md`

## Summary

Add a full admin section to YANT that allows designated administrators to manage users (list, disable/enable, promote/demote admin, delete), moderate notes (browse, view read-only, delete), manage public notes (list, unpublish), oversee sharing relationships (list, revoke), and view platform metrics on a dashboard. Admin status is bootstrapped via an environment variable and then managed in-app by existing admins. All admin actions are recorded in an append-only audit log.

## Technical Context

**Language/Version**: Go 1.25  
**Primary Dependencies**: chi/v5 (routing), goldmark (markdown rendering), scs/v2 (sessions), modernc.org/sqlite (database), htmx (frontend interactivity)  
**Storage**: SQLite (metadata, audit log) + Markdown files (note content, source of truth)  
**Testing**: `go test` with ≥75% line coverage gate (Makefile), ≥90% target (constitution)  
**Target Platform**: Linux server, Docker (Debian bookworm-slim)  
**Project Type**: Web application (monorepo: `backend/` + `frontend/`)  
**Performance Goals**: Dashboard loads in <2s, admin actions complete in <1s  
**Constraints**: Single SQLite writer (MaxOpenConns=1), no new external dependencies  
**Scale/Scope**: Designed for small-to-medium deployments (<10k users, <100k notes)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [x] **I. Markdown-first storage** — Notes remain on disk as Markdown files. SQLite stores only admin metadata (is_admin flag, disabled flag, audit log). Admin note deletion removes both the DB record and the filesystem file. No change to the source-of-truth model.
- [x] **II. Simplicity** — No new external dependencies. Admin functionality uses existing patterns (chi routes, Go templates, htmx). Admin role is a simple boolean flag on the user table, not a separate RBAC system.
- [x] **III. Monorepo** — All admin code lives in the existing `backend/` and `frontend/` directories. New admin templates follow the established `frontend/templates/admin/` convention.
- [x] **IV. Integration testing** — Admin handlers will have integration tests covering authorization checks, CRUD operations, cascade behavior, and audit logging. Coverage target ≥90% for new code.
- [x] **V. Simple web UI** — Admin pages are server-rendered Go templates with htmx for interactive elements (search filtering, confirmation dialogs). No heavy JS frameworks.
- [x] **VI. Commit & test discipline** — Implementation will proceed in small, testable increments. Each commit will have passing tests before creation.

## Project Structure

### Documentation (this feature)

```text
specs/017-admin-dashboard/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   └── admin-routes.md  # Route definitions
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```text
backend/
├── cmd/server/
│   └── main.go                      # Modified: add ADMIN_USER env var, admin route group
├── internal/
│   ├── auth/
│   │   └── auth.go                  # Modified: add disabled-user check middleware
│   ├── handlers/
│   │   ├── admin.go                 # New: admin HTTP handlers (dashboard, users, notes, shares, audit)
│   │   └── admin_test.go            # New: admin handler tests
│   └── models/
│       ├── admin.go                 # New: admin data model functions (metrics, audit log, user admin ops)
│       ├── admin_test.go            # New: admin model tests
│       └── models.go               # Modified: schema migration (is_admin, disabled columns, audit_log table)

frontend/
└── templates/
    ├── base.html                    # Modified: conditional admin nav link
    └── admin/                       # New directory
        ├── dashboard.html           # Dashboard with metrics
        ├── users.html               # User list with search
        ├── user-detail.html         # User detail view
        ├── notes.html               # Note list with filters
        ├── note-detail.html         # Read-only note view
        ├── public-notes.html        # Public notes list
        ├── shares.html              # Shares overview
        └── audit-log.html           # Audit log view
```

**Structure Decision**: Follows existing monorepo layout. Admin handlers and models are separate files within existing packages (not a new package), consistent with how shares, public notes, and todos were added. Admin templates live in a new `admin/` subdirectory under templates.

## Complexity Tracking

No constitution violations. No complexity justifications needed.
