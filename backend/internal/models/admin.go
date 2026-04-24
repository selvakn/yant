package models

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// UserStats holds a user record with aggregate counts for the admin view.
type UserStats struct {
	ID        int64
	Username  string
	NoteCount int
	CreatedAt time.Time
}

// AdminNoteEntry holds note metadata for the admin user-detail view.
type AdminNoteEntry struct {
	ID        int64
	Slug      string
	Title     string
	Archived  bool
	Tags      []string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ListAllUsers returns every user with their note count, ordered by username.
func ListAllUsers(db *DB) ([]UserStats, error) {
	rows, err := db.Query(`
		SELECT u.id, u.username, u.created_at, COUNT(n.id) AS note_count
		FROM users u
		LEFT JOIN notes n ON n.user_id = u.id
		GROUP BY u.id
		ORDER BY u.username ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []UserStats
	for rows.Next() {
		var us UserStats
		var ca string
		if err := rows.Scan(&us.ID, &us.Username, &ca, &us.NoteCount); err != nil {
			return nil, err
		}
		us.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		result = append(result, us)
	}
	return result, rows.Err()
}

// ListAllNotesForUser returns every note (active + archived) for a given user.
func ListAllNotesForUser(db *DB, userID int64) ([]AdminNoteEntry, error) {
	rows, err := db.Query(`
		SELECT id, slug, title, archived, created_at, updated_at
		FROM notes
		WHERE user_id = ?
		ORDER BY updated_at DESC`, userID)
	if err != nil {
		return nil, err
	}

	var notes []AdminNoteEntry
	for rows.Next() {
		var n AdminNoteEntry
		var ca, ua string
		var archived int
		if err := rows.Scan(&n.ID, &n.Slug, &n.Title, &archived, &ca, &ua); err != nil {
			rows.Close()
			return nil, err
		}
		n.Archived = archived == 1
		n.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		n.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
		notes = append(notes, n)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range notes {
		notes[i].Tags, _ = getTagsForNote(db, notes[i].ID)
	}
	return notes, nil
}

// AdminDeleteNote removes a single note, its metadata, and its file on disk.
func AdminDeleteNote(db *DB, noteID int64, notesDir, uploadsDir string) error {
	var userID int64
	var slug string
	err := db.QueryRow(`SELECT user_id, slug FROM notes WHERE id = ?`, noteID).Scan(&userID, &slug)
	if err != nil {
		return err
	}

	// Remove image files from disk
	imgRows, err := db.Query(`SELECT filename FROM images WHERE note_id = ?`, noteID)
	if err == nil {
		defer imgRows.Close()
		for imgRows.Next() {
			var fn string
			if err := imgRows.Scan(&fn); err == nil {
				_ = os.Remove(filepath.Join(uploadsDir, fmt.Sprintf("%d", userID), fn))
			}
		}
	}

	// Cascade deletes handle note_tags, images, note_links, note_todos, etc.
	if _, err := db.Exec(`DELETE FROM notes WHERE id = ?`, noteID); err != nil {
		return err
	}

	// Remove markdown file
	mdPath := filepath.Join(notesDir, fmt.Sprintf("%d", userID), slug+".md")
	_ = os.Remove(mdPath)

	// Remove drawing if present
	drawPath := filepath.Join(notesDir, fmt.Sprintf("%d", userID), slug+".tldraw.json")
	_ = os.Remove(drawPath)

	return nil
}

// AdminDeleteUser removes a user and all their data (notes, files, images).
func AdminDeleteUser(db *DB, userID int64, notesDir, uploadsDir string) error {
	// Collect all note IDs for the user
	rows, err := db.Query(`SELECT id FROM notes WHERE user_id = ?`, userID)
	if err != nil {
		return err
	}
	var noteIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		noteIDs = append(noteIDs, id)
	}
	rows.Close()

	for _, nid := range noteIDs {
		if err := AdminDeleteNote(db, nid, notesDir, uploadsDir); err != nil {
			return err
		}
	}

	// Delete shares granted BY this user
	if _, err := db.Exec(`DELETE FROM note_shares WHERE granted_by = ?`, userID); err != nil {
		return err
	}
	// Delete shares granted TO this user
	if _, err := db.Exec(`DELETE FROM note_shares WHERE user_id = ?`, userID); err != nil {
		return err
	}

	if _, err := db.Exec(`DELETE FROM tag_colors WHERE user_id = ?`, userID); err != nil {
		return err
	}

	if _, err := db.Exec(`DELETE FROM users WHERE id = ?`, userID); err != nil {
		return err
	}

	// Remove user directories
	_ = os.RemoveAll(filepath.Join(notesDir, fmt.Sprintf("%d", userID)))
	_ = os.RemoveAll(filepath.Join(uploadsDir, fmt.Sprintf("%d", userID)))

	return nil
}

// GetUserByID returns a user by their ID.
func GetUserByID(db *DB, id int64) (*User, error) {
	var u User
	var createdAt string
	err := db.QueryRow(
		`SELECT id, username, created_at FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Username, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &u, nil
}
