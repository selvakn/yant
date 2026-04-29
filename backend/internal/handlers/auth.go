package handlers

import (
	"log"
	"net/http"

	"github.com/selvakn/yant/internal/auth"
	"github.com/selvakn/yant/internal/models"
)

// LoginGET renders the login page.
func (h *Handler) LoginGET(w http.ResponseWriter, r *http.Request) {
	errorMsg := ""
	if r.URL.Query().Get("error") == "denied" {
		errorMsg = "GitHub authorization was denied. Please try again."
	} else if r.URL.Query().Get("error") == "failed" {
		errorMsg = "Sign in failed. Please try again."
	} else if r.URL.Query().Get("error") == "disabled" {
		errorMsg = "Your account has been disabled. Please contact an administrator."
	}
	h.render(w, r, "login.html", map[string]any{
		"Error":     errorMsg,
		"HasGitHub": h.github != nil,
	})
}

// GitHubLoginGET redirects the user to GitHub's authorization page.
func (h *Handler) GitHubLoginGET(w http.ResponseWriter, r *http.Request) {
	if h.github == nil {
		http.Error(w, "GitHub OAuth not configured", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, h.github.AuthorizeURL(r), http.StatusFound)
}

// GitHubCallbackGET handles the OAuth callback from GitHub.
func (h *Handler) GitHubCallbackGET(w http.ResponseWriter, r *http.Request) {
	if h.github == nil {
		http.Error(w, "GitHub OAuth not configured", http.StatusInternalServerError)
		return
	}

	if errParam := r.URL.Query().Get("error"); errParam != "" {
		http.Redirect(w, r, "/login?error=denied", http.StatusFound)
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		http.Redirect(w, r, "/login?error=failed", http.StatusFound)
		return
	}

	username, err := h.github.ExchangeCode(r, code, state)
	if err != nil {
		log.Printf("github oauth error: %v", err)
		http.Redirect(w, r, "/login?error=failed", http.StatusFound)
		return
	}

	user, err := models.GetOrCreateUser(h.db, username)
	if err != nil {
		log.Printf("get/create user error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if user.Disabled {
		http.Redirect(w, r, "/login?error=disabled", http.StatusFound)
		return
	}

	if h.adminUser != "" && user.Username == h.adminUser && !user.IsAdmin {
		if ok, err := models.BootstrapAdmin(h.db, user.Username); err != nil {
			log.Printf("bootstrap admin: %v", err)
		} else if ok {
			user.IsAdmin = true
		}
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
