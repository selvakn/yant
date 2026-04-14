package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/selvakn/yant/internal/models"
	"github.com/selvakn/yant/internal/storage"
)

// TodoTogglePUT toggles a todo checkbox in a note's markdown.
// PUT /notes/{slug}/todo with JSON body {"line": N, "checked": bool}
func (h *Handler) TodoTogglePUT(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")

	note, err := models.GetNote(h.db, userID, slug)
	if err != nil || note == nil {
		http.Error(w, `{"error":"note not found"}`, http.StatusNotFound)
		return
	}

	var req struct {
		Line    int  `json:"line"`
		Checked bool `json:"checked"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}

	body, err := storage.ReadNote(h.notesDir, userID, slug)
	if err != nil {
		http.Error(w, `{"error":"read failed"}`, http.StatusInternalServerError)
		return
	}

	newBody, err := models.ToggleTodoInMarkdown(body, req.Line, req.Checked)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	if err := storage.WriteNote(h.notesDir, userID, slug, newBody); err != nil {
		http.Error(w, `{"error":"write failed"}`, http.StatusInternalServerError)
		return
	}

	// Re-sync todos and tags
	todos := models.ParseTodos(newBody)
	if err := models.SyncTodos(h.db, note.ID, todos); err != nil {
		log.Printf("sync todos after toggle: %v", err)
	}
	tags := models.ParseTags(newBody)
	if err := models.SyncTags(h.db, note.ID, tags); err != nil {
		log.Printf("sync tags after toggle: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`)) //nolint:errcheck
}

// TodosListGET renders the aggregated pending todos view.
// GET /todos
func (h *Handler) TodosListGET(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	tag := r.URL.Query().Get("tag")

	todos, err := models.ListPendingTodos(h.db, userID, tag)
	if err != nil {
		log.Printf("list todos: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	tags, err := models.ListTagsForUser(h.db, userID, false)
	if err != nil {
		log.Printf("list tags for todos: %v", err)
	}

	data := h.baseData(r)
	data["Todos"] = todos
	data["Tags"] = tags
	data["ActiveTag"] = tag

	h.render(w, r, "todos/list.html", data)
}
