package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/selvakn/yant/internal/models"
)

// TagsListGET returns tags for the logged-in user.
// For htmx requests (HX-Request header), renders the sidebar partial.
// Otherwise returns JSON (includes Color field).
func (h *Handler) TagsListGET(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)

	tags, err := models.ListTagsForUser(h.db, userID, false)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		todoCount := models.CountPendingTodos(h.db, userID)
		publicCount := models.CountPublishedNotes(h.db, userID)
		sharedCount := models.CountSharedNotesForUser(h.db, userID)
		overdueCount := models.CountOverdueTodos(h.db, userID)
		isAdmin := models.IsUserAdmin(h.db, userID)
		blogCount := 0
		if isAdmin {
			blogCount = models.CountBlogPostsForUser(h.db, userID)
		}
		h.renderPartial(w, r, "tags/sidebar.html", map[string]any{
			"Tags":         tags,
			"TodoCount":    todoCount,
			"PublicCount":  publicCount,
			"SharedCount":  sharedCount,
			"OverdueCount": overdueCount,
			"BlogCount":    blogCount,
			"IsAdmin":      isAdmin,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tags) //nolint:errcheck
}

// TagColorPUT sets the color for a tag.
func (h *Handler) TagColorPUT(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	tagName := chi.URLParam(r, "name")

	var req struct {
		Color string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	// Validate color is from palette
	valid := false
	for _, c := range models.ColorPalette {
		if c == req.Color {
			valid = true
			break
		}
	}
	if !valid {
		http.Error(w, "invalid color", http.StatusBadRequest)
		return
	}

	if err := models.SetTagColor(h.db, userID, tagName, req.Color); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true}) //nolint:errcheck
}
