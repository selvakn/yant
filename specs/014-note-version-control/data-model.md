# Data Model: Note Version Control

**Feature**: 014-note-version-control
**Date**: 2026-04-14

## Overview

This feature introduces **no new database tables**. Git is the storage and query engine for version history. The data model describes the domain entities as they flow through the application.

## Entities

### Version

Represents a single point-in-time snapshot of a note file, derived from a git commit.

| Field | Type | Source | Description |
| ----- | ---- | ------ | ----------- |
| CommitHash | string | `git log` `%H` | Full SHA-1 hash — the unique version identifier |
| ShortHash | string | `git log` `%h` | Abbreviated hash for display |
| Timestamp | time.Time | `git log` `%aI` | Author date in ISO 8601 |
| Message | string | `git log` `%s` | Commit subject line (e.g., `update: my-note`) |
| Insertions | int | `git log` `--numstat` | Lines added in this commit for this file |
| Deletions | int | `git log` `--numstat` | Lines removed in this commit for this file |

**Identity**: `CommitHash` is globally unique within the repository.

**Ordering**: Versions are ordered by `Timestamp` descending (newest first) for display.

### DiffLine

Represents a single line in a unified diff output, parsed for template rendering.

| Field | Type | Description |
| ----- | ---- | ----------- |
| Type | string | One of: `add`, `remove`, `context`, `header` |
| Content | string | The line text (without the leading `+`/`-`/` ` prefix) |
| OldLineNo | int | Line number in the old version (0 if addition) |
| NewLineNo | int | Line number in the new version (0 if deletion) |

### DiffResult

Container for a complete diff between two versions.

| Field | Type | Description |
| ----- | ---- | ----------- |
| OldCommit | string | Commit hash of the older version |
| NewCommit | string | Commit hash of the newer version |
| OldDate | time.Time | Timestamp of the older version |
| NewDate | time.Time | Timestamp of the newer version |
| Lines | []DiffLine | Parsed diff lines |
| HasDrawingChange | bool | Whether the tldraw file changed between these versions |

## Relationships

```text
Note (1) ──── (*) Version
  │                 │
  │                 ├── CommitHash identifies content at that point
  │                 └── git show {hash}:{path} retrieves content
  │
  └── slug + userID maps to filesystem path
        └── {notesDir}/{userID}/{slug}.md
        └── {notesDir}/{userID}/{slug}.tldraw.json (optional)
```

## State Transitions

```text
[No Repo] ──git init + seed──▸ [Initialized]

Per note lifecycle:
  create  ──commit──▸  Version 1
  update  ──commit──▸  Version N   (only if content changed)
  rename  ──commit──▸  Version N+1 (git tracks via --follow)
  revert  ──commit──▸  Version N+1 (new commit with old content)
  delete  ──commit──▸  Final commit (file removed)
  archive ──(no commit)──        (archive is metadata-only in SQLite)
  restore ──(no commit)──        (restore is metadata-only in SQLite)
```

## Validation Rules

- CommitHash must be a valid 40-character hex string when used in URLs/queries.
- ShortHash is display-only; all operations use full CommitHash.
- Timestamp must parse as valid ISO 8601.
- Insertions and Deletions are non-negative integers.
- DiffLine.Type must be one of the four allowed values.

## No SQLite Changes

Git stores all version data. No schema migrations, no new tables, no index changes. The existing `notes` table (slug, title, timestamps, archived flag) is unchanged. This is consistent with Constitution Principle I: markdown files remain the source of truth, and git tracks their evolution without introducing a derived cache.
