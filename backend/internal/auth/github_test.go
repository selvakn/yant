package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/selvakn/yant/internal/auth"
)

func TestAuthorizeURL_ContainsExpectedParams(t *testing.T) {
	sm := scs.New()
	auth.SessionManager = sm

	gh := &auth.GitHubOAuth{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
	}

	handler := sm.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := gh.AuthorizeURL(r)
		w.Write([]byte(u)) //nolint:errcheck
	}))

	req := httptest.NewRequest("GET", "http://localhost:8080/login", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	url := rr.Body.String()
	if url == "" {
		t.Fatal("expected non-empty authorize URL")
	}
	if !strings.Contains(url, "client_id=test-client-id") {
		t.Errorf("URL missing client_id: %s", url)
	}
	if !strings.Contains(url, "scope=read") {
		t.Errorf("URL missing scope: %s", url)
	}
	if !strings.Contains(url, "state=") {
		t.Errorf("URL missing state: %s", url)
	}
	if !strings.Contains(url, "github.com/login/oauth/authorize") {
		t.Errorf("URL missing GitHub authorize path: %s", url)
	}
}

func TestAuthorizeURL_UsesBaseURL(t *testing.T) {
	sm := scs.New()
	auth.SessionManager = sm

	gh := &auth.GitHubOAuth{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		BaseURL:      "https://notes.example.com",
	}

	handler := sm.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := gh.AuthorizeURL(r)
		w.Write([]byte(u)) //nolint:errcheck
	}))

	req := httptest.NewRequest("GET", "http://localhost:8080/login", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	url := rr.Body.String()
	if !strings.Contains(url, "redirect_uri=https%3A%2F%2Fnotes.example.com%2Fauth%2Fgithub%2Fcallback") {
		t.Errorf("expected BaseURL in redirect_uri, got: %s", url)
	}
}

func TestAuthorizeURL_XForwardedProto(t *testing.T) {
	sm := scs.New()
	auth.SessionManager = sm

	gh := &auth.GitHubOAuth{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
	}

	handler := sm.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := gh.AuthorizeURL(r)
		w.Write([]byte(u)) //nolint:errcheck
	}))

	req := httptest.NewRequest("GET", "http://localhost:8080/login", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	url := rr.Body.String()
	if !strings.Contains(url, "redirect_uri=https%3A%2F%2Flocalhost%3A8080%2Fauth%2Fgithub%2Fcallback") {
		t.Errorf("expected https callback from X-Forwarded-Proto, got: %s", url)
	}
}

func TestExchangeCode_InvalidState(t *testing.T) {
	sm := scs.New()
	auth.SessionManager = sm

	gh := &auth.GitHubOAuth{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
	}

	handler := sm.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sm.Put(r.Context(), "oauth_state", "correct-state")
		_, err := gh.ExchangeCode(r, "some-code", "wrong-state")
		if err == nil {
			t.Error("expected error for mismatched state")
			return
		}
		if !strings.Contains(err.Error(), "invalid oauth state") {
			t.Errorf("expected 'invalid oauth state' error, got: %v", err)
		}
	}))

	req := httptest.NewRequest("GET", "http://localhost:8080/callback", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
}

func TestExchangeCode_EmptyState(t *testing.T) {
	sm := scs.New()
	auth.SessionManager = sm

	gh := &auth.GitHubOAuth{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
	}

	handler := sm.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := gh.ExchangeCode(r, "some-code", "any-state")
		if err == nil {
			t.Error("expected error when no state stored in session")
		}
	}))

	req := httptest.NewRequest("GET", "http://localhost:8080/callback", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
}

func TestExchangeCode_MockGitHubEndpoints(t *testing.T) {
	// This test demonstrates the mock setup; full integration requires
	// injectable URLs in GitHubOAuth (a future enhancement).
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{ //nolint:errcheck
			"access_token": "gho_test_token",
		})
	}))
	defer tokenServer.Close()

	userServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer gho_test_token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{ //nolint:errcheck
			"login": "testuser",
		})
	}))
	defer userServer.Close()

	t.Log("Mock servers available at", tokenServer.URL, userServer.URL)
}
