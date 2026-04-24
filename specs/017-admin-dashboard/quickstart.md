# Quickstart: Admin Dashboard and User Management

**Feature**: 017-admin-dashboard  
**Date**: 2026-04-24

## Prerequisites

- Go 1.25+ installed
- GitHub OAuth app configured (existing setup)
- Make installed

## Configuration

Add the `ADMIN_USER` environment variable to your `.env` file or export it:

```bash
# .env
ADMIN_CLIENT_ID=your-github-client-id
GITHUB_CLIENT_SECRET=your-github-client-secret
ADMIN_USER=your-github-username
```

The `ADMIN_USER` value is the GitHub username of the first admin. This user will be granted admin access on startup (or on first login if they haven't signed up yet).

## Build and Run

```bash
make build
ADMIN_USER=your-github-username make run
```

Or with Docker:

```bash
make docker-build
docker run --rm -p 8080:8080 \
  -e GITHUB_CLIENT_ID=... \
  -e GITHUB_CLIENT_SECRET=... \
  -e ADMIN_USER=your-github-username \
  -v yant-data:/data \
  yant:latest
```

## Verify Admin Access

1. Open `http://localhost:8080` and sign in with the GitHub account matching `ADMIN_USER`
2. After login, an "Admin" link appears in the navigation bar
3. Click it to access the admin dashboard

## Admin Operations

### Promote Another Admin

1. Go to Admin > Users
2. Find the user to promote
3. Click "Promote to Admin"
4. The user will see the Admin link on their next page load

### Disable a User

1. Go to Admin > Users
2. Find the user to disable
3. Click "Disable" — takes effect immediately (active session terminated)

### Moderate Content

1. Go to Admin > Notes to browse all notes
2. Filter by owner, public status, or shared status
3. Click a note to view its content (read-only)
4. Click "Delete" for policy violations (confirmation required with impact summary)

## Running Tests

```bash
make test          # all tests including admin
make coverage      # with coverage gate
```

## Files Changed

New files:
- `backend/internal/handlers/admin.go`
- `backend/internal/handlers/admin_test.go`
- `backend/internal/models/admin.go`
- `backend/internal/models/admin_test.go`
- `frontend/templates/admin/*.html` (8 templates)

Modified files:
- `backend/cmd/server/main.go` (new env var, admin route group)
- `backend/internal/auth/auth.go` (disabled-user check)
- `backend/internal/models/models.go` (schema migration)
- `frontend/templates/base.html` (admin nav link)
