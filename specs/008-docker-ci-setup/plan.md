# Implementation Plan: Docker Packaging & CI/CD Pipeline

**Branch**: `008-docker-ci-setup` | **Date**: 2026-04-06 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/008-docker-ci-setup/spec.md`

## Summary

Add Docker containerization with a multi-stage Dockerfile (frontend build → Go build → minimal runtime), Makefile convenience targets, and a GitHub Actions CI/CD pipeline that builds, tests, scans (govulncheck + Trivy), and publishes images to GitHub Container Registry.

## Technical Context

**Language/Version**: Go 1.25 + Node.js 24 LTS (build only)  
**Primary Dependencies**: Docker, GitHub Actions, Trivy, govulncheck  
**Storage**: SQLite (via modernc.org/sqlite, pure Go) + filesystem  
**Testing**: `make test` (go test), `make lint` (go vet)  
**Target Platform**: Linux containers (amd64)  
**Project Type**: Web application  
**Performance Goals**: Image build < 5 minutes, final image < 50 MB  
**Constraints**: No CGO required; single-binary deployment  
**Scale/Scope**: Single-user note-taking app; single container deployment

## Constitution Check

- [x] **I. Markdown-first storage** — Docker volumes preserve filesystem-based note storage. No changes to storage model.
- [x] **II. Simplicity** — Single Dockerfile, single workflow file, minimal Makefile additions. No new application dependencies.
- [x] **III. Monorepo** — All Docker/CI files live at the repo root alongside existing backend/frontend structure.
- [x] **IV. Integration testing** — CI runs existing `make test` before building images. No new test infrastructure needed.
- [x] **V. Simple web UI** — No UI changes. Frontend assets are built and bundled as-is.
- [x] **VI. Commit & test discipline** — CI enforces test-before-build. Each implementation step can be committed independently with tests passing.

## Project Structure

### Documentation (this feature)

```text
specs/008-docker-ci-setup/
├── plan.md
├── research.md
├── quickstart.md
├── checklists/
│   └── requirements.md
└── tasks.md
```

### Source Code (repository root)

```text
Dockerfile                          # Multi-stage Docker build
.dockerignore                       # Build context exclusions
.github/
└── workflows/
    └── ci.yml                      # Build, test, scan, publish workflow
Makefile                            # Updated with docker-build, docker-run targets
```

**Structure Decision**: All Docker/CI files live at the repository root following standard conventions. No changes to existing `backend/`, `frontend/`, or `frontend-build/` directories.

## Complexity Tracking

No constitution violations. This feature adds build/deployment infrastructure only — no application code changes.
