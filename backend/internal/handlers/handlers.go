package handlers

import (
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/selvakn/yant/internal/auth"
	"github.com/selvakn/yant/internal/embedding"
	"github.com/selvakn/yant/internal/models"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// Handler holds shared dependencies for all HTTP handlers.
type Handler struct {
	db                    *models.DB
	tmplDir               string
	notesDir              string
	uploadsDir            string
	github                *auth.GitHubOAuth
	embedder              *embedding.Embedder
	md                    goldmark.Markdown
	semanticSearchEnabled bool
	searchDebounceMS      int
}

// New creates a Handler with the given dependencies.
// tmplDir is the path to the frontend/templates directory.
func New(db *models.DB, tmplDir, notesDir, uploadsDir string, github *auth.GitHubOAuth, embedder *embedding.Embedder, semanticSearch bool, debounceMS int) *Handler {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
	)
	return &Handler{
		db: db, tmplDir: tmplDir, notesDir: notesDir, uploadsDir: uploadsDir,
		github: github, embedder: embedder, md: md, semanticSearchEnabled: semanticSearch,
		searchDebounceMS: debounceMS,
	}
}

// baseData returns common template data for every page.
func (h *Handler) baseData(r *http.Request) map[string]any {
	username := usernameFromSession(r)
	return map[string]any{
		"Username":         username,
		"SearchDebounceMS": h.searchDebounceMS,
		"IsAdmin":          auth.IsAdmin(username),
	}
}

// render parses base.html + the named page template and executes the "base" template.
// Each page file must define {{define "content"}}...{{end}}.
func (h *Handler) render(w http.ResponseWriter, r *http.Request, page string, data map[string]any) {
	base := h.baseData(r)
	for k, v := range data {
		base[k] = v
	}

	tmpl, err := h.loadTemplate(page)
	if err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "base", base); err != nil {
		http.Error(w, "render error: "+err.Error(), http.StatusInternalServerError)
	}
}

// loadTemplate parses base.html together with the named page file.
// The page file is resolved relative to tmplDir.
func (h *Handler) loadTemplate(page string) (*template.Template, error) {
	base := filepath.Join(h.tmplDir, "base.html")
	// page may include subdirectory, e.g. "notes/list.html"
	pagePath := filepath.Join(h.tmplDir, page)
	return template.ParseFiles(base, pagePath)
}

// renderPartial parses and executes only the page template (no base.html wrapper).
// Used for htmx partial responses that should not include the full layout.
func (h *Handler) renderPartial(w http.ResponseWriter, r *http.Request, page string, data map[string]any) {
	base := h.baseData(r)
	for k, v := range data {
		base[k] = v
	}

	pagePath := filepath.Join(h.tmplDir, page)
	tmpl, err := template.ParseFiles(pagePath)
	if err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "content", base); err != nil {
		http.Error(w, "render error: "+err.Error(), http.StatusInternalServerError)
	}
}

// RenderError renders an appropriate error page.
func (h *Handler) RenderError(w http.ResponseWriter, r *http.Request, code int, msg string) {
	w.WriteHeader(code)
	page := "404.html"
	if code == 403 {
		page = "403.html"
	}
	data := h.baseData(r)
	data["Message"] = msg
	data["Code"] = code
	tmpl, err := h.loadTemplate(page)
	if err != nil {
		http.Error(w, msg, code)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, msg, code)
	}
}
