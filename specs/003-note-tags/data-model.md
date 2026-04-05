# Data model: Seamless note tags (delta)

No schema migration. Existing tables:

- `notes` — unchanged
- `note_tags(note_id, tag_name)` — `tag_name` remains lowercase text; values now may include hyphens when present in parsed tokens

**Source of truth**: Markdown file body. **Derived**: `note_tags` rows after `SyncTags` on create/update.

**Client/server agreement**: Hashtag token grammar MUST match `ParseTags` in `backend/internal/models/models.go` (documented in code comment; mirrored in editor script).
