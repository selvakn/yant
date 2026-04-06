# Quickstart: Docker Packaging & CI/CD Pipeline

**Feature**: 008-docker-ci-setup

## Local Docker Usage

### Build the image

```bash
make docker-build
```

### Run the container

```bash
make docker-run
```

This starts the application on `http://localhost:8080` with data persisted in a `yant-data` Docker volume.

### Custom configuration

```bash
docker run -p 9090:8080 \
  -v yant-data:/data \
  -e PORT=8080 \
  ghcr.io/<owner>/yant:latest
```

## CI/CD Pipeline

### Trigger conditions

| Event | Tests | Build | Scan | Publish |
|-------|-------|-------|------|---------|
| Push to `main` | Yes | Yes | Yes | Yes (`latest` + SHA) |
| Tagged release `v*` | Yes | Yes | Yes | Yes (version + `latest` + SHA) |
| Pull request | Yes | Yes | Yes | No |

### Image tags

- `ghcr.io/<owner>/yant:latest` — latest main branch build
- `ghcr.io/<owner>/yant:<sha>` — specific commit (7-char SHA)
- `ghcr.io/<owner>/yant:<version>` — tagged release (e.g., `1.0.0`)

### Security scans

- **govulncheck**: Scans Go dependencies for known CVEs
- **Trivy**: Scans the built container image for OS and library vulnerabilities
- Results appear in the repository's **Security → Code scanning alerts** tab

## Verification Steps

1. **Docker build works**: `make docker-build` completes without errors
2. **Container runs**: `make docker-run` → visit `http://localhost:8080` → login → create a note
3. **Data persists**: Stop the container, run again, verify notes still exist
4. **CI workflow**: Push to a branch → check GitHub Actions → verify test/build/scan jobs run
5. **GHCR publish**: Push to main → verify image appears at `ghcr.io/<owner>/yant`
6. **Security tab**: After CI runs → check Security → Code scanning alerts for scan results
