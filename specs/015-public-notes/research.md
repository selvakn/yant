# Research: Public Note Sharing

## Decision 1: Public URL Path Prefix

**Decision**: Use `/p/{token}` for public note URLs (not `/notes/{slug}` or `/share/{token}`).

**Rationale**: Short, memorable, clearly distinct from the private `/notes/{slug}` namespace. The `/p/` prefix signals "public" to anyone reading the URL. Using a separate path also makes route-level authorization obvious — anything under `/p/` is unauthenticated, anything under `/notes/` is authenticated.

**Alternatives considered**:
- `/notes/{slug}?token=...`: Mixes private and public paths, makes middleware logic harder to reason about. Rejected.
- `/share/{token}`: Longer, no real benefit over `/p/`. Rejected.
- Dedicated subdomain (`public.example.com`): Requires DNS/cert work, overkill for v1. Rejected.

## Decision 2: Token Format

**Decision**: 16-byte random from `crypto/rand`, encoded as base64url (~22 chars, no padding).

**Rationale**: 128 bits of entropy is the industry standard for capability URLs. Base64url is URL-safe without encoding. 22 characters is short enough to share comfortably but unguessable. This is the same approach used by Google Docs, Dropbox, and most link-sharing platforms.

**Alternatives considered**:
- UUID v4: Longer (36 chars with hyphens), less dense encoding. Rejected for brevity.
- Short hash (8 chars): Insufficient entropy — vulnerable to enumeration over time. Rejected.
- Human-readable slug (e.g., "purple-pony-42"): Fun but lower entropy and risks collisions. Rejected.

## Decision 3: Token Persistence Across Toggles

**Decision**: The `public_notes` row is created on first publish and retained on unpublish (only the `published` flag is flipped). Re-publishing reuses the same token.

**Rationale**: The spec requires "previously shared links continue to work when the note is re-published." Keeping the row in place (vs. deleting on unpublish and regenerating) is simpler and satisfies this requirement. Storage cost is negligible (one row per shared-at-least-once note).

**Alternatives considered**:
- Delete row on unpublish: Breaks the spec requirement. Rejected.
- Separate `tokens` table with regenerate-on-demand: Over-engineered for v1. Rejected.

## Decision 4: Wiki-Link Handling in Public Notes

**Decision**: When rendering a public note, wiki-links (`[[Title]]`) are resolved only against notes that are also published. If the target is private, render the link text as plain text (no `<a>` tag), preserving the text but not creating a hyperlink.

**Rationale**: The spec mandates "Wiki-links to notes that are not public MUST NOT reveal the existence or any content of those notes." Plain text preservation maintains reading flow without leaking information. If a target happens to be public, linking to its `/p/{token}` URL is appropriate so readers can follow cross-references.

**Alternatives considered**:
- Strip `[[...]]` entirely: Loses the author's intent (the reader doesn't see there was a link). Rejected.
- Render as broken link (e.g., red underline): Reveals existence of the reference, inconsistent UX. Rejected.
- Hyperlink to login page: Invites auth attempts, leaks existence. Rejected.

## Decision 5: Public Image URL Rewriting

**Decision**: In the rendered HTML of a public note, rewrite `/uploads/{username}/{filename}` image URLs to `/p/{token}/uploads/{filename}` and serve these via a dedicated public image handler that validates the token-to-note-to-image relationship.

**Rationale**: Images embedded in a public note must be accessible to unauthenticated visitors, but the existing `/uploads/{username}/...` handler enforces session ownership. Rewriting to a token-scoped URL keeps the authorization model explicit and auditable — the handler only serves images that belong to the published note.

**Alternatives considered**:
- Make `/uploads/...` public for images belonging to public notes: Requires DB lookup on every image request and coupling; more surface area. Rejected.
- Embed images as base64 data URLs: Bloats HTML, inefficient for repeat views. Rejected.

## Decision 6: Checkbox Rendering in Public View

**Decision**: Render checkboxes as static HTML (`<input type="checkbox" disabled>` or CSS-styled spans). No `data-slug`/`data-line` attributes. No HTMX. Todos are purely visual.

**Rationale**: Visitors must not be able to mutate the note. The simplest way to enforce this is to omit the machinery that enables mutation. The `@due` badge rendering continues to work since it's pure display logic.

**Alternatives considered**:
- Remove checkboxes entirely: Loses information — the owner's task list is valuable context. Rejected.
- Session-gated mutations with fallback to read-only: Same end result but more complex handler code. Rejected.

## Decision 7: Template Strategy

**Decision**: Public note rendering uses a separate minimal template (`templates/public/note.html`) that does NOT extend `base.html` — it defines its own complete HTML document.

**Rationale**: `base.html` includes the site nav, sidebar, and session-aware UI. Reusing it for public pages would either require heavy conditional logic (is user logged in? hide sidebar? strip nav?) or risk leakage of owner UI elements. A standalone template makes the "no owner UI" requirement a property of the template, not something to remember to check. It also loads faster (no sidebar HTMX request) and has a cleaner mobile appearance.

**Alternatives considered**:
- Reuse base.html with conditionals: More code paths, easier to accidentally leak UI. Rejected.
- Use a different layout template like `public_base.html`: Same practical effect as a standalone template but adds an indirection. Rejected for simplicity.

## Decision 8: SEO / Robots

**Decision**: Public note page includes `<meta name="robots" content="noindex,nofollow">` by default.

**Rationale**: Spec explicitly says "Public notes are indexed by search engines only if explicitly desired by the owner in a future enhancement — v1 defaults to `noindex`." This prevents accidental SEO exposure of notes shared via link-sharing model.

**Alternatives considered**:
- No robots meta: Search engines may index. Violates spec. Rejected.
- Per-note opt-in: Extra UI for v1. Deferred.
