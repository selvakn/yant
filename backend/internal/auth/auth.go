package auth

import (
	"net/http"

	"github.com/alexedwards/scs/v2"
)

// SessionManager is the package-level SCS session manager.
var SessionManager *scs.SessionManager

func init() {
	SessionManager = scs.New()
	// Uses in-memory store by default — appropriate for mock auth.
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

// CurrentUsername returns the username stored in the current session.
func CurrentUsername(r *http.Request) string {
	return SessionManager.GetString(r.Context(), "username")
}
