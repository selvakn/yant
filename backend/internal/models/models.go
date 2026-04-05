package models

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	_ "modernc.org/sqlite"
)

// DB wraps *sql.DB.
type DB struct {
	*sql.DB
}

// User represents an application user.
type User struct {
	ID        int64
	Username  string
	CreatedAt time.Time
}

// Note represents a note's metadata (body lives on disk).
type Note struct {
	ID        int64
	UserID    int64
	Slug      string
	Title     string
	Tags      []string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Image tracks an uploaded image file.
type Image struct {
	ID       int64
	NoteID   int64
	Filename string
	Original string
	MimeType string
	Size     int64
}

// TagCount holds a tag name and the number of notes with that tag.
type TagCount struct {
	Name  string
	Count int
}

// Open opens (or creates) the SQLite database at path.
func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // SQLite WAL is fine with 1 writer
	return &DB{db}, nil
}

// InitSchema creates tables and indexes if they do not exist.
func InitSchema(db *DB) error {
	_, err := db.Exec(`
		PRAGMA journal_mode=WAL;
		PRAGMA foreign_keys=ON;

		CREATE TABLE IF NOT EXISTS users (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			username   TEXT    UNIQUE NOT NULL,
			created_at TEXT    NOT NULL
		);

		CREATE TABLE IF NOT EXISTS notes (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id    INTEGER NOT NULL REFERENCES users(id),
			slug       TEXT    NOT NULL,
			title      TEXT    NOT NULL DEFAULT 'Untitled Note',
			created_at TEXT    NOT NULL,
			updated_at TEXT    NOT NULL,
			UNIQUE(user_id, slug)
		);

		CREATE TABLE IF NOT EXISTS note_tags (
			note_id  INTEGER NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
			tag_name TEXT    NOT NULL,
			PRIMARY KEY(note_id, tag_name)
		);

		CREATE TABLE IF NOT EXISTS images (
			id        INTEGER PRIMARY KEY AUTOINCREMENT,
			note_id   INTEGER NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
			filename  TEXT    NOT NULL,
			original  TEXT    NOT NULL,
			mime_type TEXT    NOT NULL,
			size      INTEGER NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_note_user      ON notes(user_id);
		CREATE INDEX IF NOT EXISTS idx_tag_name_note  ON note_tags(tag_name, note_id);
		CREATE INDEX IF NOT EXISTS idx_image_note     ON images(note_id);
	`)
	return err
}

// ── Users ────────────────────────────────────────────────────────────────────

func GetUserByUsername(db *DB, username string) (*User, error) {
	row := db.QueryRow(
		`SELECT id, username, created_at FROM users WHERE username = ?`, username)
	return scanUser(row)
}

func CreateUser(db *DB, username string) (*User, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Exec(
		`INSERT INTO users(username, created_at) VALUES(?, ?)`, username, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &User{ID: id, Username: username, CreatedAt: time.Now().UTC()}, nil
}

func GetOrCreateUser(db *DB, username string) (*User, error) {
	u, err := GetUserByUsername(db, username)
	if err == sql.ErrNoRows {
		return CreateUser(db, username)
	}
	return u, err
}

func scanUser(row *sql.Row) (*User, error) {
	var u User
	var createdAt string
	err := row.Scan(&u.ID, &u.Username, &createdAt)
	if err != nil {
		return nil, err
	}
	u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &u, nil
}

// ── Notes ────────────────────────────────────────────────────────────────────

// GenerateSlug creates a URL-safe slug from title, appending -2, -3 on collision.
func GenerateSlug(db *DB, userID int64, title string) (string, error) {
	base := slugify(title)
	if base == "" {
		base = "untitled-note"
	}
	slug := base
	for i := 2; ; i++ {
		var count int
		err := db.QueryRow(
			`SELECT COUNT(*) FROM notes WHERE user_id=? AND slug=?`, userID, slug).Scan(&count)
		if err != nil {
			return "", err
		}
		if count == 0 {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
}

func slugify(s string) string {
	if s == "" {
		return ""
	}
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else if unicode.IsSpace(r) || r == '-' || r == '_' {
			b.WriteRune('-')
		}
	}
	// Collapse multiple dashes
	re := regexp.MustCompile(`-{2,}`)
	result := re.ReplaceAllString(b.String(), "-")
	return strings.Trim(result, "-")
}

func CreateNote(db *DB, userID int64, title, slug string) (*Note, error) {
	if title == "" {
		title = "Untitled Note"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Exec(
		`INSERT INTO notes(user_id, slug, title, created_at, updated_at) VALUES(?,?,?,?,?)`,
		userID, slug, title, now, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	t, _ := time.Parse(time.RFC3339, now)
	return &Note{
		ID: id, UserID: userID, Slug: slug, Title: title,
		CreatedAt: t, UpdatedAt: t,
	}, nil
}

func GetNote(db *DB, userID int64, slug string) (*Note, error) {
	row := db.QueryRow(
		`SELECT id, user_id, slug, title, created_at, updated_at FROM notes WHERE user_id=? AND slug=?`,
		userID, slug)
	n, err := scanNote(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	n.Tags, err = getTagsForNote(db, n.ID)
	return n, err
}

func ListNotes(db *DB, userID int64, tag string) ([]*Note, error) {
	var rows *sql.Rows
	var err error
	if tag == "" {
		rows, err = db.Query(
			`SELECT id, user_id, slug, title, created_at, updated_at FROM notes WHERE user_id=? ORDER BY updated_at DESC`,
			userID)
	} else {
		rows, err = db.Query(
			`SELECT n.id, n.user_id, n.slug, n.title, n.created_at, n.updated_at
			 FROM notes n JOIN note_tags t ON t.note_id=n.id
			 WHERE n.user_id=? AND t.tag_name=? ORDER BY n.updated_at DESC`,
			userID, strings.ToLower(tag))
	}
	if err != nil {
		return nil, err
	}

	// Collect all notes before closing rows — avoids nested query deadlock with MaxOpenConns(1)
	var notes []*Note
	for rows.Next() {
		n, err := scanNoteRow(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		notes = append(notes, n)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Now fetch tags in a separate pass (connection is free)
	for _, n := range notes {
		n.Tags, _ = getTagsForNote(db, n.ID)
	}
	return notes, nil
}

func UpdateNote(db *DB, userID int64, slug, title string) (*Note, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(
		`UPDATE notes SET title=?, updated_at=? WHERE user_id=? AND slug=?`,
		title, now, userID, slug)
	if err != nil {
		return nil, err
	}
	return GetNote(db, userID, slug)
}

func DeleteNote(db *DB, userID int64, slug string) error {
	_, err := db.Exec(`DELETE FROM notes WHERE user_id=? AND slug=?`, userID, slug)
	return err
}

func scanNote(row *sql.Row) (*Note, error) {
	var n Note
	var ca, ua string
	err := row.Scan(&n.ID, &n.UserID, &n.Slug, &n.Title, &ca, &ua)
	if err != nil {
		return nil, err
	}
	n.CreatedAt, _ = time.Parse(time.RFC3339, ca)
	n.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
	return &n, nil
}

func scanNoteRow(rows *sql.Rows) (*Note, error) {
	var n Note
	var ca, ua string
	err := rows.Scan(&n.ID, &n.UserID, &n.Slug, &n.Title, &ca, &ua)
	if err != nil {
		return nil, err
	}
	n.CreatedAt, _ = time.Parse(time.RFC3339, ca)
	n.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
	return &n, nil
}

func getTagsForNote(db *DB, noteID int64) ([]string, error) {
	rows, err := db.Query(`SELECT tag_name FROM note_tags WHERE note_id=? ORDER BY tag_name`, noteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tags []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

// ── Tags ─────────────────────────────────────────────────────────────────────

// Tag tokens in note bodies: #word with letters, digits, underscore, hyphen.
// Keep in sync with tag parsing in frontend editor (notes/editor.html).
var tagRe = regexp.MustCompile(`#([a-zA-Z0-9_-]+)`)

// ParseTags extracts unique lowercase hashtag names from Markdown body.
func ParseTags(body string) []string {
	matches := tagRe.FindAllStringSubmatch(body, -1)
	seen := make(map[string]struct{})
	var tags []string
	for _, m := range matches {
		t := strings.ToLower(m[1])
		if _, ok := seen[t]; !ok {
			seen[t] = struct{}{}
			tags = append(tags, t)
		}
	}
	return tags
}

// SyncTags replaces all tag associations for a note.
func SyncTags(db *DB, noteID int64, tags []string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.Exec(`DELETE FROM note_tags WHERE note_id=?`, noteID); err != nil {
		return err
	}
	for _, t := range tags {
		if _, err := tx.Exec(`INSERT INTO note_tags(note_id, tag_name) VALUES(?,?)`, noteID, t); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ListTagsForUser returns all tag names and note counts for a user.
func ListTagsForUser(db *DB, userID int64) ([]TagCount, error) {
	rows, err := db.Query(
		`SELECT t.tag_name, COUNT(*) as cnt
		 FROM note_tags t JOIN notes n ON n.id=t.note_id
		 WHERE n.user_id=?
		 GROUP BY t.tag_name ORDER BY cnt DESC, t.tag_name ASC`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []TagCount
	for rows.Next() {
		var tc TagCount
		if err := rows.Scan(&tc.Name, &tc.Count); err != nil {
			return nil, err
		}
		result = append(result, tc)
	}
	return result, rows.Err()
}

// ── Images ───────────────────────────────────────────────────────────────────

func CreateImage(db *DB, noteID int64, filename, original, mimeType string, size int64) (*Image, error) {
	res, err := db.Exec(
		`INSERT INTO images(note_id, filename, original, mime_type, size) VALUES(?,?,?,?,?)`,
		noteID, filename, original, mimeType, size)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Image{ID: id, NoteID: noteID, Filename: filename, Original: original, MimeType: mimeType, Size: size}, nil
}

func GetImagesForNote(db *DB, noteID int64) ([]*Image, error) {
	rows, err := db.Query(`SELECT id, note_id, filename, original, mime_type, size FROM images WHERE note_id=?`, noteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var imgs []*Image
	for rows.Next() {
		var img Image
		if err := rows.Scan(&img.ID, &img.NoteID, &img.Filename, &img.Original, &img.MimeType, &img.Size); err != nil {
			return nil, err
		}
		imgs = append(imgs, &img)
	}
	return imgs, rows.Err()
}

// DeleteImagesForNote removes image records for a note and returns filenames to delete from disk.
func DeleteImagesForNote(db *DB, noteID int64) ([]string, error) {
	rows, err := db.Query(`SELECT filename FROM images WHERE note_id=?`, noteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var filenames []string
	for rows.Next() {
		var fn string
		if err := rows.Scan(&fn); err != nil {
			return nil, err
		}
		filenames = append(filenames, fn)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	_, err = db.Exec(`DELETE FROM images WHERE note_id=?`, noteID)
	return filenames, err
}

// ── Rebuild ──────────────────────────────────────────────────────────────────

// RebuildDB rebuilds the SQLite index from markdown files on disk.
func RebuildDB(db *DB, notesRoot, uploadsRoot string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	for _, tbl := range []string{"images", "note_tags", "notes", "users"} {
		if _, err := tx.Exec(`DELETE FROM ` + tbl); err != nil {
			return err
		}
	}

	userDirs, err := os.ReadDir(notesRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return tx.Commit()
		}
		return err
	}

	titleRe := regexp.MustCompile(`(?m)^#\s+(.+)`)

	for _, ud := range userDirs {
		if !ud.IsDir() {
			continue
		}
		// Insert user
		now := time.Now().UTC().Format(time.RFC3339)
		res, err := tx.Exec(`INSERT OR IGNORE INTO users(username, created_at) VALUES(?,?)`, ud.Name(), now)
		if err != nil {
			return err
		}
		userID, _ := res.LastInsertId()
		if userID == 0 {
			tx.QueryRow(`SELECT id FROM users WHERE username=?`, ud.Name()).Scan(&userID) //nolint:errcheck
		}

		mdFiles, err := filepath.Glob(filepath.Join(notesRoot, ud.Name(), "*.md"))
		if err != nil {
			return err
		}
		for _, f := range mdFiles {
			info, err := os.Stat(f)
			if err != nil {
				continue
			}
			slug := strings.TrimSuffix(filepath.Base(f), ".md")
			content, err := os.ReadFile(f)
			if err != nil {
				continue
			}
			body := string(content)
			title := slug
			if m := titleRe.FindStringSubmatch(body); len(m) > 1 {
				title = strings.TrimSpace(m[1])
			}
			mtime := info.ModTime().UTC().Format(time.RFC3339)
			res2, err := tx.Exec(
				`INSERT OR IGNORE INTO notes(user_id,slug,title,created_at,updated_at) VALUES(?,?,?,?,?)`,
				userID, slug, title, mtime, mtime)
			if err != nil {
				return err
			}
			noteID, _ := res2.LastInsertId()
			for _, tag := range ParseTags(body) {
				tx.Exec(`INSERT OR IGNORE INTO note_tags(note_id,tag_name) VALUES(?,?)`, noteID, tag) //nolint:errcheck
			}
		}
	}
	return tx.Commit()
}
