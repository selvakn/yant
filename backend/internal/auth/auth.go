package auth

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
)

// SessionManager is the package-level SCS session manager.
var SessionManager *scs.SessionManager

// adminUsername holds the GitHub username of the admin user, set via ADMIN_USER env var.
var adminUsername string

func init() {
	SessionManager = scs.New()
	// In-memory store by default — replaced by ConfigureSessionStore at server startup.
}

// SetAdminUser configures the admin username. Call once at startup.
func SetAdminUser(username string) {
	adminUsername = username
}

// IsAdmin returns true if the given username matches the configured admin user.
func IsAdmin(username string) bool {
	return adminUsername != "" && username == adminUsername
}

// ConfigureSessionStore switches the SessionManager to a persistent SQLite-backed
// store so user sessions survive server restarts. Call this once at server startup
// after the DB has been opened and InitSchema has run.
//
// Sessions are kept for 30 days. Expired rows are cleaned up every hour by a
// goroutine owned by the store.
func ConfigureSessionStore(db *sql.DB) {
	SessionManager.Store = sqlite3store.NewWithCleanupInterval(db, time.Hour)
	SessionManager.Lifetime = 30 * 24 * time.Hour
}

// RequireLogin is middleware that redirects unauthenticated requests to /login.
func RequireLogin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if SessionManager.GetString(r.Context(), "username") == "" {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAdmin is middleware that returns 403 for non-admin users.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username := SessionManager.GetString(r.Context(), "username")
		if !IsAdmin(username) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// CurrentUsername returns the username stored in the current session.
func CurrentUsername(r *http.Request) string {
	return SessionManager.GetString(r.Context(), "username")
}
