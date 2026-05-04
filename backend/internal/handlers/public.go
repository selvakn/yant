package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/selvakn/yant/internal/models"
	"github.com/selvakn/yant/internal/storage"
)

// publicCheckboxRe matches rendered goldmark task list checkboxes.
var publicCheckboxRe = regexp.MustCompile(`<input[^>]*type="checkbox"[^>]*>`)

// PublishPUT marks a note as publicly shared and returns the public URL.
// PUT /notes/{slug}/publish
func (h *Handler) PublishPUT(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")

	note, err := models.GetNote(h.db, userID, slug)
	if err != nil || note == nil {
		http.Error(w, `{"error":"note not found"}`, http.StatusNotFound)
		return
	}

	token, err := models.PublishNote(h.db, note.ID)
	if err != nil {
		http.Error(w, `{"error":"publish failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
		"ok":         true,
		"token":      token,
		"public_url": "/p/" + token,
	})
}

// UnpublishPUT marks a note as private. The token is preserved.
// PUT /notes/{slug}/unpublish
func (h *Handler) UnpublishPUT(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")

	note, err := models.GetNote(h.db, userID, slug)
	if err != nil || note == nil {
		http.Error(w, `{"error":"note not found"}`, http.StatusNotFound)
		return
	}

	if err := models.UnpublishNote(h.db, note.ID); err != nil {
		http.Error(w, `{"error":"unpublish failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true}) //nolint:errcheck
}

// PublicNoteGET renders a publicly shared note without requiring authentication.
// GET /p/{token}
func (h *Handler) PublicNoteGET(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	note, err := models.GetNoteByToken(h.db, token)
	if err != nil || note == nil {
		h.publicNotFound(w)
		return
	}

	body, err := storage.ReadNote(h.notesDir, note.UserID, note.Slug)
	if err != nil {
		body = ""
	}

	// Resolve wiki-links: only to other public notes (private targets → plain text)
	body = models.ResolveWikiLinksPublic(h.db, note.UserID, body)

	var buf bytes.Buffer
	if err := h.md.Convert([]byte(body), &buf); err != nil {
		buf.WriteString("<p>Error rendering markdown</p>")
	}

	html := buf.String()

	// Render checkboxes as inert (strip data attributes, keep disabled)
	html = publicCheckboxRe.ReplaceAllStringFunc(html, func(match string) string {
		checked := ""
		if strings.Contains(match, "checked") {
			checked = ` checked`
		}
		return fmt.Sprintf(`<input type="checkbox" disabled%s class="todo-checkbox todo-readonly">`, checked)
	})

	// Render @due(YYYY-MM-DD) as styled badges (same as private reader)
	now := time.Now().Truncate(24 * time.Hour)
	html = dueBadgeRe.ReplaceAllStringFunc(html, func(match string) string {
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

	// Rewrite private image URLs (/uploads/{username}/{filename}) to public ones (/p/{token}/uploads/{filename})
	html = rewritePublicImageURLs(html, token)

	// Build a short description for Open Graph (first 200 chars of body, stripped)
	description := truncateForMeta(stripMarkdown(body), 200)

	drawings, _ := models.ListDrawings(h.db, note.ID)
	hasLegacyDrawing := len(drawings) == 0 && storage.DrawingExists(h.notesDir, note.UserID, note.Slug)
	legacyDrawingType := ""
	if hasLegacyDrawing {
		legacyDrawingType = string(storage.DetectDrawingType(h.notesDir, note.UserID, note.Slug))
	}

	data := map[string]any{
		"Title":             note.Title,
		"BodyHTML":          template.HTML(html), //nolint:gosec
		"Drawings":          drawings,
		"HasLegacyDrawing":  hasLegacyDrawing,
		"LegacyDrawingType": legacyDrawingType,
		"Token":             token,
		"Description":       description,
		"TldrawLicenseKey":  h.tldrawLicenseKey,
	}

	pagePath := filepath.Join(h.tmplDir, "public", "note.html")
	tmpl, err := template.ParseFiles(pagePath)
	if err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "render error: "+err.Error(), http.StatusInternalServerError)
	}
}

// PublicImageServeGET serves an image embedded in a public note.
// GET /p/{token}/uploads/{filename}
func (h *Handler) PublicImageServeGET(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	filename := chi.URLParam(r, "filename")

	note, err := models.GetNoteByToken(h.db, token)
	if err != nil || note == nil {
		http.NotFound(w, r)
		return
	}

	// Prevent path traversal
	if strings.Contains(filename, "/") || strings.Contains(filename, "..") {
		http.NotFound(w, r)
		return
	}

	// Only serve if the image belongs to this note
	belongs, err := imageBelongsToNote(h.db, note.ID, filename)
	if err != nil || !belongs {
		http.NotFound(w, r)
		return
	}

	path := storage.UploadPath(h.uploadsDir, note.UserID, filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	ext := strings.ToLower(filepath.Ext(filename))
	ct := map[string]string{
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
		".webp": "image/webp",
	}[ext]
	if ct == "" {
		ct = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ct)
	http.ServeFile(w, r, path)
}

// PublicDrawingGET serves the drawing JSON snapshot for a public note's drawing.
// GET /p/{token}/drawing
func (h *Handler) PublicDrawingGET(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	note, err := models.GetNoteByToken(h.db, token)
	if err != nil || note == nil {
		http.NotFound(w, r)
		return
	}

	data, dt, err := storage.ReadDrawing(h.notesDir, note.UserID, note.Slug)
	if os.IsNotExist(err) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "no drawing"}) //nolint:errcheck
		return
	}
	if err != nil {
		http.Error(w, "read error", http.StatusInternalServerError)
		return
	}

	writeDrawingResponse(w, data, dt)
}

// PublicDrawingByIDGET serves a specific drawing JSON for a public note.
// GET /p/{token}/drawings/{drawingID}
func (h *Handler) PublicDrawingByIDGET(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	drawingID := chi.URLParam(r, "drawingID")

	note, err := models.GetNoteByToken(h.db, token)
	if err != nil || note == nil {
		http.NotFound(w, r)
		return
	}

	h.migrateLegacyDrawingIfNeeded(note.UserID, note.ID, note.Slug)

	d, err := models.GetDrawing(h.db, note.ID, drawingID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	dt := storage.DrawingType(d.ToolType)
	data, err := storage.ReadDrawingByID(h.notesDir, note.UserID, note.Slug, drawingID, dt)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	writeDrawingResponse(w, data, dt)
}

// PublicDrawingsListGET returns drawing metadata for a public note.
// GET /p/{token}/drawings
func (h *Handler) PublicDrawingsListGET(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	note, err := models.GetNoteByToken(h.db, token)
	if err != nil || note == nil {
		http.NotFound(w, r)
		return
	}

	h.migrateLegacyDrawingIfNeeded(note.UserID, note.ID, note.Slug)

	drawings, err := models.ListDrawings(h.db, note.ID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	type drawingJSON struct {
		DrawingID   string `json:"drawing_id"`
		DisplayName string `json:"display_name"`
		ToolType    string `json:"tool_type"`
	}
	var result []drawingJSON
	for _, d := range drawings {
		result = append(result, drawingJSON{
			DrawingID:   d.DrawingID,
			DisplayName: d.DisplayName,
			ToolType:    d.ToolType,
		})
	}

	if result == nil {
		result = []drawingJSON{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"drawings": result}) //nolint:errcheck
}

// PublicNotesListGET renders the owner's list of currently-published notes.
// GET /public
func (h *Handler) PublicNotesListGET(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)

	list, err := models.ListPublishedNotes(h.db, userID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	data := h.baseData(r)
	data["Notes"] = list
	h.render(w, r, "public/list.html", data)
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// publicNotFound serves a plain "not found" page with no information leakage.
func (h *Handler) publicNotFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`<!DOCTYPE html><html><head><meta charset="UTF-8"><title>Not found</title><meta name="robots" content="noindex,nofollow"><style>body{font-family:system-ui;text-align:center;padding:4rem 2rem;color:#1e293b}h1{font-size:2rem}p{color:#64748b}</style></head><body><h1>Note not found</h1><p>This note is not available.</p></body></html>`)) //nolint:errcheck
}

// imageBelongsToNote returns true if the image exists in the images table for the given note.
func imageBelongsToNote(db *models.DB, noteID int64, filename string) (bool, error) {
	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM images WHERE note_id = ? AND filename = ?",
		noteID, filename).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// rewritePublicImageURLs replaces /uploads/{username}/{filename} with /p/{token}/uploads/{filename}.
// Access control is enforced at serve time by imageBelongsToNote.
func rewritePublicImageURLs(html, token string) string {
	re := regexp.MustCompile(`/uploads/([^/"\s]+)/([^"\s]+)`)
	return re.ReplaceAllStringFunc(html, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		return "/p/" + token + "/uploads/" + parts[2]
	})
}

// stripMarkdown removes markdown syntax for a plain-text description.
func stripMarkdown(s string) string {
	s = regexp.MustCompile(`[#*_\[\]`+"`"+`>|]`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// truncateForMeta cuts a string to max runes, adding ellipsis.
func truncateForMeta(s string, max int) string {
	if len([]rune(s)) <= max {
		return s
	}
	return string([]rune(s)[:max]) + "…"
}
