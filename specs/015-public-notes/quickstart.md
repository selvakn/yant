# Quickstart: Public Note Sharing

## Prerequisites

- Go 1.25+, `make deps` completed
- On branch `015-public-notes`

## Build & Run

```bash
make build && make run
```

## Test

```bash
make test
make coverage
make integration-test
```

## Feature Usage

### Publishing a Note

1. Open any note in the reader view (`/notes/{slug}`)
2. Click the `⋯` menu
3. Click **Publish**
4. A share URL appears with a **Copy** button
5. Share the URL with anyone — they can read the note without signing in

### Unpublishing

1. Open the same note in reader view
2. Click the `⋯` menu
3. Click **Unpublish** — the URL stops working immediately

### Viewing All Public Notes

- Click **Public notes** in the sidebar (shows a count badge when you have published notes)
- URL: `/public`
- From this page you can copy any share URL or unpublish any note

### Visiting a Public Note

- Paste the share URL in any browser (no login required)
- URL format: `/p/{token}` (e.g., `/p/abc123xyz...`)
- The note renders read-only with no sidebar, nav, or edit controls

## Security Notes

- Anyone with the URL can read the note — treat share URLs like passwords
- Archiving or deleting a public note revokes access immediately
- Wiki-links in public notes that reference your private notes render as plain text (no leakage)
- Public notes are excluded from search engines (`noindex`)

## Key Files

| Purpose                       | Path                                       |
| ----------------------------- | ------------------------------------------ |
| Public share model + queries  | `backend/internal/models/public.go`        |
| Public handlers               | `backend/internal/handlers/public.go`      |
| Minimal public template       | `frontend/templates/public/note.html`      |
| Owner's public notes list     | `frontend/templates/public/list.html`      |
| Schema                        | `backend/internal/models/models.go`        |
| Routes                        | `backend/cmd/server/main.go`               |
