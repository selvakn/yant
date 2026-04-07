package models

import (
	"encoding/json"
	"fmt"
	"html/template"
	"time"

	"github.com/selvakn/yant/internal/storage"
)

const (
	DefaultSimilarityThreshold = 0.9
	DefaultMaxResults          = 20
)

// SemanticSearch performs a KNN vector search against stored embeddings,
// then merges in text-fallback results for notes without embeddings.
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

	// KNN query: collect note_ids and distances, then close rows before further DB calls
	// (SQLite MaxOpenConns=1 means nested queries cause deadlock)
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

	// Hydrate matched notes (connection is now free)
	var semanticResults []SearchResult
	seenIDs := make(map[int64]bool)

	for _, m := range matches {
		if len(semanticResults) >= maxResults {
			break
		}

		note, err := getNoteByID(db, m.noteID)
		if err != nil || note == nil {
			continue
		}
		note.Tags, _ = getTagsForNote(db, m.noteID)

		body, _ := storage.ReadNote(notesDir, note.UserID, note.Slug)

		semanticResults = append(semanticResults, SearchResult{
			Note:           note,
			Body:           body,
			Score:          int(m.similarity * 1000),
			TitleHighlight: template.HTML(template.HTMLEscapeString(note.Title)),
			TagsHighlight:  highlightTagsPlain(note.Tags),
			BodySnippet:    BodySnippet(body, query, 150),
		})
		seenIDs[m.noteID] = true
	}

	// Text-based fallback for notes without embeddings
	fallbackResults, err := textFallbackSearch(db, notesDir, userID, query, archived, seenIDs)
	if err != nil {
		return nil, err
	}

	remaining := maxResults - len(semanticResults)
	if remaining > 0 && len(fallbackResults) > 0 {
		if len(fallbackResults) > remaining {
			fallbackResults = fallbackResults[:remaining]
		}
		semanticResults = append(semanticResults, fallbackResults...)
	}

	return semanticResults, nil
}

// textFallbackSearch uses existing fuzzy search for notes that lack embeddings.
func textFallbackSearch(db *DB, notesDir string, userID int64, query string, archived bool, excludeIDs map[int64]bool) ([]SearchResult, error) {
	allResults, err := SearchNotes(db, notesDir, userID, query, archived)
	if err != nil {
		return nil, err
	}

	var fallback []SearchResult
	for _, r := range allResults {
		if !excludeIDs[r.Note.ID] {
			fallback = append(fallback, r)
		}
	}
	return fallback, nil
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
