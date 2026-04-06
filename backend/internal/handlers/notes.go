package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/yuin/goldmark"

	"github.com/selvakn/yant/internal/models"
	"github.com/selvakn/yant/internal/storage"
)

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

	// Parse and sync tags from the initial body
	tags := models.ParseTags(body)
	if len(tags) > 0 {
		_ = models.SyncTags(h.db, note.ID, tags)
	}

	linkedTitles := models.ParseNoteLinks(body)
	if len(linkedTitles) > 0 {
		_ = models.SyncLinks(h.db, note.ID, userID, linkedTitles)
	}

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

	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(body), &buf); err != nil {
		buf.WriteString("<p>Error rendering markdown</p>")
	}

	backlinks, _ := models.GetBacklinks(h.db, note.ID)

	data := map[string]any{
		"Note":       note,
		"BodyHTML":   template.HTML(buf.String()), //nolint:gosec
		"Backlinks":  backlinks,
		"HasDrawing": storage.DrawingExists(h.notesDir, userID, slug),
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

	data := map[string]any{
		"Note":       note,
		"Body":       body,
		"HasDrawing": storage.DrawingExists(h.notesDir, userID, slug),
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

	tags := models.ParseTags(body)
	_ = models.SyncTags(h.db, note.ID, tags)

	linkedTitles := models.ParseNoteLinks(body)
	_ = models.SyncLinks(h.db, note.ID, userID, linkedTitles)

	// htmx: signal client to redirect to editor
	w.Header().Set("HX-Redirect", fmt.Sprintf("/notes/%s/edit", slug))
	w.WriteHeader(http.StatusOK)
}

// NotesSearchGET searches notes for the logged-in user.
func (h *Handler) NotesSearchGET(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	query := r.URL.Query().Get("q")

	results, err := models.SearchNotes(h.db, h.notesDir, userID, query, false)
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

	if err := models.ArchiveNote(h.db, userID, slug); err != nil {
		http.NotFound(w, r)
		return
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

	if err := models.DeleteNote(h.db, userID, slug); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	_ = storage.DeleteNoteFile(h.notesDir, userID, slug)
	_ = storage.DeleteDrawing(h.notesDir, userID, slug)

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
