package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/selvakn/yant/internal/models"
	"github.com/selvakn/yant/internal/storage"
	"github.com/selvakn/yant/internal/versioning"
)

var checkboxRe = regexp.MustCompile(`<input[^>]*type="checkbox"[^>]*>`)
var dueBadgeRe = regexp.MustCompile(`@due\((\d{4}-\d{2}-\d{2})\)`)

// NotesListGET lists notes for the logged-in user, optionally filtered by tag.
func (h *Handler) NotesListGET(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	tag := r.URL.Query().Get("tag")

	notes, err := models.ListNotes(h.db, userID, tag, false)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Notes":     notes,
		"ActiveTag": tag,
	}
	h.render(w, r, "notes/list.html", data)
}

// NotesCreatePOST creates a new note and redirects to its editor.
func (h *Handler) NotesCreatePOST(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	title := r.FormValue("title")
	body := r.FormValue("body")
	if title == "" {
		title = "Untitled Note"
	}

	slug, err := models.GenerateSlug(h.db, userID, title)
	if err != nil {
		http.Error(w, "slug error", http.StatusInternalServerError)
		return
	}

	note, err := models.CreateNote(h.db, userID, title, slug)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	if err := storage.WriteNote(h.notesDir, userID, slug, body); err != nil {
		http.Error(w, "write error", http.StatusInternalServerError)
		return
	}

	relPath := fmt.Sprintf("%d/%s.md", userID, slug)
	if err := versioning.CommitFile(h.notesDir, relPath, "create: "+slug); err != nil {
		log.Printf("versioning: commit create %s: %v", slug, err)
	}

	// Parse and sync tags from the initial body
	tags := models.ParseTags(body)
	if len(tags) > 0 {
		_ = models.SyncTags(h.db, note.ID, tags)
	}

	linkedTitles := models.ParseNoteLinks(body)
	if len(linkedTitles) > 0 {
		_ = models.SyncLinks(h.db, note.ID, userID, linkedTitles)
	}

	todos := models.ParseTodos(body)
	if err := models.SyncTodos(h.db, note.ID, todos); err != nil {
		log.Printf("sync todos: %v", err)
	}

	h.generateEmbedding(note.ID, title, body)

	http.Redirect(w, r, fmt.Sprintf("/notes/%s/edit", note.Slug), http.StatusFound)
}

// NoteReaderGET renders a note in reader mode (goldmark HTML).
func (h *Handler) NoteReaderGET(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")

	note, err := models.GetNote(h.db, userID, slug)
	if err != nil || note == nil {
		http.NotFound(w, r)
		return
	}

	body, err := storage.ReadNote(h.notesDir, userID, slug)
	if err != nil {
		body = ""
	}

	// Resolve [[note title]] wiki-links to markdown links before rendering
	body = models.ResolveWikiLinks(h.db, userID, body)

	// Parse todos before rendering to map checkbox order to line numbers
	todos := models.ParseTodos(body)

	var buf bytes.Buffer
	if err := h.md.Convert([]byte(body), &buf); err != nil {
		buf.WriteString("<p>Error rendering markdown</p>")
	}

	// Post-process: make checkboxes interactive
	html := buf.String()
	todoIdx := 0
	html = checkboxRe.ReplaceAllStringFunc(html, func(match string) string {
		if todoIdx >= len(todos) {
			return match
		}
		todo := todos[todoIdx]
		todoIdx++
		checked := ""
		checkedClass := ""
		if strings.Contains(match, "checked") {
			checked = ` checked`
			checkedClass = " todo-checked"
		}
		return fmt.Sprintf(`<input type="checkbox"%s class="todo-checkbox%s" data-slug="%s" data-line="%d">`,
			checked, checkedClass, note.Slug, todo.Line)
	})

	// Post-process: render @due(YYYY-MM-DD) as styled badges
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

	backlinks, _ := models.GetBacklinks(h.db, note.ID)

	// Public share state
	share, _ := models.GetPublicShare(h.db, note.ID)
	isPublic := share != nil && share.Published
	publicURL := ""
	if isPublic {
		publicURL = "/p/" + share.Token
	}

	// Collaborator list for the Share dialog
	collaborators, _ := models.ListSharesForNote(h.db, note.ID)

	drawings, err := models.ListDrawings(h.db, note.ID)
	if err != nil {
		log.Printf("list drawings: %v", err)
		drawings = nil
	}
	hasLegacyDrawing := len(drawings) == 0 && storage.DrawingExists(h.notesDir, userID, slug)
	legacyDrawingType := ""
	if hasLegacyDrawing {
		legacyDrawingType = string(storage.DetectDrawingType(h.notesDir, userID, slug))
	}

	isBlog := models.IsBlogPost(h.db, note.ID)

	data := map[string]any{
		"Note":              note,
		"BodyHTML":          template.HTML(html), //nolint:gosec
		"Backlinks":         backlinks,
		"Drawings":          drawings,
		"HasLegacyDrawing":  hasLegacyDrawing,
		"LegacyDrawingType": legacyDrawingType,
		"IsPublic":          isPublic,
		"PublicURL":         publicURL,
		"Collaborators":     collaborators,
		"IsBlog":            isBlog,
	}
	h.render(w, r, "notes/reader.html", data)
}

// NoteEditorGET renders a note in editor mode (EasyMDE).
func (h *Handler) NoteEditorGET(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")

	note, err := models.GetNote(h.db, userID, slug)
	if err != nil || note == nil {
		http.NotFound(w, r)
		return
	}

	body, err := storage.ReadNote(h.notesDir, userID, slug)
	if err != nil {
		body = ""
	}

	drawings, err := models.ListDrawings(h.db, note.ID)
	if err != nil {
		log.Printf("list drawings: %v", err)
		drawings = nil
	}
	hasLegacyDrawing := len(drawings) == 0 && storage.DrawingExists(h.notesDir, userID, slug)
	legacyDrawingType := ""
	if hasLegacyDrawing {
		legacyDrawingType = string(storage.DetectDrawingType(h.notesDir, userID, slug))
	}

	isBlog := models.IsBlogPost(h.db, note.ID)

	data := map[string]any{
		"Note":              note,
		"Body":              body,
		"Drawings":          drawings,
		"HasLegacyDrawing":  hasLegacyDrawing,
		"LegacyDrawingType": legacyDrawingType,
		"IsBlog":            isBlog,
	}
	h.render(w, r, "notes/editor.html", data)
}

// NoteUpdateOrDelete dispatches PUT (update) or DELETE based on X-HTTP-Method-Override.
func (h *Handler) NoteUpdateOrDelete(w http.ResponseWriter, r *http.Request) {
	switch r.Header.Get("X-HTTP-Method-Override") {
	case "DELETE":
		h.noteDelete(w, r)
	default:
		h.noteUpdate(w, r)
	}
}

func (h *Handler) noteUpdate(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")

	title := r.FormValue("title")
	body := r.FormValue("body")
	if title == "" {
		title = "Untitled Note"
	}

	note, err := models.UpdateNote(h.db, userID, slug, title)
	if err != nil || note == nil {
		http.NotFound(w, r)
		return
	}

	if err := storage.WriteNote(h.notesDir, userID, slug, body); err != nil {
		http.Error(w, "write error", http.StatusInternalServerError)
		return
	}

	relPath := fmt.Sprintf("%d/%s.md", userID, slug)
	if err := versioning.CommitFile(h.notesDir, relPath, "update: "+slug); err != nil {
		log.Printf("versioning: commit update %s: %v", slug, err)
	}

	tags := models.ParseTags(body)
	_ = models.SyncTags(h.db, note.ID, tags)

	linkedTitles := models.ParseNoteLinks(body)
	_ = models.SyncLinks(h.db, note.ID, userID, linkedTitles)

	todos := models.ParseTodos(body)
	if err := models.SyncTodos(h.db, note.ID, todos); err != nil {
		log.Printf("sync todos: %v", err)
	}

	h.generateEmbedding(note.ID, title, body)

	// htmx: signal client to redirect to editor
	w.Header().Set("HX-Redirect", fmt.Sprintf("/notes/%s/edit", slug))
	w.WriteHeader(http.StatusOK)
}

// NotesSearchGET searches notes for the logged-in user.
func (h *Handler) NotesSearchGET(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	query := r.URL.Query().Get("q")

	results, err := h.searchNotes(userID, query, false)
	if err != nil {
		http.Error(w, "search error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Results": results,
		"Query":   query,
	}
	h.renderPartial(w, r, "notes/search-results.html", data)
}

// NotesArchivePUT archives a note.
func (h *Handler) NotesArchivePUT(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")

	// Fetch note ID before archiving so we can revoke the public share
	note, _ := models.GetNote(h.db, userID, slug)

	if err := models.ArchiveNote(h.db, userID, slug); err != nil {
		http.NotFound(w, r)
		return
	}

	// Archive revokes public access immediately
	if note != nil {
		_ = models.UnpublishNote(h.db, note.ID)
	}

	w.Header().Set("HX-Redirect", "/notes")
	w.WriteHeader(http.StatusOK)
}

// NotesRestorePUT restores an archived note.
func (h *Handler) NotesRestorePUT(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")

	if err := models.RestoreNote(h.db, userID, slug); err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("HX-Redirect", "/archive")
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) noteDelete(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")

	// Get note first to retrieve its ID for image cleanup
	note, err := models.GetNote(h.db, userID, slug)
	if err != nil || note == nil {
		http.NotFound(w, r)
		return
	}

	// Clean up images
	filenames, err := models.DeleteImagesForNote(h.db, note.ID)
	if err == nil {
		for _, fn := range filenames {
			_ = os.Remove(filepath.Join(h.uploadsDir, fmt.Sprintf("%d", userID), fn))
		}
	}

	_ = models.DeleteEmbedding(h.db, note.ID)

	if err := models.DeleteNote(h.db, userID, slug); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	_ = storage.DeleteNoteFile(h.notesDir, userID, slug)
	_ = storage.DeleteDrawing(h.notesDir, userID, slug)

	relPath := fmt.Sprintf("%d/%s.md", userID, slug)
	if err := versioning.CommitDelete(h.notesDir, relPath, "delete: "+slug); err != nil {
		log.Printf("versioning: commit delete %s: %v", slug, err)
	}
	drawingRelPath := fmt.Sprintf("%d/%s.tldraw.json", userID, slug)
	_ = versioning.CommitDelete(h.notesDir, drawingRelPath, "delete drawing: "+slug)

	w.Header().Set("HX-Redirect", "/notes")
	w.WriteHeader(http.StatusOK)
}

// NotesAutocompleteGET returns matching note titles as JSON for [[link]] autocomplete.
func (h *Handler) NotesAutocompleteGET(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	query := r.URL.Query().Get("q")

	results, err := models.SearchNotesByTitle(h.db, userID, query)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results) //nolint:errcheck
}

// generateEmbedding creates a vector embedding for a note if the embedder is available.
func (h *Handler) generateEmbedding(noteID int64, title, body string) {
	embedder := h.embedder.Load()
	if embedder == nil {
		return
	}
	hash := models.ContentHash(title, body)
	if !models.NeedsEmbedding(h.db, noteID, hash) {
		return
	}
	text := models.PrepareEmbeddingText(title, body)
	emb, err := embedder.Embed(text)
	if err != nil {
		log.Printf("embedding: generate failed for note %d: %v", noteID, err)
		return
	}
	if err := models.UpsertEmbedding(h.db, noteID, emb, hash); err != nil {
		log.Printf("embedding: store failed for note %d: %v", noteID, err)
	}
}

// searchNotes dispatches to semantic or text-based search based on handler config.
func (h *Handler) searchNotes(userID int64, query string, archived bool) ([]models.SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return models.SearchNotes(h.db, h.notesDir, userID, query, archived)
	}
	embedder := h.embedder.Load()
	if h.semanticSearchEnabled && embedder != nil {
		queryEmb, err := embedder.Embed(query)
		if err != nil {
			log.Printf("embedding: query embed failed, falling back to text search: %v", err)
			return models.SearchNotes(h.db, h.notesDir, userID, query, archived)
		}
		return models.SemanticSearch(h.db, h.notesDir, userID, query, queryEmb, archived,
			models.DefaultSimilarityThreshold, models.DefaultMaxResults)
	}
	return models.SearchNotes(h.db, h.notesDir, userID, query, archived)
}
