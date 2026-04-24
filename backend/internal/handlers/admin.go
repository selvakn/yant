package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/selvakn/yant/internal/auth"
	"github.com/selvakn/yant/internal/models"
	"github.com/selvakn/yant/internal/storage"
	"github.com/selvakn/yant/internal/versioning"
)

// AdminOnly ensures the current user is an admin. Non-admins receive 404.
func (h *Handler) AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := auth.SessionManager.GetInt64(r.Context(), "userID")
		if !models.IsUserAdmin(h.db, userID) {
			h.RenderError(w, r, http.StatusNotFound, "Page not found")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RegisterAdminRoutes registers all /admin routes. Mount on a router that already
// enforces login and non-disabled users.
func (h *Handler) RegisterAdminRoutes(r chi.Router) {
	r.Route("/admin", func(r chi.Router) {
		r.Use(h.AdminOnly)
		r.Get("/", h.AdminDashboardGET)
		r.Get("/users", h.AdminUsersListGET)
		r.Get("/users/{username}/delete-confirm", h.AdminUserDeleteConfirmGET)
		r.Get("/users/{username}", h.AdminUserDetailGET)
		r.Post("/users/{username}/disable", h.AdminUserDisablePOST)
		r.Post("/users/{username}/enable", h.AdminUserEnablePOST)
		r.Post("/users/{username}/promote", h.AdminUserPromotePOST)
		r.Post("/users/{username}/demote", h.AdminUserDemotePOST)
		r.Delete("/users/{username}", h.AdminUserDeleteDELETE)
		r.Get("/notes", h.AdminNotesListGET)
		r.Get("/notes/{noteID}/delete-confirm", h.AdminNoteDeleteConfirmGET)
		r.Get("/notes/{noteID}", h.AdminNoteDetailGET)
		r.Delete("/notes/{noteID}", h.AdminNoteDeleteDELETE)
		r.Get("/public-notes", h.AdminPublicNotesListGET)
		r.Post("/public-notes/{noteID}/unpublish", h.AdminPublicNoteUnpublishPOST)
		r.Get("/shares", h.AdminSharesListGET)
		r.Delete("/shares/{noteID}/{userID}", h.AdminShareRevokeDELETE)
		r.Get("/audit-log", h.AdminAuditLogGET)
	})
}

func (h *Handler) AdminDashboardGET(w http.ResponseWriter, r *http.Request) {
	metrics, err := models.GetDashboardMetrics(h.db)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.render(w, r, "admin/dashboard.html", map[string]any{
		"Title":   "Admin",
		"Metrics": metrics,
	})
}

func adminPage(r *http.Request) int {
	p := r.URL.Query().Get("page")
	if p == "" {
		return 1
	}
	n, err := strconv.Atoi(p)
	if err != nil || n < 1 {
		return 1
	}
	return n
}

func adminTotalPages(total int) int {
	if total == 0 {
		return 1
	}
	return (total + models.AdminListPageSize - 1) / models.AdminListPageSize
}

func pageNav(page, totalPages int) (prev, next int, hasPrev, hasNext bool) {
	if page > 1 {
		hasPrev = true
		prev = page - 1
	}
	if page < totalPages {
		hasNext = true
		next = page + 1
	}
	return
}

// AdminUsersListGET lists users with optional search and pagination.
func (h *Handler) AdminUsersListGET(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	page := adminPage(r)
	users, total, err := models.ListAllUsers(h.db, q, page)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	tp := adminTotalPages(total)
	pv, nx, hpv, hnx := pageNav(page, tp)
	h.render(w, r, "admin/users.html", map[string]any{
		"Title":      "Users",
		"Users":      users,
		"Search":     q,
		"Page":       page,
		"Total":      total,
		"TotalPages": tp,
		"PrevPage":   pv,
		"NextPage":   nx,
		"HasPrev":    hpv,
		"HasNext":    hnx,
		"PageSize":   models.AdminListPageSize,
	})
}

// AdminUserDetailGET shows a user and related data.
func (h *Handler) AdminUserDetailGET(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	u, err := models.GetAdminUserDetail(h.db, username)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if u == nil {
		http.NotFound(w, r)
		return
	}
	notes, notesTotal, err := models.ListAllNotes(h.db, username, "", "", 1)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	shares, shareTotal, err := models.ListAllShares(h.db, username, 1)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	published, _ := models.ListPublishedNotes(h.db, u.ID)
	impact, _ := models.GetUserImpactSummary(h.db, u.ID)
	h.render(w, r, "admin/user-detail.html", map[string]any{
		"Title":         u.Username,
		"User":          u,
		"Notes":         notes,
		"NotesTotal":    notesTotal,
		"Shares":        shares,
		"SharesTotal":   shareTotal,
		"Published":     published,
		"Impact":        impact,
		"SessionUser":   usernameFromSession(r),
		"OnlyUserAdmin": u.IsAdmin && models.CountAdminUsers(h.db) == 1,
	})
}

func (h *Handler) loadUserByName(username string) (*models.AdminUserView, int64, error) {
	u, err := models.GetAdminUserDetail(h.db, username)
	if err != nil {
		return nil, 0, err
	}
	if u == nil {
		return nil, 0, nil
	}
	return u, u.ID, nil
}

// AdminUserDisablePOST disables a user account.
func (h *Handler) AdminUserDisablePOST(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	adminName := usernameFromSession(r)
	u, id, err := h.loadUserByName(username)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if u == nil {
		http.NotFound(w, r)
		return
	}
	if u.IsAdmin && models.CountAdminUsers(h.db) == 1 {
		http.Error(w, "cannot disable the only admin", http.StatusBadRequest)
		return
	}
	if err := models.DisableUser(h.db, id); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	_ = models.WriteAuditLog(h.db, adminName, models.AuditDisableUser, "user", fmt.Sprintf("%d", id), username)
	if isHX(r) {
		nu, _, _ := h.loadUserByName(username)
		oua := false
		if nu != nil && nu.IsAdmin {
			oua = models.CountAdminUsers(h.db) == 1
		}
		h.renderPartial(w, r, "admin/partials/user-actions.html", map[string]any{
			"User":          nu,
			"Target":        username,
			"OnlyUserAdmin": oua,
		})
		return
	}
	http.Redirect(w, r, "/admin/users/"+url.PathEscape(username), http.StatusFound)
}

// AdminUserEnablePOST re-enables a user account.
func (h *Handler) AdminUserEnablePOST(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	adminName := usernameFromSession(r)
	u, id, err := h.loadUserByName(username)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if u == nil {
		http.NotFound(w, r)
		return
	}
	if err := models.EnableUser(h.db, id); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	_ = models.WriteAuditLog(h.db, adminName, models.AuditEnableUser, "user", fmt.Sprintf("%d", id), username)
	if isHX(r) {
		nu, _, _ := h.loadUserByName(username)
		oua := false
		if nu != nil && nu.IsAdmin {
			oua = models.CountAdminUsers(h.db) == 1
		}
		h.renderPartial(w, r, "admin/partials/user-actions.html", map[string]any{
			"User":          nu,
			"Target":        username,
			"OnlyUserAdmin": oua,
		})
		return
	}
	http.Redirect(w, r, "/admin/users/"+url.PathEscape(username), http.StatusFound)
}

// AdminUserPromotePOST grants admin to a user.
func (h *Handler) AdminUserPromotePOST(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	adminName := usernameFromSession(r)
	u, id, err := h.loadUserByName(username)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if u == nil {
		http.NotFound(w, r)
		return
	}
	if err := models.PromoteAdmin(h.db, id); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	_ = models.WriteAuditLog(h.db, adminName, models.AuditPromoteAdmin, "user", fmt.Sprintf("%d", id), username)
	if isHX(r) {
		nu, _, _ := h.loadUserByName(username)
		oua := false
		if nu != nil && nu.IsAdmin {
			oua = models.CountAdminUsers(h.db) == 1
		}
		h.renderPartial(w, r, "admin/partials/user-actions.html", map[string]any{
			"User":          nu,
			"Target":        username,
			"OnlyUserAdmin": oua,
		})
		return
	}
	http.Redirect(w, r, "/admin/users/"+url.PathEscape(username), http.StatusFound)
}

// AdminUserDemotePOST removes admin from a user.
func (h *Handler) AdminUserDemotePOST(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	adminName := usernameFromSession(r)
	u, id, err := h.loadUserByName(username)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if u == nil {
		http.NotFound(w, r)
		return
	}
	if u.IsAdmin && models.CountAdminUsers(h.db) == 1 {
		http.Error(w, "cannot demote the only admin", http.StatusBadRequest)
		return
	}
	if err := models.DemoteAdmin(h.db, id); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	_ = models.WriteAuditLog(h.db, adminName, models.AuditDemoteAdmin, "user", fmt.Sprintf("%d", id), username)
	if isHX(r) {
		nu, _, _ := h.loadUserByName(username)
		oua := false
		if nu != nil && nu.IsAdmin {
			oua = models.CountAdminUsers(h.db) == 1
		}
		h.renderPartial(w, r, "admin/partials/user-actions.html", map[string]any{
			"User":          nu,
			"Target":        username,
			"OnlyUserAdmin": oua,
		})
		return
	}
	http.Redirect(w, r, "/admin/users/"+url.PathEscape(username), http.StatusFound)
}

// AdminUserDeleteConfirmGET shows a confirmation view before deleting a user.
func (h *Handler) AdminUserDeleteConfirmGET(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	u, err := models.GetAdminUserDetail(h.db, username)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if u == nil {
		http.NotFound(w, r)
		return
	}
	selfID := userIDFromSession(r)
	if u.IsAdmin && models.CountAdminUsers(h.db) == 1 {
		h.render(w, r, "admin/user-delete-confirm.html", map[string]any{
			"Title":  "Delete user",
			"User":   u,
			"Block":  "cannot delete the only admin",
			"Target": username,
		})
		return
	}
	if u.ID == selfID {
		h.render(w, r, "admin/user-delete-confirm.html", map[string]any{
			"Title":  "Delete user",
			"User":   u,
			"Block":  "cannot delete your own account from the admin panel",
			"Target": username,
		})
		return
	}
	summary, _ := models.GetUserImpactSummary(h.db, u.ID)
	h.render(w, r, "admin/user-delete-confirm.html", map[string]any{
		"Title":   "Delete user",
		"User":    u,
		"Summary": summary,
		"Target":  username,
	})
}

// AdminUserDeleteDELETE removes a user and all owned data.
func (h *Handler) AdminUserDeleteDELETE(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	adminName := usernameFromSession(r)
	u, id, err := h.loadUserByName(username)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if u == nil {
		http.NotFound(w, r)
		return
	}
	selfID := userIDFromSession(r)
	if u.ID == selfID {
		http.Error(w, "cannot delete own account", http.StatusBadRequest)
		return
	}
	if u.IsAdmin && models.CountAdminUsers(h.db) == 1 {
		http.Error(w, "cannot delete the only admin", http.StatusBadRequest)
		return
	}
	_ = os.RemoveAll(filepath.Join(h.notesDir, fmt.Sprintf("%d", id)))
	_ = os.RemoveAll(filepath.Join(h.uploadsDir, fmt.Sprintf("%d", id)))
	if err := models.DeleteUserCascade(h.db, id); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	_ = models.WriteAuditLog(h.db, adminName, models.AuditDeleteUser, "user", fmt.Sprintf("%d", id), username)
	if isHX(r) {
		w.Header().Set("HX-Redirect", "/admin/users")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/admin/users", http.StatusFound)
}

// AdminNotesListGET lists notes with filters.
func (h *Handler) AdminNotesListGET(w http.ResponseWriter, r *http.Request) {
	owner := strings.TrimSpace(r.URL.Query().Get("owner"))
	public := r.URL.Query().Get("public")
	shared := r.URL.Query().Get("shared")
	page := adminPage(r)
	notes, total, err := models.ListAllNotes(h.db, owner, public, shared, page)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	tp := adminTotalPages(total)
	pv, nx, hpv, hnx := pageNav(page, tp)
	h.render(w, r, "admin/notes.html", map[string]any{
		"Title":      "Notes",
		"Notes":      notes,
		"Owner":      owner,
		"Public":     public,
		"Shared":     shared,
		"Page":       page,
		"Total":      total,
		"TotalPages": tp,
		"PrevPage":   pv,
		"NextPage":   nx,
		"HasPrev":    hpv,
		"HasNext":    hnx,
		"PageSize":   models.AdminListPageSize,
	})
}

// AdminNoteDetailGET shows a read-only rendered note.
func (h *Handler) AdminNoteDetailGET(w http.ResponseWriter, r *http.Request) {
	noteID, err := strconv.ParseInt(chi.URLParam(r, "noteID"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	n, err := models.GetNoteForAdmin(h.db, noteID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if n == nil {
		http.NotFound(w, r)
		return
	}
	body, err := storage.ReadNote(h.notesDir, n.OwnerID, n.Slug)
	if err != nil {
		body = ""
	}
	body = models.ResolveWikiLinks(h.db, n.OwnerID, body)
	var buf bytes.Buffer
	if err := h.md.Convert([]byte(body), &buf); err != nil {
		buf.WriteString("<p>Error rendering markdown</p>")
	}
	impact, _ := models.GetNoteImpactSummary(h.db, noteID)
	h.render(w, r, "admin/note-detail.html", map[string]any{
		"Title":    n.Title,
		"Note":     n,
		"BodyHTML": template.HTML(buf.String()), //nolint:gosec
		"Summary":  impact,
	})
}

// AdminNoteDeleteConfirmGET shows a confirmation view before deleting a note.
func (h *Handler) AdminNoteDeleteConfirmGET(w http.ResponseWriter, r *http.Request) {
	noteID, err := strconv.ParseInt(chi.URLParam(r, "noteID"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	n, err := models.GetNoteForAdmin(h.db, noteID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if n == nil {
		http.NotFound(w, r)
		return
	}
	summary, _ := models.GetNoteImpactSummary(h.db, noteID)
	h.render(w, r, "admin/note-delete-confirm.html", map[string]any{
		"Title":   "Delete note",
		"Note":    n,
		"Summary": summary,
	})
}

func (h *Handler) adminDeleteNoteWithAssets(noteID int64) error {
	n, err := models.GetNoteForAdmin(h.db, noteID)
	if err != nil {
		return err
	}
	if n == nil {
		return errors.New("note not found")
	}
	uid, slug := n.OwnerID, n.Slug
	filenames, err := models.DeleteImagesForNote(h.db, n.ID)
	if err == nil {
		for _, fn := range filenames {
			_ = os.Remove(filepath.Join(h.uploadsDir, fmt.Sprintf("%d", uid), fn))
		}
	}
	_ = models.DeleteEmbedding(h.db, n.ID)
	if err := models.DeleteNote(h.db, uid, slug); err != nil {
		return err
	}
	_ = storage.DeleteNoteFile(h.notesDir, uid, slug)
	_ = storage.DeleteDrawing(h.notesDir, uid, slug)
	relPath := fmt.Sprintf("%d/%s.md", uid, slug)
	if err := versioning.CommitDelete(h.notesDir, relPath, "admin delete: "+slug); err != nil {
		log.Printf("versioning: commit delete %s: %v", slug, err)
	}
	drawingRelPath := fmt.Sprintf("%d/%s.tldraw.json", uid, slug)
	_ = versioning.CommitDelete(h.notesDir, drawingRelPath, "admin delete drawing: "+slug)
	return nil
}

// AdminNoteDeleteDELETE deletes a note and on-disk assets.
func (h *Handler) AdminNoteDeleteDELETE(w http.ResponseWriter, r *http.Request) {
	noteID, err := strconv.ParseInt(chi.URLParam(r, "noteID"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	adminName := usernameFromSession(r)
	n, err := models.GetNoteForAdmin(h.db, noteID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if n == nil {
		http.NotFound(w, r)
		return
	}
	if err := h.adminDeleteNoteWithAssets(noteID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	_ = models.WriteAuditLog(h.db, adminName, models.AuditDeleteNote, "note", fmt.Sprintf("%d", noteID), "")
	if isHX(r) {
		w.Header().Set("HX-Redirect", "/admin/notes")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/admin/notes", http.StatusFound)
}

// AdminPublicNotesListGET lists published public notes.
func (h *Handler) AdminPublicNotesListGET(w http.ResponseWriter, r *http.Request) {
	page := adminPage(r)
	notes, total, err := models.ListAllPublicNotes(h.db, page)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	tp := adminTotalPages(total)
	pv, nx, hpv, hnx := pageNav(page, tp)
	h.render(w, r, "admin/public-notes.html", map[string]any{
		"Title":      "Public notes",
		"Notes":      notes,
		"Page":       page,
		"Total":      total,
		"TotalPages": tp,
		"PrevPage":   pv,
		"NextPage":   nx,
		"HasPrev":    hpv,
		"HasNext":    hnx,
		"PageSize":   models.AdminListPageSize,
	})
}

// AdminPublicNoteUnpublishPOST unpublishes a public note.
func (h *Handler) AdminPublicNoteUnpublishPOST(w http.ResponseWriter, r *http.Request) {
	noteID, err := strconv.ParseInt(chi.URLParam(r, "noteID"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	adminName := usernameFromSession(r)
	if err := models.UnpublishNote(h.db, noteID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	_ = models.WriteAuditLog(h.db, adminName, models.AuditUnpublishNote, "public", fmt.Sprintf("%d", noteID), "")
	if isHX(r) {
		w.Header().Set("HX-Redirect", "/admin/public-notes")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/admin/public-notes", http.StatusFound)
}

// AdminSharesListGET lists all share rows.
func (h *Handler) AdminSharesListGET(w http.ResponseWriter, r *http.Request) {
	user := strings.TrimSpace(r.URL.Query().Get("user"))
	page := adminPage(r)
	shares, total, err := models.ListAllShares(h.db, user, page)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	tp := adminTotalPages(total)
	pv, nx, hpv, hnx := pageNav(page, tp)
	h.render(w, r, "admin/shares.html", map[string]any{
		"Title":      "Shares",
		"Shares":     shares,
		"User":       user,
		"Page":       page,
		"Total":      total,
		"TotalPages": tp,
		"PrevPage":   pv,
		"NextPage":   nx,
		"HasPrev":    hpv,
		"HasNext":    hnx,
		"PageSize":   models.AdminListPageSize,
	})
}

// AdminShareRevokeDELETE revokes a collaborator share.
func (h *Handler) AdminShareRevokeDELETE(w http.ResponseWriter, r *http.Request) {
	noteID, err1 := strconv.ParseInt(chi.URLParam(r, "noteID"), 10, 64)
	collabID, err2 := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	if err1 != nil || err2 != nil {
		http.NotFound(w, r)
		return
	}
	adminName := usernameFromSession(r)
	if err := models.RevokeShare(h.db, noteID, collabID); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	_ = models.WriteAuditLog(h.db, adminName, models.AuditRevokeShare, "share", fmt.Sprintf("%d:%d", noteID, collabID), "")
	if isHX(r) {
		w.Header().Set("HX-Redirect", "/admin/shares")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/admin/shares", http.StatusFound)
}

// AdminAuditLogGET shows the admin audit log.
func (h *Handler) AdminAuditLogGET(w http.ResponseWriter, r *http.Request) {
	action := strings.TrimSpace(r.URL.Query().Get("action"))
	user := strings.TrimSpace(r.URL.Query().Get("user"))
	page := adminPage(r)
	entries, total, err := models.ListAuditLog(h.db, action, user, page)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	tp := adminTotalPages(total)
	pv, nx, hpv, hnx := pageNav(page, tp)
	h.render(w, r, "admin/audit-log.html", map[string]any{
		"Title":      "Audit log",
		"Entries":    entries,
		"Action":     action,
		"User":       user,
		"Page":       page,
		"Total":      total,
		"TotalPages": tp,
		"PrevPage":   pv,
		"NextPage":   nx,
		"HasPrev":    hpv,
		"HasNext":    hnx,
		"PageSize":   models.AdminListPageSize,
	})
}

func isHX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}
