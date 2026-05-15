# yant Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-04-22

## Active Technologies
- Go 1.25+ (backend), vanilla JS + htmx (frontend) + chi/v5 (routing), goldmark + goldmark TaskList extension (markdown), scs/v2 (sessions), modernc.org/sqlite (database) (013-markdown-inline-todos)
- Markdown files (source of truth), SQLite `note_todos` table (derived cache for aggregation queries) (013-markdown-inline-todos)
- Go 1.25+ (backend), vanilla JS + htmx (frontend) + chi/v5 (routing), goldmark + GFM extension (markdown), scs/v2 (sessions), modernc.org/sqlite (015-public-notes)
- Markdown files (source of truth), SQLite `public_notes` table (public ID + published flag) (015-public-notes)
- Go 1.25+ (backend), vanilla JS + htmx (frontend) + chi/v5 (routing), goldmark + GFM extension, scs/v2 (sessions), modernc.org/sqlite (016-note-sharing)
- Markdown files (owner-scoped, unchanged), SQLite `note_shares` table (016-note-sharing)

- Go 1.22+ — `github.com/go-chi/chi/v5`, `github.com/yuin/goldmark`, `github.com/alexedwards/scs/v2`, `modernc.org/sqlite` (pure Go, no CGO), `modernc.org/sqlite/vec` (sqlite-vec vector search)
- ncnn (compiled from source, static-linked via CGO bridge) for 384-dim sentence embeddings; build with `-tags ncnn`; without tag, a stub is compiled (semantic search disabled). Model files downloaded at runtime from GitHub Release assets (convert-model.yml pipeline).
- Markdown note files (source of truth) with SQLite for metadata, tag index, and vector embeddings
- Frontend: Go-rendered HTML templates, EasyMDE, htmx, tldraw (vendored under `frontend/static/vendor/`)
- Frontend build: Node.js 24 LTS with Vite for tldraw bundle (see `frontend-build/`)
- POSIX `make` plus the standard Go toolchain (`go build`, `go test`, `go tool cover`, `go vet`)
- Docker (multi-stage build: Node.js + Go → Debian bookworm-slim runtime with ONNX Runtime)
- GitHub Actions CI/CD with GHCR publishing, govulncheck, Trivy scanning, and integration tests
- `github.com/testcontainers/testcontainers-go` for API-level integration tests

## Project Structure

```text
backend/          # Go module: cmd/server, internal packages, *_test.go
frontend/         # templates/, static/ (CSS, JS, vendored editors)
frontend-build/   # Node.js/Vite project for building tldraw bundle
specs/            # Feature specs and plans per numbered feature
Makefile          # build, test, coverage, lint, run, build-frontend, docker-build, docker-run
Dockerfile        # Multi-stage build (Node.js + Go → Alpine runtime)
.dockerignore     # Build context exclusions
.github/workflows/ci.yml  # CI/CD: test, lint, scan, build, publish
```

## Commands

From the repository root:

- `make build` — compile server to `./bin/server`
- `make test` — run all Go tests (`backend/...`)
- `make coverage` — tests with ≥75% line coverage gate on `internal/...` (excludes embedding pkg)
- `make integration-test` — run API-level integration tests against Docker image
- `make lint` — `go vet ./...` in `backend`
- `make run` — build and start the server (default `:8080`; override with `ADDR=:9090 make run`)
- `make deps` — `go mod tidy` and `go mod download` in `backend`
- `make clean` — remove `./bin` and coverage artifacts
- `make build-frontend` — build tldraw bundle (requires Node.js 24+)
- `make docker-build` — build Docker image (`DOCKER_IMAGE=yant DOCKER_TAG=latest`)
- `make docker-run` — run container with persistent data volume

## Code Style

- Go: `gofmt` / `go fmt`, idiomatic error handling, keep handlers thin and logic testable
- Templates and static assets: match existing patterns in `frontend/`

## Recent Changes
- 016-note-sharing: Added Go 1.25+ (backend), vanilla JS + htmx (frontend) + chi/v5 (routing), goldmark + GFM extension, scs/v2 (sessions), modernc.org/sqlite
- 015-public-notes: Added Go 1.25+ (backend), vanilla JS + htmx (frontend) + chi/v5 (routing), goldmark + GFM extension (markdown), scs/v2 (sessions), modernc.org/sqlite
- 013-markdown-inline-todos: Added Go 1.25+ (backend), vanilla JS + htmx (frontend) + chi/v5 (routing), goldmark + goldmark TaskList extension (markdown), scs/v2 (sessions), modernc.org/sqlite (database)


<!-- MANUAL ADDITIONS START -->

## Release Workflow

When the user asks to make a release (major/minor/patch):

1. Ensure current branch tests pass (`make test && make lint`).
2. Commit any pending changes and push the working branch.
3. Merge the feature branch to `main` (fast-forward) and push `main`.
4. Tag the release (e.g., `vX.Y.Z`), following semver from the latest `git tag --sort=-v:refname | head -1`.
5. Push the tag (`git push origin vX.Y.Z`).
6. Create the GitHub release with initial notes via `gh release create` — include a placeholder `_Pending CI build..._` under a "Docker Image" section.
7. Wait for the tag's CI run to complete (`gh run watch <id> --exit-status`).
8. Extract the published image tags from the CI logs (`gh run view <id> --log | grep "ghcr.io/selvakn/yant:"`).
9. Update the release notes via `gh release edit vX.Y.Z --notes` replacing the placeholder with the Docker pull command and alternative tags. Always include:
   - `docker pull ghcr.io/selvakn/yant:X.Y.Z`
   - `ghcr.io/selvakn/yant:X.Y`
   - `ghcr.io/selvakn/yant:latest`

This sequence runs automatically whenever the user asks to "make a release" — no need to re-confirm the Docker-image update step.

<!-- MANUAL ADDITIONS END -->

<!-- SPECKIT START -->
For additional context about technologies to be used, project structure,
shell commands, and other important information, read the current plan: specs/025-shared-note-authorship/plan.md
<!-- SPECKIT END -->
