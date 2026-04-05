<!--
  Sync Impact Report
  ==================
  Version change: 0.0.0 (template) → 1.0.0
  Modified principles: N/A (initial population)
  Added sections:
    - Core Principles (5): Markdown-First Storage, Simplicity,
      Monorepo Structure, Integration Testing, Simple Web Interface
    - Technology Constraints
    - Development Workflow
    - Governance
  Removed sections: None
  Templates requiring updates:
    ✅ plan-template.md — no changes needed, structure aligns
    ✅ spec-template.md — no changes needed, acceptance scenarios cover testing
    ✅ tasks-template.md — no changes needed, integration test phases present
    ✅ No command files exist to update
  Follow-up TODOs: None
-->

# My Notes Constitution

## Core Principles

### I. Markdown-First Storage

- All user notes MUST be stored as plain Markdown files on the
  filesystem.
- SQLite MAY be used only when a filesystem-based approach is
  insufficient (e.g., full-text search indexing, metadata caching).
- If SQLite is introduced, the Markdown files MUST remain the
  source of truth; the database is a derived, rebuildable cache.
- Notes MUST be portable: a user can copy the Markdown folder to
  another system and retain full content without the application.

### II. Simplicity

- YAGNI: features MUST NOT be added until they are actually needed.
- Minimize external dependencies; prefer the standard library and
  well-established, minimal packages.
- Start with the simplest viable implementation; refactor only when
  complexity is justified by a concrete requirement.
- No premature abstractions: three similar lines of code are
  preferable to a speculative helper.

### III. Monorepo Structure

- Frontend and backend code MUST reside in the same repository.
- The repository MUST use a clear directory separation (e.g.,
  `backend/` and `frontend/`) while sharing a single version
  control history.
- Shared types, constants, or contracts between frontend and
  backend MUST live in an explicit shared location rather than
  being duplicated.

### IV. Integration Testing (NON-NEGOTIABLE)

- Every backend endpoint and service MUST have integration tests
  that exercise the real stack (filesystem, SQLite if present,
  HTTP layer).
- Backend test coverage MUST be at or above 90% as measured by
  line coverage.
- Mocks MAY be used only for external third-party services that
  cannot be run locally; internal components MUST be tested with
  real implementations.
- Coverage MUST be measured and reported on every test run;
  a run that drops below 90% MUST fail the test suite.

### V. Simple Web Interface

- The frontend MUST be a lightweight web interface prioritizing
  content readability and fast page loads.
- Avoid heavy JavaScript frameworks when a simpler approach
  suffices; progressive enhancement is preferred.
- The interface MUST render Markdown notes faithfully, preserving
  formatting, links, and code blocks.

## Technology Constraints

- **Storage**: Markdown files as primary store; SQLite permitted
  only as a derived index/cache (see Principle I).
- **Backend**: Technology choice is open but MUST support serving
  a web interface and exposing a REST or equivalent API.
- **Frontend**: MUST be served from the same repository and
  deployable as a single unit with the backend.
- **Database migrations**: If SQLite is introduced, migrations
  MUST be versioned and reproducible. The database MUST be
  rebuildable from the Markdown source files at any time.

## Development Workflow

- **Branch strategy**: One feature branch per change; merge to
  main via pull request.
- **Testing gate**: All backend integration tests MUST pass with
  ≥90% coverage before a branch is eligible for merge.
- **Code review**: Every change MUST be reviewed before merging.
  Reviews MUST verify compliance with this constitution.
- **Commit discipline**: Atomic commits with clear messages;
  each commit SHOULD leave the project in a working state.

## Governance

- This constitution is the highest-authority document for the
  My Notes project. In case of conflict with other documentation,
  the constitution prevails.
- **Amendment procedure**: Any principle change MUST be documented
  with a rationale, reviewed, and merged as a dedicated commit.
  The Sync Impact Report (HTML comment at the top of this file)
  MUST be updated with every amendment.
- **Versioning policy**: The constitution follows semantic
  versioning — MAJOR for principle removals or incompatible
  redefinitions, MINOR for new principles or material expansions,
  PATCH for clarifications and wording fixes.
- **Compliance review**: At the start of every feature plan
  (`/speckit.plan`), the Constitution Check section MUST verify
  alignment with all active principles. Violations MUST be
  justified in the Complexity Tracking table or resolved before
  implementation begins.

**Version**: 1.0.0 | **Ratified**: 2026-04-05 | **Last Amended**: 2026-04-05
