# Research: Multiple Drawings Per Note

## Drawing ID Generation

**Decision**: Use `crypto/rand` to generate 8-character alphanumeric tokens (a-z, 0-9) for drawing IDs.

**Rationale**: 8 chars from a 36-char alphabet gives ~2.8 trillion combinations — collision-free at the per-note scale. Go's `crypto/rand` is secure and stdlib-only. The IDs are safe for file names, URL paths, and markdown syntax markers.

**Alternatives considered**:
- UUID v4: too long for syntax markers and file names; 36 chars with hyphens is unwieldy in `![[draw:...]]`.
- Incrementing integers: not collision-safe if drawings are deleted and recreated; also reveals ordering information unnecessarily.
- Slug from display name: rejected because display name is mutable, and we want a stable ID.

## Drawing Marker Syntax

**Decision**: Use `![[draw:<id>]]` as the marker syntax. The goldmark rendering pipeline will be extended with a custom inline parser that matches this pattern and replaces it with a `<div>` placeholder for client-side hydration.

**Rationale**: The `![[...]]` syntax is consistent with wiki-link style (`[[...]]`) already used in the project for note linking. The `draw:` prefix avoids ambiguity. Goldmark supports custom parsers/renderers without forking.

**Alternatives considered**:
- HTML comment markers (`<!-- draw:id -->`): invisible in editors, hard to move/edit.
- Fenced code block (` ```draw:id` `): semantically wrong, interferes with code rendering.
- Custom shortcode (`{{draw id}}`): conflicts with Go template syntax.

## Goldmark Extension Approach

**Decision**: Create a custom goldmark inline parser that matches `![[draw:<id>]]` and renders it as `<div class="drawing-embed" data-drawing-id="<id>"></div>`. Client-side JS then hydrates these placeholders.

**Rationale**: Inline parser is the lightest-weight extension. The div placeholder is simple to hydrate, and the `data-drawing-id` attribute is easy to query from JS. This follows the same pattern as mermaid diagram rendering in the project.

**Alternatives considered**:
- Post-process the HTML string with regex: fragile, error-prone with nested elements.
- AST transformer: heavier than needed for a simple inline replacement.

## File Naming Convention

**Decision**: Multi-drawing files use `<slug>--<drawingID>.<tool>.json` (e.g., `my-note--abc12345.excalidraw.json`). The `--` double-dash separator distinguishes from slugs that use single dashes.

**Rationale**: Double-dash is uncommon in note slugs (enforced by the slug generation), making it a safe separator. File names remain human-readable and glob-friendly.

**Alternatives considered**:
- Subdirectory per note (`<slug>/drawings/<id>.json`): changes storage layout significantly, breaks existing glob patterns.
- Underscore separator (`<slug>_<id>.json`): underscore already appears in slugs, making parsing ambiguous.

## Legacy Detection

**Decision**: At read time, `DetectDrawings()` checks for both legacy files (`<slug>.tldraw.json`, `<slug>.excalidraw.json`) and new-format files (`<slug>--*.json`). Legacy files are served transparently. On first edit or when adding a second drawing, the legacy file is renamed to `<slug>--<newID>.<tool>.json`, a `note_drawings` row is inserted with display name "Drawing 1", and a `![[draw:<newID>]]` marker is appended to the markdown.

**Rationale**: Lazy migration avoids a risky bulk rename on deployment. Users see no change until they interact with drawings. The rename happens atomically within the save transaction.

## SQLite Table Design

**Decision**: New `note_drawings` table:

```sql
CREATE TABLE IF NOT EXISTS note_drawings (
    drawing_id  TEXT    NOT NULL,
    note_id     INTEGER NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    display_name TEXT   NOT NULL,
    tool_type   TEXT    NOT NULL CHECK (tool_type IN ('tldraw','excalidraw')),
    created_at  TEXT    NOT NULL,
    updated_at  TEXT    NOT NULL,
    PRIMARY KEY (drawing_id, note_id)
);
CREATE INDEX IF NOT EXISTS idx_note_drawings_note ON note_drawings(note_id);
```

**Rationale**: Composite PK on (drawing_id, note_id) ensures uniqueness per note. ON DELETE CASCADE handles note deletion. Consistent with existing table patterns (text dates, foreign keys).

## Rebuild-DB Support

**Decision**: The `--rebuild-db` command scans for `<slug>--<id>.<tool>.json` files and legacy `<slug>.<tool>.json` files, inserting rows into `note_drawings`. For legacy files, the drawing_id is generated and the display name defaults to "Drawing 1". For new-format files, the ID is extracted from the filename.

**Rationale**: Maintains the constitution's principle that SQLite is a rebuildable cache.
