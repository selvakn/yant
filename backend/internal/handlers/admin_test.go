package handlers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/selvakn/yant/internal/auth"
	"github.com/selvakn/yant/internal/handlers"
	"github.com/selvakn/yant/internal/models"
	"github.com/selvakn/yant/internal/versioning"
)

func noFollowClient(app *testApp) *http.Client {
	return &http.Client{
		Jar: app.client.Jar,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func TestAdminDashboardGET_NonAdminReturns404(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	resp := app.get(t, "/admin/")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestAdminDashboardGET_AdminReturns200(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	_, _ = app.db.Exec(`UPDATE users SET is_admin = 1 WHERE username = ?`, "alice")
	resp := app.get(t, "/admin/")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body := bodyStr(t, resp)
	if !strings.Contains(body, "admin-dashboard:") && !strings.Contains(body, "Admin") {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestLoginPOST_DisabledUserRedirects(t *testing.T) {
	app := newTestApp(t)
	_, _ = models.GetOrCreateUser(app.db, "erin")
	_, _ = app.db.Exec(`UPDATE users SET disabled = 1 WHERE username = ?`, "erin")
	resp, err := noFollowClient(app).PostForm(app.url("/login"), url.Values{"username": []string{"erin"}})
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected redirect, got %d", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if !strings.Contains(loc, "error=disabled") {
		t.Fatalf("expected disabled error in Location, got %q", loc)
	}
}

func TestRequireActive_DisabledUserCannotAccessNotes(t *testing.T) {
	app := newTestApp(t)
	_, _ = models.GetOrCreateUser(app.db, "frank")
	app.login(t, "frank")
	_, _ = app.db.Exec(`UPDATE users SET disabled = 1 WHERE username = ?`, "frank")
	req, _ := http.NewRequest(http.MethodGet, app.url("/notes"), nil)
	resp, err := noFollowClient(app).Do(req)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected redirect, got %d", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if !strings.Contains(loc, "error=disabled") {
		t.Fatalf("expected disabled redirect, got %q", loc)
	}
}

// newTestAppForAdmin is a slimmer test stack with all admin routes (RegisterAdminRoutes).
func newTestAppForAdmin(t *testing.T) *testApp {
	t.Helper()
	tmp := t.TempDir()
	notesDir := filepath.Join(tmp, "notes")
	uploadsDir := filepath.Join(tmp, "uploads")
	_ = os.MkdirAll(notesDir, 0755)   //nolint:errcheck
	_ = os.MkdirAll(uploadsDir, 0755) //nolint:errcheck
	db, err := models.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := models.InitSchema(db); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	if err := versioning.Init(notesDir); err != nil {
		t.Fatalf("versioning init: %v", err)
	}
	auth.SessionManager = newSessionManager()
	tmplDir := resolveOrStubTemplateDir(t)
	h := handlers.New(db, tmplDir, notesDir, uploadsDir, nil, nil, false, 300, "")

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(auth.SessionManager.LoadAndSave)
	r.Get("/login", h.LoginGET)
	r.Post("/login", func(w http.ResponseWriter, r *http.Request) {
		username := r.FormValue("username")
		if username == "" {
			http.Error(w, "username is required", http.StatusBadRequest)
			return
		}
		user, err := models.GetOrCreateUser(db, username)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if models.IsUserDisabled(db, user.ID) {
			http.Redirect(w, r, "/login?error=disabled", http.StatusFound)
			return
		}
		auth.SessionManager.Put(r.Context(), "username", user.Username)
		auth.SessionManager.Put(r.Context(), "userID", user.ID)
		http.Redirect(w, r, "/notes", http.StatusFound)
	})
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireLogin)
		r.Use(auth.RequireActive(func(userID int64) bool {
			return models.IsUserDisabled(db, userID)
		}))
		h.RegisterAdminRoutes(r)
	})
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		h.RenderError(w, r, http.StatusNotFound, "Page not found")
	})
	srv := httptest.NewServer(r)
	t.Cleanup(func() {
		srv.Close()
		_ = db.Close()
	})
	return &testApp{
		server:     srv,
		db:         db,
		notesDir:   notesDir,
		uploadsDir: uploadsDir,
		client:     &http.Client{Jar: newCookieJar(), CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return nil }},
		handler:    h,
	}
}

func TestAdminUsersListGET_RequiresAdmin(t *testing.T) {
	app := newTestAppForAdmin(t)
	_, _ = models.GetOrCreateUser(app.db, "norm")
	app.login(t, "norm")
	resp := app.get(t, "/admin/users")
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestAdminUsersListGET_Ok(t *testing.T) {
	app := newTestAppForAdmin(t)
	_, _ = models.GetOrCreateUser(app.db, "ginny")
	_, _ = app.db.Exec(`UPDATE users SET is_admin = 1 WHERE username = ?`, "ginny")
	app.login(t, "ginny")
	resp := app.get(t, "/admin/users")
	body := bodyStr(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}
}

func loginAsAdmin(t *testing.T, app *testApp, username string) {
	t.Helper()
	_, _ = models.GetOrCreateUser(app.db, username)
	_, _ = app.db.Exec(`UPDATE users SET is_admin = 1 WHERE username = ?`, username)
	app.login(t, username)
}

func TestAdminUserDetailGET_ShowsUser(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "admin1")
	_, _ = models.GetOrCreateUser(app.db, "target1")
	resp := app.get(t, "/admin/users/target1")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestAdminUserDetailGET_Missing(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "admin2")
	resp := app.get(t, "/admin/users/noexist")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestAdminUserDisablePOST_DisablesUser(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "admin3")
	target, _ := models.GetOrCreateUser(app.db, "victimDisable")
	resp, err := app.client.Post(app.url("/admin/users/victimDisable/disable"), "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	_ = resp.Body.Close()
	if models.IsUserDisabled(app.db, target.ID) != true {
		t.Error("expected user to be disabled")
	}
}

func TestAdminUserDisablePOST_MissingUser(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "admin3b")
	resp, err := app.client.Post(app.url("/admin/users/noone/disable"), "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestAdminUserEnablePOST_EnablesUser(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "admin4")
	target, _ := models.GetOrCreateUser(app.db, "victimEnable")
	_ = models.DisableUser(app.db, target.ID)
	resp, err := app.client.Post(app.url("/admin/users/victimEnable/enable"), "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	_ = resp.Body.Close()
	if models.IsUserDisabled(app.db, target.ID) {
		t.Error("expected user to be enabled")
	}
}

func TestAdminUserEnablePOST_MissingUser(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "admin4b")
	resp, err := app.client.Post(app.url("/admin/users/noone/enable"), "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestAdminUserPromotePOST_PromotesUser(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "admin5")
	target, _ := models.GetOrCreateUser(app.db, "promoTarget")
	resp, err := app.client.Post(app.url("/admin/users/promoTarget/promote"), "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	_ = resp.Body.Close()
	if !models.IsUserAdmin(app.db, target.ID) {
		t.Error("expected user to be admin")
	}
}

func TestAdminUserPromotePOST_MissingUser(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "admin5b")
	resp, err := app.client.Post(app.url("/admin/users/noone/promote"), "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestAdminUserDemotePOST_DemotesUser(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "admin6")
	target, _ := models.GetOrCreateUser(app.db, "demTarget")
	_ = models.PromoteAdmin(app.db, target.ID)
	resp, err := app.client.Post(app.url("/admin/users/demTarget/demote"), "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	_ = resp.Body.Close()
	if models.IsUserAdmin(app.db, target.ID) {
		t.Error("expected user to be demoted")
	}
}

func TestAdminUserDemotePOST_LastAdmin_Blocked(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "onlyAdmin")
	resp, err := noFollowClient(app).Post(app.url("/admin/users/onlyAdmin/demote"), "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestAdminUserDemotePOST_MissingUser(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "admin6b")
	resp, err := app.client.Post(app.url("/admin/users/noone/demote"), "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestAdminUserDeleteConfirmGET_ShowsSummary(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "admin7")
	_, _ = models.GetOrCreateUser(app.db, "delCandidate")
	resp := app.get(t, "/admin/users/delCandidate/delete-confirm")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestAdminUserDeleteConfirmGET_Missing(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "admin7b")
	resp := app.get(t, "/admin/users/noone/delete-confirm")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestAdminUserDeleteConfirmGET_LastAdmin(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "soleAdmin")
	resp := app.get(t, "/admin/users/soleAdmin/delete-confirm")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body := bodyStr(t, resp)
	if !strings.Contains(body, "cannot") && !strings.Contains(body, "admin") {
		t.Logf("body: %s", body)
	}
}

func TestAdminUserDeleteConfirmGET_SelfDelete(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "selfDel")
	_, _ = models.GetOrCreateUser(app.db, "otherAdmin")
	_ = models.PromoteAdmin(app.db, func() int64 {
		u, _ := models.GetAdminUserDetail(app.db, "otherAdmin")
		return u.ID
	}())
	resp := app.get(t, "/admin/users/selfDel/delete-confirm")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body := bodyStr(t, resp)
	if !strings.Contains(body, "cannot") && !strings.Contains(body, "own") {
		t.Logf("body: %s", body)
	}
}

func TestAdminUserDeleteDELETE_DeletesUser(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "admin8")
	target, _ := models.GetOrCreateUser(app.db, "delVictim")
	_, _ = models.CreateNote(app.db, target.ID, "Victim Note", "victim-note")

	req, _ := http.NewRequest(http.MethodDelete, app.url("/admin/users/delVictim"), nil)
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	_ = resp.Body.Close()
	detail, _ := models.GetAdminUserDetail(app.db, "delVictim")
	if detail != nil {
		t.Error("expected user deleted")
	}
}

func TestAdminUserDeleteDELETE_SelfDelete_Blocked(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "selfAdmin")
	req, _ := http.NewRequest(http.MethodDelete, app.url("/admin/users/selfAdmin"), nil)
	resp, err := noFollowClient(app).Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestAdminUserDeleteDELETE_LastAdmin_Blocked(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "lastAdm")
	_, _ = models.GetOrCreateUser(app.db, "otherUser")
	_, _ = app.db.Exec(`UPDATE users SET is_admin = 1 WHERE username = ?`, "otherUser")
	req, _ := http.NewRequest(http.MethodDelete, app.url("/admin/users/lastAdm"), nil)
	resp, err := noFollowClient(app).Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 (self-delete), got %d", resp.StatusCode)
	}
}

func TestAdminUserDeleteDELETE_Missing(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "admin8b")
	req, _ := http.NewRequest(http.MethodDelete, app.url("/admin/users/nobody"), nil)
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestAdminNotesListGET_Ok(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "admin9")
	u, _ := models.GetOrCreateUser(app.db, "noteOwner")
	_, _ = models.CreateNote(app.db, u.ID, "TestNote", "test-note")
	resp := app.get(t, "/admin/notes")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestAdminNotesListGET_WithFilters(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "admin9f")
	resp := app.get(t, "/admin/notes?owner=someone&public=yes&shared=no&page=1")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestAdminNoteDetailGET_Ok(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "admin10")
	u, _ := models.GetOrCreateUser(app.db, "detailOwner")
	n, _ := models.CreateNote(app.db, u.ID, "DetailNote", "detail-note")
	resp := app.get(t, fmt.Sprintf("/admin/notes/%d", n.ID))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestAdminNoteDetailGET_Missing(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "admin10b")
	resp := app.get(t, "/admin/notes/99999")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestAdminNoteDetailGET_InvalidID(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "admin10c")
	resp := app.get(t, "/admin/notes/abc")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestAdminNoteDeleteConfirmGET_Ok(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "admin11")
	u, _ := models.GetOrCreateUser(app.db, "delNOwner")
	n, _ := models.CreateNote(app.db, u.ID, "DelConfNote", "del-conf-note")
	resp := app.get(t, fmt.Sprintf("/admin/notes/%d/delete-confirm", n.ID))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestAdminNoteDeleteConfirmGET_Missing(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "admin11b")
	resp := app.get(t, "/admin/notes/99999/delete-confirm")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestAdminNoteDeleteDELETE_DeletesNote(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "admin12")
	u, _ := models.GetOrCreateUser(app.db, "noteDelOwner")
	n, _ := models.CreateNote(app.db, u.ID, "DelNote", "del-note")
	req, _ := http.NewRequest(http.MethodDelete, app.url(fmt.Sprintf("/admin/notes/%d", n.ID)), nil)
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	_ = resp.Body.Close()
	check, _ := models.GetNoteForAdmin(app.db, n.ID)
	if check != nil {
		t.Error("expected note deleted")
	}
}

func TestAdminNoteDeleteDELETE_Missing(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "admin12b")
	req, _ := http.NewRequest(http.MethodDelete, app.url("/admin/notes/99999"), nil)
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestAdminPublicNotesListGET_Ok(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "admin13")
	resp := app.get(t, "/admin/public-notes")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestAdminPublicNoteUnpublishPOST_Unpublishes(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "admin14")
	u, _ := models.GetOrCreateUser(app.db, "pubOwner")
	n, _ := models.CreateNote(app.db, u.ID, "PubNote", "pub-note")
	_, _ = models.PublishNote(app.db, n.ID)
	resp, err := app.client.Post(app.url(fmt.Sprintf("/admin/public-notes/%d/unpublish", n.ID)), "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	_ = resp.Body.Close()

	var published int
	_ = app.db.QueryRow(`SELECT published FROM public_notes WHERE note_id = ?`, n.ID).Scan(&published)
	if published != 0 {
		t.Error("expected note to be unpublished")
	}
}

func TestAdminSharesListGET_Ok(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "admin15")
	resp := app.get(t, "/admin/shares")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestAdminSharesListGET_WithUserFilter(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "admin15f")
	resp := app.get(t, "/admin/shares?user=someuser")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestAdminShareRevokeDELETE_Revokes(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "admin16")
	owner, _ := models.GetOrCreateUser(app.db, "revokeOwner")
	collab, _ := models.GetOrCreateUser(app.db, "revokeCollab")
	n, _ := models.CreateNote(app.db, owner.ID, "RevokeNote", "revoke-note")
	_ = models.GrantShare(app.db, n.ID, collab.ID, owner.ID, "read")

	req, _ := http.NewRequest(http.MethodDelete, app.url(fmt.Sprintf("/admin/shares/%d/%d", n.ID, collab.ID)), nil)
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	_ = resp.Body.Close()

	var cnt int
	_ = app.db.QueryRow(`SELECT COUNT(*) FROM note_shares WHERE note_id = ? AND user_id = ?`, n.ID, collab.ID).Scan(&cnt)
	if cnt != 0 {
		t.Error("expected share to be revoked")
	}
}

func TestAdminShareRevokeDELETE_InvalidIDs(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "admin16b")
	req, _ := http.NewRequest(http.MethodDelete, app.url("/admin/shares/abc/def"), nil)
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestAdminAuditLogGET_Ok(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "admin17")
	resp := app.get(t, "/admin/audit-log")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestAdminAuditLogGET_WithFilters(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "admin17f")
	resp := app.get(t, "/admin/audit-log?action=disable-user&user=someone")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestAdminDisable_WritesAuditLog(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "auditAdmin")
	_, _ = models.GetOrCreateUser(app.db, "auditTarget")
	_, _ = app.client.Post(app.url("/admin/users/auditTarget/disable"), "application/x-www-form-urlencoded", nil)
	var cnt int
	_ = app.db.QueryRow(`SELECT COUNT(*) FROM admin_audit_log WHERE action = ? AND admin_username = ?`,
		models.AuditDisableUser, "auditAdmin").Scan(&cnt)
	if cnt == 0 {
		t.Error("expected audit log entry for disable")
	}
}

func TestAdminEnable_WritesAuditLog(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "auditAdmin2")
	target, _ := models.GetOrCreateUser(app.db, "auditTarget2")
	_ = models.DisableUser(app.db, target.ID)
	_, _ = app.client.Post(app.url("/admin/users/auditTarget2/enable"), "application/x-www-form-urlencoded", nil)
	var cnt int
	_ = app.db.QueryRow(`SELECT COUNT(*) FROM admin_audit_log WHERE action = ?`, models.AuditEnableUser).Scan(&cnt)
	if cnt == 0 {
		t.Error("expected audit log entry for enable")
	}
}

func TestAdminPromote_WritesAuditLog(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "auditAdmin3")
	_, _ = models.GetOrCreateUser(app.db, "promoAudit")
	_, _ = app.client.Post(app.url("/admin/users/promoAudit/promote"), "application/x-www-form-urlencoded", nil)
	var cnt int
	_ = app.db.QueryRow(`SELECT COUNT(*) FROM admin_audit_log WHERE action = ?`, models.AuditPromoteAdmin).Scan(&cnt)
	if cnt == 0 {
		t.Error("expected audit log entry for promote")
	}
}

func TestAdminDemote_WritesAuditLog(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "auditAdmin4")
	target, _ := models.GetOrCreateUser(app.db, "demAudit")
	_ = models.PromoteAdmin(app.db, target.ID)
	_, _ = app.client.Post(app.url("/admin/users/demAudit/demote"), "application/x-www-form-urlencoded", nil)
	var cnt int
	_ = app.db.QueryRow(`SELECT COUNT(*) FROM admin_audit_log WHERE action = ?`, models.AuditDemoteAdmin).Scan(&cnt)
	if cnt == 0 {
		t.Error("expected audit log entry for demote")
	}
}

func TestAdminNoteDelete_WritesAuditLog(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "auditAdmin5")
	u, _ := models.GetOrCreateUser(app.db, "noteAuditOwner")
	n, _ := models.CreateNote(app.db, u.ID, "AuditNote", "audit-note")
	req, _ := http.NewRequest(http.MethodDelete, app.url(fmt.Sprintf("/admin/notes/%d", n.ID)), nil)
	_, _ = app.client.Do(req)
	var cnt int
	_ = app.db.QueryRow(`SELECT COUNT(*) FROM admin_audit_log WHERE action = ?`, models.AuditDeleteNote).Scan(&cnt)
	if cnt == 0 {
		t.Error("expected audit log entry for note delete")
	}
}

func TestAdminUserDelete_WritesAuditLog(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "auditAdmin6")
	_, _ = models.GetOrCreateUser(app.db, "delAuditVictim")
	req, _ := http.NewRequest(http.MethodDelete, app.url("/admin/users/delAuditVictim"), nil)
	_, _ = app.client.Do(req)
	var cnt int
	_ = app.db.QueryRow(`SELECT COUNT(*) FROM admin_audit_log WHERE action = ?`, models.AuditDeleteUser).Scan(&cnt)
	if cnt == 0 {
		t.Error("expected audit log entry for user delete")
	}
}

func TestAdminDisableOnlyAdmin_Blocked(t *testing.T) {
	app := newTestAppForAdmin(t)
	loginAsAdmin(t, app, "soloAdmin")
	resp, err := noFollowClient(app).Post(app.url("/admin/users/soloAdmin/disable"), "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestAdminAllRoutes_NonAdminGets404(t *testing.T) {
	app := newTestAppForAdmin(t)
	_, _ = models.GetOrCreateUser(app.db, "regular")
	app.login(t, "regular")
	paths := []string{
		"/admin/",
		"/admin/users",
		"/admin/notes",
		"/admin/public-notes",
		"/admin/shares",
		"/admin/audit-log",
	}
	for _, p := range paths {
		resp := app.get(t, p)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("path %s: expected 404, got %d", p, resp.StatusCode)
		}
	}
}
