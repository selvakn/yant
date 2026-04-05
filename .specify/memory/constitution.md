<!--
  Sync Impact Report
  ==================
  Version change: 1.0.0 → 1.1.0
  Modified principles: None renamed
  Added sections:
    - VI. Commit & Test Discipline (NON-NEGOTIABLE)
  Removed sections: None
  Development Workflow: materially expanded (commit cadence, test-before-commit,
    fix-failures-first)
  Templates requiring updates:
    ✅ .specify/templates/plan-template.md — Constitution Check gates expanded
    ✅ .specify/templates/tasks-template.md — constitution alignment note + Notes
    ✅ .specify/templates/spec-template.md — no change (scope unchanged)
    ✅ .cursor/commands/speckit.constitution.md — command paths note for sync
    ✅ .cursor/commands/speckit.implement.md — pre-commit test gate bullets
    ✅ CLAUDE.md — manual addition for Principle VI
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

### VI. Commit & Test Discipline (NON-NEGOTIABLE)

- Work MUST be committed frequently in small, incremental steps so
  that history remains reviewable and reversible.
- Before every commit, the full project test suite MUST pass (same
  command and scope the project uses for CI or release gates).
- Commits MUST NOT be created while any test is failing.
- If tests fail, the author MUST fix the failing tests or the code
  under test before starting new work, additional features, or
  further commits. Skipping or deferring a failing test to “the next
  step” is NOT permitted.
- **Rationale**: Frequent green commits preserve a known-good baseline,
  reduce merge and debug cost, and keep coverage and integration
  guarantees from Principle IV enforceable in daily work.

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
- **Commit and test gate (Principle VI)**: During implementation,
  run the full test suite before each commit; do not commit on
  red; fix failures before proceeding.
- **Code review**: Every change MUST be reviewed before merging.
  Reviews MUST verify compliance with this constitution.
- **Commit discipline**: Commits MUST be frequent and scoped;
  each commit MUST leave the project in a passing state (tests
  green per Principle VI).

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

**Version**: 1.1.0 | **Ratified**: 2026-04-05 | **Last Amended**: 2026-04-05
