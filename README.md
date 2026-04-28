# YANT - Yet Another Note Taking

A self-hosted note-taking application built with Go and plain Markdown files. Notes are stored as regular `.md` files on disk, so you can always access them outside the app. SQLite is used only as a derived index for search and metadata -- the Markdown files are the source of truth.

## Features

- Write and edit notes in Markdown with a live preview editor (EasyMDE)
- Inline image uploads, pasted or dragged into the editor
- Tag notes with hashtags directly in the note body (e.g. `#project`, `#idea`)
- Tags are extracted automatically and shown as filterable chips in the sidebar
- Customizable tag colors with a built-in color picker
- Semantic search powered by all-MiniLM-L6-v2 embeddings and sqlite-vec, with fuzzy text fallback
- Configurable search debounce, similarity threshold, and feature toggle
- Mermaid diagrams rendered inline using standard ` ```mermaid ` code blocks
- Freehand sketches and diagrams per note with a choice of **Excalidraw** or **tldraw**, stored as editable JSON
- Note version control using Git -- view history, browse previous versions, compare diffs, and revert changes
- Inline Markdown todos (`- [ ]` / `- [x]`) with a cross-note aggregation view
- Public note sharing via capability URLs (no login required to view)
- Share notes with specific users for read-only or read-write collaboration
- Admin dashboard for user management and system overview
- Archive notes to move them out of the main list without deleting them
- Archived notes have their own section with search and tag filtering
- Restore archived notes back to the active list at any time
- Auto-save in both the text editor and the drawing canvas
- Collapsible sidebar with keyboard shortcuts
- Mobile-responsive UI
- Keyboard navigation in search results (arrow keys, enter, escape)
- Database can be rebuilt from the Markdown files at any time (`--rebuild-db`)

## Requirements

- Go 1.25 or later
- Node.js 24 LTS (only needed to rebuild the tldraw/Excalidraw drawing components)
- Make

## Getting started

Clone the repository and run:

    make build-frontend   # first time only: install npm deps and build vendor assets
    make run

This compiles the server and starts it on http://localhost:8080.

Authentication requires a GitHub OAuth App. Create one at https://github.com/settings/developers and set the authorization callback URL to `http://localhost:8080/auth/github/callback`. Then provide the credentials:

    export GITHUB_CLIENT_ID=your_client_id
    export GITHUB_CLIENT_SECRET=your_client_secret
    make run

Accounts are created automatically on first sign-in using the GitHub username.

Data is stored in three directories at the repository root by default:

    notes.db       SQLite index (rebuildable)
    notes/         Markdown files
    uploads/       Uploaded images

Override these with environment variables or flags:

    make run ADDR=:9090 DB=./mydata.db NOTES_DIR=./mynotes UPLOADS_DIR=./myuploads

Semantic search requires the ONNX Runtime shared library (libonnxruntime.so) to be installed on the host for local development. It is bundled automatically in the Docker image. If the library is not found, the server falls back to text-based fuzzy search.

When running behind a reverse proxy that terminates TLS, set the external base URL so OAuth callbacks use the correct scheme:

    -base-url    External base URL (e.g. https://notes.example.com, env: BASE_URL)

Configuration flags for semantic search:

    -semantic-search    Enable/disable semantic search (default: true, env: SEMANTIC_SEARCH)
    -search-debounce    Search debounce in milliseconds (default: 300, env: SEARCH_DEBOUNCE_MS)
    -onnx-lib           Path to libonnxruntime.so (env: ONNXRUNTIME_LIB_PATH)

## Build

    make build              # compile server binary to ./bin/server
    make build-frontend     # build all frontend vendor assets (requires Node.js)

`make build-frontend` pulls htmx, EasyMDE, mermaid, tldraw, and Excalidraw via npm and copies their dist files to `frontend/static/vendor/`. This must be run once after cloning and whenever frontend dependencies are updated.

## Test

    make test               # run all Go tests
    make lint               # run go vet
    make coverage           # run tests and enforce coverage threshold
    make integration-test   # run integration tests against Docker image (requires docker-build)

## Docker

Build and run with Docker -- no Go or Node.js installation required:

    make docker-build
    make docker-run

The container listens on port 8080 and stores all data in a Docker volume called `yant-data`. Data persists across container restarts.

You can also run it directly:

    docker build -t yant .
    docker run --rm -p 8080:8080 -v yant-data:/data \
      -e GITHUB_CLIENT_ID=your_id \
      -e GITHUB_CLIENT_SECRET=your_secret \
      yant

Or use docker-compose. Create a `docker-compose.yaml`:

    services:
      yant:
        image: ghcr.io/selvakn/yant:6.0.0
        ports:
          - "8080:8080"
        volumes:
          - yant-data:/data
        environment:
          - GITHUB_CLIENT_ID=your_id
          - GITHUB_CLIENT_SECRET=your_secret
          - BASE_URL=https://notes.example.com
        restart: unless-stopped

    volumes:
      yant-data:

Then run:

    docker compose up -d

The Docker image uses a multi-stage build (Node.js for the frontend bundle, Go for the server, Debian bookworm-slim for the runtime with ONNX Runtime). Semantic search works out of the box -- the embedding model is compiled into the binary and the ONNX Runtime library is included in the image.

## CI/CD

The repository includes a GitHub Actions workflow (`.github/workflows/ci.yml`) that runs on every push to main and on pull requests:

- Runs the test suite and linter
- Scans Go dependencies for known vulnerabilities (govulncheck)
- Builds the Docker image and scans it with Trivy
- Runs integration tests against the Docker image
- Publishes the image to GitHub Container Registry on pushes to main and tagged releases

## Project structure

    backend/            Go module: server, handlers, models, storage, tests
    frontend/           HTML templates and static assets (CSS, JS, vendored libraries)
    frontend-build/     Node.js/Vite project for building the tldraw and Excalidraw components
    specs/              Feature specifications and planning documents
    Makefile            Build, test, run, and Docker targets
    Dockerfile          Multi-stage container build

## TODO

- Export notes as PDF or HTML
- Keyboard shortcuts for common actions
