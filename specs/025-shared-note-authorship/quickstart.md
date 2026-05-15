# Quickstart: Shared Note Authorship & Indicators

## Prerequisites

- Two user accounts in the app (e.g., `alice` and `bob`)
- Note sharing (feature 016) deployed and working

## Testing version history authorship

1. Log in as `alice`.
2. Create a note; make and save an edit.
3. In another browser/incognito, log in as `bob`.
4. Log in as `alice`; share the note with `bob` (edit permission).
5. Log in as `bob`; open the shared note and make an edit.
6. Log in as `alice`; open the note's History page (`/notes/{slug}/history`).
   - Alice's edit should show author: `alice`
   - Bob's edit should show author: `bob`
   - The initial creation commit shows author: `alice`
7. As `bob`, visit `/shared/alice/{slug}/history`.
   - Same history list with authors should appear.

## Testing "last updated by" on the reader page

1. Log in as `bob`; edit a shared note.
2. Log in as `alice`; open the note reader.
   - The topbar should show: `Jan 2, 2026 at 3:04 PM · by bob`

## Testing share indicators on the notes list

1. Log in as `alice`; share a note with `bob`.
2. Open `/notes` as `alice`.
   - The shared note should show an outgoing badge: e.g., `↑ Shared with 1`
3. As `alice`, revoke `bob`'s access.
4. Refresh `/notes`.
   - The outgoing badge should be gone.

## Testing incoming share indicators on the shared list

1. Log in as `bob`; open `/shared`.
   - Notes shared with `bob` should show "Shared by alice" styled as a badge.

## Running tests

```bash
make test          # unit + integration tests
make coverage      # verify ≥90% coverage on internal/...
```
