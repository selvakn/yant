package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/selvakn/my-notes/internal/models"
)

// TagsListGET returns tags for the logged-in user.
// For htmx requests (HX-Request header), renders the sidebar partial.
// Otherwise returns JSON.
func (h *Handler) TagsListGET(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)

	tags, err := models.ListTagsForUser(h.db, userID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		h.renderPartial(w, r, "tags/sidebar.html", map[string]any{"Tags": tags})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tags) //nolint:errcheck
}
