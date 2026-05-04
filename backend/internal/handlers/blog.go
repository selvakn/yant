package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/selvakn/yant/internal/markdown"
	"github.com/selvakn/yant/internal/models"
	"github.com/selvakn/yant/internal/storage"
)

const defaultBlogPageSize = 10

func parseBlogPage(r *http.Request) int {
	raw := r.URL.Query().Get("page")
	if raw == "" {
		return 1
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 1
	}
	return n
}

// BlogIndexGET handles GET /blog
func (h *Handler) BlogIndexGET(w http.ResponseWriter, r *http.Request) {
	page := parseBlogPage(r)

	posts, err := models.ListBlogPosts(h.db, page, defaultBlogPageSize)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	h.blogExcerpts(posts)

	total := models.CountBlogPosts(h.db)
	totalPages := 1
	if total > 0 {
		totalPages = (total + defaultBlogPageSize - 1) / defaultBlogPageSize
	}

	data := map[string]any{
		"Posts":      posts,
		"Page":       page,
		"TotalPages": totalPages,
		"HasPrev":    page > 1,
		"HasNext":    page < totalPages,
		"AllTags":    models.ListBlogTags(h.db),
		"Tag":        "",
	}

	h.renderBlog(w, "index.html", data)
}

// BlogPostGET handles GET /blog/{username}/{slug}
func (h *Handler) BlogPostGET(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	slug := chi.URLParam(r, "slug")

	bp, err := models.GetBlogPost(h.db, username, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	body, err := storage.ReadNote(h.notesDir, bp.Note.UserID, bp.Note.Slug)
	if err != nil {
		body = ""
	}

	resolved := models.ResolveWikiLinksForBlog(h.db, bp.Note.UserID, body)

	var buf bytes.Buffer
	if err := h.md.Convert([]byte(resolved), &buf); err != nil {
		buf.WriteString("<p>Error rendering markdown</p>")
	}

	htmlStr := buf.String()
	now := time.Now().Truncate(24 * time.Hour)
	htmlStr = dueBadgeRe.ReplaceAllStringFunc(htmlStr, func(match string) string {
		m := dueBadgeRe.FindStringSubmatch(match)
		date := m[1]
		t, err := time.Parse("2006-01-02", date)
		if err != nil {
			return match
		}
		class := "todo-due"
		if t.Before(now) {
			class += " todo-overdue"
		}
		return fmt.Sprintf(`<span class="%s">%s</span>`, class, t.Format("Jan 2, 2006"))
	})

	drawings, err := models.ListDrawings(h.db, bp.Note.ID)
	if err != nil {
		drawings = nil
	}

	prevPost, nextPost := models.GetAdjacentBlogPosts(h.db, bp.PublishedAt)

	description := markdown.GenerateExcerpt(body, 200)

	meta := make([]struct {
		DrawingID   string `json:"drawing_id"`
		DisplayName string `json:"display_name"`
	}, len(drawings))
	for i, d := range drawings {
		meta[i].DrawingID = d.DrawingID
		meta[i].DisplayName = d.DisplayName
	}
	drawingsJSON, err := json.Marshal(meta)
	if err != nil {
		drawingsJSON = []byte("[]")
	}

	data := map[string]any{
		"Post":         bp,
		"BodyHTML":     template.HTML(htmlStr), //nolint:gosec
		"Drawings":     drawings,
		"DrawingsJSON": template.JS(drawingsJSON),
		"PrevPost":     prevPost,
		"NextPost":     nextPost,
		"Title":        bp.Note.Title,
		"Description":  description,
	}

	h.renderBlog(w, "post.html", data)
}

// BlogTagGET handles GET /blog/tag/{tag}
func (h *Handler) BlogTagGET(w http.ResponseWriter, r *http.Request) {
	tag := chi.URLParam(r, "tag")
	page := parseBlogPage(r)

	posts, err := models.ListBlogPostsByTag(h.db, tag, page, defaultBlogPageSize)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	h.blogExcerpts(posts)

	total := models.CountBlogPostsByTag(h.db, tag)
	totalPages := 1
	if total > 0 {
		totalPages = (total + defaultBlogPageSize - 1) / defaultBlogPageSize
	}

	data := map[string]any{
		"Posts":      posts,
		"Page":       page,
		"TotalPages": totalPages,
		"HasPrev":    page > 1,
		"HasNext":    page < totalPages,
		"AllTags":    models.ListBlogTags(h.db),
		"Tag":        tag,
	}

	h.renderBlog(w, "index.html", data)
}

// BlogDrawingSVGGET handles GET /blog/{username}/{slug}/drawings/{drawingID}/svg
func (h *Handler) BlogDrawingSVGGET(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	slug := chi.URLParam(r, "slug")
	drawingID := chi.URLParam(r, "drawingID")

	user, err := models.GetUserByUsername(h.db, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	note, err := models.GetNote(h.db, user.ID, slug)
	if err != nil || note == nil {
		http.NotFound(w, r)
		return
	}

	if !models.IsBlogPost(h.db, note.ID) {
		http.NotFound(w, r)
		return
	}

	if _, err := models.GetDrawing(h.db, note.ID, drawingID); err != nil {
		http.NotFound(w, r)
		return
	}

	svg, err := storage.ReadDrawingSVG(h.notesDir, note.UserID, note.Slug, drawingID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(svg)
}

func (h *Handler) renderBlog(w http.ResponseWriter, page string, data map[string]any) {
	funcMap := template.FuncMap{
		"add":      func(a, b int) int { return a + b },
		"subtract": func(a, b int) int { return a - b },
	}
	base := filepath.Join(h.tmplDir, "blog", "base.html")
	pagePath := filepath.Join(h.tmplDir, "blog", page)
	tmpl, err := template.New("").Funcs(funcMap).ParseFiles(base, pagePath)
	if err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "blog-base", data); err != nil {
		http.Error(w, "render error: "+err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) blogExcerpts(posts []*models.BlogPost) {
	for _, bp := range posts {
		body, _ := storage.ReadNote(h.notesDir, bp.Note.UserID, bp.Note.Slug)
		bp.Excerpt = markdown.GenerateExcerpt(body, 200)
	}
}
