package models

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const AdminListPageSize = 25

type AdminUserView struct {
	ID             int64
	Username       string
	IsAdmin        bool
	Disabled       bool
	CreatedAt      time.Time
	NoteCount      int
	TotalSizeBytes int64
	LastActive     time.Time
}

type AuditLogEntry struct {
	ID            int64
	AdminUsername string
	Action        string
	TargetType    string
	TargetID      string
	Details       string
	CreatedAt     time.Time
}

type DashboardMetrics struct {
	TotalUsers        int
	ActiveUsers30d    int
	TotalNotes        int
	NotesCreated7d    int
	TotalPublicNotes  int
	TotalActiveShares int
}

type AdminNoteView struct {
	ID            int64
	Slug          string
	Title         string
	OwnerUsername string
	OwnerID       int64
	Archived      bool
	IsPublic      bool
	ShareCount    int
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Tags          []string
}

type AdminShareView struct {
	NoteID         int64
	NoteTitle      string
	NoteSlug       string
	OwnerUsername  string
	CollabUsername string
	CollabID       int64
	Permission     string
	GrantedAt      time.Time
}

type AdminPublicNoteView struct {
	NoteID        int64
	NoteSlug      string
	NoteTitle     string
	OwnerUsername string
	Token         string
	PublishedAt   time.Time
}

type UserImpactSummary struct {
	Username        string
	NoteCount       int
	ShareCount      int
	PublicNoteCount int
}

type NoteImpactSummary struct {
	NoteID     int64
	Title      string
	Owner      string
	ShareCount int
	IsPublic   bool
}

const (
	AuditDisableUser   = "disable-user"
	AuditEnableUser    = "enable-user"
	AuditDeleteUser    = "delete-user"
	AuditPromoteAdmin  = "promote-admin"
	AuditDemoteAdmin   = "demote-admin"
	AuditDeleteNote    = "delete-note"
	AuditUnpublishNote = "unpublish-note"
	AuditRevokeShare   = "revoke-share"
)

func WriteAuditLog(db *DB, adminUsername, action, targetType, targetID, details string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(`INSERT INTO admin_audit_log (admin_username, action, target_type, target_id, details, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		adminUsername, action, targetType, targetID, details, now)
	return err
}

func BootstrapAdmin(db *DB, username string) (bool, error) {
	res, err := db.Exec(`UPDATE users SET is_admin = 1 WHERE username = ?`, username)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func IsUserAdmin(db *DB, userID int64) bool {
	var isAdmin int
	err := db.QueryRow(`SELECT is_admin FROM users WHERE id = ? AND disabled = 0`, userID).Scan(&isAdmin)
	if err != nil {
		return false
	}
	return isAdmin == 1
}

func IsUserDisabled(db *DB, userID int64) bool {
	var disabled int
	err := db.QueryRow(`SELECT disabled FROM users WHERE id = ?`, userID).Scan(&disabled)
	if err != nil {
		return false
	}
	return disabled == 1
}

func GetDashboardMetrics(db *DB) (*DashboardMetrics, error) {
	m := &DashboardMetrics{}
	if err := db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&m.TotalUsers); err != nil {
		return nil, err
	}
	thirtyDaysAgo := time.Now().UTC().Add(-30 * 24 * time.Hour).Format(time.RFC3339)
	if err := db.QueryRow(`SELECT COUNT(DISTINCT n.user_id) FROM notes n WHERE n.updated_at > ?`, thirtyDaysAgo).Scan(&m.ActiveUsers30d); err != nil {
		return nil, err
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM notes`).Scan(&m.TotalNotes); err != nil {
		return nil, err
	}
	sevenDaysAgo := time.Now().UTC().Add(-7 * 24 * time.Hour).Format(time.RFC3339)
	if err := db.QueryRow(`SELECT COUNT(*) FROM notes WHERE created_at > ?`, sevenDaysAgo).Scan(&m.NotesCreated7d); err != nil {
		return nil, err
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM public_notes WHERE published = 1`).Scan(&m.TotalPublicNotes); err != nil {
		return nil, err
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM note_shares`).Scan(&m.TotalActiveShares); err != nil {
		return nil, err
	}
	return m, nil
}

func ListAllUsers(db *DB, search string, page int) ([]AdminUserView, int, error) {
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * AdminListPageSize

	var countQuery, listQuery string
	var args []any

	if search != "" {
		like := "%" + strings.ToLower(search) + "%"
		countQuery = `SELECT COUNT(*) FROM users WHERE LOWER(username) LIKE ?`
		listQuery = `SELECT u.id, u.username, u.is_admin, u.disabled, u.created_at,
			(SELECT COUNT(*) FROM notes WHERE user_id = u.id) as note_count,
			COALESCE((SELECT SUM(size_bytes) FROM notes WHERE user_id = u.id), 0) as total_size_bytes,
			COALESCE((SELECT MAX(updated_at) FROM notes WHERE user_id = u.id), u.created_at) as last_active
			FROM users u WHERE LOWER(u.username) LIKE ?
			ORDER BY u.created_at DESC LIMIT ? OFFSET ?`
		args = []any{like}
	} else {
		countQuery = `SELECT COUNT(*) FROM users`
		listQuery = `SELECT u.id, u.username, u.is_admin, u.disabled, u.created_at,
			(SELECT COUNT(*) FROM notes WHERE user_id = u.id) as note_count,
			COALESCE((SELECT SUM(size_bytes) FROM notes WHERE user_id = u.id), 0) as total_size_bytes,
			COALESCE((SELECT MAX(updated_at) FROM notes WHERE user_id = u.id), u.created_at) as last_active
			FROM users u ORDER BY u.created_at DESC LIMIT ? OFFSET ?`
	}

	var total int
	var err error
	if search != "" {
		err = db.QueryRow(countQuery, args[0]).Scan(&total)
	} else {
		err = db.QueryRow(countQuery).Scan(&total)
	}
	if err != nil {
		return nil, 0, err
	}

	queryArgs := append(args, AdminListPageSize, offset)
	rows, err := db.Query(listQuery, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []AdminUserView
	for rows.Next() {
		var u AdminUserView
		var createdAt, lastActive string
		var isAdmin, disabled int
		if err := rows.Scan(&u.ID, &u.Username, &isAdmin, &disabled, &createdAt, &u.NoteCount, &u.TotalSizeBytes, &lastActive); err != nil {
			return nil, 0, err
		}
		u.IsAdmin = isAdmin == 1
		u.Disabled = disabled == 1
		u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		u.LastActive, _ = time.Parse(time.RFC3339, lastActive)
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func GetAdminUserDetail(db *DB, username string) (*AdminUserView, error) {
	var u AdminUserView
	var createdAt, lastActive string
	var isAdmin, disabled int
	err := db.QueryRow(`SELECT u.id, u.username, u.is_admin, u.disabled, u.created_at,
		(SELECT COUNT(*) FROM notes WHERE user_id = u.id) as note_count,
		COALESCE((SELECT SUM(size_bytes) FROM notes WHERE user_id = u.id), 0) as total_size_bytes,
		COALESCE((SELECT MAX(updated_at) FROM notes WHERE user_id = u.id), u.created_at) as last_active
		FROM users u WHERE u.username = ?`, username).Scan(&u.ID, &u.Username, &isAdmin, &disabled, &createdAt, &u.NoteCount, &u.TotalSizeBytes, &lastActive)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.IsAdmin = isAdmin == 1
	u.Disabled = disabled == 1
	u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	u.LastActive, _ = time.Parse(time.RFC3339, lastActive)
	return &u, nil
}

func DisableUser(db *DB, userID int64) error {
	_, err := db.Exec(`UPDATE users SET disabled = 1 WHERE id = ?`, userID)
	if err != nil {
		return err
	}
	_, _ = db.Exec(`DELETE FROM sessions`)
	return nil
}

func EnableUser(db *DB, userID int64) error {
	_, err := db.Exec(`UPDATE users SET disabled = 0 WHERE id = ?`, userID)
	return err
}

func PromoteAdmin(db *DB, userID int64) error {
	_, err := db.Exec(`UPDATE users SET is_admin = 1 WHERE id = ?`, userID)
	return err
}

func DemoteAdmin(db *DB, userID int64) error {
	_, err := db.Exec(`UPDATE users SET is_admin = 0 WHERE id = ?`, userID)
	return err
}

func CountAdminUsers(db *DB) int {
	var count int
	_ = db.QueryRow(`SELECT COUNT(*) FROM users WHERE is_admin = 1 AND disabled = 0`).Scan(&count)
	return count
}

func GetUserImpactSummary(db *DB, userID int64) (*UserImpactSummary, error) {
	s := &UserImpactSummary{}
	_ = db.QueryRow(`SELECT username FROM users WHERE id = ?`, userID).Scan(&s.Username)
	_ = db.QueryRow(`SELECT COUNT(*) FROM notes WHERE user_id = ?`, userID).Scan(&s.NoteCount)
	_ = db.QueryRow(`SELECT COUNT(*) FROM note_shares WHERE note_id IN (SELECT id FROM notes WHERE user_id = ?) OR user_id = ?`, userID, userID).Scan(&s.ShareCount)
	_ = db.QueryRow(`SELECT COUNT(*) FROM public_notes p JOIN notes n ON n.id = p.note_id WHERE n.user_id = ? AND p.published = 1`, userID).Scan(&s.PublicNoteCount)
	return s, nil
}

func DeleteUserCascade(db *DB, userID int64) error {
	if _, err := db.Exec(`DELETE FROM note_shares WHERE user_id = ?`, userID); err != nil {
		return err
	}
	if _, err := db.Exec(`DELETE FROM notes WHERE user_id = ?`, userID); err != nil {
		return err
	}
	_, err := db.Exec(`DELETE FROM users WHERE id = ?`, userID)
	return err
}

func ListAllNotes(db *DB, ownerFilter, publicFilter, sharedFilter string, page int) ([]AdminNoteView, int, error) {
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * AdminListPageSize

	var conditions []string
	var args []any

	if ownerFilter != "" {
		conditions = append(conditions, "u.username = ?")
		args = append(args, ownerFilter)
	}
	if publicFilter == "yes" {
		conditions = append(conditions, "EXISTS (SELECT 1 FROM public_notes p WHERE p.note_id = n.id AND p.published = 1)")
	} else if publicFilter == "no" {
		conditions = append(conditions, "NOT EXISTS (SELECT 1 FROM public_notes p WHERE p.note_id = n.id AND p.published = 1)")
	}
	if sharedFilter == "yes" {
		conditions = append(conditions, "EXISTS (SELECT 1 FROM note_shares s WHERE s.note_id = n.id)")
	} else if sharedFilter == "no" {
		conditions = append(conditions, "NOT EXISTS (SELECT 1 FROM note_shares s WHERE s.note_id = n.id)")
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM notes n JOIN users u ON u.id = n.user_id %s`, whereClause)
	if err := db.QueryRow(countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQ := fmt.Sprintf(`SELECT n.id, n.slug, n.title, u.username, n.user_id, n.archived, n.created_at, n.updated_at,
		EXISTS (SELECT 1 FROM public_notes p WHERE p.note_id = n.id AND p.published = 1) as is_public,
		(SELECT COUNT(*) FROM note_shares s WHERE s.note_id = n.id) as share_count
		FROM notes n JOIN users u ON u.id = n.user_id %s
		ORDER BY n.updated_at DESC LIMIT ? OFFSET ?`, whereClause)

	queryArgs := append(args, AdminListPageSize, offset)
	rows, err := db.Query(listQ, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	type rowValues struct {
		n         AdminNoteView
		archived  int
		isPublic  int
		createdAt string
		updatedAt string
	}
	var scanned []rowValues
	for rows.Next() {
		var rv rowValues
		if err := rows.Scan(&rv.n.ID, &rv.n.Slug, &rv.n.Title, &rv.n.OwnerUsername, &rv.n.OwnerID, &rv.archived, &rv.createdAt, &rv.updatedAt, &rv.isPublic, &rv.n.ShareCount); err != nil {
			_ = rows.Close()
			return nil, 0, err
		}
		scanned = append(scanned, rv)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, 0, err
	}
	if err := rows.Close(); err != nil {
		return nil, 0, err
	}

	notes := make([]AdminNoteView, 0, len(scanned))
	for _, rv := range scanned {
		n := rv.n
		n.Archived = rv.archived == 1
		n.IsPublic = rv.isPublic == 1
		n.CreatedAt, _ = time.Parse(time.RFC3339, rv.createdAt)
		n.UpdatedAt, _ = time.Parse(time.RFC3339, rv.updatedAt)
		n.Tags, _ = getTagsForNote(db, n.ID)
		notes = append(notes, n)
	}
	return notes, total, nil
}

func GetNoteForAdmin(db *DB, noteID int64) (*AdminNoteView, error) {
	var n AdminNoteView
	var archived, isPublic int
	var createdAt, updatedAt string
	err := db.QueryRow(`SELECT n.id, n.slug, n.title, u.username, n.user_id, n.archived, n.created_at, n.updated_at,
		EXISTS (SELECT 1 FROM public_notes p WHERE p.note_id = n.id AND p.published = 1) as is_public,
		(SELECT COUNT(*) FROM note_shares s WHERE s.note_id = n.id) as share_count
		FROM notes n JOIN users u ON u.id = n.user_id WHERE n.id = ?`, noteID).Scan(
		&n.ID, &n.Slug, &n.Title, &n.OwnerUsername, &n.OwnerID, &archived, &createdAt, &updatedAt, &isPublic, &n.ShareCount)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	n.Archived = archived == 1
	n.IsPublic = isPublic == 1
	n.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	n.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	n.Tags, _ = getTagsForNote(db, n.ID)
	return &n, nil
}

func GetNoteImpactSummary(db *DB, noteID int64) (*NoteImpactSummary, error) {
	s := &NoteImpactSummary{NoteID: noteID}
	_ = db.QueryRow(`SELECT n.title, u.username FROM notes n JOIN users u ON u.id = n.user_id WHERE n.id = ?`, noteID).Scan(&s.Title, &s.Owner)
	_ = db.QueryRow(`SELECT COUNT(*) FROM note_shares WHERE note_id = ?`, noteID).Scan(&s.ShareCount)
	var isPublic int
	_ = db.QueryRow(`SELECT COUNT(*) FROM public_notes WHERE note_id = ? AND published = 1`, noteID).Scan(&isPublic)
	s.IsPublic = isPublic > 0
	return s, nil
}

func AdminDeleteNote(db *DB, noteID int64) error {
	_, err := db.Exec(`DELETE FROM notes WHERE id = ?`, noteID)
	return err
}

func ListAllPublicNotes(db *DB, page int) ([]AdminPublicNoteView, int, error) {
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * AdminListPageSize

	var total int
	if err := db.QueryRow(`SELECT COUNT(*) FROM public_notes WHERE published = 1`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := db.Query(`SELECT p.note_id, n.slug, n.title, u.username, p.token, p.published_at
		FROM public_notes p
		JOIN notes n ON n.id = p.note_id
		JOIN users u ON u.id = n.user_id
		WHERE p.published = 1
		ORDER BY p.published_at DESC LIMIT ? OFFSET ?`, AdminListPageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var notes []AdminPublicNoteView
	for rows.Next() {
		var n AdminPublicNoteView
		var publishedAt string
		if err := rows.Scan(&n.NoteID, &n.NoteSlug, &n.NoteTitle, &n.OwnerUsername, &n.Token, &publishedAt); err != nil {
			return nil, 0, err
		}
		n.PublishedAt, _ = time.Parse(time.RFC3339, publishedAt)
		notes = append(notes, n)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return notes, total, nil
}

func ListAllShares(db *DB, userFilter string, page int) ([]AdminShareView, int, error) {
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * AdminListPageSize

	var whereClause string
	var args []any
	if userFilter != "" {
		whereClause = "WHERE owner.username = ? OR collab.username = ?"
		args = []any{userFilter, userFilter}
	}

	var total int
	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM note_shares s
		JOIN notes n ON n.id = s.note_id
		JOIN users owner ON owner.id = n.user_id
		JOIN users collab ON collab.id = s.user_id %s`, whereClause)
	if err := db.QueryRow(countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQ := fmt.Sprintf(`SELECT s.note_id, n.title, n.slug, owner.username, collab.username, s.user_id, s.permission, s.granted_at
		FROM note_shares s
		JOIN notes n ON n.id = s.note_id
		JOIN users owner ON owner.id = n.user_id
		JOIN users collab ON collab.id = s.user_id %s
		ORDER BY s.granted_at DESC LIMIT ? OFFSET ?`, whereClause)

	queryArgs := append(args, AdminListPageSize, offset)
	rows, err := db.Query(listQ, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var shares []AdminShareView
	for rows.Next() {
		var s AdminShareView
		var grantedAt string
		if err := rows.Scan(&s.NoteID, &s.NoteTitle, &s.NoteSlug, &s.OwnerUsername, &s.CollabUsername, &s.CollabID, &s.Permission, &grantedAt); err != nil {
			return nil, 0, err
		}
		s.GrantedAt, _ = time.Parse(time.RFC3339, grantedAt)
		shares = append(shares, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return shares, total, nil
}

func ListAuditLog(db *DB, actionFilter, userFilter string, page int) ([]AuditLogEntry, int, error) {
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * AdminListPageSize

	var conditions []string
	var args []any
	if actionFilter != "" {
		conditions = append(conditions, "action = ?")
		args = append(args, actionFilter)
	}
	if userFilter != "" {
		conditions = append(conditions, "(target_id = ? OR admin_username = ?)")
		args = append(args, userFilter, userFilter)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM admin_audit_log %s`, whereClause)
	if err := db.QueryRow(countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQ := fmt.Sprintf(`SELECT id, admin_username, action, target_type, target_id, COALESCE(details, ''), created_at
		FROM admin_audit_log %s ORDER BY created_at DESC LIMIT ? OFFSET ?`, whereClause)

	queryArgs := append(args, AdminListPageSize, offset)
	rows, err := db.Query(listQ, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var entries []AuditLogEntry
	for rows.Next() {
		var e AuditLogEntry
		var createdAt string
		if err := rows.Scan(&e.ID, &e.AdminUsername, &e.Action, &e.TargetType, &e.TargetID, &e.Details, &createdAt); err != nil {
			return nil, 0, err
		}
		e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return entries, total, nil
}
