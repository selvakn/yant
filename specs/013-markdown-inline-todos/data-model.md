# Data Model: Inline Markdown Todos

## Entities

### Todo Item (derived from markdown)

A parsed representation of a `- [ ]` or `- [x]` line in a note's markdown body.

| Field      | Type        | Description                                              |
| ---------- | ----------- | -------------------------------------------------------- |
| note_id    | INTEGER     | FK → notes(id), CASCADE on delete                        |
| line       | INTEGER     | 1-based line number in the markdown file                 |
| text       | TEXT        | Task text (without `- [ ]` prefix and `@due(...)` suffix)|
| due_date   | TEXT (date) | ISO 8601 date from `@due(YYYY-MM-DD)`, nullable          |
| completed  | BOOLEAN     | false for `- [ ]`, true for `- [x]`                      |

**Primary Key**: (note_id, line)  
**Indexes**: (completed, due_date) for efficient pending-todos queries

### Schema DDL

```sql
CREATE TABLE IF NOT EXISTS note_todos (
    note_id   INTEGER NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    line      INTEGER NOT NULL,
    text      TEXT    NOT NULL,
    due_date  TEXT,
    completed BOOLEAN NOT NULL DEFAULT 0,
    PRIMARY KEY (note_id, line)
);
CREATE INDEX IF NOT EXISTS idx_note_todos_pending ON note_todos (completed, due_date);
```

## Relationships

```
notes 1──* note_todos   (a note has zero or more todo items)
notes 1──* note_tags    (a note has zero or more tags — existing)
```

The todos view joins these: `note_todos` → `notes` → `note_tags` to display each todo with its parent note's tags.

## State Transitions

```
Pending [ ] ──click──→ Complete [x]
Complete [x] ──click──→ Pending [ ]    (in reader view)
Complete [x] ──manual edit──→ Pending [ ]  (in editor)
```

Toggle is bidirectional in reader view. In the aggregated todos view, only pending→complete is shown (completed items are hidden from the view).

## Sync Lifecycle

```
User saves note (editor auto-save or explicit save)
  → ParseTodos(markdown body) extracts []TodoItem
  → SyncTodos(db, noteID, []TodoItem):
      DELETE FROM note_todos WHERE note_id = ?
      INSERT INTO note_todos VALUES (?, ?, ?, ?, ?) for each item

User toggles checkbox (reader or todos view)
  → Read markdown from disk
  → Toggle - [ ] ↔ - [x] on the specified line
  → Write markdown back to disk
  → Re-run ParseTodos + SyncTodos
```

## Parsed Todo Struct

```
TodoItem:
  Line      int       // 1-based line number
  Text      string    // task text without markers
  DueDate   *string   // "2026-04-20" or nil
  Completed bool      // true if [x]
```

## Aggregated View Query

```
SELECT t.line, t.text, t.due_date, t.completed,
       n.slug, n.title
FROM note_todos t
JOIN notes n ON n.id = t.note_id
WHERE n.user_id = ? AND n.archived = 0 AND t.completed = 0
ORDER BY
  CASE WHEN t.due_date IS NULL THEN 1 ELSE 0 END,
  t.due_date ASC,
  n.title ASC
```

Tags are loaded separately per note (or via a subquery/join on `note_tags`) for display in the view.
