package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexedwards/scs/v2"

	"github.com/selvakn/yant/internal/auth"
)

func TestIsAdmin_ReturnsTrueForConfiguredAdmin(t *testing.T) {
	// Arrange
	auth.SetAdminUser("admin-user")
	defer auth.SetAdminUser("")

	// Act & Assert
	if !auth.IsAdmin("admin-user") {
		t.Error("expected IsAdmin to return true for configured admin")
	}
}

func TestIsAdmin_ReturnsFalseForOtherUser(t *testing.T) {
	// Arrange
	auth.SetAdminUser("admin-user")
	defer auth.SetAdminUser("")

	// Act & Assert
	if auth.IsAdmin("regular-user") {
		t.Error("expected IsAdmin to return false for non-admin")
	}
}

func TestIsAdmin_ReturnsFalseWhenNoAdminConfigured(t *testing.T) {
	// Arrange
	auth.SetAdminUser("")

	// Act & Assert
	if auth.IsAdmin("anyone") {
		t.Error("expected IsAdmin to return false when no admin configured")
	}
}

func TestIsAdmin_ReturnsFalseForEmptyUsername(t *testing.T) {
	// Arrange
	auth.SetAdminUser("admin-user")
	defer auth.SetAdminUser("")

	// Act & Assert
	if auth.IsAdmin("") {
		t.Error("expected IsAdmin to return false for empty username")
	}
}

func TestRequireAdmin_BlocksNonAdmin(t *testing.T) {
	// Arrange
	sm := scs.New()
	auth.SessionManager = sm
	auth.SetAdminUser("admin-user")
	defer auth.SetAdminUser("")

	setCookie := setSession(t, sm, "regular-user")

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := sm.LoadAndSave(auth.RequireAdmin(next))

	// Act
	req := httptest.NewRequest("GET", "/admin/users", nil)
	req.AddCookie(setCookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Assert
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin, got %d", rr.Code)
	}
}

func TestRequireAdmin_AllowsAdmin(t *testing.T) {
	// Arrange
	sm := scs.New()
	auth.SessionManager = sm
	auth.SetAdminUser("admin-user")
	defer auth.SetAdminUser("")

	setCookie := setSession(t, sm, "admin-user")

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := sm.LoadAndSave(auth.RequireAdmin(next))

	// Act
	req := httptest.NewRequest("GET", "/admin/users", nil)
	req.AddCookie(setCookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Assert
	if !called {
		t.Error("expected admin handler to be called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for admin, got %d", rr.Code)
	}
}

func TestRequireAdmin_BlocksUnauthenticated(t *testing.T) {
	// Arrange
	sm := scs.New()
	auth.SessionManager = sm
	auth.SetAdminUser("admin-user")
	defer auth.SetAdminUser("")

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := sm.LoadAndSave(auth.RequireAdmin(next))

	// Act
	req := httptest.NewRequest("GET", "/admin/users", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Assert
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for unauthenticated, got %d", rr.Code)
	}
}

func setSession(t *testing.T, sm *scs.SessionManager, username string) *http.Cookie {
	t.Helper()
	setHandler := sm.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sm.Put(r.Context(), "username", username)
		w.WriteHeader(http.StatusOK)
	}))
	setReq := httptest.NewRequest("POST", "/login", nil)
	setRR := httptest.NewRecorder()
	setHandler.ServeHTTP(setRR, setReq)

	for _, c := range setRR.Result().Cookies() {
		if c.Name == "session" {
			return c
		}
	}
	t.Fatal("no session cookie set")
	return nil
}
