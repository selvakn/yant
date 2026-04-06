# Research: Fuzzy Search for Notes

**Feature**: 006-fuzzy-search
**Date**: 2026-04-05

## Research Questions

### 1. Fuzzy Search Library Selection

**Decision**: Use `github.com/lithammer/fuzzysearch/fuzzy`

**Rationale**:
- Lightweight, zero dependencies
- Simple API: `fuzzy.RankMatchFold(pattern, target)` returns score (-1 = no match)
- Case-insensitive matching built-in via `Fold` variants
- Handles typos via character-skip matching (not Levenshtein, but good for substring matching)
- Active maintenance, widely used

**Alternatives Considered**:

| Library | Pros | Cons | Rejected Because |
|---------|------|------|------------------|
| `sahilm/fuzzy` | Good scoring, highlighting | More complex API | Overkill for simple use case |
| SQLite FTS5 | Powerful, persistent index | Setup complexity, schema changes | Violates Principle II (simplicity) |
| Custom Levenshtein | No dependencies | Implementation time, edge cases | Reinventing the wheel |

### 2. Search Architecture

**Decision**: Server-side search via htmx endpoint

**Rationale**:
- Consistent with existing htmx patterns in project
- No need to send all note bodies to client
- Debounce on client, single API call per search
- Returns HTML partial (search-results.html) for htmx swap

**Flow**:
1. User types in search input (with `hx-trigger="keyup changed delay:250ms"`)
2. htmx calls `GET /notes/search?q={query}`
3. Handler loads all user notes with bodies, scores each against query
4. Returns ranked, highlighted HTML fragment
5. htmx swaps into results container

**Alternatives Considered**:

| Approach | Pros | Cons | Rejected Because |
|----------|------|------|------------------|
| Pure client-side | No server round-trip | Must load all bodies upfront, memory heavy | Doesn't scale to 500+ notes |
| WebSocket streaming | Real-time updates | Complexity, overkill | Violates Principle II |

### 3. Relevance Ranking Strategy

**Decision**: Weighted scoring with field multipliers

**Rationale**:
- Title match: weight 3x (most important)
- Tag match: weight 2x (secondary)
- Body match: weight 1x (least specific)
- Final score = sum of (field_score × weight) for each field with match
- Sort descending by score

**Implementation**:
```go
func ScoreNote(query string, note *NoteWithBody) int {
    score := 0
    if s := fuzzy.RankMatchFold(query, note.Title); s >= 0 {
        score += (s + 1) * 3
    }
    for _, tag := range note.Tags {
        if s := fuzzy.RankMatchFold(query, tag); s >= 0 {
            score += (s + 1) * 2
            break // count tag match once
        }
    }
    if s := fuzzy.RankMatchFold(query, note.Body); s >= 0 {
        score += (s + 1) * 1
    }
    return score
}
```

### 4. Highlighting Strategy

**Decision**: Client-side highlighting via JavaScript after htmx swap

**Rationale**:
- Server returns plain text with data attributes for match positions
- Simpler server code; no HTML escaping edge cases
- Use `<mark>` tags for highlighting
- Actually, simpler: use CSS `::highlight()` or wrap matched text in `<mark>` on server

**Revised Decision**: Server-side highlighting in Go template

- Escape HTML in content first
- Wrap matched substrings in `<mark>` tags
- Template uses `template.HTML` for safe rendering

### 5. Keyboard Navigation

**Decision**: JavaScript event handlers on search results list

**Implementation**:
- Track `selectedIndex` in JS state
- Arrow Up/Down: change selection, add `.selected` class
- Enter: navigate to selected note's href
- Escape: clear search input, reset selection

### 6. Performance Considerations

**Decision**: Acceptable for <500 notes per user

**Rationale**:
- Each search loads all notes from DB + reads all bodies from disk
- For 500 notes × ~2KB avg body = ~1MB read per search
- With SSD + OS caching, this completes in <100ms
- If scale increases, can add SQLite FTS5 later (out of scope for MVP)

**Mitigations**:
- 250ms debounce prevents excessive API calls
- Consider caching note bodies in memory per session (future enhancement, not MVP)

## Summary

| Decision | Choice |
|----------|--------|
| Fuzzy library | `lithammer/fuzzysearch/fuzzy` |
| Architecture | Server-side htmx endpoint |
| Ranking | Weighted scores (title 3x, tags 2x, body 1x) |
| Highlighting | Server-side `<mark>` wrapping |
| Keyboard nav | Client-side JS |
| Performance | Acceptable for MVP; scale later if needed |
