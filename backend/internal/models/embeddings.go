package models

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// PrepareEmbeddingText combines title and body into a single string for embedding.
// Truncates to approximately maxChars to stay within model input limits.
func PrepareEmbeddingText(title, body string) string {
	text := title
	body = strings.TrimSpace(body)
	if body != "" {
		text = title + "\n\n" + body
	}
	const maxChars = 8000
	if len(text) > maxChars {
		text = text[:maxChars]
	}
	return text
}

// ContentHash returns a SHA-256 hash of the text to detect content changes.
func ContentHash(title, body string) string {
	text := PrepareEmbeddingText(title, body)
	h := sha256.Sum256([]byte(text))
	return fmt.Sprintf("%x", h)
}

// NeedsEmbedding checks if a note needs a new embedding by comparing content hashes.
func NeedsEmbedding(db *DB, noteID int64, currentHash string) bool {
	var storedHash string
	err := db.QueryRow(`SELECT content_hash FROM note_embeddings WHERE note_id=?`, noteID).Scan(&storedHash)
	if err != nil {
		return true
	}
	return storedHash != currentHash
}

// UpsertEmbedding stores or updates the embedding for a note in both
// note_embeddings (metadata) and vec_note_embeddings (vector data).
func UpsertEmbedding(db *DB, noteID int64, embedding []float32, contentHash string) error {
	now := time.Now().UTC().Format(time.RFC3339)

	vecJSON, err := json.Marshal(embedding)
	if err != nil {
		return fmt.Errorf("marshal embedding: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	_, err = tx.Exec(
		`INSERT INTO note_embeddings(note_id, content_hash, updated_at) VALUES(?,?,?)
		 ON CONFLICT(note_id) DO UPDATE SET content_hash=excluded.content_hash, updated_at=excluded.updated_at`,
		noteID, contentHash, now)
	if err != nil {
		return fmt.Errorf("upsert note_embeddings: %w", err)
	}

	// Delete then insert for vec virtual table (no ON CONFLICT support)
	_, _ = tx.Exec(`DELETE FROM vec_note_embeddings WHERE note_id=?`, noteID)
	_, err = tx.Exec(
		`INSERT INTO vec_note_embeddings(note_id, embedding) VALUES(?, vec_f32(?))`,
		noteID, string(vecJSON))
	if err != nil {
		return fmt.Errorf("insert vec_note_embeddings: %w", err)
	}

	return tx.Commit()
}

// DeleteEmbedding removes embedding data for a note from both tables.
func DeleteEmbedding(db *DB, noteID int64) error {
	_, _ = db.Exec(`DELETE FROM vec_note_embeddings WHERE note_id=?`, noteID)
	_, err := db.Exec(`DELETE FROM note_embeddings WHERE note_id=?`, noteID)
	return err
}

// NoteInfo holds minimal note data for embedding operations.
type NoteInfo struct {
	ID     int64
	UserID int64
	Slug   string
	Title  string
}

// NotesWithoutEmbeddings returns IDs and slugs of notes that have no embedding.
func NotesWithoutEmbeddings(db *DB) ([]NoteInfo, error) {
	rows, err := db.Query(
		`SELECT n.id, n.user_id, n.slug, n.title FROM notes n
		 LEFT JOIN note_embeddings e ON e.note_id = n.id
		 WHERE e.note_id IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []NoteInfo
	for rows.Next() {
		var n NoteInfo
		if err := rows.Scan(&n.ID, &n.UserID, &n.Slug, &n.Title); err != nil {
			return nil, err
		}
		result = append(result, n)
	}
	return result, rows.Err()
}
