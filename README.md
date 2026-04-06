# YANT - Yet Another Note Taking

A self-hosted note-taking application built with Go and plain Markdown files. Notes are stored as regular `.md` files on disk, so you can always access them outside the app. SQLite is used only as a derived index for search and metadata -- the Markdown files are the source of truth.

## Features

- Write and edit notes in Markdown with a live preview editor (EasyMDE)
- Inline image uploads, pasted or dragged into the editor
- Tag notes with hashtags directly in the note body (e.g. `#project`, `#idea`)
- Tags are extracted automatically and shown as filterable chips in the sidebar
- Customizable tag colors with a built-in color picker
- Fuzzy search across note titles, tags, and body content, filtering as you type
- Freehand sketches and diagrams per note using tldraw, stored as editable JSON
- Archive notes to move them out of the main list without deleting them
- Archived notes have their own section with search and tag filtering
- Restore archived notes back to the active list at any time
- Auto-save in both the text editor and the drawing canvas
- Keyboard navigation in search results (arrow keys, enter, escape)
- Database can be rebuilt from the Markdown files at any time (`--rebuild-db`)

## Requirements

- Go 1.22 or later
- Node.js 24 LTS (only needed to rebuild the tldraw drawing component)
- Make

## Getting started

Clone the repository and run:

    make run

This compiles the server and starts it on http://localhost:8080. Sign in with any username -- accounts are created on first login.

Data is stored in three directories at the repository root by default:

    notes.db       SQLite index (rebuildable)
    notes/         Markdown files
    uploads/       Uploaded images

Override these with environment variables or flags:

    make run ADDR=:9090 DB=./mydata.db NOTES_DIR=./mynotes UPLOADS_DIR=./myuploads

## Build

    make build              # compile server binary to ./bin/server
    make build-frontend     # rebuild tldraw bundle (requires Node.js)

The tldraw bundle is already committed under `frontend/static/vendor/`, so you only need `make build-frontend` if you modify the drawing component source in `frontend-build/`.

## Test

    make test       # run all Go tests
    make lint       # run go vet
    make coverage   # run tests and enforce 90% line coverage

## Docker

Build and run with Docker -- no Go or Node.js installation required:

    make docker-build
    make docker-run

The container listens on port 8080 and stores all data in a Docker volume called `yant-data`. Data persists across container restarts.

You can also run it directly:

    docker build -t yant .
    docker run --rm -p 8080:8080 -v yant-data:/data yant

The Docker image uses a multi-stage build (Node.js for the frontend bundle, Go for the server, Alpine for the runtime) and comes out around 25 MB.

## CI/CD

The repository includes a GitHub Actions workflow (`.github/workflows/ci.yml`) that runs on every push to main and on pull requests:

- Runs the test suite and linter
- Scans Go dependencies for known vulnerabilities (govulncheck)
- Builds the Docker image and scans it with Trivy
- Publishes the image to GitHub Container Registry on pushes to main and tagged releases

## Project structure

    backend/            Go module: server, handlers, models, storage, tests
    frontend/           HTML templates and static assets (CSS, JS, vendored libraries)
    frontend-build/     Node.js/Vite project for building the tldraw component
    specs/              Feature specifications and planning documents
    Makefile            Build, test, run, and Docker targets
    Dockerfile          Multi-stage container build

## TODO

- Proper user authentication (the current login accepts any username with no password)
- Note sharing and collaboration
- Export notes as PDF or HTML
- Full-text search using SQLite FTS instead of in-memory fuzzy matching
- Mobile-friendly responsive layout
- Keyboard shortcuts for common actions
- Note versioning and history
