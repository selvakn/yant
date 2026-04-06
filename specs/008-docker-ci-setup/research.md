# Research: Docker Packaging & CI/CD Pipeline

**Feature**: 008-docker-ci-setup  
**Date**: 2026-04-06

## Key Decisions

### 1. Docker Base Image

**Decision**: Use `golang:1.25-bookworm` for build stages, `debian:bookworm-slim` for runtime.

**Rationale**: The Go build stage needs the Go toolchain. The runtime stage only needs glibc (for modernc.org/sqlite which uses CGO-free pure Go but still needs libc). Alpine would be smaller but can cause subtle issues with DNS resolution and libc compatibility. `debian:bookworm-slim` is ~28MB and well-supported.

**Alternative rejected**: `scratch` or `distroless` — these lack a shell and basic utilities, making debugging harder. The application stores files on disk and needs a functional filesystem layer with proper user management.

### 2. Multi-Stage Build Strategy

**Decision**: Three-stage Dockerfile:
1. **frontend-builder**: Node.js stage to build tldraw bundle
2. **backend-builder**: Go stage to compile the server binary
3. **runtime**: Minimal Debian image with just the binary + assets

**Rationale**: Separates concerns, minimizes final image size, avoids shipping build tools. The frontend build produces `tldraw-bundle.js` and `tldraw-bundle.css` that get copied into `frontend/static/vendor/`.

### 3. GitHub Actions Workflow Design

**Decision**: Single workflow file with build/test/scan/publish jobs:
- `test`: Run `make test` and `make lint`
- `build-and-push`: Build Docker image, scan with Trivy, push to GHCR
- Conditional publish: only on main branch pushes and tags, not PRs

**Rationale**: Single workflow keeps CI simple (Principle II — simplicity). Job dependencies ensure tests pass before building/publishing.

### 4. Security Scanning Tools

**Decision**: 
- **govulncheck**: Official Go vulnerability scanner for dependency CVEs
- **Trivy**: Industry-standard container image scanner (OS packages + app dependencies)
- Both upload SARIF results to GitHub Security tab

**Rationale**: govulncheck is maintained by the Go team and understands Go module graphs. Trivy is the most widely adopted open-source container scanner with native GitHub Actions support and SARIF output.

### 5. Image Tagging Strategy

**Decision**:
- Main branch push: `latest` + git SHA (short, 7 chars)
- Tagged release (e.g., `v1.2.3`): `1.2.3` + `latest` + git SHA
- PR builds: build and scan but do not push

**Rationale**: SHA tags provide immutable references for deployments. `latest` provides convenience. Semantic version tags (without `v` prefix) follow container registry conventions.

### 6. Container Runtime Configuration

**Decision**: Use environment variables with sensible defaults:
- `PORT=8080`
- `DB_PATH=/data/notes.db`
- `NOTES_DIR=/data/notes`
- `UPLOADS_DIR=/data/uploads`

All persistent data under `/data` so users mount a single volume.

**Rationale**: Single volume mount simplifies deployment. Environment variables are the standard container configuration mechanism.

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Go 1.25 image not available | Low | High | Pin to latest available Go version, update when 1.25 ships |
| Trivy scan false positives | Medium | Low | Use `.trivyignore` for known false positives |
| GHCR rate limiting | Low | Low | GitHub Actions has generous limits for same-org pushes |
| Large image size | Low | Medium | Multi-stage build + slim base keeps image small |
