package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"

	"github.com/selvakn/yant/internal/models"
	"github.com/selvakn/yant/internal/storage"
	"github.com/selvakn/yant/internal/versioning"
)

// DrawingGET returns the drawing JSON for a note wrapped with a type field, or 404 if no drawing exists.
func (h *Handler) DrawingGET(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")

	note, err := models.GetNote(h.db, userID, slug)
	if err != nil || note == nil {
		http.NotFound(w, r)
		return
	}

	data, dt, err := storage.ReadDrawing(h.notesDir, userID, slug)
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

// DrawingPUT creates or updates the drawing JSON for a note.
// Accepts ?type=excalidraw|tldraw query param (default: tldraw).
func (h *Handler) DrawingPUT(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")

	note, err := models.GetNote(h.db, userID, slug)
	if err != nil || note == nil {
		http.NotFound(w, r)
		return
	}

	dt := parseDrawingType(r.URL.Query().Get("type"))

	existing := storage.DetectDrawingType(h.notesDir, userID, slug)
	if existing != storage.DrawingNone && existing != dt {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": "drawing exists with different tool type"}) //nolint:errcheck
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}

	if err := storage.WriteDrawing(h.notesDir, userID, slug, dt, body); err != nil {
		http.Error(w, "write error", http.StatusInternalServerError)
		return
	}

	relPath := storage.DrawingRelPath(userID, slug, dt)
	if err := versioning.CommitFile(h.notesDir, relPath, "update drawing: "+slug); err != nil {
		log.Printf("versioning: commit drawing %s: %v", slug, err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true}) //nolint:errcheck
}

// DrawingDELETE removes the drawing file (any type) for a note.
func (h *Handler) DrawingDELETE(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")

	note, err := models.GetNote(h.db, userID, slug)
	if err != nil || note == nil {
		http.NotFound(w, r)
		return
	}

	dt := storage.DetectDrawingType(h.notesDir, userID, slug)

	if err := storage.DeleteDrawing(h.notesDir, userID, slug); err != nil {
		http.Error(w, "delete error", http.StatusInternalServerError)
		return
	}

	if dt != storage.DrawingNone {
		relPath := storage.DrawingRelPath(userID, slug, dt)
		if err := versioning.CommitDelete(h.notesDir, relPath, "delete drawing: "+slug); err != nil {
			log.Printf("versioning: commit delete drawing %s: %v", slug, err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true}) //nolint:errcheck
}

// DrawingTypeForNote returns the detected drawing type for a note.
func (h *Handler) DrawingTypeForNote(userID int64, slug string) storage.DrawingType {
	return storage.DetectDrawingType(h.notesDir, userID, slug)
}

func parseDrawingType(raw string) storage.DrawingType {
	if raw == "excalidraw" {
		return storage.DrawingExcalidraw
	}
	return storage.DrawingTldraw
}

// writeDrawingResponse writes a type-wrapped drawing response.
// The data key is "data" for excalidraw, "document" for tldraw.
// Raw content bytes are embedded directly without re-encoding.
func writeDrawingResponse(w http.ResponseWriter, data []byte, dt storage.DrawingType) {
	w.Header().Set("Content-Type", "application/json")

	typeStr := string(dt)
	key := "document"
	if dt == storage.DrawingExcalidraw {
		key = "data"
	}

	w.Write([]byte(fmt.Sprintf(`{"type":%q,%q:`, typeStr, key))) //nolint:errcheck
	w.Write(data)                                                 //nolint:errcheck
	w.Write([]byte(`}`))                                          //nolint:errcheck
}
