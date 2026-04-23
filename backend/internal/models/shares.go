package models

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Role constants for share-based access.
const (
	RoleOwner  = "owner"
	RoleEditor = "editor"
	RoleReader = "reader"
)

// PermissionRead / PermissionEdit are the two share levels.
const (
	PermissionRead = "read"
	PermissionEdit = "edit"
)

// ErrNoAccess is returned when the viewer has no ownership or share grant on a note.
var ErrNoAccess = errors.New("no access to note")

// ErrSelfShare is returned when an owner attempts to share a note with themselves.
var ErrSelfShare = errors.New("cannot share with yourself")

// NoteShare represents a single share grant on a note.
type NoteShare struct {
	NoteID     int64
	UserID     int64
	Permission string
	GrantedAt  time.Time
	GrantedBy  int64
}

// NoteCollaborator is the per-note view of a collaborator, used by the share dialog.
type NoteCollaborator struct {
	Username   string
	Permission string
	GrantedAt  time.Time
}

// SharedNoteSummary is used when listing notes shared with a viewer.
type SharedNoteSummary struct {
	Slug          string
	Title         string
	OwnerUsername string
	Permission    string
	UpdatedAt     time.Time
	Tags          []string
}

// GrantShare inserts or updates a share grant. permission must be "read" or "edit".
// Caller is responsible for preventing self-shares (ErrSelfShare).
func GrantShare(db *DB, noteID, collaboratorID, grantorID int64, permission string) error {
	if permission != PermissionRead && permission != PermissionEdit {
		return fmt.Errorf("invalid permission %q", permission)
	}
	if collaboratorID == grantorID {
		return ErrSelfShare
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(`
		INSERT INTO note_shares (note_id, user_id, permission, granted_at, granted_by)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(note_id, user_id) DO UPDATE SET
			permission = excluded.permission,
			granted_at = excluded.granted_at,
			granted_by = excluded.granted_by`,
		noteID, collaboratorID, permission, now, grantorID)
	return err
}

// RevokeShare removes a share grant. Returns nil even if the row didn't exist.
func RevokeShare(db *DB, noteID, collaboratorID int64) error {
	_, err := db.Exec("DELETE FROM note_shares WHERE note_id = ? AND user_id = ?", noteID, collaboratorID)
	return err
}

// GetShareByViewerAndNote returns (permission, true) if the viewer has a share on the note,
// or ("", false) if not.
func GetShareByViewerAndNote(db *DB, noteID, viewerID int64) (string, bool) {
	var permission string
	err := db.QueryRow(
		"SELECT permission FROM note_shares WHERE note_id = ? AND user_id = ?",
		noteID, viewerID).Scan(&permission)
	if err != nil {
		return "", false
	}
	return permission, true
}

// ListSharesForNote returns all collaborators on a note, ordered by username.
func ListSharesForNote(db *DB, noteID int64) ([]NoteCollaborator, error) {
	rows, err := db.Query(`
		SELECT u.username, s.permission, s.granted_at
		FROM note_shares s
		JOIN users u ON u.id = s.user_id
		WHERE s.note_id = ?
		ORDER BY u.username`, noteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []NoteCollaborator
	for rows.Next() {
		var c NoteCollaborator
		var grantedAt string
		if err := rows.Scan(&c.Username, &c.Permission, &grantedAt); err != nil {
			return nil, err
		}
		c.GrantedAt, _ = time.Parse(time.RFC3339, grantedAt)
		out = append(out, c)
	}
	return out, nil
}

// ListSharedNotesForUser returns notes that have been shared WITH the given user
// (i.e. they are a collaborator, not the owner). Excludes archived notes.
func ListSharedNotesForUser(db *DB, viewerID int64) ([]SharedNoteSummary, error) {
	rows, err := db.Query(`
		SELECT n.slug, n.title, u.username, s.permission, n.updated_at
		FROM note_shares s
		JOIN notes n ON n.id = s.note_id
		JOIN users u ON u.id = n.user_id
		WHERE s.user_id = ? AND n.archived = 0
		ORDER BY n.updated_at DESC`, viewerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SharedNoteSummary
	for rows.Next() {
		var s SharedNoteSummary
		var updatedAt string
		if err := rows.Scan(&s.Slug, &s.Title, &s.OwnerUsername, &s.Permission, &updatedAt); err != nil {
			return nil, err
		}
		s.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		out = append(out, s)
	}
	return out, nil
}

// CountSharedNotesForUser returns the number of non-archived notes shared with the viewer.
func CountSharedNotesForUser(db *DB, viewerID int64) int {
	var count int
	db.QueryRow(`
		SELECT COUNT(*) FROM note_shares s
		JOIN notes n ON n.id = s.note_id
		WHERE s.user_id = ? AND n.archived = 0`, viewerID).Scan(&count) //nolint:errcheck
	return count
}

// GetNoteForViewer resolves a note by (ownerUsername, slug) and determines the viewer's role.
// Role is one of RoleOwner, RoleEditor, RoleReader. Returns ErrNoAccess if the viewer has
// no relationship to the note, or sql.ErrNoRows if the note does not exist.
func GetNoteForViewer(db *DB, viewerID int64, ownerUsername, slug string) (*Note, string, error) {
	row := db.QueryRow(`
		SELECT n.id, n.user_id, n.slug, n.title, n.archived, n.created_at, n.updated_at,
			CASE
				WHEN n.user_id = ? THEN 'owner'
				WHEN s.permission = 'edit' THEN 'editor'
				WHEN s.permission = 'read' THEN 'reader'
				ELSE ''
			END AS role
		FROM notes n
		JOIN users u ON u.id = n.user_id
		LEFT JOIN note_shares s ON s.note_id = n.id AND s.user_id = ?
		WHERE u.username = ? AND n.slug = ?`,
		viewerID, viewerID, ownerUsername, slug)

	var (
		n           Note
		archivedInt int
		createdAt   string
		updatedAt   string
		role        string
	)
	err := row.Scan(&n.ID, &n.UserID, &n.Slug, &n.Title, &archivedInt, &createdAt, &updatedAt, &role)
	if err == sql.ErrNoRows {
		return nil, "", sql.ErrNoRows
	}
	if err != nil {
		return nil, "", err
	}
	if role == "" {
		return nil, "", ErrNoAccess
	}
	n.Archived = archivedInt != 0
	n.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	n.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	n.Tags, _ = getTagsForNote(db, n.ID)
	return &n, role, nil
}

// ResolveWikiLinksForViewer resolves [[Title]] wiki-links for a shared-note reader.
// Links are created only for target notes that are ALSO shared with the viewer; all
// other [[Title]] occurrences render as plain text, preventing information leakage
// about the owner's private notes.
func ResolveWikiLinksForViewer(db *DB, viewerID, ownerID int64, ownerUsername, body string) string {
	return noteLinkRe.ReplaceAllStringFunc(body, func(match string) string {
		title := strings.TrimSpace(match[2 : len(match)-2])

		var slug string
		err := db.QueryRow(`
			SELECT n.slug
			FROM notes n
			JOIN note_shares s ON s.note_id = n.id
			WHERE n.user_id = ? AND LOWER(n.title) = LOWER(?)
			  AND s.user_id = ? AND n.archived = 0`,
			ownerID, title, viewerID).Scan(&slug)
		if err != nil {
			return title
		}
		return fmt.Sprintf("[%s](/shared/%s/%s)", title, ownerUsername, slug)
	})
}

// UpdateNoteByID updates a note's title and updated_at timestamp, bypassing ownership check.
// Callers MUST have independently authorized the edit (typically via GetNoteForViewer).
func UpdateNoteByID(db *DB, noteID int64, title string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec("UPDATE notes SET title = ?, updated_at = ? WHERE id = ?", title, now, noteID)
	return err
}
