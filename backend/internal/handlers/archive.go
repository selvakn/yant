package handlers

import (
	"net/http"

	"github.com/selvakn/yant/internal/models"
)

// ArchiveListGET lists archived notes for the logged-in user, optionally filtered by tag.
func (h *Handler) ArchiveListGET(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	tag := r.URL.Query().Get("tag")

	notes, err := models.ListNotes(h.db, userID, tag, true)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	tags, err := models.ListTagsForUser(h.db, userID, true)
	if err != nil {
		tags = nil
	}

	data := map[string]any{
		"Notes":     notes,
		"Tags":      tags,
		"ActiveTag": tag,
	}
	h.render(w, r, "archive/list.html", data)
}

// ArchiveSearchGET searches archived notes for the logged-in user.
func (h *Handler) ArchiveSearchGET(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	query := r.URL.Query().Get("q")

	results, err := h.searchNotes(userID, query, true)
	if err != nil {
		http.Error(w, "search error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Results": results,
		"Query":   query,
	}
	h.renderPartial(w, r, "archive/search-results.html", data)
}

// ArchiveTagsGET returns tags for archived notes.
func (h *Handler) ArchiveTagsGET(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)

	tags, err := models.ListTagsForUser(h.db, userID, true)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	h.renderPartial(w, r, "archive/tags-sidebar.html", map[string]any{"Tags": tags})
}
