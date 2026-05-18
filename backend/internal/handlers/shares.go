package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
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

	drawings, err := models.ListDrawings(h.db, note.ID)
	if err != nil {
		log.Printf("list drawings (shared reader): %v", err)
		drawings = nil
	}
	hasLegacyDrawing := len(drawings) == 0 && storage.DrawingExists(h.notesDir, note.UserID, slug)
	legacyDrawingType := ""
	if hasLegacyDrawing {
		legacyDrawingType = string(storage.DetectDrawingType(h.notesDir, note.UserID, slug))
	}

	sharedRelPath := fmt.Sprintf("%d/%s.md", note.UserID, slug)
	lastEditor := ""
	if vs, err := versioning.Log(h.notesDir, sharedRelPath, 1, 0); err == nil && len(vs) > 0 {
		if name := vs[0].AuthorName; name != "" && name != "yant" {
			lastEditor = name
		}
	}

	data := h.baseData(r)
	data["Note"] = note
	data["BodyHTML"] = template.HTML(html) //nolint:gosec
	data["OwnerUsername"] = ownerUsername
	data["Role"] = role
	data["CanEdit"] = role == models.RoleEditor
	data["Drawings"] = drawings
	data["HasLegacyDrawing"] = hasLegacyDrawing
	data["LegacyDrawingType"] = legacyDrawingType
	data["LastEditor"] = lastEditor

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
	drawings, derr := models.ListDrawings(h.db, note.ID)
	if derr != nil {
		log.Printf("list drawings (shared editor): %v", derr)
		drawings = nil
	}
	hasLegacyDrawing := len(drawings) == 0 && storage.DrawingExists(h.notesDir, note.UserID, slug)
	legacyDrawingType := ""
	if hasLegacyDrawing {
		legacyDrawingType = string(storage.DetectDrawingType(h.notesDir, note.UserID, slug))
	}
	data["Drawings"] = drawings
	data["HasLegacyDrawing"] = hasLegacyDrawing
	data["LegacyDrawingType"] = legacyDrawingType
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

// SharedDrawingGET serves the legacy single-file drawing JSON for a shared note.
// GET /shared/{username}/{slug}/drawing
func (h *Handler) SharedDrawingGET(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	ownerUsername := chi.URLParam(r, "username")
	slug := chi.URLParam(r, "slug")

	note, role, err := models.GetNoteForViewer(h.db, userID, ownerUsername, slug)
	if err != nil || note == nil || role == models.RoleOwner || note.Archived {
		http.NotFound(w, r)
		return
	}

	data, dt, err := storage.ReadDrawing(h.notesDir, note.UserID, slug)
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

// SharedDrawingsListGET returns drawing metadata for a note shared with the viewer.
// GET /shared/{username}/{slug}/drawings
func (h *Handler) SharedDrawingsListGET(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	ownerUsername := chi.URLParam(r, "username")
	slug := chi.URLParam(r, "slug")

	note, role, err := models.GetNoteForViewer(h.db, userID, ownerUsername, slug)
	if err != nil || note == nil || role == models.RoleOwner || note.Archived {
		http.NotFound(w, r)
		return
	}

	h.migrateLegacyDrawingIfNeeded(note.UserID, note.ID, slug)

	drawings, err := models.ListDrawings(h.db, note.ID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	body, _ := storage.ReadNote(h.notesDir, note.UserID, slug)
	type drawingJSON struct {
		DrawingID   string `json:"drawing_id"`
		DisplayName string `json:"display_name"`
		ToolType    string `json:"tool_type"`
		HasMarker   bool   `json:"has_marker"`
	}
	var result []drawingJSON
	for _, d := range drawings {
		marker := "![[draw:" + d.DrawingID + "]]"
		result = append(result, drawingJSON{
			DrawingID:   d.DrawingID,
			DisplayName: d.DisplayName,
			ToolType:    d.ToolType,
			HasMarker:   strings.Contains(body, marker),
		})
	}
	if result == nil {
		result = []drawingJSON{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"drawings": result}) //nolint:errcheck
}

// SharedDrawingByIDGET serves drawing JSON by ID for a shared note.
// GET /shared/{username}/{slug}/drawings/{drawingID}
func (h *Handler) SharedDrawingByIDGET(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	ownerUsername := chi.URLParam(r, "username")
	slug := chi.URLParam(r, "slug")
	drawingID := chi.URLParam(r, "drawingID")

	note, role, err := models.GetNoteForViewer(h.db, userID, ownerUsername, slug)
	if err != nil || note == nil || role == models.RoleOwner || note.Archived {
		http.NotFound(w, r)
		return
	}

	h.migrateLegacyDrawingIfNeeded(note.UserID, note.ID, slug)

	d, err := models.GetDrawing(h.db, note.ID, drawingID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	dt := storage.DrawingType(d.ToolType)
	data, err := storage.ReadDrawingByID(h.notesDir, note.UserID, slug, drawingID, dt)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "no drawing content"}) //nolint:errcheck
		return
	}

	writeDrawingResponse(w, data, dt)
}

// sharedDrawingEditorAuth validates that the viewer has edit permission on the
// given shared note and returns the note plus the collaborator's username for
// git attribution. On any failure it writes an appropriate response and
// returns (nil, "", false).
func (h *Handler) sharedDrawingEditorAuth(w http.ResponseWriter, r *http.Request, ownerUsername, slug string) (*models.Note, string, bool) {
	userID := userIDFromSession(r)
	note, role, err := models.GetNoteForViewer(h.db, userID, ownerUsername, slug)
	if err != nil || note == nil || note.Archived {
		http.NotFound(w, r)
		return nil, "", false
	}
	if role != models.RoleEditor {
		// Owners use /notes/...; readers have no write access. Both map to 404
		// to mirror the silent-not-found behavior of the read endpoints.
		http.NotFound(w, r)
		return nil, "", false
	}
	editorUsername := usernameFromSession(r)
	return note, editorUsername, true
}

// SharedDrawingsCreatePOST creates a new drawing on a shared note.
// POST /shared/{username}/{slug}/drawings
func (h *Handler) SharedDrawingsCreatePOST(w http.ResponseWriter, r *http.Request) {
	ownerUsername := chi.URLParam(r, "username")
	slug := chi.URLParam(r, "slug")

	note, _, ok := h.sharedDrawingEditorAuth(w, r, ownerUsername, slug)
	if !ok {
		return
	}

	h.migrateLegacyDrawingIfNeeded(note.UserID, note.ID, slug)

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

// SharedDrawingByIDPUT saves drawing content by drawing ID on a shared note.
// PUT /shared/{username}/{slug}/drawings/{drawingID}
func (h *Handler) SharedDrawingByIDPUT(w http.ResponseWriter, r *http.Request) {
	ownerUsername := chi.URLParam(r, "username")
	slug := chi.URLParam(r, "slug")
	drawingID := chi.URLParam(r, "drawingID")

	note, editorUsername, ok := h.sharedDrawingEditorAuth(w, r, ownerUsername, slug)
	if !ok {
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
	if err := storage.WriteDrawingByID(h.notesDir, note.UserID, slug, drawingID, dt, body); err != nil {
		http.Error(w, "write error", http.StatusInternalServerError)
		return
	}

	models.UpdateDrawingTimestamp(h.db, note.ID, drawingID)

	relPath := storage.DrawingRelPathByID(note.UserID, slug, drawingID, dt)
	if err := versioning.CommitFileAs(
		h.notesDir, relPath,
		"update drawing: "+slug+"/"+drawingID,
		editorUsername,
		editorUsername+"@yant.local",
	); err != nil {
		log.Printf("versioning: commit drawing %s/%s: %v", slug, drawingID, err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true}) //nolint:errcheck
}

// SharedDrawingByIDRenamePATCH renames a drawing on a shared note.
// PATCH /shared/{username}/{slug}/drawings/{drawingID}
func (h *Handler) SharedDrawingByIDRenamePATCH(w http.ResponseWriter, r *http.Request) {
	ownerUsername := chi.URLParam(r, "username")
	slug := chi.URLParam(r, "slug")
	drawingID := chi.URLParam(r, "drawingID")

	note, _, ok := h.sharedDrawingEditorAuth(w, r, ownerUsername, slug)
	if !ok {
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

// SharedDrawingByIDDELETE deletes a drawing on a shared note.
// DELETE /shared/{username}/{slug}/drawings/{drawingID}
func (h *Handler) SharedDrawingByIDDELETE(w http.ResponseWriter, r *http.Request) {
	ownerUsername := chi.URLParam(r, "username")
	slug := chi.URLParam(r, "slug")
	drawingID := chi.URLParam(r, "drawingID")

	note, editorUsername, ok := h.sharedDrawingEditorAuth(w, r, ownerUsername, slug)
	if !ok {
		return
	}

	d, err := models.GetDrawing(h.db, note.ID, drawingID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	dt := storage.DrawingType(d.ToolType)
	if err := storage.DeleteDrawingByID(h.notesDir, note.UserID, slug, drawingID, dt); err != nil {
		http.Error(w, "delete error", http.StatusInternalServerError)
		return
	}
	_ = storage.DeleteDrawingSVG(h.notesDir, note.UserID, slug, drawingID)

	models.DeleteDrawingRecord(h.db, note.ID, drawingID)

	relPath := storage.DrawingRelPathByID(note.UserID, slug, drawingID, dt)
	if err := versioning.CommitDeleteAs(
		h.notesDir, relPath,
		"delete drawing: "+slug+"/"+drawingID,
		editorUsername,
		editorUsername+"@yant.local",
	); err != nil {
		log.Printf("versioning: commit delete drawing %s/%s: %v", slug, drawingID, err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true}) //nolint:errcheck
}

// SharedDrawingSVGPUT stores a pre-rendered SVG preview for a drawing on a shared note.
// PUT /shared/{username}/{slug}/drawings/{drawingID}/svg
func (h *Handler) SharedDrawingSVGPUT(w http.ResponseWriter, r *http.Request) {
	ownerUsername := chi.URLParam(r, "username")
	slug := chi.URLParam(r, "slug")
	drawingID := chi.URLParam(r, "drawingID")

	note, _, ok := h.sharedDrawingEditorAuth(w, r, ownerUsername, slug)
	if !ok {
		return
	}

	if _, err := models.GetDrawing(h.db, note.ID, drawingID); err != nil {
		http.NotFound(w, r)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 5<<20))
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}

	if err := storage.WriteDrawingSVG(h.notesDir, note.UserID, slug, drawingID, body); err != nil {
		http.Error(w, "write error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true}) //nolint:errcheck
}

// SharedNoteHistoryGET handles GET /shared/{username}/{slug}/history
// Collaborators with any permission can view the full version history of a shared note.
func (h *Handler) SharedNoteHistoryGET(w http.ResponseWriter, r *http.Request) {
	viewerID := userIDFromSession(r)
	ownerUsername := chi.URLParam(r, "username")
	slug := chi.URLParam(r, "slug")

	note, role, err := models.GetNoteForViewer(h.db, viewerID, ownerUsername, slug)
	if err != nil || note == nil || role == "" {
		http.NotFound(w, r)
		return
	}
	if role == models.RoleOwner {
		// Owners use the standard history route
		http.Redirect(w, r, "/notes/"+slug+"/history", http.StatusFound)
		return
	}
	if note.Archived {
		http.NotFound(w, r)
		return
	}

	page, perPage := parsePagination(r)
	relPath := fmt.Sprintf("%d/%s.md", note.UserID, slug)

	versions, err := versioning.Log(h.notesDir, relPath, perPage+1, (page-1)*perPage)
	if err != nil {
		http.Error(w, "version history error", http.StatusInternalServerError)
		return
	}

	hasMore := len(versions) > perPage
	if hasMore {
		versions = versions[:perPage]
	}

	data := h.baseData(r)
	data["Note"] = note
	data["OwnerUsername"] = ownerUsername
	data["Role"] = role
	data["Versions"] = versions
	data["Page"] = page
	data["PerPage"] = perPage
	data["HasMore"] = hasMore
	data["PrevPage"] = page - 1
	data["NextPage"] = page + 1
	h.render(w, r, "shared/history.html", data)
}

// SharedNoteVersionDiffGET handles GET /shared/{username}/{slug}/history/{commit}/diff
func (h *Handler) SharedNoteVersionDiffGET(w http.ResponseWriter, r *http.Request) {
	viewerID := userIDFromSession(r)
	ownerUsername := chi.URLParam(r, "username")
	slug := chi.URLParam(r, "slug")
	commit := chi.URLParam(r, "commit")

	if !versioning.ValidCommitHash(commit) {
		http.Error(w, "invalid version identifier", http.StatusBadRequest)
		return
	}

	note, role, err := models.GetNoteForViewer(h.db, viewerID, ownerUsername, slug)
	if err != nil || note == nil || role == "" {
		http.NotFound(w, r)
		return
	}
	if role == models.RoleOwner {
		http.Redirect(w, r, "/notes/"+slug+"/history/"+commit+"/diff", http.StatusFound)
		return
	}
	if note.Archived {
		http.NotFound(w, r)
		return
	}

	against := r.URL.Query().Get("against")
	if against == "" {
		parent, err := versioning.ParentCommit(h.notesDir, commit)
		if err != nil {
			http.Error(w, "no previous version to compare against", http.StatusBadRequest)
			return
		}
		against = parent
	} else if !versioning.ValidCommitHash(against) {
		http.Error(w, "invalid comparison version", http.StatusBadRequest)
		return
	}

	relPath := fmt.Sprintf("%d/%s.md", note.UserID, slug)
	rawDiff, err := versioning.Diff(h.notesDir, relPath, against, commit)
	if err != nil {
		http.Error(w, "diff error", http.StatusInternalServerError)
		return
	}

	newVersion, err := versioning.GetVersion(h.notesDir, commit)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	oldVersion, err := versioning.GetVersion(h.notesDir, against)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	diff := versioning.DiffResult{
		OldCommit: against,
		NewCommit: commit,
		OldDate:   oldVersion.Timestamp,
		NewDate:   newVersion.Timestamp,
		Lines:     versioning.ParseDiff(rawDiff),
	}

	data := h.baseData(r)
	data["Note"] = note
	data["OwnerUsername"] = ownerUsername
	data["Diff"] = diff
	data["OldVersion"] = oldVersion
	data["NewVersion"] = newVersion
	h.render(w, r, "shared/diff.html", data)
}

// SharedNoteVersionRevertPOST handles POST /shared/{username}/{slug}/history/{commit}/revert
// Only editors may revert.
func (h *Handler) SharedNoteVersionRevertPOST(w http.ResponseWriter, r *http.Request) {
	viewerID := userIDFromSession(r)
	ownerUsername := chi.URLParam(r, "username")
	slug := chi.URLParam(r, "slug")
	commit := chi.URLParam(r, "commit")

	if !versioning.ValidCommitHash(commit) {
		http.Error(w, "invalid version identifier", http.StatusBadRequest)
		return
	}

	note, role, err := models.GetNoteForViewer(h.db, viewerID, ownerUsername, slug)
	if err != nil || note == nil || role == "" {
		http.NotFound(w, r)
		return
	}
	if role == models.RoleOwner {
		http.Redirect(w, r, "/notes/"+slug+"/history/"+commit+"/revert", http.StatusSeeOther)
		return
	}
	if role != models.RoleEditor {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	relPath := fmt.Sprintf("%d/%s.md", note.UserID, slug)
	oldContent, err := versioning.Show(h.notesDir, relPath, commit)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	currentContent, _ := storage.ReadNote(h.notesDir, note.UserID, slug)
	if currentContent == oldContent {
		w.Header().Set("HX-Redirect", fmt.Sprintf("/shared/%s/%s", ownerUsername, slug))
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := storage.WriteNote(h.notesDir, note.UserID, slug, oldContent); err != nil {
		http.Error(w, "write error", http.StatusInternalServerError)
		return
	}

	editorUsername := usernameFromSession(r)
	v, _ := versioning.GetVersion(h.notesDir, commit)
	shortHash := commit[:8]
	if v != nil {
		shortHash = v.ShortHash
	}
	commitMsg := fmt.Sprintf("revert: %s to %s", slug, shortHash)
	if err := versioning.CommitFileAs(h.notesDir, relPath, commitMsg, editorUsername, editorUsername+"@yant.local"); err != nil {
		log.Printf("versioning: commit shared revert %s: %v", slug, err)
	}

	tags := models.ParseTags(oldContent)
	_ = models.SyncTags(h.db, note.ID, tags)
	linkedTitles := models.ParseNoteLinks(oldContent)
	_ = models.SyncLinks(h.db, note.ID, note.UserID, linkedTitles)
	todos := models.ParseTodos(oldContent)
	_ = models.SyncTodos(h.db, note.ID, todos)
	h.generateEmbedding(note.ID, note.Title, oldContent)

	w.Header().Set("HX-Redirect", fmt.Sprintf("/shared/%s/%s", ownerUsername, slug))
	w.WriteHeader(http.StatusOK)
}

// ─── regex shared between this file and notes.go ─────────────────────────────

var sharedCheckboxRe = regexp.MustCompile(`<input[^>]*type="checkbox"[^>]*>`)
