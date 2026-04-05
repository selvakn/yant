package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"

	"github.com/selvakn/my-notes/internal/models"
	"github.com/selvakn/my-notes/internal/storage"
)

// DrawingGET returns the tldraw JSON for a note, or 404 if no drawing exists.
func (h *Handler) DrawingGET(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")

	note, err := models.GetNote(h.db, userID, slug)
	if err != nil || note == nil {
		http.NotFound(w, r)
		return
	}

	data, err := storage.ReadDrawing(h.notesDir, userID, slug)
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

	w.Header().Set("Content-Type", "application/json")
	w.Write(data) //nolint:errcheck
}

// DrawingPUT creates or updates the tldraw JSON for a note.
func (h *Handler) DrawingPUT(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")

	note, err := models.GetNote(h.db, userID, slug)
	if err != nil || note == nil {
		http.NotFound(w, r)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20)) // 10MB limit
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}

	if err := storage.WriteDrawing(h.notesDir, userID, slug, body); err != nil {
		http.Error(w, "write error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true}) //nolint:errcheck
}

// DrawingDELETE removes the tldraw JSON for a note.
func (h *Handler) DrawingDELETE(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")

	note, err := models.GetNote(h.db, userID, slug)
	if err != nil || note == nil {
		http.NotFound(w, r)
		return
	}

	if err := storage.DeleteDrawing(h.notesDir, userID, slug); err != nil {
		http.Error(w, "delete error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true}) //nolint:errcheck
}

// DrawingExistsForNote returns true if a drawing file exists for the given note.
func (h *Handler) DrawingExistsForNote(userID int64, slug string) bool {
	return storage.DrawingExists(h.notesDir, userID, slug)
}
