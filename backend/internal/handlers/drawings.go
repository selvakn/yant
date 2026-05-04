package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

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

// DrawingsListGET returns all drawings for a note as JSON.
func (h *Handler) DrawingsListGET(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")

	note, err := models.GetNote(h.db, userID, slug)
	if err != nil || note == nil {
		http.NotFound(w, r)
		return
	}

	drawings, err := models.ListDrawings(h.db, note.ID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	body, _ := storage.ReadNote(h.notesDir, userID, slug)
	type drawingJSON struct {
		DrawingID   string `json:"drawing_id"`
		DisplayName string `json:"display_name"`
		ToolType    string `json:"tool_type"`
		HasMarker   bool   `json:"has_marker"`
		CreatedAt   string `json:"created_at"`
		UpdatedAt   string `json:"updated_at"`
	}
	var result []drawingJSON
	for _, d := range drawings {
		marker := "![[draw:" + d.DrawingID + "]]"
		result = append(result, drawingJSON{
			DrawingID:   d.DrawingID,
			DisplayName: d.DisplayName,
			ToolType:    d.ToolType,
			HasMarker:   strings.Contains(body, marker),
			CreatedAt:   d.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   d.UpdatedAt.Format(time.RFC3339),
		})
	}
	if result == nil {
		result = []drawingJSON{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"drawings": result}) //nolint:errcheck
}

// DrawingsCreatePOST creates a new drawing for a note.
func (h *Handler) DrawingsCreatePOST(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")

	note, err := models.GetNote(h.db, userID, slug)
	if err != nil || note == nil {
		http.NotFound(w, r)
		return
	}

	var req struct {
		DisplayName string `json:"display_name"`
		ToolType    string `json:"tool_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	drawing, err := models.CreateDrawing(h.db, note.ID, req.DisplayName, req.ToolType)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}) //nolint:errcheck
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{ //nolint:errcheck
		"drawing_id":   drawing.DrawingID,
		"display_name": drawing.DisplayName,
		"tool_type":    drawing.ToolType,
		"marker":       "![[draw:" + drawing.DrawingID + "]]",
	})
}

// DrawingByIDGET returns the drawing content by drawing ID.
func (h *Handler) DrawingByIDGET(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")
	drawingID := chi.URLParam(r, "drawingID")

	note, err := models.GetNote(h.db, userID, slug)
	if err != nil || note == nil {
		http.NotFound(w, r)
		return
	}

	d, err := models.GetDrawing(h.db, note.ID, drawingID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	dt := storage.DrawingType(d.ToolType)
	data, err := storage.ReadDrawingByID(h.notesDir, userID, slug, drawingID, dt)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "no drawing content"}) //nolint:errcheck
		return
	}

	writeDrawingResponse(w, data, dt)
}

// DrawingByIDPUT saves drawing content by drawing ID.
func (h *Handler) DrawingByIDPUT(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")
	drawingID := chi.URLParam(r, "drawingID")

	note, err := models.GetNote(h.db, userID, slug)
	if err != nil || note == nil {
		http.NotFound(w, r)
		return
	}

	d, err := models.GetDrawing(h.db, note.ID, drawingID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}

	dt := storage.DrawingType(d.ToolType)
	if err := storage.WriteDrawingByID(h.notesDir, userID, slug, drawingID, dt, body); err != nil {
		http.Error(w, "write error", http.StatusInternalServerError)
		return
	}

	models.UpdateDrawingTimestamp(h.db, note.ID, drawingID)

	relPath := storage.DrawingRelPathByID(userID, slug, drawingID, dt)
	if err := versioning.CommitFile(h.notesDir, relPath, "update drawing: "+slug+"/"+drawingID); err != nil {
		log.Printf("versioning: commit drawing %s/%s: %v", slug, drawingID, err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true}) //nolint:errcheck
}

// DrawingByIDRenamePATCH renames a drawing's display name.
func (h *Handler) DrawingByIDRenamePATCH(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")
	drawingID := chi.URLParam(r, "drawingID")

	note, err := models.GetNote(h.db, userID, slug)
	if err != nil || note == nil {
		http.NotFound(w, r)
		return
	}

	var req struct {
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := models.RenameDrawing(h.db, note.ID, drawingID, req.DisplayName); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}) //nolint:errcheck
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "display_name": req.DisplayName}) //nolint:errcheck
}

// DrawingByIDDELETE deletes a specific drawing by ID.
func (h *Handler) DrawingByIDDELETE(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")
	drawingID := chi.URLParam(r, "drawingID")

	note, err := models.GetNote(h.db, userID, slug)
	if err != nil || note == nil {
		http.NotFound(w, r)
		return
	}

	d, err := models.GetDrawing(h.db, note.ID, drawingID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	dt := storage.DrawingType(d.ToolType)
	if err := storage.DeleteDrawingByID(h.notesDir, userID, slug, drawingID, dt); err != nil {
		http.Error(w, "delete error", http.StatusInternalServerError)
		return
	}

	models.DeleteDrawingRecord(h.db, note.ID, drawingID)

	relPath := storage.DrawingRelPathByID(userID, slug, drawingID, dt)
	if err := versioning.CommitDelete(h.notesDir, relPath, "delete drawing: "+slug+"/"+drawingID); err != nil {
		log.Printf("versioning: commit delete drawing %s/%s: %v", slug, drawingID, err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true}) //nolint:errcheck
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
