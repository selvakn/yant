package models

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	_ "modernc.org/sqlite"
	_ "modernc.org/sqlite/vec"
)

// DB wraps *sql.DB.
type DB struct {
	*sql.DB
}

// User represents an application user.
type User struct {
	ID        int64
	Username  string
	IsAdmin   bool
	Disabled  bool
	CreatedAt time.Time
}

// Note represents a note's metadata (body lives on disk).
type Note struct {
	ID        int64
	UserID    int64
	Slug      string
	Title     string
	Tags      []string
	Archived  bool
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

type NoteDrawing struct {
	DrawingID   string
	NoteID      int64
	DisplayName string
	ToolType    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

const drawingIDLength = 8
const drawingIDChars = "abcdefghijklmnopqrstuvwxyz0123456789"

func GenerateDrawingID() string {
	b := make([]byte, drawingIDLength)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(drawingIDChars))))
		b[i] = drawingIDChars[n.Int64()]
	}
	return string(b)
}

// TagCount holds a tag name, count, and color.
type TagCount struct {
	Name  string
	Count int
	Color string
}

// BlogPost represents a note published as a blog post.
type BlogPost struct {
	Note        *Note
	Username    string
	PublishedAt time.Time
	Excerpt     string
	Tags        []string
}

// ColorPalette is the fixed 10-color palette for tags.
var ColorPalette = []string{
	"#001219", // Ink Black
	"#005f73", // Dark Teal
	"#0a9396", // Dark Cyan
	"#94d2bd", // Pearl Aqua
	"#e9d8a6", // Vanilla Custard
	"#ee9b00", // Golden Orange
	"#ca6702", // Burnt Caramel
	"#bb3e03", // Rusty Spice
	"#ae2012", // Oxidized Iron
	"#9b2226", // Brown Red
}

// Resource-limit constants. Enforced server-side; hardcoded in this version.
const MaxNotesPerUser = 25

// ErrNoteLimitReached is returned by CreateNote when a regular user has reached the note cap.
var ErrNoteLimitReached = errors.New("note limit reached")

// AutoTagColor returns a deterministic color for a tag name using hash.
func AutoTagColor(tagName string) string {
	h := uint32(0)
	for _, c := range tagName {
		h = h*31 + uint32(c)
	}
	return ColorPalette[int(h)%len(ColorPalette)]
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
			archived   INTEGER NOT NULL DEFAULT 0,
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

		CREATE TABLE IF NOT EXISTS tag_colors (
			user_id  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			tag_name TEXT    NOT NULL,
			color    TEXT    NOT NULL,
			PRIMARY KEY(user_id, tag_name)
		);

		CREATE TABLE IF NOT EXISTS note_links (
			source_note_id INTEGER NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
			target_note_id INTEGER NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
			PRIMARY KEY(source_note_id, target_note_id)
		);

		CREATE TABLE IF NOT EXISTS note_todos (
			note_id   INTEGER NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
			line      INTEGER NOT NULL,
			text      TEXT    NOT NULL,
			due_date  TEXT,
			completed BOOLEAN NOT NULL DEFAULT 0,
			PRIMARY KEY (note_id, line)
		);

		CREATE TABLE IF NOT EXISTS public_notes (
			note_id      INTEGER PRIMARY KEY REFERENCES notes(id) ON DELETE CASCADE,
			token        TEXT    NOT NULL UNIQUE,
			published    BOOLEAN NOT NULL DEFAULT 1,
			published_at TEXT    NOT NULL,
			updated_at   TEXT    NOT NULL
		);

		CREATE TABLE IF NOT EXISTS note_shares (
			note_id     INTEGER NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
			user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			permission  TEXT    NOT NULL CHECK (permission IN ('read','edit')),
			granted_at  TEXT    NOT NULL,
			granted_by  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			PRIMARY KEY (note_id, user_id)
		);

		CREATE TABLE IF NOT EXISTS sessions (
			token  TEXT PRIMARY KEY,
			data   BLOB NOT NULL,
			expiry REAL NOT NULL
		);
		CREATE INDEX IF NOT EXISTS sessions_expiry_idx ON sessions (expiry);

		CREATE INDEX IF NOT EXISTS idx_note_user      ON notes(user_id);
		CREATE INDEX IF NOT EXISTS idx_tag_name_note  ON note_tags(tag_name, note_id);
		CREATE INDEX IF NOT EXISTS idx_image_note     ON images(note_id);
		CREATE INDEX IF NOT EXISTS idx_note_links_target ON note_links(target_note_id);
		CREATE INDEX IF NOT EXISTS idx_note_todos_pending ON note_todos(completed, due_date);
		CREATE INDEX IF NOT EXISTS idx_public_notes_token ON public_notes (token);
		CREATE INDEX IF NOT EXISTS idx_note_shares_user ON note_shares (user_id);
	`)
	if err != nil {
		return err
	}

	return migrateSchema(db)
}

// migrateSchema applies incremental schema changes to existing databases.
func migrateSchema(db *DB) error {
	// Add 'archived' column if missing (007-note-archive)
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('notes') WHERE name='archived'`).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		if _, err := db.Exec(`ALTER TABLE notes ADD COLUMN archived INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_note_archived ON notes(user_id, archived)`)
	if err != nil {
		return err
	}

	// Add note_embeddings table for semantic search (011-semantic-search)
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS note_embeddings (
		note_id      INTEGER PRIMARY KEY REFERENCES notes(id) ON DELETE CASCADE,
		content_hash TEXT    NOT NULL,
		updated_at   TEXT    NOT NULL
	)`)
	if err != nil {
		return err
	}

	// sqlite-vec virtual table for KNN search
	_, err = db.Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS vec_note_embeddings USING vec0(
		note_id INTEGER PRIMARY KEY,
		embedding float[384] distance_metric=cosine
	)`)
	if err != nil {
		return err
	}

	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('users') WHERE name='is_admin'`).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		if _, err := db.Exec(`ALTER TABLE users ADD COLUMN is_admin INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}

	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('users') WHERE name='disabled'`).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		if _, err := db.Exec(`ALTER TABLE users ADD COLUMN disabled INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_users_admin ON users(is_admin)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS admin_audit_log (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		admin_username  TEXT    NOT NULL,
		action          TEXT    NOT NULL,
		target_type     TEXT    NOT NULL,
		target_id       TEXT    NOT NULL,
		details         TEXT,
		created_at      TEXT    NOT NULL
	)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_audit_log_created ON admin_audit_log(created_at)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_audit_log_action ON admin_audit_log(action)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS note_drawings (
		drawing_id   TEXT    NOT NULL,
		note_id      INTEGER NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
		display_name TEXT    NOT NULL,
		tool_type    TEXT    NOT NULL CHECK (tool_type IN ('tldraw','excalidraw')),
		created_at   TEXT    NOT NULL,
		updated_at   TEXT    NOT NULL,
		PRIMARY KEY (drawing_id, note_id)
	)`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_note_drawings_note ON note_drawings(note_id)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS blog_posts (
		note_id      INTEGER PRIMARY KEY REFERENCES notes(id) ON DELETE CASCADE,
		published_at TEXT    NOT NULL
	)`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_blog_posts_published ON blog_posts(published_at DESC)`)
	if err != nil {
		return err
	}

	// Add size_bytes column to notes for per-user storage tracking (026-resource-limits-defense)
	err = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('notes') WHERE name='size_bytes'`).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		if _, err := db.Exec(`ALTER TABLE notes ADD COLUMN size_bytes INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}

	return nil
}

// ── Users ────────────────────────────────────────────────────────────────────

func GetUserByUsername(db *DB, username string) (*User, error) {
	row := db.QueryRow(
		`SELECT id, username, created_at, is_admin, disabled FROM users WHERE username = ?`, username)
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
	nowT := time.Now().UTC()
	return &User{ID: id, Username: username, IsAdmin: false, Disabled: false, CreatedAt: nowT}, nil
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
	var isAdmin, disabled int
	err := row.Scan(&u.ID, &u.Username, &createdAt, &isAdmin, &disabled)
	if err != nil {
		return nil, err
	}
	u.IsAdmin = isAdmin == 1
	u.Disabled = disabled == 1
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

// CreateNote creates a note for userID. For non-admin users the total note count must be
// below MaxNotesPerUser; if the limit is reached ErrNoteLimitReached is returned. The
// count check and insert run inside a single transaction so concurrent requests cannot
// both slip past the limit (MaxOpenConns(1) serialises all DB access).
func CreateNote(db *DB, userID int64, title, slug string, sizeBytes int64, isAdmin bool) (*Note, error) {
	if title == "" {
		title = "Untitled Note"
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	if !isAdmin {
		var count int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM notes WHERE user_id = ?`, userID).Scan(&count); err != nil {
			return nil, err
		}
		if count >= MaxNotesPerUser {
			return nil, ErrNoteLimitReached
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := tx.Exec(
		`INSERT INTO notes(user_id, slug, title, size_bytes, created_at, updated_at) VALUES(?,?,?,?,?,?)`,
		userID, slug, title, sizeBytes, now, now)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
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
		`SELECT id, user_id, slug, title, archived, created_at, updated_at FROM notes WHERE user_id=? AND slug=?`,
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

func ListNotes(db *DB, userID int64, tag string, archived bool) ([]*Note, error) {
	var rows *sql.Rows
	var err error
	archivedVal := 0
	if archived {
		archivedVal = 1
	}
	if tag == "" {
		rows, err = db.Query(
			`SELECT id, user_id, slug, title, archived, created_at, updated_at FROM notes WHERE user_id=? AND archived=? ORDER BY updated_at DESC`,
			userID, archivedVal)
	} else {
		rows, err = db.Query(
			`SELECT n.id, n.user_id, n.slug, n.title, n.archived, n.created_at, n.updated_at
			 FROM notes n JOIN note_tags t ON t.note_id=n.id
			 WHERE n.user_id=? AND n.archived=? AND t.tag_name=? ORDER BY n.updated_at DESC`,
			userID, archivedVal, strings.ToLower(tag))
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

// ArchiveNote sets archived=1 for the specified note.
func ArchiveNote(db *DB, userID int64, slug string) error {
	res, err := db.Exec(`UPDATE notes SET archived=1 WHERE user_id=? AND slug=?`, userID, slug)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// RestoreNote sets archived=0 for the specified note.
func RestoreNote(db *DB, userID int64, slug string) error {
	res, err := db.Exec(`UPDATE notes SET archived=0 WHERE user_id=? AND slug=?`, userID, slug)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func UpdateNote(db *DB, userID int64, slug, title string, sizeBytes int64) (*Note, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(
		`UPDATE notes SET title=?, size_bytes=?, updated_at=? WHERE user_id=? AND slug=?`,
		title, sizeBytes, now, userID, slug)
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
	var archived int
	err := row.Scan(&n.ID, &n.UserID, &n.Slug, &n.Title, &archived, &ca, &ua)
	if err != nil {
		return nil, err
	}
	n.Archived = archived == 1
	n.CreatedAt, _ = time.Parse(time.RFC3339, ca)
	n.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
	return &n, nil
}

func scanNoteRow(rows *sql.Rows) (*Note, error) {
	var n Note
	var ca, ua string
	var archived int
	err := rows.Scan(&n.ID, &n.UserID, &n.Slug, &n.Title, &archived, &ca, &ua)
	if err != nil {
		return nil, err
	}
	n.Archived = archived == 1
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
	if err := tx.Commit(); err != nil {
		return err
	}

	hasBlogTag := false
	for _, t := range tags {
		if t == "blog" {
			hasBlogTag = true
			break
		}
	}
	if hasBlogTag {
		_ = PublishBlogPost(db, noteID)
	} else {
		_ = UnpublishBlogPost(db, noteID)
	}
	return nil
}

// ListTagsForUser returns all tag names, counts, and colors for a user.
// If archived is true, returns tags from archived notes only; otherwise from active notes.
func ListTagsForUser(db *DB, userID int64, archived bool) ([]TagCount, error) {
	archivedVal := 0
	if archived {
		archivedVal = 1
	}
	rows, err := db.Query(
		`SELECT t.tag_name, COUNT(*) as cnt, COALESCE(c.color, '') as color
		 FROM note_tags t
		 JOIN notes n ON n.id=t.note_id
		 LEFT JOIN tag_colors c ON c.user_id=n.user_id AND c.tag_name=t.tag_name
		 WHERE n.user_id=? AND n.archived=?
		 GROUP BY t.tag_name ORDER BY cnt DESC, t.tag_name ASC`,
		userID, archivedVal)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []TagCount
	for rows.Next() {
		var tc TagCount
		if err := rows.Scan(&tc.Name, &tc.Count, &tc.Color); err != nil {
			return nil, err
		}
		if tc.Color == "" {
			tc.Color = AutoTagColor(tc.Name)
		}
		result = append(result, tc)
	}
	return result, rows.Err()
}

// GetTagColor returns the color for a tag, using auto-assignment if not set.
func GetTagColor(db *DB, userID int64, tagName string) string {
	var color string
	err := db.QueryRow(
		`SELECT color FROM tag_colors WHERE user_id=? AND tag_name=?`,
		userID, tagName).Scan(&color)
	if err != nil || color == "" {
		return AutoTagColor(tagName)
	}
	return color
}

// SetTagColor sets or updates the color for a tag.
func SetTagColor(db *DB, userID int64, tagName, color string) error {
	_, err := db.Exec(
		`INSERT INTO tag_colors(user_id, tag_name, color) VALUES(?,?,?)
		 ON CONFLICT(user_id, tag_name) DO UPDATE SET color=excluded.color`,
		userID, tagName, color)
	return err
}

// ── Note Links ───────────────────────────────────────────────────────────────

// noteLinkRe matches [[note title]] wiki-link syntax.
var noteLinkRe = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

// replaceNoteWikiLinks replaces each [[title]] via resolveFn. Segments that are part of a
// drawing embed (![[draw:<id>]]) are copied verbatim so markdown can parse drawing markers.
func replaceNoteWikiLinks(body string, resolveFn func(title string) string) string {
	var b strings.Builder
	last := 0
	for _, loc := range noteLinkRe.FindAllStringIndex(body, -1) {
		start, end := loc[0], loc[1]
		if start > 0 && body[start-1] == '!' {
			b.WriteString(body[last:end])
			last = end
			continue
		}
		b.WriteString(body[last:start])
		match := body[start:end]
		title := strings.TrimSpace(match[2 : len(match)-2])
		b.WriteString(resolveFn(title))
		last = end
	}
	b.WriteString(body[last:])
	return b.String()
}

// ParseNoteLinks extracts unique linked note titles from markdown body.
func ParseNoteLinks(body string) []string {
	seen := make(map[string]struct{})
	var titles []string
	for _, loc := range noteLinkRe.FindAllStringIndex(body, -1) {
		start, end := loc[0], loc[1]
		if start > 0 && body[start-1] == '!' {
			continue
		}
		match := body[start:end]
		t := strings.TrimSpace(match[2 : len(match)-2])
		lower := strings.ToLower(t)
		if _, ok := seen[lower]; !ok && t != "" {
			seen[lower] = struct{}{}
			titles = append(titles, t)
		}
	}
	return titles
}

// SyncLinks replaces all outgoing links for a note based on [[title]] references.
func SyncLinks(db *DB, sourceNoteID, userID int64, linkedTitles []string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.Exec(`DELETE FROM note_links WHERE source_note_id=?`, sourceNoteID); err != nil {
		return err
	}
	for _, title := range linkedTitles {
		var targetID int64
		err := tx.QueryRow(
			`SELECT id FROM notes WHERE user_id=? AND LOWER(title)=LOWER(?)`,
			userID, title).Scan(&targetID)
		if err != nil {
			continue
		}
		if targetID == sourceNoteID {
			continue
		}
		tx.Exec(`INSERT OR IGNORE INTO note_links(source_note_id, target_note_id) VALUES(?,?)`, //nolint:errcheck
			sourceNoteID, targetID)
	}
	return tx.Commit()
}

// BacklinkNote holds minimal info for a backlink entry.
type BacklinkNote struct {
	Slug  string
	Title string
}

// GetBacklinks returns notes that link to the given note.
func GetBacklinks(db *DB, noteID int64) ([]BacklinkNote, error) {
	rows, err := db.Query(
		`SELECT n.slug, n.title FROM note_links l
		 JOIN notes n ON n.id = l.source_note_id
		 WHERE l.target_note_id = ?
		 ORDER BY n.title ASC`, noteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []BacklinkNote
	for rows.Next() {
		var b BacklinkNote
		if err := rows.Scan(&b.Slug, &b.Title); err != nil {
			return nil, err
		}
		result = append(result, b)
	}
	return result, rows.Err()
}

// ResolveNoteLink resolves a note title to a slug for a given user (case-insensitive).
func ResolveNoteLink(db *DB, userID int64, title string) (string, bool) {
	var slug string
	err := db.QueryRow(
		`SELECT slug FROM notes WHERE user_id=? AND LOWER(title)=LOWER(?)`,
		userID, title).Scan(&slug)
	if err != nil {
		return "", false
	}
	return slug, true
}

// SearchNotesByTitle returns notes matching a title prefix (for autocomplete).
func SearchNotesByTitle(db *DB, userID int64, query string) ([]BacklinkNote, error) {
	rows, err := db.Query(
		`SELECT slug, title FROM notes WHERE user_id=? AND LOWER(title) LIKE LOWER(?) ORDER BY title ASC LIMIT 10`,
		userID, "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []BacklinkNote
	for rows.Next() {
		var b BacklinkNote
		if err := rows.Scan(&b.Slug, &b.Title); err != nil {
			return nil, err
		}
		result = append(result, b)
	}
	return result, rows.Err()
}

// ResolveWikiLinks replaces [[title]] in markdown with [title](/notes/slug) links
// for titles that match an existing note, or leaves them as plain text otherwise.
func ResolveWikiLinks(db *DB, userID int64, body string) string {
	return replaceNoteWikiLinks(body, func(title string) string {
		if slug, ok := ResolveNoteLink(db, userID, title); ok {
			return fmt.Sprintf("[%s](/notes/%s)", title, slug)
		}
		return title
	})
}

// CountNotesForUser returns the total number of notes owned by userID.
func CountNotesForUser(db *DB, userID int64) (int, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM notes WHERE user_id = ?`, userID).Scan(&n)
	return n, err
}

// CountImagesForNote returns the lifetime number of images uploaded to noteID.
// Deleted images are not removed from the DB, so this gives the cumulative count.
func CountImagesForNote(db *DB, noteID int64) (int, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM images WHERE note_id = ?`, noteID).Scan(&n)
	return n, err
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

// ── Note drawings ────────────────────────────────────────────────────────────

const maxDrawingNameLength = 100

func CreateDrawing(db *DB, noteID int64, displayName, toolType string) (NoteDrawing, error) {
	if displayName == "" || len(displayName) > maxDrawingNameLength {
		return NoteDrawing{}, fmt.Errorf("display name must be 1-%d characters", maxDrawingNameLength)
	}
	if toolType != "tldraw" && toolType != "excalidraw" {
		return NoteDrawing{}, fmt.Errorf("invalid tool type: %s", toolType)
	}
	id := GenerateDrawingID()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(`INSERT INTO note_drawings (drawing_id, note_id, display_name, tool_type, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		id, noteID, displayName, toolType, now, now)
	if err != nil {
		return NoteDrawing{}, err
	}
	t, _ := time.Parse(time.RFC3339, now)
	return NoteDrawing{DrawingID: id, NoteID: noteID, DisplayName: displayName, ToolType: toolType, CreatedAt: t, UpdatedAt: t}, nil
}

func GetDrawing(db *DB, noteID int64, drawingID string) (*NoteDrawing, error) {
	var d NoteDrawing
	var created, updated string
	err := db.QueryRow(`SELECT drawing_id, note_id, display_name, tool_type, created_at, updated_at FROM note_drawings WHERE note_id=? AND drawing_id=?`, noteID, drawingID).
		Scan(&d.DrawingID, &d.NoteID, &d.DisplayName, &d.ToolType, &created, &updated)
	if err != nil {
		return nil, err
	}
	d.CreatedAt, _ = time.Parse(time.RFC3339, created)
	d.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return &d, nil
}

func ListDrawings(db *DB, noteID int64) ([]NoteDrawing, error) {
	rows, err := db.Query(`SELECT drawing_id, note_id, display_name, tool_type, created_at, updated_at FROM note_drawings WHERE note_id=? ORDER BY created_at`, noteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var drawings []NoteDrawing
	for rows.Next() {
		var d NoteDrawing
		var created, updated string
		if err := rows.Scan(&d.DrawingID, &d.NoteID, &d.DisplayName, &d.ToolType, &created, &updated); err != nil {
			return nil, err
		}
		d.CreatedAt, _ = time.Parse(time.RFC3339, created)
		d.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
		drawings = append(drawings, d)
	}
	return drawings, rows.Err()
}

func RenameDrawing(db *DB, noteID int64, drawingID, newName string) error {
	if newName == "" || len(newName) > maxDrawingNameLength {
		return fmt.Errorf("display name must be 1-%d characters", maxDrawingNameLength)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Exec(`UPDATE note_drawings SET display_name=?, updated_at=? WHERE note_id=? AND drawing_id=?`, newName, now, noteID, drawingID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("drawing not found")
	}
	return nil
}

func DeleteDrawingRecord(db *DB, noteID int64, drawingID string) error {
	_, err := db.Exec(`DELETE FROM note_drawings WHERE note_id=? AND drawing_id=?`, noteID, drawingID)
	return err
}

// UpsertDrawingRecord re-inserts a drawing record if it was deleted (e.g. during revert).
// It is a no-op if the record already exists, preserving any existing display name.
func UpsertDrawingRecord(db *DB, noteID int64, drawingID, toolType string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(
		`INSERT OR IGNORE INTO note_drawings (drawing_id, note_id, display_name, tool_type, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		drawingID, noteID, "Drawing", toolType, now, now,
	)
	return err
}

func UpdateDrawingTimestamp(db *DB, noteID int64, drawingID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(`UPDATE note_drawings SET updated_at=? WHERE note_id=? AND drawing_id=?`, now, noteID, drawingID)
	return err
}

func InsertDrawingRecord(db *DB, drawingID string, noteID int64, displayName, toolType string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(`INSERT OR IGNORE INTO note_drawings (drawing_id, note_id, display_name, tool_type, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		drawingID, noteID, displayName, toolType, now, now)
	return err
}

// ── Blog ─────────────────────────────────────────────────────────────────────

const defaultBlogPageSize = 10

// PublishBlogPost creates a blog_posts row for the note if one doesn't exist.
func PublishBlogPost(db *DB, noteID int64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(
		`INSERT OR IGNORE INTO blog_posts(note_id, published_at) VALUES(?, ?)`,
		noteID, now)
	return err
}

// UnpublishBlogPost removes the blog_posts row for a note.
func UnpublishBlogPost(db *DB, noteID int64) error {
	_, err := db.Exec(`DELETE FROM blog_posts WHERE note_id = ?`, noteID)
	return err
}

// IsBlogPost checks if a note has a blog_posts row.
func IsBlogPost(db *DB, noteID int64) bool {
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM blog_posts WHERE note_id = ?`, noteID).Scan(&count)
	return count > 0
}

// GetBlogPost retrieves a single blog post by username and slug.
func GetBlogPost(db *DB, slug string) (*BlogPost, error) {
	var n Note
	var ca, ua, pa string
	var archived int
	var uname string
	err := db.QueryRow(
		`SELECT n.id, n.user_id, n.slug, n.title, n.archived, n.created_at, n.updated_at,
		        u.username, bp.published_at
		 FROM blog_posts bp
		 JOIN notes n ON n.id = bp.note_id
		 JOIN users u ON u.id = n.user_id
		 WHERE n.slug = ? AND n.archived = 0
		 ORDER BY bp.published_at ASC LIMIT 1`,
		slug).Scan(
		&n.ID, &n.UserID, &n.Slug, &n.Title, &archived, &ca, &ua,
		&uname, &pa)
	if err != nil {
		return nil, err
	}
	n.Archived = archived == 1
	n.CreatedAt, _ = time.Parse(time.RFC3339, ca)
	n.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
	publishedAt, _ := time.Parse(time.RFC3339, pa)

	tags, _ := getTagsForNote(db, n.ID)
	filteredTags := filterBlogTag(tags)

	return &BlogPost{
		Note:        &n,
		Username:    uname,
		PublishedAt: publishedAt,
		Tags:        filteredTags,
	}, nil
}

// ListBlogPosts returns paginated blog posts ordered by published_at DESC.
func ListBlogPosts(db *DB, page, pageSize int) ([]*BlogPost, error) {
	if pageSize <= 0 {
		pageSize = defaultBlogPageSize
	}
	offset := (page - 1) * pageSize

	rows, err := db.Query(
		`SELECT n.id, n.user_id, n.slug, n.title, n.archived, n.created_at, n.updated_at,
		        u.username, bp.published_at
		 FROM blog_posts bp
		 JOIN notes n ON n.id = bp.note_id
		 JOIN users u ON u.id = n.user_id
		 WHERE n.archived = 0
		 ORDER BY bp.published_at DESC
		 LIMIT ? OFFSET ?`, pageSize, offset)
	if err != nil {
		return nil, err
	}
	return scanBlogPosts(db, rows)
}

// ListBlogPostsByTag returns paginated blog posts filtered by tag, ordered by published_at DESC.
func ListBlogPostsByTag(db *DB, tag string, page, pageSize int) ([]*BlogPost, error) {
	if pageSize <= 0 {
		pageSize = defaultBlogPageSize
	}
	offset := (page - 1) * pageSize

	rows, err := db.Query(
		`SELECT n.id, n.user_id, n.slug, n.title, n.archived, n.created_at, n.updated_at,
		        u.username, bp.published_at
		 FROM blog_posts bp
		 JOIN notes n ON n.id = bp.note_id
		 JOIN users u ON u.id = n.user_id
		 JOIN note_tags t ON t.note_id = n.id
		 WHERE n.archived = 0 AND t.tag_name = ?
		 ORDER BY bp.published_at DESC
		 LIMIT ? OFFSET ?`, strings.ToLower(tag), pageSize, offset)
	if err != nil {
		return nil, err
	}
	return scanBlogPosts(db, rows)
}

// CountBlogPosts returns the total number of non-archived blog posts.
func CountBlogPosts(db *DB) int {
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM blog_posts bp JOIN notes n ON n.id = bp.note_id WHERE n.archived = 0`).Scan(&count)
	return count
}

// CountBlogPostsForUser returns the number of non-archived blog posts owned by the given user.
func CountBlogPostsForUser(db *DB, userID int64) int {
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM blog_posts bp JOIN notes n ON n.id = bp.note_id WHERE n.archived = 0 AND n.user_id = ?`, userID).Scan(&count)
	return count
}

// CountBlogPostsByTag returns the count of non-archived blog posts with a specific tag.
func CountBlogPostsByTag(db *DB, tag string) int {
	var count int
	db.QueryRow(
		`SELECT COUNT(*) FROM blog_posts bp
		 JOIN notes n ON n.id = bp.note_id
		 JOIN note_tags t ON t.note_id = n.id
		 WHERE n.archived = 0 AND t.tag_name = ?`, strings.ToLower(tag)).Scan(&count)
	return count
}

// ListBlogTags returns all tags used by non-archived blog posts, excluding the "blog" tag itself.
func ListBlogTags(db *DB) []TagCount {
	rows, err := db.Query(
		`SELECT t.tag_name, COUNT(*) as cnt
		 FROM note_tags t
		 JOIN blog_posts bp ON bp.note_id = t.note_id
		 JOIN notes n ON n.id = t.note_id
		 WHERE n.archived = 0 AND t.tag_name != 'blog'
		 GROUP BY t.tag_name
		 ORDER BY cnt DESC, t.tag_name ASC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var result []TagCount
	for rows.Next() {
		var tc TagCount
		if err := rows.Scan(&tc.Name, &tc.Count); err != nil {
			return result
		}
		tc.Color = AutoTagColor(tc.Name)
		result = append(result, tc)
	}
	return result
}

// GetAdjacentBlogPosts returns the previous and next blog posts relative to the given published_at.
func GetAdjacentBlogPosts(db *DB, publishedAt time.Time) (prev *BlogPost, next *BlogPost) {
	pa := publishedAt.Format(time.RFC3339)

	// Previous (older) post
	var pn Note
	var pca, pua, ppa, puname string
	var parchived int
	err := db.QueryRow(
		`SELECT n.id, n.user_id, n.slug, n.title, n.archived, n.created_at, n.updated_at,
		        u.username, bp.published_at
		 FROM blog_posts bp
		 JOIN notes n ON n.id = bp.note_id
		 JOIN users u ON u.id = n.user_id
		 WHERE n.archived = 0 AND bp.published_at < ?
		 ORDER BY bp.published_at DESC LIMIT 1`, pa).Scan(
		&pn.ID, &pn.UserID, &pn.Slug, &pn.Title, &parchived, &pca, &pua, &puname, &ppa)
	if err == nil {
		pn.Archived = parchived == 1
		pn.CreatedAt, _ = time.Parse(time.RFC3339, pca)
		pn.UpdatedAt, _ = time.Parse(time.RFC3339, pua)
		ppat, _ := time.Parse(time.RFC3339, ppa)
		prev = &BlogPost{Note: &pn, Username: puname, PublishedAt: ppat}
	}

	// Next (newer) post
	var nn Note
	var nca, nua, npa, nuname string
	var narchived int
	err = db.QueryRow(
		`SELECT n.id, n.user_id, n.slug, n.title, n.archived, n.created_at, n.updated_at,
		        u.username, bp.published_at
		 FROM blog_posts bp
		 JOIN notes n ON n.id = bp.note_id
		 JOIN users u ON u.id = n.user_id
		 WHERE n.archived = 0 AND bp.published_at > ?
		 ORDER BY bp.published_at ASC LIMIT 1`, pa).Scan(
		&nn.ID, &nn.UserID, &nn.Slug, &nn.Title, &narchived, &nca, &nua, &nuname, &npa)
	if err == nil {
		nn.Archived = narchived == 1
		nn.CreatedAt, _ = time.Parse(time.RFC3339, nca)
		nn.UpdatedAt, _ = time.Parse(time.RFC3339, nua)
		npat, _ := time.Parse(time.RFC3339, npa)
		next = &BlogPost{Note: &nn, Username: nuname, PublishedAt: npat}
	}

	return prev, next
}

// ResolveWikiLinksForBlog replaces [[title]] in markdown:
// - If the target note is also a blog post → markdown link to /blog/<slug>
// - Otherwise → plain text (title only)
func ResolveWikiLinksForBlog(db *DB, userID int64, body string) string {
	return replaceNoteWikiLinks(body, func(title string) string {
		slug, ok := ResolveNoteLink(db, userID, title)
		if !ok {
			return title
		}
		var noteID int64
		err := db.QueryRow(`SELECT id FROM notes WHERE user_id = ? AND slug = ?`, userID, slug).Scan(&noteID)
		if err != nil {
			return title
		}
		if IsBlogPost(db, noteID) {
			return fmt.Sprintf("[%s](/blog/%s)", title, slug)
		}
		return title
	})
}

func scanBlogPosts(db *DB, rows *sql.Rows) ([]*BlogPost, error) {
	defer rows.Close()
	var posts []*BlogPost
	for rows.Next() {
		var n Note
		var ca, ua, pa string
		var archived int
		var username string
		if err := rows.Scan(&n.ID, &n.UserID, &n.Slug, &n.Title, &archived, &ca, &ua, &username, &pa); err != nil {
			return nil, err
		}
		n.Archived = archived == 1
		n.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		n.UpdatedAt, _ = time.Parse(time.RFC3339, ua)
		publishedAt, _ := time.Parse(time.RFC3339, pa)
		posts = append(posts, &BlogPost{
			Note:        &n,
			Username:    username,
			PublishedAt: publishedAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, bp := range posts {
		tags, _ := getTagsForNote(db, bp.Note.ID)
		bp.Tags = filterBlogTag(tags)
	}
	return posts, nil
}

func filterBlogTag(tags []string) []string {
	filtered := make([]string, 0, len(tags))
	for _, t := range tags {
		if t != "blog" {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// ── Rebuild ──────────────────────────────────────────────────────────────────

// RebuildDB rebuilds the SQLite index from markdown files on disk.
func RebuildDB(db *DB, notesRoot, uploadsRoot string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	for _, tbl := range []string{"blog_posts", "note_todos", "note_links", "images", "note_tags", "notes", "users"} {
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

	// Second pass: rebuild note links from [[title]] references
	noteRows, err := tx.Query(`SELECT id, user_id FROM notes`)
	if err != nil {
		return err
	}
	type noteRef struct {
		id     int64
		userID int64
	}
	var allNotes []noteRef
	for noteRows.Next() {
		var nr noteRef
		if err := noteRows.Scan(&nr.id, &nr.userID); err != nil {
			noteRows.Close()
			return err
		}
		allNotes = append(allNotes, nr)
	}
	noteRows.Close()

	for _, nr := range allNotes {
		var slug string
		tx.QueryRow(`SELECT slug FROM notes WHERE id=?`, nr.id).Scan(&slug) //nolint:errcheck
		// Find user directory name
		var username string
		tx.QueryRow(`SELECT username FROM users WHERE id=?`, nr.userID).Scan(&username) //nolint:errcheck
		mdPath := filepath.Join(notesRoot, username, slug+".md")
		content, err := os.ReadFile(mdPath)
		if err != nil {
			continue
		}
		for _, title := range ParseNoteLinks(string(content)) {
			var targetID int64
			err := tx.QueryRow(
				`SELECT id FROM notes WHERE user_id=? AND LOWER(title)=LOWER(?)`,
				nr.userID, title).Scan(&targetID)
			if err != nil || targetID == nr.id {
				continue
			}
			tx.Exec(`INSERT OR IGNORE INTO note_links(source_note_id, target_note_id) VALUES(?,?)`, nr.id, targetID) //nolint:errcheck
		}
	}

	// Third pass: rebuild todos from markdown content
	for _, nr := range allNotes {
		var slug string
		tx.QueryRow(`SELECT slug FROM notes WHERE id=?`, nr.id).Scan(&slug) //nolint:errcheck
		var username string
		tx.QueryRow(`SELECT username FROM users WHERE id=?`, nr.userID).Scan(&username) //nolint:errcheck
		mdPath := filepath.Join(notesRoot, username, slug+".md")
		content, err := os.ReadFile(mdPath)
		if err != nil {
			continue
		}
		for _, todo := range ParseTodos(string(content)) {
			var due *string
			if todo.DueDate != "" {
				due = &todo.DueDate
			}
			tx.Exec(`INSERT INTO note_todos(note_id, line, text, due_date, completed) VALUES(?,?,?,?,?)`, //nolint:errcheck
				nr.id, todo.Line, todo.Text, due, todo.Completed)
		}
	}

	// Fourth pass: rebuild blog_posts from note_tags
	blogRows, err := tx.Query(`SELECT n.id, n.created_at FROM notes n JOIN note_tags t ON t.note_id = n.id WHERE t.tag_name = 'blog'`)
	if err != nil {
		return err
	}
	var blogNotes []struct {
		id        int64
		createdAt string
	}
	for blogRows.Next() {
		var bn struct {
			id        int64
			createdAt string
		}
		if err := blogRows.Scan(&bn.id, &bn.createdAt); err != nil {
			blogRows.Close()
			return err
		}
		blogNotes = append(blogNotes, bn)
	}
	blogRows.Close()

	for _, bn := range blogNotes {
		tx.Exec(`INSERT OR IGNORE INTO blog_posts(note_id, published_at) VALUES(?, ?)`, bn.id, bn.createdAt) //nolint:errcheck
	}

	if err := rebuildNoteDrawingsFromDisk(tx, notesRoot); err != nil {
		return err
	}

	return tx.Commit()
}

func rebuildNoteDrawingsFromDisk(tx *sql.Tx, notesRoot string) error {
	rows, err := tx.Query(`SELECT n.id, n.user_id, n.slug, u.username FROM notes n JOIN users u ON n.user_id = u.id`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type noteRow struct {
		id       int64
		userID   int64
		slug     string
		username string
	}
	var noteRows []noteRow
	for rows.Next() {
		var n noteRow
		if err := rows.Scan(&n.id, &n.userID, &n.slug, &n.username); err != nil {
			return err
		}
		noteRows = append(noteRows, n)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	const ins = `INSERT OR IGNORE INTO note_drawings (drawing_id, note_id, display_name, tool_type, created_at, updated_at) VALUES (?,?,?,?,?,?)`

	for _, n := range noteRows {
		dirs := []string{
			filepath.Join(notesRoot, n.username),
			filepath.Join(notesRoot, fmt.Sprintf("%d", n.userID)),
		}
		seenDir := map[string]bool{}
		legacyInserted := false
		for _, dir := range dirs {
			if seenDir[dir] {
				continue
			}
			seenDir[dir] = true
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				name := e.Name()
				if tool, ok := matchLegacyDrawingFilename(name, n.slug); ok {
					if legacyInserted {
						continue
					}
					legacyInserted = true
					id := GenerateDrawingID()
					if _, err := tx.Exec(ins, id, n.id, "Drawing 1", tool, now, now); err != nil {
						return err
					}
					continue
				}
				drawingID, tool, ok := matchNewDrawingFilename(name, n.slug)
				if !ok {
					continue
				}
				if _, err := tx.Exec(ins, drawingID, n.id, "Drawing", tool, now, now); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func matchLegacyDrawingFilename(name, slug string) (tool string, ok bool) {
	switch name {
	case slug + ".tldraw.json":
		return "tldraw", true
	case slug + ".excalidraw.json":
		return "excalidraw", true
	default:
		return "", false
	}
}

func matchNewDrawingFilename(name, slug string) (drawingID, tool string, ok bool) {
	prefix := slug + "--"
	if !strings.HasPrefix(name, prefix) {
		return "", "", false
	}
	rest := name[len(prefix):]
	switch {
	case strings.HasSuffix(rest, ".tldraw.json"):
		id := strings.TrimSuffix(rest, ".tldraw.json")
		if id == "" {
			return "", "", false
		}
		return id, "tldraw", true
	case strings.HasSuffix(rest, ".excalidraw.json"):
		id := strings.TrimSuffix(rest, ".excalidraw.json")
		if id == "" {
			return "", "", false
		}
		return id, "excalidraw", true
	default:
		return "", "", false
	}
}
