package models

import (
	"html/template"
	"sort"
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"

	"github.com/selvakn/my-notes/internal/storage"
)

// SearchResult represents a note that matched a search query.
type SearchResult struct {
	Note           *Note
	Body           string
	Score          int
	TitleHighlight template.HTML
	TagsHighlight  []template.HTML
	BodySnippet    template.HTML
}

// SearchNotes searches notes by title, tags, and body using fuzzy matching.
// Returns results sorted by relevance score (descending).
func SearchNotes(db *DB, notesDir string, userID int64, query string) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if len(query) > 200 {
		query = query[:200]
	}

	notes, err := ListNotes(db, userID, "")
	if err != nil {
		return nil, err
	}

	if query == "" {
		var results []SearchResult
		for _, note := range notes {
			body, _ := storage.ReadNote(notesDir, userID, note.Slug)
			results = append(results, SearchResult{
				Note:           note,
				Body:           body,
				Score:          0,
				TitleHighlight: template.HTML(template.HTMLEscapeString(note.Title)),
				BodySnippet:    BodySnippet(body, "", 150),
			})
		}
		return results, nil
	}

	var results []SearchResult
	for _, note := range notes {
		body, _ := storage.ReadNote(notesDir, userID, note.Slug)
		score := ScoreNote(query, note, body)
		if score > 0 {
			results = append(results, SearchResult{
				Note:           note,
				Body:           body,
				Score:          score,
				TitleHighlight: HighlightMatch(note.Title, query),
				TagsHighlight:  HighlightTags(note.Tags, query),
				BodySnippet:    BodySnippet(body, query, 150),
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results, nil
}

// ScoreNote calculates relevance score for a note against a query.
// Title matches: 3x weight, Tag matches: 2x weight, Body matches: 1x weight.
func ScoreNote(query string, note *Note, body string) int {
	score := 0

	if s := fuzzy.RankMatchFold(query, note.Title); s >= 0 {
		score += (s + 1) * 3
	}

	for _, tag := range note.Tags {
		if s := fuzzy.RankMatchFold(query, tag); s >= 0 {
			score += (s + 1) * 2
			break
		}
	}

	if s := fuzzy.RankMatchFold(query, body); s >= 0 {
		score += (s + 1) * 1
	}

	return score
}

// HighlightMatch wraps matched substrings in <mark> tags using case-insensitive search.
func HighlightMatch(text, query string) template.HTML {
	if query == "" || text == "" {
		return template.HTML(template.HTMLEscapeString(text))
	}

	lowerText := strings.ToLower(text)
	lowerQuery := strings.ToLower(query)

	idx := strings.Index(lowerText, lowerQuery)
	if idx == -1 {
		return template.HTML(template.HTMLEscapeString(text))
	}

	var result strings.Builder
	result.WriteString(template.HTMLEscapeString(text[:idx]))
	result.WriteString("<mark>")
	result.WriteString(template.HTMLEscapeString(text[idx : idx+len(query)]))
	result.WriteString("</mark>")
	result.WriteString(template.HTMLEscapeString(text[idx+len(query):]))

	return template.HTML(result.String())
}

// HighlightTags applies highlighting to each tag that matches.
func HighlightTags(tags []string, query string) []template.HTML {
	var result []template.HTML
	for _, tag := range tags {
		result = append(result, HighlightMatch(tag, query))
	}
	return result
}

// BodySnippet extracts a snippet around the first match with highlighting.
func BodySnippet(body, query string, maxLen int) template.HTML {
	if body == "" {
		return ""
	}

	body = strings.ReplaceAll(body, "\n", " ")
	body = strings.ReplaceAll(body, "\r", " ")

	if query == "" {
		if len(body) > maxLen {
			return template.HTML(template.HTMLEscapeString(body[:maxLen]) + "…")
		}
		return template.HTML(template.HTMLEscapeString(body))
	}

	lowerBody := strings.ToLower(body)
	lowerQuery := strings.ToLower(query)
	idx := strings.Index(lowerBody, lowerQuery)

	if idx == -1 {
		if fuzzy.MatchFold(query, body) {
			idx = 0
		}
	}

	if idx == -1 {
		if len(body) > maxLen {
			return template.HTML(template.HTMLEscapeString(body[:maxLen]) + "…")
		}
		return template.HTML(template.HTMLEscapeString(body))
	}

	start := idx - 50
	if start < 0 {
		start = 0
	}
	end := start + maxLen
	if end > len(body) {
		end = len(body)
	}

	snippet := body[start:end]
	prefix := ""
	suffix := ""
	if start > 0 {
		prefix = "…"
	}
	if end < len(body) {
		suffix = "…"
	}

	highlighted := HighlightMatch(snippet, query)
	return template.HTML(prefix + string(highlighted) + suffix)
}
