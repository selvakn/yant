# Feature Specification: Docker Packaging & CI/CD Pipeline

**Feature Branch**: `008-docker-ci-setup`  
**Created**: 2026-04-06  
**Status**: Draft  
**Input**: User description: "Add docker packaging setup. It should be easy to use and make scripts for building. Also add github workflows for building and publishing the images. Use github container registry. in the github actions, setup security scanners for the code and dependencies. Follow best practices for building and maintaining modern applications."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Build and Run Application via Docker (Priority: P1)

A developer clones the repository and wants to run the application locally without installing Go, Node.js, or any other build tooling. They use a single Docker command to build the image and run the application with persistent data.

**Why this priority**: This is the foundational capability — containerizing the application enables all subsequent CI/CD workflows and simplifies local development for contributors.

**Independent Test**: Can be fully tested by running `docker build` and `docker run` on a fresh clone, then accessing the application in a browser and creating/editing notes.

**Acceptance Scenarios**:

1. **Given** a fresh clone of the repository, **When** the developer runs the Docker build command, **Then** the application image is built successfully including the Go backend and frontend-build assets.
2. **Given** a built Docker image, **When** the developer runs the container with mounted volumes for notes and uploads, **Then** the application starts, serves on the configured port, and data persists across container restarts.
3. **Given** a running container, **When** the developer creates notes, uploads images, and adds drawings, **Then** all features work identically to a non-containerized deployment.

---

### User Story 2 - Automated Image Build and Publish on Push (Priority: P1)

When a developer pushes code to the main branch or creates a release tag, the CI pipeline automatically builds the Docker image, runs security scans, and publishes the image to GitHub Container Registry (GHCR).

**Why this priority**: Automated builds and publishing ensure every release is consistently built, scanned, and available for deployment without manual intervention.

**Independent Test**: Can be tested by pushing a commit to the main branch and verifying the image appears in GHCR with the correct tags.

**Acceptance Scenarios**:

1. **Given** a push to the main branch, **When** the GitHub Actions workflow triggers, **Then** the Docker image is built, scanned, and published to `ghcr.io/<owner>/yant` with a `latest` tag and the git SHA tag.
2. **Given** a tagged release (e.g., `v1.0.0`), **When** the workflow triggers, **Then** the image is published with the release version tag (e.g., `1.0.0`) in addition to `latest`.
3. **Given** a pull request, **When** the workflow triggers, **Then** the image is built and scanned but NOT published to the registry.

---

### User Story 3 - Security Scanning in CI (Priority: P1)

The CI pipeline automatically scans the codebase, dependencies, and built container image for known vulnerabilities on every push and pull request, preventing insecure code from reaching production.

**Why this priority**: Security scanning is a critical best practice that should gate every build, catching vulnerabilities before they are deployed.

**Independent Test**: Can be tested by introducing a known vulnerable dependency and verifying the scan detects and reports it.

**Acceptance Scenarios**:

1. **Given** a push or pull request, **When** the CI workflow runs, **Then** Go dependencies are scanned for known CVEs using `govulncheck` or equivalent.
2. **Given** a built Docker image, **When** the CI workflow runs, **Then** the container image is scanned for OS-level and library vulnerabilities (e.g., via Trivy).
3. **Given** the security scan finds critical or high-severity vulnerabilities, **When** the scan results are generated, **Then** they are reported as GitHub Security findings visible in the Security tab.

---

### User Story 4 - Convenient Build Scripts (Priority: P2)

A developer has access to simple Make targets or scripts to build, tag, and run Docker images locally without memorizing Docker CLI flags.

**Why this priority**: Convenience scripts reduce friction and ensure consistent build practices, but are not blocking for core functionality.

**Independent Test**: Can be tested by running each Make target and verifying the expected output.

**Acceptance Scenarios**:

1. **Given** the Makefile exists, **When** the developer runs the Docker build target, **Then** the image is built with a sensible default tag.
2. **Given** a built image, **When** the developer runs the Docker run target, **Then** the container starts with proper port mapping and volume mounts for data persistence.

---

### Edge Cases

- What happens when the SQLite database file does not exist at container startup? The application should create it automatically.
- How does the container handle graceful shutdown (SIGTERM)? The Go server should stop accepting new connections and drain existing ones.
- What happens when the container runs as a non-root user and the mounted volume has root ownership? File permissions should be documented.
- How are the frontend-build Node.js dependencies handled during Docker build? They should be built in a separate stage and only the output artifacts copied.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The project MUST include a Dockerfile that produces a minimal, production-ready container image.
- **FR-002**: The Docker build MUST use multi-stage builds — separate stages for building the frontend (Node.js), building the backend (Go), and the final runtime image.
- **FR-003**: The final runtime image MUST NOT contain build tools (Go, Node.js, npm) or source code — only the compiled binary, frontend assets, and templates.
- **FR-004**: The container MUST run as a non-root user for security.
- **FR-005**: The Dockerfile MUST support configuring the listen port, database path, notes directory, and uploads directory via environment variables or command-line flags.
- **FR-006**: The Makefile MUST include targets for building and running the Docker image locally.
- **FR-007**: A GitHub Actions workflow MUST build the Docker image on pushes to the main branch and on pull requests.
- **FR-008**: The workflow MUST publish the built image to GitHub Container Registry (`ghcr.io`) on pushes to main and on tagged releases.
- **FR-009**: The workflow MUST NOT publish images from pull request builds.
- **FR-010**: Images MUST be tagged with the git SHA, `latest` (for main branch pushes), and the version number (for tagged releases).
- **FR-011**: The CI pipeline MUST run Go dependency vulnerability scanning on every push and pull request.
- **FR-012**: The CI pipeline MUST run container image vulnerability scanning on the built image.
- **FR-013**: Security scan results MUST be uploaded to GitHub Security (SARIF format) for visibility in the repository's Security tab.
- **FR-014**: The CI pipeline MUST run the existing test suite (`make test`) as a prerequisite before building the image.
- **FR-015**: The CI pipeline MUST run Go static analysis (`go vet`) on every push and pull request.
- **FR-016**: A `.dockerignore` file MUST be present to exclude unnecessary files from the build context.

### Key Entities

- **Container Image**: The packaged application artifact containing the compiled binary, static assets, and templates. Tagged with version identifiers.
- **Build Pipeline**: The automated GitHub Actions workflow that builds, tests, scans, and optionally publishes the container image.
- **Security Report**: Scan results in SARIF format uploaded to GitHub's security findings.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A developer can build and run the application from a fresh clone using only Docker in under 5 minutes (excluding image download time).
- **SC-002**: The final container image size is under 50 MB (excluding mounted data volumes).
- **SC-003**: The CI pipeline completes (build + test + scan + publish) in under 10 minutes.
- **SC-004**: Every push to main and every pull request triggers automated security scanning with results visible in GitHub.
- **SC-005**: Published images on GHCR are correctly tagged and pullable by any authorized user.
- **SC-006**: The application runs correctly inside the container with all features functional (notes, tags, search, drawings, images, archive).

## Assumptions

- The repository is hosted on GitHub and GitHub Actions is available.
- The GitHub Container Registry (ghcr.io) is enabled for the repository owner/organization.
- The `GITHUB_TOKEN` provided by GitHub Actions has sufficient permissions to push to GHCR (default for public repos, needs package write permission for private repos).
- The Go version used in the Docker build matches the project's `go.mod` requirement (Go 1.25+).
- The Node.js version used for building the frontend is a current LTS release (22.x).
- Pure Go SQLite driver (`modernc.org/sqlite`) is used, so no CGO or C compiler is needed in the build or runtime stage.
- The existing `make test` and `make lint` targets are the authoritative test/lint commands.
- Security scanning tools (govulncheck, Trivy) are freely available for open-source use.
