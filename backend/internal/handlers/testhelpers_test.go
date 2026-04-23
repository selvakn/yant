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
		filepath.Join("notes", "history.html"),
		filepath.Join("notes", "version.html"),
		filepath.Join("notes", "diff.html"),
		filepath.Join("tags", "sidebar.html"),
		filepath.Join("public", "note.html"),
		filepath.Join("public", "list.html"),
		filepath.Join("shared", "list.html"),
		filepath.Join("shared", "reader.html"),
		filepath.Join("shared", "editor.html"),
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
	os.MkdirAll(filepath.Join(dir, "todos"), 0755)     //nolint:errcheck
	os.MkdirAll(filepath.Join(dir, "public"), 0755)    //nolint:errcheck
	os.MkdirAll(filepath.Join(dir, "shared"), 0755)    //nolint:errcheck

	stubs := map[string]string{
		"base.html": `{{define "base"}}<!DOCTYPE html><html><body>{{block "content" .}}{{end}}</body></html>{{end}}`,
		filepath.Join("notes", "list.html"):    `{{define "content"}}{{range .Notes}}<li>{{.Title}}</li>{{end}}{{end}}`,
		filepath.Join("notes", "editor.html"):  `{{define "content"}}editor:{{.Body}}{{end}}`,
		filepath.Join("notes", "reader.html"):  `{{define "content"}}reader:{{.BodyHTML}}{{end}}`,
		filepath.Join("notes", "history.html"): `{{define "content"}}history:{{range .Versions}}<div class="version">{{.ShortHash}}|{{.Message}}|{{.Timestamp.Format "2006-01-02"}}</div>{{end}}{{if not .Versions}}No version history{{end}}{{end}}`,
		filepath.Join("notes", "version.html"): `{{define "content"}}version:{{.Version.ShortHash}}|{{.BodyHTML}}{{if .IsHistorical}}|historical{{end}}{{end}}`,
		filepath.Join("notes", "diff.html"):    `{{define "content"}}diff:{{.OldVersion.ShortHash}}..{{.NewVersion.ShortHash}}|{{range .Diff.Lines}}{{.Type}}:{{.Content}};{{end}}{{end}}`,
		filepath.Join("tags", "sidebar.html"):  `{{define "content"}}{{range .Tags}}<a>{{.Name}}</a>{{end}}{{end}}`,
		filepath.Join("todos", "list.html"):    `{{define "content"}}{{range .Todos}}<li>{{.Text}}</li>{{end}}{{end}}`,
		filepath.Join("public", "note.html"):   `<!DOCTYPE html><html><head><title>{{.Title}}</title></head><body class="public-page"><h1>{{.Title}}</h1><div class="markdown-body">{{.BodyHTML}}</div></body></html>`,
		filepath.Join("public", "list.html"):   `{{define "content"}}public-notes:{{range .Notes}}<li>{{.Title}}|{{.Token}}</li>{{end}}{{end}}`,
		filepath.Join("shared", "list.html"):   `{{define "content"}}shared-notes:{{range .SharedNotes}}<li>{{.Title}}|{{.OwnerUsername}}|{{.Permission}}</li>{{end}}{{end}}`,
		filepath.Join("shared", "reader.html"): `{{define "content"}}shared-reader:{{.Note.Title}}|{{.OwnerUsername}}|role:{{.Role}}|canEdit:{{.CanEdit}}{{.BodyHTML}}{{end}}`,
		filepath.Join("shared", "editor.html"): `{{define "content"}}shared-editor:{{.Note.Title}}|{{.OwnerUsername}}|{{.Body}}{{end}}`,
		"login.html":                           `{{define "content"}}{{if .Error}}<div class="login-error">{{.Error}}</div>{{end}}<a href="/auth/github">Sign in with GitHub</a>{{end}}`,
		"404.html":                             `{{define "content"}}404{{end}}`,
		"403.html":                             `{{define "content"}}403{{end}}`,
	}
	for name, content := range stubs {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write stub %s: %v", name, err)
		}
	}
	return dir
}
