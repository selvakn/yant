package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/selvakn/yant/internal/models"
	"github.com/selvakn/yant/internal/storage"
	"github.com/selvakn/yant/internal/versioning"
)

// ─── Owner-side share management ─────────────────────────────────────────────

// ShareCreatePUT handles PUT /notes/{slug}/share
// Body: {"username": "...", "permission": "read"|"edit"}
func (h *Handler) ShareCreatePUT(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")

	// Owner lookup on the note
	note, err := models.GetNote(h.db, userID, slug)
	if err != nil || note == nil {
		http.Error(w, `{"error":"note not found"}`, http.StatusNotFound)
		return
	}

	var req struct {
		Username   string `json:"username"`
		Permission string `json:"permission"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		http.Error(w, `{"error":"username required"}`, http.StatusBadRequest)
		return
	}
	if req.Permission != models.PermissionRead && req.Permission != models.PermissionEdit {
		http.Error(w, `{"error":"permission must be read or edit"}`, http.StatusBadRequest)
		return
	}

	target, err := models.GetUserByUsername(h.db, req.Username)
	if err != nil || target == nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}

	if target.ID == userID {
		http.Error(w, `{"error":"cannot share with yourself"}`, http.StatusBadRequest)
		return
	}

	if err := models.GrantShare(h.db, note.ID, target.ID, userID, req.Permission); err != nil {
		if errors.Is(err, models.ErrSelfShare) {
			http.Error(w, `{"error":"cannot share with yourself"}`, http.StatusBadRequest)
			return
		}
		log.Printf("grant share: %v", err)
		http.Error(w, `{"error":"grant failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
		"ok":         true,
		"username":   req.Username,
		"permission": req.Permission,
	})
}

// ShareDeletePUT handles DELETE /notes/{slug}/share/{username}
func (h *Handler) ShareDeletePUT(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")
	username := chi.URLParam(r, "username")

	note, err := models.GetNote(h.db, userID, slug)
	if err != nil || note == nil {
		http.Error(w, `{"error":"note not found"}`, http.StatusNotFound)
		return
	}

	target, err := models.GetUserByUsername(h.db, username)
	if err != nil || target == nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}

	if err := models.RevokeShare(h.db, note.ID, target.ID); err != nil {
		log.Printf("revoke share: %v", err)
		http.Error(w, `{"error":"revoke failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true}) //nolint:errcheck
}

// ShareListGET handles GET /notes/{slug}/shares
func (h *Handler) ShareListGET(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")

	note, err := models.GetNote(h.db, userID, slug)
	if err != nil || note == nil {
		http.Error(w, `{"error":"note not found"}`, http.StatusNotFound)
		return
	}

	collabs, err := models.ListSharesForNote(h.db, note.ID)
	if err != nil {
		log.Printf("list shares: %v", err)
		http.Error(w, `{"error":"list failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"collaborators": collabs}) //nolint:errcheck
}

// ─── Recipient-side shared-note views ────────────────────────────────────────

// SharedNotesListGET handles GET /shared — lists notes shared WITH the viewer.
func (h *Handler) SharedNotesListGET(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)

	notes, err := models.ListSharedNotesForUser(h.db, userID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	data := h.baseData(r)
	data["SharedNotes"] = notes
	h.render(w, r, "shared/list.html", data)
}

// SharedNoteReaderGET handles GET /shared/{username}/{slug}
func (h *Handler) SharedNoteReaderGET(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	ownerUsername := chi.URLParam(r, "username")
	slug := chi.URLParam(r, "slug")

	note, role, err := models.GetNoteForViewer(h.db, userID, ownerUsername, slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if role == models.RoleOwner {
		// Owners should use /notes/{slug}, not the shared path
		http.Redirect(w, r, "/notes/"+slug, http.StatusFound)
		return
	}
	if note.Archived {
		http.NotFound(w, r)
		return
	}

	body, err := storage.ReadNote(h.notesDir, note.UserID, slug)
	if err != nil {
		body = ""
	}

	// Wiki-links: only link to targets also shared with the viewer
	body = models.ResolveWikiLinksForViewer(h.db, userID, note.UserID, ownerUsername, body)

	var buf bytes.Buffer
	if err := h.md.Convert([]byte(body), &buf); err != nil {
		buf.WriteString("<p>Error rendering markdown</p>")
	}
	html := buf.String()

	// Render checkboxes as inert (no toggling for shared notes in v1)
	html = sharedCheckboxRe.ReplaceAllStringFunc(html, func(match string) string {
		checked := ""
		if strings.Contains(match, "checked") {
			checked = ` checked`
		}
		return fmt.Sprintf(`<input type="checkbox" disabled%s class="todo-checkbox todo-readonly">`, checked)
	})

	// @due badges (same as private reader)
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

	data := h.baseData(r)
	data["Note"] = note
	data["BodyHTML"] = template.HTML(html) //nolint:gosec
	data["OwnerUsername"] = ownerUsername
	data["Role"] = role
	data["CanEdit"] = role == models.RoleEditor
	data["HasDrawing"] = storage.DrawingExists(h.notesDir, note.UserID, slug)
	data["DrawingType"] = string(storage.DetectDrawingType(h.notesDir, note.UserID, slug))

	h.render(w, r, "shared/reader.html", data)
}

// SharedNoteEditorGET handles GET /shared/{username}/{slug}/edit
func (h *Handler) SharedNoteEditorGET(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	ownerUsername := chi.URLParam(r, "username")
	slug := chi.URLParam(r, "slug")

	note, role, err := models.GetNoteForViewer(h.db, userID, ownerUsername, slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if role == models.RoleOwner {
		http.Redirect(w, r, "/notes/"+slug+"/edit", http.StatusFound)
		return
	}
	if role != models.RoleEditor {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if note.Archived {
		http.NotFound(w, r)
		return
	}

	body, err := storage.ReadNote(h.notesDir, note.UserID, slug)
	if err != nil {
		body = ""
	}

	data := h.baseData(r)
	data["Note"] = note
	data["Body"] = body
	data["OwnerUsername"] = ownerUsername
	data["HasDrawing"] = storage.DrawingExists(h.notesDir, note.UserID, slug)
	data["DrawingType"] = string(storage.DetectDrawingType(h.notesDir, note.UserID, slug))
	h.render(w, r, "shared/editor.html", data)
}

// SharedNoteUpdate handles POST /shared/{username}/{slug}
// Mirrors noteUpdate but writes to the owner's file and uses the collaborator's identity
// for git attribution.
func (h *Handler) SharedNoteUpdate(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	ownerUsername := chi.URLParam(r, "username")
	slug := chi.URLParam(r, "slug")

	note, role, err := models.GetNoteForViewer(h.db, userID, ownerUsername, slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if role != models.RoleEditor && role != models.RoleOwner {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	title := r.FormValue("title")
	body := r.FormValue("body")
	if title == "" {
		title = "Untitled Note"
	}

	if err := models.UpdateNoteByID(h.db, note.ID, title); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	if err := storage.WriteNote(h.notesDir, note.UserID, slug, body); err != nil {
		http.Error(w, "write error", http.StatusInternalServerError)
		return
	}

	// Git attribution — commit as the viewer (collaborator or owner; both identified by username)
	editorUsername := usernameFromSession(r)
	relPath := fmt.Sprintf("%d/%s.md", note.UserID, slug)
	commitErr := versioning.CommitFileAs(
		h.notesDir, relPath,
		"update: "+slug,
		editorUsername,
		editorUsername+"@yant.local",
	)
	if commitErr != nil {
		log.Printf("versioning: commit update %s: %v", slug, commitErr)
	}

	// Sync tags, links, todos, embedding
	tags := models.ParseTags(body)
	_ = models.SyncTags(h.db, note.ID, tags)

	linkedTitles := models.ParseNoteLinks(body)
	_ = models.SyncLinks(h.db, note.ID, note.UserID, linkedTitles)

	todos := models.ParseTodos(body)
	if err := models.SyncTodos(h.db, note.ID, todos); err != nil {
		log.Printf("sync todos: %v", err)
	}

	h.generateEmbedding(note.ID, title, body)

	w.Header().Set("HX-Redirect", fmt.Sprintf("/shared/%s/%s/edit", ownerUsername, slug))
	w.WriteHeader(http.StatusOK)
}

// ─── regex shared between this file and notes.go ─────────────────────────────

var sharedCheckboxRe = regexp.MustCompile(`<input[^>]*type="checkbox"[^>]*>`)
