package handlers

import (
	"net/http"

	"github.com/selvakn/my-notes/internal/auth"
	"github.com/selvakn/my-notes/internal/models"
)

// LoginGET renders the login page.
func (h *Handler) LoginGET(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "login.html", nil)
}

// LoginPOST handles username-only login. Auto-creates user if unknown.
func (h *Handler) LoginPOST(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	if username == "" {
		http.Error(w, "username is required", http.StatusBadRequest)
		return
	}

	user, err := models.GetOrCreateUser(h.db, username)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	auth.SessionManager.Put(r.Context(), "username", user.Username)
	auth.SessionManager.Put(r.Context(), "userID", user.ID)
	http.Redirect(w, r, "/notes", http.StatusFound)
}

// LogoutPOST clears the session and redirects to /login.
func (h *Handler) LogoutPOST(w http.ResponseWriter, r *http.Request) {
	_ = auth.SessionManager.Destroy(r.Context())
	http.Redirect(w, r, "/login", http.StatusFound)
}

// usernameFromSession is a helper used by multiple handlers.
func usernameFromSession(r *http.Request) string {
	return auth.CurrentUsername(r)
}

// userIDFromSession returns the session user ID (int64).
func userIDFromSession(r *http.Request) int64 {
	return auth.SessionManager.GetInt64(r.Context(), "userID")
}
