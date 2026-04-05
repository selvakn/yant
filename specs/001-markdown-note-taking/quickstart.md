# Quickstart: Markdown Note Taking App

**Date**: 2026-04-05
**Branch**: `001-markdown-note-taking`

## Prerequisites

- Go 1.22+
- Modern web browser (Chrome, Firefox, Safari, Edge)

## Setup

```bash
# From repository root
cd backend

# Download dependencies (pure Go — no C compiler needed)
go mod download
```

## Run the App

```bash
# From backend/
go run ./cmd/server

# Or build first
go build -o notes-server ./cmd/server
./notes-server
```

Open http://localhost:8080 in your browser.

**Flags**:
```
-addr   string   listen address (default ":8080")
-db     string   SQLite DB path (default "notes.db")
-notes  string   markdown storage root (default "notes/")
-uploads string  image storage root (default "uploads/")
-rebuild-db      rebuild SQLite index from markdown files, then exit
```

## Usage

1. **Login**: Enter any username. New accounts are created automatically.
2. **Create a note**: Click "New Note" on the home screen.
3. **Edit**: Type a title and Markdown content in the EasyMDE editor.
4. **Add images**: Drag and drop image files onto the editor area.
5. **Add tags**: Type `#tagname` anywhere in the note body.
6. **Reader mode**: Click "View" to see goldmark-rendered Markdown.
7. **Navigate by tag**: Use the tag sidebar to filter notes.

## Run Tests

```bash
# From backend/
go test ./... -cover -coverprofile=coverage.out

# Check coverage (must be ≥90%)
go tool cover -func=coverage.out | tail -1

# Run only integration tests (handlers package)
go test ./internal/handlers/... -v

# Run only unit tests (models, storage)
go test ./internal/models/... ./internal/storage/... -v
```

## Rebuild DB from Files

If the SQLite index gets out of sync with the markdown files:

```bash
./notes-server --rebuild-db
```

## Project Layout

```
backend/          Go backend (binary, tests, markdown/image storage)
  cmd/server/     Main entry point
  internal/       All application logic (not exported)
frontend/         Web UI (templates + vendored static assets)
  templates/      Go html/template files served at runtime
  static/vendor/  EasyMDE, htmx (no build step)
```

## Key Design Decisions

- **Markdown files are the source of truth** — SQLite indexes metadata
  only and can be rebuilt at any time.
- **No CGO** — `modernc.org/sqlite` is pure Go; `go build` requires
  no C toolchain.
- **No npm/build step** — EasyMDE and htmx are vendored static files.
- **Mock auth** — login with username only; real auth is a future
  iteration.
