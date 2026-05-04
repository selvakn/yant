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
		filepath.Join("blog", "base.html"),
		filepath.Join("blog", "index.html"),
		filepath.Join("blog", "post.html"),
		filepath.Join("admin", "dashboard.html"),
		filepath.Join("admin", "users.html"),
		filepath.Join("admin", "user-detail.html"),
		filepath.Join("admin", "notes.html"),
		filepath.Join("admin", "note-detail.html"),
		filepath.Join("admin", "public-notes.html"),
		filepath.Join("admin", "shares.html"),
		filepath.Join("admin", "audit-log.html"),
		filepath.Join("admin", "user-delete-confirm.html"),
		filepath.Join("admin", "note-delete-confirm.html"),
		filepath.Join("admin", "partials", "user-actions.html"),
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
	os.MkdirAll(filepath.Join(dir, "notes"), 0755)             //nolint:errcheck
	os.MkdirAll(filepath.Join(dir, "tags"), 0755)              //nolint:errcheck
	os.MkdirAll(filepath.Join(dir, "todos"), 0755)             //nolint:errcheck
	os.MkdirAll(filepath.Join(dir, "public"), 0755)            //nolint:errcheck
	os.MkdirAll(filepath.Join(dir, "shared"), 0755)            //nolint:errcheck
	os.MkdirAll(filepath.Join(dir, "blog"), 0755)              //nolint:errcheck
	os.MkdirAll(filepath.Join(dir, "admin"), 0755)             //nolint:errcheck
	os.MkdirAll(filepath.Join(dir, "admin", "partials"), 0755) //nolint:errcheck

	stubs := map[string]string{
		"base.html":                                             `{{define "base"}}<!DOCTYPE html><html><body>{{block "content" .}}{{end}}</body></html>{{end}}`,
		filepath.Join("notes", "list.html"):                     `{{define "content"}}{{range .Notes}}<li>{{.Title}}</li>{{end}}{{end}}`,
		filepath.Join("notes", "editor.html"):                   `{{define "content"}}editor:{{.Body}}{{end}}`,
		filepath.Join("notes", "reader.html"):                   `{{define "content"}}reader:{{.BodyHTML}}{{end}}`,
		filepath.Join("notes", "history.html"):                  `{{define "content"}}history:{{range .Versions}}<div class="version">{{.ShortHash}}|{{.Message}}|{{.Timestamp.Format "2006-01-02"}}</div>{{end}}{{if not .Versions}}No version history{{end}}{{end}}`,
		filepath.Join("notes", "version.html"):                  `{{define "content"}}version:{{.Version.ShortHash}}|{{.BodyHTML}}{{if .IsHistorical}}|historical{{end}}{{end}}`,
		filepath.Join("notes", "diff.html"):                     `{{define "content"}}diff:{{.OldVersion.ShortHash}}..{{.NewVersion.ShortHash}}|{{range .Diff.Lines}}{{.Type}}:{{.Content}};{{end}}{{end}}`,
		filepath.Join("tags", "sidebar.html"):                   `{{define "content"}}{{range .Tags}}<a>{{.Name}}</a>{{end}}{{end}}`,
		filepath.Join("todos", "list.html"):                     `{{define "content"}}{{range .Todos}}<li>{{.Text}}</li>{{end}}{{end}}`,
		filepath.Join("public", "note.html"):                    `<!DOCTYPE html><html><head><title>{{.Title}}</title></head><body class="public-page"><h1>{{.Title}}</h1><div class="markdown-body">{{.BodyHTML}}</div></body></html>`,
		filepath.Join("public", "list.html"):                    `{{define "content"}}public-notes:{{range .Notes}}<li>{{.Title}}|{{.Token}}</li>{{end}}{{end}}`,
		filepath.Join("shared", "list.html"):                    `{{define "content"}}shared-notes:{{range .SharedNotes}}<li>{{.Title}}|{{.OwnerUsername}}|{{.Permission}}</li>{{end}}{{end}}`,
		filepath.Join("shared", "reader.html"):                  `{{define "content"}}shared-reader:{{.Note.Title}}|{{.OwnerUsername}}|role:{{.Role}}|canEdit:{{.CanEdit}}{{.BodyHTML}}{{end}}`,
		filepath.Join("blog", "base.html"):                     `{{define "blog-base"}}<!DOCTYPE html><html><body>{{template "blog-content" .}}</body></html>{{end}}`,
		filepath.Join("blog", "index.html"): `{{define "blog-content"}}{{if .Posts}}{{range .Posts}}<article class="blog-post-card"><h2 class="blog-post-card-title"><a href="/blog/{{.Username}}/{{.Note.Slug}}">{{.Note.Title}}</a></h2><p class="blog-post-card-excerpt">{{.Excerpt}}</p></article>{{end}}{{else}}{{if .Tag}}<p>No posts tagged &ldquo;{{.Tag}}&rdquo; yet.</p>{{else}}<p>No posts yet.</p>{{end}}{{end}}{{end}}`,
		filepath.Join("blog", "post.html"): `{{define "title"}}x{{end}}{{define "meta"}}{{end}}{{define "blog-content"}}<article class="blog-post"><h1 class="blog-post-title">{{.Post.Note.Title}}</h1><div class="blog-post-body">{{.BodyHTML}}</div></article>{{end}}`,
		filepath.Join("shared", "editor.html"):                  `{{define "content"}}shared-editor:{{.Note.Title}}|{{.OwnerUsername}}|{{.Body}}{{end}}`,
		filepath.Join("admin", "dashboard.html"):                `{{define "content"}}admin-dashboard:{{.Metrics.TotalUsers}}|{{.Metrics.TotalNotes}}{{end}}`,
		filepath.Join("admin", "users.html"):                    `{{define "content"}}{{range .Users}}<span>{{.Username}}</span>{{end}}users-page{{end}}`,
		filepath.Join("admin", "user-detail.html"):              `{{define "content"}}user-detail:{{.User.Username}}{{end}}`,
		filepath.Join("admin", "notes.html"):                    `{{define "content"}}{{range .Notes}}<span>{{.Title}}</span>{{end}}notes-page{{end}}`,
		filepath.Join("admin", "note-detail.html"):              `{{define "content"}}note-detail:{{.Note.Title}}|{{.BodyHTML}}{{end}}`,
		filepath.Join("admin", "public-notes.html"):             `{{define "content"}}{{range .Notes}}<span>{{.NoteTitle}}</span>{{end}}public{{end}}`,
		filepath.Join("admin", "shares.html"):                   `{{define "content"}}{{range .Shares}}<span>{{.NoteTitle}}</span>{{end}}shares{{end}}`,
		filepath.Join("admin", "audit-log.html"):                `{{define "content"}}{{range .Entries}}<span>{{.Action}}</span>{{end}}audit{{end}}`,
		filepath.Join("admin", "user-delete-confirm.html"):      `{{define "content"}}user-del-confirm:{{.Block}}{{if .User}}{{.User.Username}}{{end}}{{end}}`,
		filepath.Join("admin", "note-delete-confirm.html"):      `{{define "content"}}note-del-confirm:{{.Note.Title}}{{end}}`,
		filepath.Join("admin", "partials", "user-actions.html"): `{{define "content"}}<div id="user-actions-inner">user-actions</div>{{end}}`,
		"login.html": `{{define "content"}}YANT{{if .HasGitHub}}<a href="/auth/github">Sign in with GitHub</a>{{end}}{{if .Error}}<div class="login-error">{{.Error}}</div>{{end}}{{end}}`,
		"404.html":   `{{define "content"}}404{{end}}`,
		"403.html":   `{{define "content"}}403{{end}}`,
	}
	for name, content := range stubs {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write stub %s: %v", name, err)
		}
	}
	return dir
}
