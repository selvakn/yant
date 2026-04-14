# Research: Note Version Control

**Feature**: 014-note-version-control
**Date**: 2026-04-14

## Decision 1: Git Integration Approach

**Decision**: Shell out to the `git` binary via `os/exec` from the Go backend.

**Rationale**:
- Constitution Principle II (Simplicity) favors zero new dependencies when the standard library suffices.
- The operations required (init, add, commit, log, show, diff) are simple, well-documented CLI commands.
- `os/exec` is stdlib — no new module dependencies.
- Git binary is universally available on Linux, macOS, and Docker images (easily added to Dockerfile).
- The alternative (`go-git/go-git/v5`) is a large pure-Go library with many transitive dependencies, adding build complexity disproportionate to the feature scope.

**Alternatives considered**:
- `github.com/go-git/go-git/v5` (Apache 2.0, pure Go): Full programmatic git. Rejected because it adds ~30+ transitive dependencies for operations easily done with 6 CLI commands. The project already avoids CGO but does not avoid external binaries (git is already used for the project's own VCS).
- `github.com/libgit2/git2go` (LGPL-2.1 + CGO): Rejected — restrictive license and requires CGO, which the project avoids.

## Decision 2: Repository Structure

**Decision**: Single git repository initialized at the notes directory root (`{notesDir}/`), covering all users.

**Rationale**:
- All note files already live under a shared root with per-user subdirectories (`{notesDir}/{userID}/`).
- A single repo is simpler to initialize, manage, and back up.
- History queries are scoped to specific file paths, so cross-user data exposure is not a concern at the git layer — the application layer already enforces user isolation via session-based auth.

**Alternatives considered**:
- One repo per user (`{notesDir}/{userID}/.git`): Better isolation but adds complexity for init, cleanup, and makes the notes directory structure dependent on git. Rejected per Principle II.

## Decision 3: Commit Strategy

**Decision**: One commit per save operation (note create, update, delete, drawing save). Commit message encodes the action and note slug for human readability and machine parseability.

**Rationale**:
- Atomic commits per operation give the clearest history per file.
- `git log --follow -- {filepath}` naturally returns the history for a single note, including across renames.
- Commit messages like `create: {slug}`, `update: {slug}`, `delete: {slug}`, `revert: {slug} to {shortHash}` enable filtering and display in the UI.

**Alternatives considered**:
- Batch commits (periodic snapshots): Loses per-save granularity. Rejected — defeats the purpose of version control.
- Squash on revert: Would lose intermediate history. Rejected — spec requires non-destructive reverts (FR-008).

## Decision 4: Diff Generation

**Decision**: Use `git diff {commit1} {commit2} -- {filepath}` output, parsed into a structured format for template rendering.

**Rationale**:
- Git's unified diff format is the standard for source-level diffs.
- Parsing unified diff is straightforward: lines starting with `+` (additions), `-` (deletions), and space (context).
- No external diff library needed — git handles the computation.

**Alternatives considered**:
- Go-native diff libraries (`github.com/sergi/go-diff`): Adds a dependency for something git already does. Rejected per Principle II.

## Decision 5: Rename Tracking

**Decision**: Use `git log --follow` to track history across file renames. When a note slug changes (file rename), git detects the rename heuristically.

**Rationale**:
- `--follow` is the standard git mechanism for tracking file renames.
- The existing `noteUpdate` handler already handles slug changes (update DB slug + rename file). Adding a git `mv` or sequential `rm`/`add` in the same commit preserves the rename chain.

## Decision 6: Drawing Diff Strategy

**Decision**: For drawing changes, retrieve the tldraw JSON at each version and render two read-only tldraw canvases side by side in the browser.

**Rationale**:
- Raw JSON diff of tldraw data is not meaningful to users.
- Tldraw is already vendored in the frontend and supports read-only rendering.
- The backend serves drawing JSON at a specific commit via `git show {commit}:{drawingPath}`; the frontend loads two instances.

## Decision 7: Seeding Existing Notes

**Decision**: On server startup, if the notes directory is not a git repo, initialize it and commit all existing files as an initial version.

**Rationale**:
- Ensures every note has at least one version entry in history from day one (FR-016).
- One-time operation; subsequent starts skip if `.git` already exists.
- Commit message: `seed: initial version of all notes`.

## Decision 8: Docker Image Changes

**Decision**: Add `git` package to the Dockerfile runtime stage.

**Rationale**:
- The runtime image needs the git binary for version control operations.
- `git` is a small package (~10MB) with no significant image size impact.
- Already present in build stages; only needs adding to the final runtime stage.
