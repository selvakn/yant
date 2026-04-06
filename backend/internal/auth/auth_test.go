package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexedwards/scs/v2"

	"github.com/selvakn/yant/internal/auth"
)

func TestRequireLogin_RedirectsWhenNoSession(t *testing.T) {
	sm := scs.New()
	auth.SessionManager = sm

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := sm.LoadAndSave(auth.RequireLogin(next))

	req := httptest.NewRequest("GET", "/notes", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Errorf("expected 302 redirect, got %d", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/login" {
		t.Errorf("expected redirect to /login, got %q", loc)
	}
}

func TestRequireLogin_AllowsAuthenticatedRequest(t *testing.T) {
	sm := scs.New()
	auth.SessionManager = sm

	// Simulate setting up a session with username via a login handler
	var sessionToken string
	loginHandler := sm.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sm.Put(r.Context(), "username", "alice")
		sessionToken = ""
		w.WriteHeader(http.StatusOK)
	}))

	loginReq := httptest.NewRequest("POST", "/login", nil)
	loginRR := httptest.NewRecorder()
	loginHandler.ServeHTTP(loginRR, loginReq)

	// Extract session cookie
	var cookie *http.Cookie
	for _, c := range loginRR.Result().Cookies() {
		if c.Name == "session" {
			cookie = c
			sessionToken = c.Value
			break
		}
	}
	_ = sessionToken

	if cookie == nil {
		t.Skip("no session cookie set, skipping authenticated test")
	}

	// Now make a request with the session cookie
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := sm.LoadAndSave(auth.RequireLogin(next))
	req := httptest.NewRequest("GET", "/notes", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("expected next handler to be called for authenticated request")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestCurrentUsername_ReturnsEmpty_WhenNoSession(t *testing.T) {
	sm := scs.New()
	auth.SessionManager = sm

	var username string
	handler := sm.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username = auth.CurrentUsername(r)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if username != "" {
		t.Errorf("expected empty username, got %q", username)
	}
}

func TestCurrentUsername_ReturnsUsername_WhenSessionSet(t *testing.T) {
	sm := scs.New()
	auth.SessionManager = sm

	// First request: set username
	var setCookie *http.Cookie
	setHandler := sm.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sm.Put(r.Context(), "username", "testuser")
		w.WriteHeader(http.StatusOK)
	}))
	setReq := httptest.NewRequest("POST", "/login", nil)
	setRR := httptest.NewRecorder()
	setHandler.ServeHTTP(setRR, setReq)
	for _, c := range setRR.Result().Cookies() {
		if c.Name == "session" {
			setCookie = c
			break
		}
	}
	if setCookie == nil {
		t.Skip("no session cookie")
	}

	// Second request: read username
	var username string
	getHandler := sm.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username = auth.CurrentUsername(r)
		w.WriteHeader(http.StatusOK)
	}))
	getReq := httptest.NewRequest("GET", "/", nil)
	getReq.AddCookie(setCookie)
	getRR := httptest.NewRecorder()
	getHandler.ServeHTTP(getRR, getReq)

	if username != "testuser" {
		t.Errorf("expected 'testuser', got %q", username)
	}
}
