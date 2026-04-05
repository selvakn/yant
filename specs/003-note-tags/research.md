# Research: Seamless note tags

## Decision: Keep tags in Markdown as `#tokens`

**Rationale**: Matches existing architecture (`ParseTags`, `SyncTags`, portable files). UI only mirrors and edits that representation.

## Decision: Chip strip + quick-add input

**Alternatives considered**: (1) YAML front matter only — rejected (harder for casual typing). (2) Separate tag API without body — rejected (violates Markdown-first). (3) Inline-only `#` — kept as optional; chips add discoverability and one-click remove.

## Decision: Debounced refresh from editor content

**Rationale**: Avoids lag on every keystroke; 150ms debounce is a common balance.

## Decision: Extend regex to allow `-` inside tags

**Rationale**: Spec assumes hyphenated words; `#my-tag` should be one tag, not `my` and `tag`.

## Suggestions: `GET /tags` JSON

**Rationale**: Already implemented (`TagsListGET` without `HX-Request`). Client populates `<datalist>` or filters for autocomplete.
