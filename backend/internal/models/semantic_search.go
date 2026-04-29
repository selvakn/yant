package models

import (
	"encoding/json"
	"fmt"
	"html/template"
	"sort"
	"strings"
	"time"

	"github.com/selvakn/yant/internal/storage"
)

const (
	DefaultSimilarityThreshold = 0.9
	DefaultMaxResults          = 20

	// Notes with embedding text shorter than this get a proportional score
	// discount. Prevents short/generic text from producing "hub" vectors that
	// falsely match many queries.
	minEmbeddingContentLen = 100

	// Bonus added to text-search scores when the query appears as an exact
	// case-insensitive substring in the title or body. Ensures literal matches
	// compete with semantic scores (which are on a 0-1000 scale).
	exactMatchTitleBonus = 500
	exactMatchBodyBonus  = 300
)

// SemanticSearch performs a KNN vector search against stored embeddings,
// then merges in text-based results, re-ranking the combined set so that
// exact text matches can outrank weak semantic hits.
func SemanticSearch(db *DB, notesDir string, userID int64, query string, queryEmbedding []float32, archived bool, threshold float64, maxResults int) ([]SearchResult, error) {
	if threshold <= 0 {
		threshold = DefaultSimilarityThreshold
	}
	if maxResults <= 0 {
		maxResults = DefaultMaxResults
	}

	vecJSON, err := json.Marshal(queryEmbedding)
	if err != nil {
		return nil, fmt.Errorf("marshal query embedding: %w", err)
	}

	archivedVal := 0
	if archived {
		archivedVal = 1
	}

	// KNN query: collect note_ids and distances, then close rows before
	// further DB calls (SQLite MaxOpenConns=1 means nested queries deadlock).
	type knnMatch struct {
		noteID     int64
		similarity float64
	}

	rows, err := db.Query(
		`SELECT v.note_id, v.distance
		 FROM vec_note_embeddings v
		 JOIN notes n ON n.id = v.note_id
		 WHERE v.embedding MATCH vec_f32(?)
		   AND k = ?
		   AND n.user_id = ?
		   AND n.archived = ?
		 ORDER BY v.distance ASC`,
		string(vecJSON), maxResults*2, userID, archivedVal)
	if err != nil {
		return nil, fmt.Errorf("knn query: %w", err)
	}

	var matches []knnMatch
	for rows.Next() {
		var noteID int64
		var distance float64
		if err := rows.Scan(&noteID, &distance); err != nil {
			rows.Close()
			return nil, err
		}
		similarity := 1.0 - distance
		if similarity >= threshold {
			matches = append(matches, knnMatch{noteID: noteID, similarity: similarity})
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	merged := make(map[int64]SearchResult)

	for _, m := range matches {
		note, err := getNoteByID(db, m.noteID)
		if err != nil || note == nil {
			continue
		}
		note.Tags, _ = getTagsForNote(db, m.noteID)

		body, _ := storage.ReadNote(notesDir, note.UserID, note.Slug)

		score := contentLengthAdjustedScore(m.similarity, note.Title, body)

		merged[m.noteID] = SearchResult{
			Note:           note,
			Body:           body,
			Score:          score,
			TitleHighlight: template.HTML(template.HTMLEscapeString(note.Title)),
			TagsHighlight:  highlightTagsPlain(note.Tags),
			BodySnippet:    BodySnippet(body, query, 150),
		}
	}

	// Run text-based search across ALL notes so exact text matches compete
	// fairly with semantic hits.
	textResults, err := SearchNotes(db, notesDir, userID, query, archived)
	if err != nil {
		return nil, err
	}

	lowerQuery := strings.ToLower(query)
	for _, tr := range textResults {
		tr.Score += exactMatchBonus(lowerQuery, tr.Note.Title, tr.Body)

		existing, exists := merged[tr.Note.ID]
		if !exists {
			merged[tr.Note.ID] = tr
		} else if tr.Score > existing.Score {
			if hasHighlight(existing.TitleHighlight) {
				tr.TitleHighlight = existing.TitleHighlight
			}
			if existing.BodySnippet != "" && hasHighlight(existing.BodySnippet) {
				tr.BodySnippet = existing.BodySnippet
			}
			merged[tr.Note.ID] = tr
		} else {
			if hasHighlight(tr.TitleHighlight) {
				existing.TitleHighlight = tr.TitleHighlight
			}
			if tr.BodySnippet != "" && !hasHighlight(existing.BodySnippet) {
				existing.BodySnippet = tr.BodySnippet
			}
			merged[tr.Note.ID] = existing
		}
	}

	results := make([]SearchResult, 0, len(merged))
	for _, r := range merged {
		results = append(results, r)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > maxResults {
		results = results[:maxResults]
	}

	return results, nil
}

// contentLengthAdjustedScore discounts semantic similarity for notes whose
// embedding text is very short. Short, generic texts (e.g. "Untitled Note")
// produce "hub" vectors with deceptively high cosine similarity to many
// queries.
func contentLengthAdjustedScore(similarity float64, title, body string) int {
	contentLen := len(PrepareEmbeddingText(title, body))
	lengthFactor := 1.0
	if contentLen < minEmbeddingContentLen {
		lengthFactor = float64(contentLen) / float64(minEmbeddingContentLen)
	}
	return int(similarity * lengthFactor * 1000)
}

// exactMatchBonus returns a score bonus when the query appears as a literal
// substring (case-insensitive) in the title or body.
func exactMatchBonus(lowerQuery, title, body string) int {
	bonus := 0
	if strings.Contains(strings.ToLower(title), lowerQuery) {
		bonus += exactMatchTitleBonus
	}
	if strings.Contains(strings.ToLower(body), lowerQuery) {
		bonus += exactMatchBodyBonus
	}
	return bonus
}

func hasHighlight(h template.HTML) bool {
	return strings.Contains(string(h), "<mark>")
}

func getNoteByID(db *DB, noteID int64) (*Note, error) {
	var n Note
	var ca, ua string
	var archived int
	err := db.QueryRow(
		`SELECT id, user_id, slug, title, archived, created_at, updated_at FROM notes WHERE id=?`,
		noteID).Scan(&n.ID, &n.UserID, &n.Slug, &n.Title, &archived, &ca, &ua)
	if err != nil {
		return nil, err
	}
	n.Archived = archived == 1
	n.CreatedAt, _ = time.Parse(time.RFC3339, ca)
	n.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
	return &n, nil
}

func highlightTagsPlain(tags []string) []template.HTML {
	var result []template.HTML
	for _, t := range tags {
		result = append(result, template.HTML(template.HTMLEscapeString(t)))
	}
	return result
}
