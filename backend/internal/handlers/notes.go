package handlers

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/yuin/goldmark"

	"github.com/selvakn/my-notes/internal/models"
	"github.com/selvakn/my-notes/internal/storage"
)

// NotesListGET lists notes for the logged-in user, optionally filtered by tag.
func (h *Handler) NotesListGET(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	tag := r.URL.Query().Get("tag")

	notes, err := models.ListNotes(h.db, userID, tag)
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

	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(body), &buf); err != nil {
		buf.WriteString("<p>Error rendering markdown</p>")
	}

	data := map[string]any{
		"Note":     note,
		"BodyHTML": template.HTML(buf.String()), //nolint:gosec
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
		"Note": note,
		"Body": body,
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

	// htmx: signal client to redirect to editor
	w.Header().Set("HX-Redirect", fmt.Sprintf("/notes/%s/edit", slug))
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

	w.Header().Set("HX-Redirect", "/notes")
	w.WriteHeader(http.StatusOK)
}
