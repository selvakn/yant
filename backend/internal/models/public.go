package models

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

// PublicNote represents the publishing state of a note.
type PublicNote struct {
	NoteID      int64
	Token       string
	Published   bool
	PublishedAt time.Time
	UpdatedAt   time.Time
}

// PublicNoteSummary is used by the owner's public notes list.
type PublicNoteSummary struct {
	Slug        string
	Title       string
	Token       string
	PublishedAt time.Time
}

// GenerateShareToken returns a URL-safe random token with 128 bits of entropy.
func GenerateShareToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// PublishNote marks a note as publicly shared, generating a token on first publish.
// Subsequent publishes reuse the existing token.
// Returns the token.
func PublishNote(db *DB, noteID int64) (string, error) {
	// Check if a row already exists
	var existingToken string
	err := db.QueryRow("SELECT token FROM public_notes WHERE note_id = ?", noteID).Scan(&existingToken)
	now := time.Now().UTC().Format(time.RFC3339)

	if err == nil {
		// Row exists — flip published to true, keep token
		_, err := db.Exec(
			"UPDATE public_notes SET published = 1, updated_at = ? WHERE note_id = ?",
			now, noteID)
		if err != nil {
			return "", err
		}
		return existingToken, nil
	}
	if err != sql.ErrNoRows {
		return "", err
	}

	// Generate a new token and insert
	token, err := GenerateShareToken()
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	_, err = db.Exec(
		"INSERT INTO public_notes (note_id, token, published, published_at, updated_at) VALUES (?, ?, 1, ?, ?)",
		noteID, token, now, now)
	if err != nil {
		return "", err
	}
	return token, nil
}

// UnpublishNote marks a note's public share as not published. Token is preserved.
// Returns nil (no error) even if the note was not previously published.
func UnpublishNote(db *DB, noteID int64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(
		"UPDATE public_notes SET published = 0, updated_at = ? WHERE note_id = ?",
		now, noteID)
	return err
}

// GetNoteByToken returns the note associated with a public token if and only if
// the note is currently published AND not archived. Returns (nil, nil) if not
// found, expired, or archived.
func GetNoteByToken(db *DB, token string) (*Note, error) {
	row := db.QueryRow(`
		SELECT n.id, n.user_id, n.slug, n.title, n.archived, n.created_at, n.updated_at
		FROM public_notes p
		JOIN notes n ON n.id = p.note_id
		WHERE p.token = ? AND p.published = 1 AND n.archived = 0`, token)
	n, err := scanNote(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	n.Tags, _ = getTagsForNote(db, n.ID)
	return n, nil
}

// GetPublicShare returns the public share state for a given note.
// Returns (nil, nil) if no share row exists (never published).
func GetPublicShare(db *DB, noteID int64) (*PublicNote, error) {
	var p PublicNote
	var publishedAt, updatedAt string
	err := db.QueryRow(
		"SELECT note_id, token, published, published_at, updated_at FROM public_notes WHERE note_id = ?",
		noteID,
	).Scan(&p.NoteID, &p.Token, &p.Published, &publishedAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.PublishedAt, _ = time.Parse(time.RFC3339, publishedAt)
	p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &p, nil
}

// ListPublishedNotes returns all currently-published notes for a user.
func ListPublishedNotes(db *DB, userID int64) ([]PublicNoteSummary, error) {
	rows, err := db.Query(`
		SELECT n.slug, n.title, p.token, p.published_at
		FROM public_notes p
		JOIN notes n ON n.id = p.note_id
		WHERE n.user_id = ? AND p.published = 1 AND n.archived = 0
		ORDER BY p.published_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PublicNoteSummary
	for rows.Next() {
		var s PublicNoteSummary
		var publishedAt string
		if err := rows.Scan(&s.Slug, &s.Title, &s.Token, &publishedAt); err != nil {
			return nil, err
		}
		s.PublishedAt, _ = time.Parse(time.RFC3339, publishedAt)
		out = append(out, s)
	}
	return out, nil
}

// CountPublishedNotes returns the number of currently-published notes for a user.
func CountPublishedNotes(db *DB, userID int64) int {
	var count int
	db.QueryRow(`
		SELECT COUNT(*) FROM public_notes p
		JOIN notes n ON n.id = p.note_id
		WHERE n.user_id = ? AND p.published = 1 AND n.archived = 0`,
		userID).Scan(&count) //nolint:errcheck
	return count
}

// ResolveWikiLinksPublic resolves [[Title]] wiki-links for public rendering.
// Only resolves to other *published* notes belonging to the same user.
// Private notes render as plain text (no hyperlink), preventing information leakage.
func ResolveWikiLinksPublic(db *DB, userID int64, body string) string {
	return noteLinkRe.ReplaceAllStringFunc(body, func(match string) string {
		title := strings.TrimSpace(match[2 : len(match)-2])

		// Look up the target note's ID + public token, restricted to same user and published
		var token string
		err := db.QueryRow(`
			SELECT p.token
			FROM notes n
			JOIN public_notes p ON p.note_id = n.id
			WHERE n.user_id = ? AND LOWER(n.title) = LOWER(?) AND p.published = 1 AND n.archived = 0`,
			userID, title).Scan(&token)
		if err != nil {
			// Target doesn't exist, is private, or is archived — render as plain text
			return title
		}
		return fmt.Sprintf("[%s](/p/%s)", title, token)
	})
}
