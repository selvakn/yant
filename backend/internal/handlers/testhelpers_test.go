package handlers_test

import (
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"testing"

	"github.com/alexedwards/scs/v2"
	"golang.org/x/net/publicsuffix"
)

func newSessionManager() *scs.SessionManager {
	return scs.New()
}

func newCookieJar() http.CookieJar {
	jar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	return jar
}

// resolveOrStubTemplateDir returns the real frontend/templates directory if it exists
// with all required templates present, otherwise creates stub templates in a temp dir.
func resolveOrStubTemplateDir(t *testing.T) string {
	t.Helper()

	// When tests run from backend/internal/handlers, frontend is at ../../../frontend
	candidates := []string{
		"../../../frontend/templates",
		"../../../../frontend/templates",
	}
	for _, c := range candidates {
		if hasRequiredTemplates(c) {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return createStubTemplateDir(t)
}

func hasRequiredTemplates(dir string) bool {
	required := []string{
		"base.html",
		filepath.Join("notes", "list.html"),
		filepath.Join("notes", "editor.html"),
		filepath.Join("notes", "reader.html"),
		filepath.Join("tags", "sidebar.html"),
		"login.html",
	}
	for _, f := range required {
		if _, err := os.Stat(filepath.Join(dir, f)); os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// createStubTemplateDir writes minimal stub HTML templates to a temp directory.
func createStubTemplateDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "notes"), 0755)     //nolint:errcheck
	os.MkdirAll(filepath.Join(dir, "tags"), 0755)      //nolint:errcheck

	stubs := map[string]string{
		"base.html": `{{define "base"}}<!DOCTYPE html><html><body>{{block "content" .}}{{end}}</body></html>{{end}}`,
		filepath.Join("notes", "list.html"):   `{{define "content"}}{{range .Notes}}<li>{{.Title}}</li>{{end}}{{end}}`,
		filepath.Join("notes", "editor.html"): `{{define "content"}}editor:{{.Body}}{{end}}`,
		filepath.Join("notes", "reader.html"): `{{define "content"}}reader:{{.BodyHTML}}{{end}}`,
		filepath.Join("tags", "sidebar.html"): `{{define "content"}}{{range .Tags}}<a>{{.Name}}</a>{{end}}{{end}}`,
		"login.html":                          `{{define "content"}}<form method="POST" action="/login"><input name="username"><button>Sign In</button></form>{{end}}`,
		"404.html":                            `{{define "content"}}404{{end}}`,
		"403.html":                            `{{define "content"}}403{{end}}`,
	}
	for name, content := range stubs {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write stub %s: %v", name, err)
		}
	}
	return dir
}
