# Tasks: Docker Packaging & CI/CD Pipeline

**Input**: Design documents from `/specs/008-docker-ci-setup/`
**Prerequisites**: plan.md, spec.md, research.md

**Tests**: Existing test suite (`make test`, `make lint`) — no new application tests needed. Docker build verification is the primary validation.

**Organization**: Tasks grouped by deliverable for incremental implementation.

## Format: `[ID] [P?] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- Exact file paths included

---

## Phase 1: Docker Build Infrastructure

**Purpose**: Dockerfile and .dockerignore for building the application image

- [x] T001 Create `.dockerignore` to exclude unnecessary files from build context
- [x] T002 Create multi-stage `Dockerfile` with frontend-builder, backend-builder, and runtime stages
- [x] T003 Verify Docker build completes successfully: `docker build -t yant .`

**Checkpoint**: `docker build` produces a working image

---

## Phase 2: Makefile Targets

**Purpose**: Convenience scripts for Docker operations

- [x] T004 Add `docker-build` target to `Makefile`
- [x] T005 Add `docker-run` target to `Makefile` with port mapping and volume mount

**Checkpoint**: `make docker-build` and `make docker-run` work correctly

---

## Phase 3: GitHub Actions CI Workflow

**Purpose**: Automated build, test, scan, and publish pipeline

- [x] T006 Create `.github/workflows/ci.yml` with test job (make test + make lint)
- [x] T007 [P] Add Docker build-and-push job using `docker/build-push-action`
- [x] T008 [P] Add govulncheck security scanning step
- [x] T009 [P] Add Trivy container image scanning step with SARIF upload
- [x] T010 Configure conditional publishing (main branch + tags only, not PRs)
- [x] T011 Configure image tagging (latest, SHA, version from tags)

**Checkpoint**: Workflow file is syntactically valid and covers all required steps

---

## Phase 4: Polish & Validation

**Purpose**: Final validation and documentation

- [x] T012 Update `.gitignore` if needed for any new artifacts
- [x] T013 Verify Docker image runs correctly with all features (HTTP 200 confirmed)
- [x] T014 Verify image size is under 50 MB target (25.1 MB achieved)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1**: No dependencies — start immediately
- **Phase 2**: Depends on Phase 1 (Dockerfile must exist)
- **Phase 3**: Depends on Phase 1 (references Dockerfile)
- **Phase 4**: Depends on all previous phases

### Parallel Opportunities

```text
After Phase 1 completes:
├── Phase 2 (Makefile targets)
└── Phase 3 (GitHub Actions workflow) [can parallelize]
```

Within Phase 3, T007-T009 can run in parallel (different sections of the same file, but logically independent).

---

## Notes

- Constitution Principle VI: Run `make test` before each commit; fix failures before proceeding.
- No new application tests are needed — this feature adds build/deployment infrastructure only.
- The Go version in the Dockerfile should match `go.mod` (currently Go 1.25).
- Commit after each phase when tests pass.
