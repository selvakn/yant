package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/selvakn/yant/internal/auth"
	"github.com/selvakn/yant/internal/models"
	"github.com/selvakn/yant/internal/storage"
)

// AdminUsersListGET renders the admin dashboard listing all users.
func (h *Handler) AdminUsersListGET(w http.ResponseWriter, r *http.Request) {
	users, err := models.ListAllUsers(h.db)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	h.render(w, r, "admin/users.html", map[string]any{
		"Users":   users,
		"IsAdmin": true,
	})
}

// AdminUserDetailGET renders all notes for a specific user.
func (h *Handler) AdminUserDetailGET(w http.ResponseWriter, r *http.Request) {
	targetID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	targetUser, err := models.GetUserByID(h.db, targetID)
	if err != nil || targetUser == nil {
		http.NotFound(w, r)
		return
	}

	notes, err := models.ListAllNotesForUser(h.db, targetID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	h.render(w, r, "admin/user-notes.html", map[string]any{
		"TargetUser": targetUser,
		"Notes":      notes,
		"IsAdmin":    true,
	})
}

// AdminUserNotesGET renders a single note body for admin preview (htmx partial).
func (h *Handler) AdminNotePreviewGET(w http.ResponseWriter, r *http.Request) {
	noteID, err := strconv.ParseInt(chi.URLParam(r, "noteID"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var userID int64
	var slug string
	if err := h.db.QueryRow(`SELECT user_id, slug FROM notes WHERE id = ?`, noteID).Scan(&userID, &slug); err != nil {
		http.NotFound(w, r)
		return
	}

	body, err := storage.ReadNote(h.notesDir, userID, slug)
	if err != nil {
		body = "(file not found)"
	}

	const maxPreviewLen = 2000
	if len(body) > maxPreviewLen {
		body = body[:maxPreviewLen] + "\n\n... (truncated)"
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, body)
}

// AdminDeleteNoteDELETE deletes a specific note (admin action).
func (h *Handler) AdminDeleteNoteDELETE(w http.ResponseWriter, r *http.Request) {
	noteID, err := strconv.ParseInt(chi.URLParam(r, "noteID"), 10, 64)
	if err != nil {
		http.Error(w, "invalid note ID", http.StatusBadRequest)
		return
	}

	adminUser := auth.CurrentUsername(r)
	log.Printf("ADMIN: user %q deleting note ID %d", adminUser, noteID)

	if err := models.AdminDeleteNote(h.db, noteID, h.notesDir, h.uploadsDir); err != nil {
		log.Printf("admin delete note %d: %v", noteID, err)
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// AdminDeleteUserDELETE deletes a user and all their data (admin action).
func (h *Handler) AdminDeleteUserDELETE(w http.ResponseWriter, r *http.Request) {
	targetID, err := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	if err != nil {
		http.Error(w, "invalid user ID", http.StatusBadRequest)
		return
	}

	adminUser := auth.CurrentUsername(r)
	currentUserID := userIDFromSession(r)

	if targetID == currentUserID {
		http.Error(w, "cannot delete yourself", http.StatusBadRequest)
		return
	}

	targetUser, err := models.GetUserByID(h.db, targetID)
	if err != nil || targetUser == nil {
		http.NotFound(w, r)
		return
	}

	log.Printf("ADMIN: user %q deleting user %q (ID %d)", adminUser, targetUser.Username, targetID)

	if err := models.AdminDeleteUser(h.db, targetID, h.notesDir, h.uploadsDir); err != nil {
		log.Printf("admin delete user %d: %v", targetID, err)
		http.Error(w, "delete failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", "/admin/users")
	w.WriteHeader(http.StatusOK)
}
