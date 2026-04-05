# my-notes Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-04-05

## Active Technologies
- Go 1.22+ + github.com/go-chi/chi/v5, github.com/yuin/goldmark, github.com/alexedwards/scs/v2, modernc.org/sqlite, EasyMDE 2.x (JS vendored), htmx 2.x (JS vendored) (001-markdown-note-taking)
- Markdown files (source of truth) + SQLite via modernc.org/sqlite (pure Go, no CGO) (001-markdown-note-taking)
- Go 1.22+ (existing project requirement); POSIX `make` + Standard Go toolchain (`go build`, `go test`, `go tool cover`, `go vet`); POSIX `awk`, `grep`, `rm` (002-makefile-build-scripts)
- N/A (build tooling only) (002-makefile-build-scripts)

- Python 3.12 + Flask 3.x, EasyMDE (JS, no build step), htmx (JS, no build step) (001-markdown-note-taking)

## Project Structure

```text
backend/
frontend/
tests/
```

## Commands

cd src [ONLY COMMANDS FOR ACTIVE TECHNOLOGIES][ONLY COMMANDS FOR ACTIVE TECHNOLOGIES] pytest [ONLY COMMANDS FOR ACTIVE TECHNOLOGIES][ONLY COMMANDS FOR ACTIVE TECHNOLOGIES] ruff check .

## Code Style

Python 3.12: Follow standard conventions

## Recent Changes
- 002-makefile-build-scripts: Added Go 1.22+ (existing project requirement); POSIX `make` + Standard Go toolchain (`go build`, `go test`, `go tool cover`, `go vet`); POSIX `awk`, `grep`, `rm`
- 001-markdown-note-taking: Added Go 1.22+ + github.com/go-chi/chi/v5, github.com/yuin/goldmark, github.com/alexedwards/scs/v2, modernc.org/sqlite, EasyMDE 2.x (JS vendored), htmx 2.x (JS vendored)

- 001-markdown-note-taking: Added Python 3.12 + Flask 3.x, EasyMDE (JS, no build step), htmx (JS, no build step)

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
