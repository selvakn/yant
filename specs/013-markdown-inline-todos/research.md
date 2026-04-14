# Research: Inline Markdown Todos

## Decision 1: Goldmark TaskList Extension

**Decision**: Use `github.com/yuin/goldmark/extension.TaskList` — goldmark's built-in task list extension.

**Rationale**: It's part of the goldmark module already in `go.mod` (no new dependency). It parses GFM-style `- [ ]`/`- [x]` syntax and renders `<input type="checkbox">` elements. The extension outputs `disabled` checkboxes by default; a custom renderer wrapper removes `disabled` and adds data attributes for the toggle endpoint.

**Alternatives considered**:
- Custom regex-based HTML post-processing: Fragile, doesn't handle nested lists or edge cases. Rejected.
- Client-side JS parsing: Would require duplicating parsing logic in JS. Rejected — server-side rendering is simpler and consistent with the existing goldmark pipeline.

## Decision 2: @due Annotation Parsing

**Decision**: Parse `@due(YYYY-MM-DD)` using a regex applied during todo extraction, and render it as a `<span class="todo-due" data-date="YYYY-MM-DD">Apr 20, 2026</span>` badge in the HTML output.

**Rationale**: The `@due()` syntax is a convention applied after goldmark rendering via a simple HTML post-processing step (regex replace on the rendered HTML string). This avoids writing a custom goldmark AST node, keeping complexity minimal. The same regex is used during todo parsing for the DB sync (extracting due dates for sorting).

**Alternatives considered**:
- Custom goldmark inline parser for `@due()`: More correct AST-wise, but significantly more complex to implement. The simple post-processing approach is sufficient for a single annotation type. Rejected for now — can upgrade later if more annotations are needed.
- Store due date separately from markdown: Violates markdown-first principle. Rejected.

## Decision 3: Todo Storage Strategy

**Decision**: Derived `note_todos` SQLite table, rebuilt from markdown on every note save. Same pattern as `note_tags` and `note_links`.

**Rationale**: Follows the established pattern in the codebase. The `note_tags` table is synced from markdown via `ParseTags()` → `SyncTags()` on every save. Todos will follow the same flow: `ParseTodos()` → `SyncTodos()`. This keeps markdown as source of truth while enabling efficient SQL queries for the aggregated view.

**Alternatives considered**:
- Parse todos on-the-fly from markdown files for every request: Too slow for aggregation across all notes. Rejected.
- Separate todo file per note: Violates markdown-first principle and adds file management complexity. Rejected.

## Decision 4: Checkbox Toggle Mechanism

**Decision**: `PUT /notes/{slug}/todo` with JSON body `{"line": N, "checked": true/false}`. Handler reads markdown from disk, finds the todo line by line number, toggles `[ ]`↔`[x]`, writes back, and re-syncs the `note_todos` table.

**Rationale**: Line-number-based identification is simple and reliable. The spec clarified that concurrent edit conflicts use "last write wins" — no locking needed. This is consistent with how the existing auto-save works (POST with full body, last write wins).

**Alternatives considered**:
- Content-hash-based identification (find line by matching text): Fragile when identical todo text exists. The spec explicitly says identical todos are distinguished by line position. Rejected.
- Full-body save (like the editor): Heavier payload for a single checkbox toggle. The line-based approach is more efficient for the common case. Rejected.

## Decision 5: Goldmark Instance Lifecycle

**Decision**: Create a configured goldmark instance once in the `Handler` constructor (`handlers.New()`) and reuse it for all rendering calls. Replace the current `goldmark.Convert()` call with `h.md.Convert()`.

**Rationale**: The current code creates a default goldmark instance per render via `goldmark.Convert()`. A single configured instance with the TaskList extension is more efficient and centralizes markdown configuration.

**Alternatives considered**:
- Global package-level goldmark instance: Works but makes testing harder (can't swap configuration). Rejected.
- Per-request instance: Wasteful. Rejected.
