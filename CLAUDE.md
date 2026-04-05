# my-notes Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-04-05

## Active Technologies

- Go 1.22+ — `github.com/go-chi/chi/v5`, `github.com/yuin/goldmark`, `github.com/alexedwards/scs/v2`, `modernc.org/sqlite` (pure Go, no CGO)
- Markdown note files (source of truth) with SQLite for metadata and tag index
- Frontend: Go-rendered HTML templates, EasyMDE, htmx, tldraw (vendored under `frontend/static/vendor/`)
- Frontend build: Node.js 18+ with Vite for tldraw bundle (see `frontend-build/`)
- POSIX `make` plus the standard Go toolchain (`go build`, `go test`, `go tool cover`, `go vet`)

## Project Structure

```text
backend/          # Go module: cmd/server, internal packages, *_test.go
frontend/         # templates/, static/ (CSS, JS, vendored editors)
frontend-build/   # Node.js/Vite project for building tldraw bundle
specs/            # Feature specs and plans per numbered feature
Makefile          # build, test, coverage, lint, run, build-frontend
```

## Commands

From the repository root:

- `make build` — compile server to `./bin/server`
- `make test` — run all Go tests (`backend/...`)
- `make coverage` — tests with ≥90% line coverage gate on `internal/...`
- `make lint` — `go vet ./...` in `backend`
- `make run` — build and start the server (default `:8080`; override with `ADDR=:9090 make run`)
- `make deps` — `go mod tidy` and `go mod download` in `backend`
- `make clean` — remove `./bin` and coverage artifacts
- `make build-frontend` — build tldraw bundle (requires Node.js 18+)

## Code Style

- Go: `gofmt` / `go fmt`, idiomatic error handling, keep handlers thin and logic testable
- Templates and static assets: match existing patterns in `frontend/`

## Recent Changes

- 001-markdown-note-taking: Go server + chi, goldmark, session auth, SQLite, EasyMDE + htmx
- 002-makefile-build-scripts: `make` targets for build, test, coverage, lint, run
- 003-note-tags: Editor tag bar with chips and quick-add; hyphenated hashtags
- 004-tldraw-diagrams: Drawing canvas per note using tldraw; frontend build system

<!-- MANUAL ADDITIONS START -->
- Git: Frequent commits; run the full test suite before every commit; if tests fail, fix tests or code before continuing (see `.specify/memory/constitution.md`, Principle VI).
<!-- MANUAL ADDITIONS END -->
