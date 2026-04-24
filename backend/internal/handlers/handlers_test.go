package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
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

// ── Test helpers ─────────────────────────────────────────────────────────────

type testApp struct {
	server     *httptest.Server
	db         *models.DB
	notesDir   string
	uploadsDir string
	client     *http.Client
	handler    *handlers.Handler
}

func newTestApp(t *testing.T) *testApp {
	t.Helper()
	tmp := t.TempDir()
	notesDir := filepath.Join(tmp, "notes")
	uploadsDir := filepath.Join(tmp, "uploads")
	os.MkdirAll(notesDir, 0755)   //nolint:errcheck
	os.MkdirAll(uploadsDir, 0755) //nolint:errcheck

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
	// Test-only direct login route (replaces the removed username-only login)
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
	r.Post("/logout", h.LogoutPOST)

	// Public (unauthenticated) routes
	r.Get("/p/{token}", h.PublicNoteGET)
	r.Get("/p/{token}/uploads/{filename}", h.PublicImageServeGET)
	r.Get("/p/{token}/drawing", h.PublicDrawingGET)

	r.Group(func(r chi.Router) {
		r.Use(auth.RequireLogin)
		r.Use(auth.RequireActive(func(userID int64) bool {
			return models.IsUserDisabled(db, userID)
		}))
		r.Get("/notes", h.NotesListGET)
		r.Get("/notes/search", h.NotesSearchGET)
		r.Get("/notes/autocomplete", h.NotesAutocompleteGET)
		r.Post("/notes", h.NotesCreatePOST)
		r.Get("/notes/{slug}", h.NoteReaderGET)
		r.Get("/notes/{slug}/edit", h.NoteEditorGET)
		r.Post("/notes/{slug}", h.NoteUpdateOrDelete)
		r.Put("/notes/{slug}/archive", h.NotesArchivePUT)
		r.Put("/notes/{slug}/restore", h.NotesRestorePUT)
		r.Post("/notes/{slug}/images", h.ImageUploadPOST)
		r.Get("/notes/{slug}/drawing", h.DrawingGET)
		r.Put("/notes/{slug}/drawing", h.DrawingPUT)
		r.Delete("/notes/{slug}/drawing", h.DrawingDELETE)
		r.Get("/tags", h.TagsListGET)
		r.Put("/tags/{name}/color", h.TagColorPUT)
		r.Get("/uploads/{username}/{filename}", h.ImageServeGET)

		r.Get("/notes/{slug}/history", h.NoteHistoryGET)
		r.Get("/notes/{slug}/history/{commit}", h.NoteVersionGET)
		r.Get("/notes/{slug}/history/{commit}/diff", h.NoteVersionDiffGET)
		r.Get("/notes/{slug}/history/{commit}/drawing", h.NoteVersionDrawingGET)
		r.Post("/notes/{slug}/history/{commit}/revert", h.NoteVersionRevertPOST)

		r.Get("/todos", h.TodosListGET)
		r.Put("/notes/{slug}/todo", h.TodoTogglePUT)

		r.Put("/notes/{slug}/publish", h.PublishPUT)
		r.Put("/notes/{slug}/unpublish", h.UnpublishPUT)
		r.Get("/public", h.PublicNotesListGET)

		r.Put("/notes/{slug}/share", h.ShareCreatePUT)
		r.Delete("/notes/{slug}/share/{username}", h.ShareDeletePUT)
		r.Get("/notes/{slug}/shares", h.ShareListGET)

		r.Get("/shared", h.SharedNotesListGET)
		r.Get("/shared/{username}/{slug}", h.SharedNoteReaderGET)
		r.Get("/shared/{username}/{slug}/edit", h.SharedNoteEditorGET)
		r.Post("/shared/{username}/{slug}", h.SharedNoteUpdate)

		r.Get("/archive", h.ArchiveListGET)
		r.Get("/archive/search", h.ArchiveSearchGET)
		r.Get("/archive/tags", h.ArchiveTagsGET)

		h.RegisterAdminRoutes(r)

		// Test-only error trigger routes
		r.Get("/test/error/403", func(w http.ResponseWriter, r *http.Request) {
			h.RenderError(w, r, http.StatusForbidden, "Access denied")
		})
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		h.RenderError(w, r, http.StatusNotFound, "Page not found")
	})

	srv := httptest.NewServer(r)
	t.Cleanup(func() {
		srv.Close()
		db.Close()
	})

	jar := newCookieJar()
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil // follow redirects
		},
	}

	return &testApp{
		server:     srv,
		db:         db,
		notesDir:   notesDir,
		uploadsDir: uploadsDir,
		client:     client,
		handler:    h,
	}
}

func (a *testApp) url(path string) string {
	return a.server.URL + path
}

// login performs a login POST and returns the final response.
func (a *testApp) login(t *testing.T, username string) *http.Response {
	t.Helper()
	resp, err := a.client.PostForm(a.url("/login"), url.Values{"username": {username}})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	return resp
}

func (a *testApp) postForm(t *testing.T, path string, vals url.Values) *http.Response {
	t.Helper()
	resp, err := a.client.PostForm(a.url(path), vals)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return resp
}

func (a *testApp) get(t *testing.T, path string) *http.Response {
	t.Helper()
	resp, err := a.client.Get(a.url(path))
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return resp
}

func (a *testApp) put(t *testing.T, path string, body io.Reader) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPut, a.url(path), body)
	if err != nil {
		t.Fatalf("PUT %s: create request: %v", path, err)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		t.Fatalf("PUT %s: %v", path, err)
	}
	return resp
}

func bodyStr(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(b)
}

// ── T011: Auth tests ──────────────────────────────────────────────────────────

func TestLoginPOST_ValidUsername(t *testing.T) {
	app := newTestApp(t)
	resp := app.login(t, "alice")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 after redirect, got %d", resp.StatusCode)
	}
	// Should end up at /notes
	if !strings.HasSuffix(resp.Request.URL.Path, "/notes") {
		t.Errorf("expected redirect to /notes, got %s", resp.Request.URL.Path)
	}
}

func TestLoginPOST_EmptyUsername(t *testing.T) {
	app := newTestApp(t)
	resp, err := app.client.PostForm(app.url("/login"), url.Values{"username": {""}})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestLoginPOST_NewUsernameAutoCreates(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "newuser")
	u, err := models.GetUserByUsername(app.db, "newuser")
	if err != nil {
		t.Fatalf("user not found: %v", err)
	}
	if u.Username != "newuser" {
		t.Errorf("expected newuser, got %s", u.Username)
	}
}

func TestLoginPOST_ExistingUserNoError(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	// Second login with same username should succeed
	resp := app.login(t, "alice")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestLogoutPOST(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	resp := app.postForm(t, "/logout", nil)
	defer resp.Body.Close()
	// After logout, /notes should redirect to login
	resp2 := app.get(t, "/notes")
	defer resp2.Body.Close()
	if !strings.HasSuffix(resp2.Request.URL.Path, "/login") {
		t.Errorf("expected redirect to /login after logout, got %s", resp2.Request.URL.Path)
	}
}

func TestGetNotes_WithoutLogin_RedirectsToLogin(t *testing.T) {
	app := newTestApp(t)
	// Fresh client with no cookies
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return nil }}
	resp, err := client.Get(app.url("/notes"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if !strings.HasSuffix(resp.Request.URL.Path, "/login") {
		t.Errorf("expected redirect to /login, got %s", resp.Request.URL.Path)
	}
}

// ── T012: Notes tests ─────────────────────────────────────────────────────────

func TestPostNotes_CreatesDBRowAndFile(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	resp := app.postForm(t, "/notes", url.Values{
		"title": {"My First Note"},
		"body":  {"Hello **world**"},
	})
	defer resp.Body.Close()

	// Check DB row
	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, err := models.ListNotes(app.db, u.ID, "", false)
	if err != nil || len(notes) == 0 {
		t.Fatalf("note not found in DB: %v", err)
	}
	note := notes[0]
	if note.Title != "My First Note" {
		t.Errorf("expected 'My First Note', got %q", note.Title)
	}

	// Check .md file exists
	mdPath := filepath.Join(app.notesDir, fmt.Sprintf("%d", u.ID), note.Slug+".md")
	if _, err := os.Stat(mdPath); os.IsNotExist(err) {
		t.Errorf("expected .md file at %s, does not exist", mdPath)
	}
}

func TestPostNotes_EmptyTitleDefaultsToUntitled(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	app.postForm(t, "/notes", url.Values{"title": {""}, "body": {""}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	if len(notes) == 0 {
		t.Fatal("no notes found")
	}
	if notes[0].Title != "Untitled Note" {
		t.Errorf("expected 'Untitled Note', got %q", notes[0].Title)
	}
}

func TestGetNoteSlug_RendersMarkdown(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Test Note"}, "body": {"# Hello"}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	resp := app.get(t, "/notes/"+slug)
	body := bodyStr(t, resp)
	if !strings.Contains(body, "<h1>") {
		t.Errorf("expected rendered <h1> in reader, got: %s", body[:min(200, len(body))])
	}
}

func TestGetNoteSlugEdit_ReturnsRawBody(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Edit Test"}, "body": {"raw **markdown**"}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	resp := app.get(t, "/notes/"+slug+"/edit")
	body := bodyStr(t, resp)
	if !strings.Contains(body, "raw **markdown**") {
		t.Errorf("expected raw markdown in editor page, got: %s", body[:min(300, len(body))])
	}
}

func TestPutNote_UpdatesDBAndFile(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Original"}, "body": {"original body"}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	req, _ := http.NewRequest("POST", app.url("/notes/"+slug), strings.NewReader(url.Values{
		"title": {"Updated Title"},
		"body":  {"updated body"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-HTTP-Method-Override", "PUT")
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	// Check DB updated
	note, _ := models.GetNote(app.db, u.ID, slug)
	if note == nil || note.Title != "Updated Title" {
		t.Errorf("DB title not updated, got: %v", note)
	}

	// Check file updated
	mdPath := filepath.Join(app.notesDir, fmt.Sprintf("%d", u.ID), slug+".md")
	content, _ := os.ReadFile(mdPath)
	if !strings.Contains(string(content), "updated body") {
		t.Errorf("file not updated: %s", string(content))
	}
}

func TestDeleteNote_RemovesDBRowAndFile(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"To Delete"}, "body": {"bye"}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug
	mdPath := filepath.Join(app.notesDir, fmt.Sprintf("%d", u.ID), slug+".md")

	req, _ := http.NewRequest("POST", app.url("/notes/"+slug), nil)
	req.Header.Set("X-HTTP-Method-Override", "DELETE")
	resp, _ := app.client.Do(req)
	resp.Body.Close()

	// DB row gone
	remaining, _ := models.ListNotes(app.db, u.ID, "", false)
	if len(remaining) != 0 {
		t.Errorf("expected 0 notes after delete, got %d", len(remaining))
	}
	// File gone
	if _, err := os.Stat(mdPath); !os.IsNotExist(err) {
		t.Errorf("expected .md file to be deleted")
	}
}

func TestGetNotes_ListsOnlyOwnNotes(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Alice Note"}, "body": {""}})
	app.postForm(t, "/logout", nil)

	// Create second client for bob
	jar2 := newCookieJar()
	client2 := &http.Client{Jar: jar2, CheckRedirect: func(req *http.Request, via []*http.Request) error { return nil }}
	client2.PostForm(app.url("/login"), url.Values{"username": {"bob"}}) //nolint:errcheck
	client2.PostForm(app.url("/notes"), url.Values{"title": {"Bob Note"}, "body": {""}}) //nolint:errcheck

	// Alice logs back in and checks her notes
	app.login(t, "alice")
	resp := app.get(t, "/notes")
	body := bodyStr(t, resp)
	if strings.Contains(body, "Bob Note") {
		t.Errorf("alice should not see bob's notes")
	}
	if !strings.Contains(body, "Alice Note") {
		t.Errorf("alice should see her own notes")
	}
}

// ── T025: Image tests ─────────────────────────────────────────────────────────

func TestImageUpload_ValidPNG(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Img Note"}, "body": {""}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	// Minimal 1x1 PNG bytes
	pngData := minimalPNG()
	body, ct := multipartFile(t, "image", "test.png", pngData)

	req, _ := http.NewRequest("POST", app.url("/notes/"+slug+"/images"), body)
	req.Header.Set("Content-Type", ct)
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	respBody := bodyStr(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, respBody)
	}
	if !strings.Contains(respBody, `"url"`) {
		t.Errorf("expected JSON with url, got: %s", respBody)
	}

	// Verify file on disk
	imgDir := filepath.Join(app.uploadsDir, fmt.Sprintf("%d", u.ID))
	entries, _ := os.ReadDir(imgDir)
	if len(entries) == 0 {
		t.Error("expected image file to be saved on disk")
	}
}

func TestImageUpload_NonImageReturns400(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"N"}, "body": {""}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	body, ct := multipartFile(t, "image", "doc.txt", []byte("not an image"))
	req, _ := http.NewRequest("POST", app.url("/notes/"+slug+"/images"), body)
	req.Header.Set("Content-Type", ct)
	resp, _ := app.client.Do(req)
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestImageServe_OwnerCanAccess(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Img"}, "body": {""}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	pngData := minimalPNG()
	body, ct := multipartFile(t, "image", "test.png", pngData)
	req, _ := http.NewRequest("POST", app.url("/notes/"+slug+"/images"), body)
	req.Header.Set("Content-Type", ct)
	uploadResp, _ := app.client.Do(req)
	uploadBody := bodyStr(t, uploadResp)

	// Extract URL from JSON response
	urlStart := strings.Index(uploadBody, `"/uploads/`)
	if urlStart == -1 {
		t.Fatalf("no upload URL in response: %s", uploadBody)
	}
	urlEnd := strings.Index(uploadBody[urlStart+1:], `"`)
	imgURL := uploadBody[urlStart+1 : urlStart+1+urlEnd]

	resp := app.get(t, imgURL)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for image, got %d", resp.StatusCode)
	}
}

func TestImageServe_OtherUserForbidden(t *testing.T) {
	app := newTestApp(t)
	// alice uploads an image
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Img"}, "body": {""}})
	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	pngData := minimalPNG()
	body, ct := multipartFile(t, "image", "test.png", pngData)
	req, _ := http.NewRequest("POST", app.url("/notes/"+notes[0].Slug+"/images"), body)
	req.Header.Set("Content-Type", ct)
	uploadResp, _ := app.client.Do(req)
	uploadBody := bodyStr(t, uploadResp)

	urlStart := strings.Index(uploadBody, `"/uploads/`)
	urlEnd := strings.Index(uploadBody[urlStart+1:], `"`)
	imgURL := uploadBody[urlStart+1 : urlStart+1+urlEnd]

	// bob tries to access alice's image
	app.postForm(t, "/logout", nil)
	app.login(t, "bob")
	resp := app.get(t, imgURL)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

func TestDeleteNote_CleansUpImages(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"With Img"}, "body": {""}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	pngData := minimalPNG()
	body, ct := multipartFile(t, "image", "test.png", pngData)
	req, _ := http.NewRequest("POST", app.url("/notes/"+slug+"/images"), body)
	req.Header.Set("Content-Type", ct)
	uploadResp, _ := app.client.Do(req)
	uploadBody := bodyStr(t, uploadResp)

	urlStart := strings.Index(uploadBody, `"/uploads/`)
	urlEnd := strings.Index(uploadBody[urlStart+1:], `"`)
	imgURL := uploadBody[urlStart+1 : urlStart+1+urlEnd]
	// Extract filename
	parts := strings.Split(imgURL, "/")
	filename := parts[len(parts)-1]

	// Delete note
	delReq, _ := http.NewRequest("POST", app.url("/notes/"+slug), nil)
	delReq.Header.Set("X-HTTP-Method-Override", "DELETE")
	delResp, _ := app.client.Do(delReq)
	delResp.Body.Close()

	// Image file should be gone
	imgPath := filepath.Join(app.uploadsDir, fmt.Sprintf("%d", u.ID), filename)
	if _, err := os.Stat(imgPath); !os.IsNotExist(err) {
		t.Errorf("expected image file to be deleted after note deletion")
	}
}

// ── T030: Tag tests ───────────────────────────────────────────────────────────

func TestPutNote_SyncsTags(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Tagged"}, "body": {""}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	req, _ := http.NewRequest("POST", app.url("/notes/"+slug), strings.NewReader(url.Values{
		"title": {"Tagged"},
		"body":  {"This has #work and #ideas tags"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-HTTP-Method-Override", "PUT")
	resp, _ := app.client.Do(req)
	resp.Body.Close()

	note, _ := models.GetNote(app.db, u.ID, slug)
	tagSet := make(map[string]bool)
	for _, tg := range note.Tags {
		tagSet[tg] = true
	}
	if !tagSet["work"] || !tagSet["ideas"] {
		t.Errorf("expected tags work and ideas, got: %v", note.Tags)
	}
}

func TestGetTags_ReturnsOnlyUserTags(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"A"}, "body": {"#alicetag"}})
	app.postForm(t, "/logout", nil)

	jar2 := newCookieJar()
	client2 := &http.Client{Jar: jar2, CheckRedirect: func(req *http.Request, via []*http.Request) error { return nil }}
	client2.PostForm(app.url("/login"), url.Values{"username": {"bob"}})       //nolint:errcheck
	client2.PostForm(app.url("/notes"), url.Values{"title": {"B"}, "body": {"#bobtag"}}) //nolint:errcheck

	// Alice checks tags
	app.login(t, "alice")
	req, _ := http.NewRequest("GET", app.url("/tags"), nil)
	req.Header.Set("Accept", "application/json")
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body := bodyStr(t, resp)
	if strings.Contains(body, "bobtag") {
		t.Errorf("alice should not see bob's tags; got: %s", body)
	}
}

func TestGetNotes_FilterByTag(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	// Note with tag
	app.postForm(t, "/notes", url.Values{"title": {"Work Note"}, "body": {"#work"}})
	// Note without tag
	app.postForm(t, "/notes", url.Values{"title": {"Personal"}, "body": {"no tags here"}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	allNotes, _ := models.ListNotes(app.db, u.ID, "", false)
	for _, n := range allNotes {
		tags := models.ParseTags(n.Title)
		_ = tags
	}

	tagged, _ := models.ListNotes(app.db, u.ID, "work", false)
	if len(tagged) == 0 {
		t.Error("expected at least 1 note with #work tag")
	}
	for _, n := range tagged {
		found := false
		for _, tg := range n.Tags {
			if tg == "work" {
				found = true
			}
		}
		if !found {
			t.Errorf("note %q missing #work tag", n.Title)
		}
	}
}

func TestTagsCaseInsensitive(t *testing.T) {
	tags1 := models.ParseTags("#Work #WORK #work")
	if len(tags1) != 1 || tags1[0] != "work" {
		t.Errorf("expected single 'work' tag, got: %v", tags1)
	}
}

// ── Tag color tests ──────────────────────────────────────────────────────────

func TestTagColorPUT_SetsColor(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	req, _ := http.NewRequest("PUT", app.url("/tags/work/color"), strings.NewReader(`{"color":"#ee9b00"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatalf("PUT color: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Verify color was set
	u, _ := models.GetUserByUsername(app.db, "alice")
	color := models.GetTagColor(app.db, u.ID, "work")
	if color != "#ee9b00" {
		t.Errorf("expected #ee9b00, got %s", color)
	}
}

func TestTagColorPUT_InvalidColor(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	req, _ := http.NewRequest("PUT", app.url("/tags/work/color"), strings.NewReader(`{"color":"#invalid"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatalf("PUT color: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid color, got %d", resp.StatusCode)
	}
}

func TestTagsListGET_IncludesColor(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Test"}, "body": {"#work content"}})

	req, _ := http.NewRequest("GET", app.url("/tags"), nil)
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatalf("GET tags: %v", err)
	}
	defer resp.Body.Close()

	var tags []models.TagCount
	json.NewDecoder(resp.Body).Decode(&tags) //nolint:errcheck

	if len(tags) == 0 {
		t.Fatal("expected at least one tag")
	}
	if tags[0].Color == "" {
		t.Error("expected tag to have a color")
	}
}

// ── Search ────────────────────────────────────────────────────────────────────

func TestNotesSearchGET_EmptyQuery_ReturnsAllNotes(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	// Create some notes
	app.postForm(t, "/notes", url.Values{"title": {"First Note"}, "body": {"Content of first note #tag1"}})
	app.postForm(t, "/notes", url.Values{"title": {"Second Note"}, "body": {"Content of second note #tag2"}})

	resp := app.get(t, "/notes/search?q=")
	body := bodyStr(t, resp)

	// Should contain both notes
	if !strings.Contains(body, "First Note") || !strings.Contains(body, "Second Note") {
		t.Errorf("expected both notes in empty search results, got: %s", body[:min(500, len(body))])
	}
}

func TestNotesSearchGET_MatchesTitle(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "bob")

	app.postForm(t, "/notes", url.Values{"title": {"Meeting Notes"}, "body": {"Discussed project timeline"}})
	app.postForm(t, "/notes", url.Values{"title": {"Shopping List"}, "body": {"Milk, eggs, bread"}})

	resp := app.get(t, "/notes/search?q=meeting")
	body := bodyStr(t, resp)

	// Title may be highlighted with <mark> tags
	if !strings.Contains(body, "meeting-notes") {
		t.Errorf("expected meeting-notes slug in results, got: %s", body[:min(500, len(body))])
	}
	if strings.Contains(body, "shopping-list") {
		t.Error("did not expect shopping-list slug in results")
	}
}

func TestNotesSearchGET_MatchesTags(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "carol")

	app.postForm(t, "/notes", url.Values{"title": {"Work Stuff"}, "body": {"Some content #work #important"}})
	app.postForm(t, "/notes", url.Values{"title": {"Personal"}, "body": {"Other content #personal"}})

	resp := app.get(t, "/notes/search?q=work")
	body := bodyStr(t, resp)

	// Should find note with #work tag
	if !strings.Contains(body, "work-stuff") {
		t.Errorf("expected work-stuff slug in results (matches tag), got: %s", body[:min(500, len(body))])
	}
}

func TestNotesSearchGET_MatchesBody(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "dave")

	app.postForm(t, "/notes", url.Values{"title": {"Recipe Ideas"}, "body": {"Make chocolate cake for birthday"}})
	app.postForm(t, "/notes", url.Values{"title": {"Todo"}, "body": {"Buy groceries"}})

	resp := app.get(t, "/notes/search?q=chocolate")
	body := bodyStr(t, resp)

	if !strings.Contains(body, "Recipe Ideas") {
		t.Error("expected Recipe Ideas in results (matches body)")
	}
}

func TestNotesSearchGET_FuzzyMatching(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "eve")

	app.postForm(t, "/notes", url.Values{"title": {"Meeting Agenda"}, "body": {"Quarterly review meeting"}})
	app.postForm(t, "/notes", url.Values{"title": {"Random"}, "body": {"Nothing special"}})

	// Fuzzy search finds "meet" in "Meeting" (prefix/substring match)
	resp := app.get(t, "/notes/search?q=meet")
	body := bodyStr(t, resp)

	if !strings.Contains(body, "meeting-agenda") {
		t.Errorf("expected fuzzy match to find meeting-agenda, got: %s", body[:min(500, len(body))])
	}
	// Random note should not match
	if strings.Contains(body, "random") {
		t.Error("did not expect random slug in results")
	}
}

func TestNotesSearchGET_NoMatches(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "frank")

	app.postForm(t, "/notes", url.Values{"title": {"Some Note"}, "body": {"Some content"}})

	resp := app.get(t, "/notes/search?q=xyz123nonexistent")
	body := bodyStr(t, resp)

	if !strings.Contains(body, "No notes found") {
		t.Error("expected 'No notes found' message")
	}
}

func TestNotesSearchGET_RequiresAuth(t *testing.T) {
	app := newTestApp(t)
	// Not logged in - use fresh client without cookies

	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return nil }}
	resp, err := client.Get(app.server.URL + "/notes/search?q=test")
	if err != nil {
		t.Fatalf("GET search: %v", err)
	}
	defer resp.Body.Close()

	// Should redirect to login
	if !strings.HasSuffix(resp.Request.URL.Path, "/login") {
		t.Errorf("expected redirect to /login, got %s", resp.Request.URL.Path)
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func multipartFile(t *testing.T, field, filename string, data []byte) (io.Reader, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile(field, filename)
	fw.Write(data) //nolint:errcheck
	w.Close()
	return &buf, w.FormDataContentType()
}

// minimalPNG returns a valid 1x1 transparent PNG.
func minimalPNG() []byte {
	// Pre-encoded minimal valid PNG bytes
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, // signature
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41, // IDAT chunk
		0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x00, 0x02, 0x00, 0x01, 0xe2, 0x21, 0xbc,
		0x33, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, // IEND chunk
		0x44, 0xae, 0x42, 0x60, 0x82,
	}
}

// ── Archive ──────────────────────────────────────────────────────────────────

func TestNotesArchivePUT_ArchivesNote(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	app.postForm(t, "/notes", url.Values{"title": {"To Archive"}, "body": {"Content"}})
	u, _ := models.GetUserByUsername(app.db, "alice")
	note, _ := models.GetNote(app.db, u.ID, "to-archive")
	if note.Archived {
		t.Fatal("note should not be archived initially")
	}

	resp := app.put(t, "/notes/to-archive/archive", nil)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	note, _ = models.GetNote(app.db, u.ID, "to-archive")
	if !note.Archived {
		t.Error("note should be archived after PUT")
	}

	// Check redirect header
	redirect := resp.Header.Get("HX-Redirect")
	if redirect != "/notes" {
		t.Errorf("expected HX-Redirect to /notes, got %q", redirect)
	}
}

func TestNotesArchivePUT_ReturnsNotFoundForMissingNote(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	resp := app.put(t, "/notes/nonexistent/archive", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestNotesArchivePUT_RequiresLogin(t *testing.T) {
	app := newTestApp(t)
	resp := app.put(t, "/notes/some-note/archive", nil)
	// Client follows redirects; should end up at login page
	if !strings.Contains(resp.Request.URL.Path, "/login") {
		t.Errorf("expected redirect to login, got path %s", resp.Request.URL.Path)
	}
}

func TestNotesRestorePUT_RestoresNote(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	app.postForm(t, "/notes", url.Values{"title": {"Archived Note"}, "body": {"Content"}})
	u, _ := models.GetUserByUsername(app.db, "alice")
	models.ArchiveNote(app.db, u.ID, "archived-note") //nolint:errcheck

	resp := app.put(t, "/notes/archived-note/restore", nil)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	note, _ := models.GetNote(app.db, u.ID, "archived-note")
	if note.Archived {
		t.Error("note should not be archived after restore")
	}

	// Check redirect header
	redirect := resp.Header.Get("HX-Redirect")
	if redirect != "/archive" {
		t.Errorf("expected HX-Redirect to /archive, got %q", redirect)
	}
}

func TestNotesRestorePUT_ReturnsNotFoundForMissingNote(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	resp := app.put(t, "/notes/nonexistent/restore", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestArchivedNotesExcludedFromMainList(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	app.postForm(t, "/notes", url.Values{"title": {"Active Note"}, "body": {"Content"}})
	app.postForm(t, "/notes", url.Values{"title": {"Archived Note"}, "body": {"Content"}})
	u, _ := models.GetUserByUsername(app.db, "alice")
	models.ArchiveNote(app.db, u.ID, "archived-note") //nolint:errcheck

	resp := app.get(t, "/notes")
	body := bodyStr(t, resp)

	if !strings.Contains(body, "Active Note") {
		t.Error("expected active note in list")
	}
	if strings.Contains(body, "Archived Note") {
		t.Error("archived note should not appear in main list")
	}
}

func TestArchivedNotesExcludedFromSearch(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	app.postForm(t, "/notes", url.Values{"title": {"Searchable Active"}, "body": {"findme content"}})
	app.postForm(t, "/notes", url.Values{"title": {"Searchable Archived"}, "body": {"findme content"}})
	u, _ := models.GetUserByUsername(app.db, "alice")
	models.ArchiveNote(app.db, u.ID, "searchable-archived") //nolint:errcheck

	resp := app.get(t, "/notes/search?q=findme")
	body := bodyStr(t, resp)

	if !strings.Contains(body, "searchable-active") {
		t.Error("expected active note slug in search results")
	}
	if strings.Contains(body, "searchable-archived") {
		t.Error("archived note should not appear in search results")
	}
}

// ── Archive List & Search ────────────────────────────────────────────────────

func TestArchiveListGET_ShowsArchivedNotes(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	app.postForm(t, "/notes", url.Values{"title": {"Active Note"}, "body": {"Content"}})
	app.postForm(t, "/notes", url.Values{"title": {"Archived One"}, "body": {"Content #work"}})
	app.postForm(t, "/notes", url.Values{"title": {"Archived Two"}, "body": {"Content #personal"}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	models.ArchiveNote(app.db, u.ID, "archived-one") //nolint:errcheck
	models.ArchiveNote(app.db, u.ID, "archived-two") //nolint:errcheck

	resp := app.get(t, "/archive")
	body := bodyStr(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(body, "Archived One") {
		t.Error("expected Archived One in archive list")
	}
	if !strings.Contains(body, "Archived Two") {
		t.Error("expected Archived Two in archive list")
	}
	if strings.Contains(body, "Active Note") {
		t.Error("active note should not appear in archive list")
	}
}

func TestArchiveListGET_FilterByTag(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	app.postForm(t, "/notes", url.Values{"title": {"Work Note"}, "body": {"#work content"}})
	app.postForm(t, "/notes", url.Values{"title": {"Personal Note"}, "body": {"#personal content"}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	models.ArchiveNote(app.db, u.ID, "work-note")     //nolint:errcheck
	models.ArchiveNote(app.db, u.ID, "personal-note") //nolint:errcheck

	resp := app.get(t, "/archive?tag=work")
	body := bodyStr(t, resp)

	if !strings.Contains(body, "Work Note") {
		t.Error("expected Work Note with tag filter")
	}
	if strings.Contains(body, "Personal Note") {
		t.Error("Personal Note should be filtered out")
	}
}

func TestArchiveListGET_EmptyArchive(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	resp := app.get(t, "/archive")
	body := bodyStr(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(body, "No archived notes") {
		t.Error("expected 'No archived notes' message")
	}
}

func TestArchiveSearchGET_SearchesArchivedNotes(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	app.postForm(t, "/notes", url.Values{"title": {"Archive Findable"}, "body": {"special content here"}})
	app.postForm(t, "/notes", url.Values{"title": {"Active Findable"}, "body": {"special content here"}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	models.ArchiveNote(app.db, u.ID, "archive-findable") //nolint:errcheck

	resp := app.get(t, "/archive/search?q=special")
	body := bodyStr(t, resp)

	if !strings.Contains(body, "archive-findable") {
		t.Error("expected archived note slug in archive search results")
	}
	if strings.Contains(body, "active-findable") {
		t.Error("active note should not appear in archive search")
	}
}

func TestArchiveSearchGET_EmptyQuery(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	app.postForm(t, "/notes", url.Values{"title": {"Archived Stuff"}, "body": {"Content"}})
	u, _ := models.GetUserByUsername(app.db, "alice")
	models.ArchiveNote(app.db, u.ID, "archived-stuff") //nolint:errcheck

	resp := app.get(t, "/archive/search?q=")
	body := bodyStr(t, resp)

	if !strings.Contains(body, "Archived Stuff") {
		t.Error("expected archived note in empty query results")
	}
}

func TestArchiveListGET_RequiresLogin(t *testing.T) {
	app := newTestApp(t)
	resp := app.get(t, "/archive")
	if !strings.Contains(resp.Request.URL.Path, "/login") {
		t.Errorf("expected redirect to login, got path %s", resp.Request.URL.Path)
	}
}

func TestArchiveTagsGET_ReturnsArchivedTags(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	app.postForm(t, "/notes", url.Values{"title": {"Active Work"}, "body": {"#work content"}})
	app.postForm(t, "/notes", url.Values{"title": {"Archived Personal"}, "body": {"#personal content"}})
	u, _ := models.GetUserByUsername(app.db, "alice")
	models.ArchiveNote(app.db, u.ID, "archived-personal") //nolint:errcheck

	resp := app.get(t, "/archive/tags")
	body := bodyStr(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(body, "personal") {
		t.Error("expected personal tag from archived notes")
	}
	if strings.Contains(body, "work") {
		t.Error("work tag should not appear (only on active notes)")
	}
}

func TestArchiveSearchGET_NoMatchesMessage(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	resp := app.get(t, "/archive/search?q=nonexistent")
	body := bodyStr(t, resp)

	if !strings.Contains(body, "No archived notes found") {
		t.Error("expected 'No archived notes found' for non-matching query")
	}
}

func TestArchivedNoteAccessibleByDirectURL(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	app.postForm(t, "/notes", url.Values{"title": {"Direct Access"}, "body": {"Secret content"}})
	u, _ := models.GetUserByUsername(app.db, "alice")
	models.ArchiveNote(app.db, u.ID, "direct-access") //nolint:errcheck

	// Reader view should still work
	resp := app.get(t, "/notes/direct-access")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("reader: expected 200, got %d", resp.StatusCode)
	}
	body := bodyStr(t, resp)
	if !strings.Contains(body, "Direct Access") {
		t.Error("expected archived note to be readable via direct URL")
	}

	// Editor view should still work
	resp = app.get(t, "/notes/direct-access/edit")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("editor: expected 200, got %d", resp.StatusCode)
	}
}

func TestRestoreFromArchive_NoteReturnsToActiveList(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	app.postForm(t, "/notes", url.Values{"title": {"Restore Me"}, "body": {"Content"}})
	u, _ := models.GetUserByUsername(app.db, "alice")
	models.ArchiveNote(app.db, u.ID, "restore-me") //nolint:errcheck

	// Verify it's in archive
	resp := app.get(t, "/archive")
	body := bodyStr(t, resp)
	if !strings.Contains(body, "Restore Me") {
		t.Error("expected note in archive before restore")
	}

	// Restore it
	app.put(t, "/notes/restore-me/restore", nil)

	// Verify it's back in active list
	resp = app.get(t, "/notes")
	body = bodyStr(t, resp)
	if !strings.Contains(body, "Restore Me") {
		t.Error("expected restored note in active list")
	}

	// Verify it's gone from archive
	resp = app.get(t, "/archive")
	body = bodyStr(t, resp)
	if strings.Contains(body, "Restore Me") {
		t.Error("restored note should not appear in archive")
	}
}

// ── Note Links & Backlinks ────────────────────────────────────────────────────

func TestNoteReader_ShowsBacklinks(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Target Note"}, "body": {"Some content"}})
	app.postForm(t, "/notes", url.Values{"title": {"Source Note"}, "body": {"Links to [[Target Note]]"}})

	// Save source note with link via update to trigger SyncLinks
	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	for _, n := range notes {
		if n.Title == "Source Note" {
			req, _ := http.NewRequest("POST", app.url("/notes/"+n.Slug), strings.NewReader(url.Values{
				"title": {"Source Note"},
				"body":  {"Links to [[Target Note]]"},
			}.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("X-HTTP-Method-Override", "PUT")
			resp, _ := app.client.Do(req)
			resp.Body.Close()
		}
	}

	// View the target note - should show backlink
	resp := app.get(t, "/notes/target-note")
	body := bodyStr(t, resp)
	if !strings.Contains(body, "Linked from") {
		t.Error("expected 'Linked from' section in reader")
	}
	if !strings.Contains(body, "Source Note") {
		t.Error("expected 'Source Note' in backlinks")
	}
}

func TestNoteReader_NoBacklinksSection(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Standalone"}, "body": {"No links here"}})

	resp := app.get(t, "/notes/standalone")
	body := bodyStr(t, resp)
	if strings.Contains(body, "Linked from") {
		t.Error("should not show 'Linked from' when no backlinks exist")
	}
}

func TestNoteReader_WikiLinksRendered(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Page A"}, "body": {"Hello"}})
	app.postForm(t, "/notes", url.Values{"title": {"Page B"}, "body": {"See [[Page A]]"}})

	resp := app.get(t, "/notes/page-b")
	body := bodyStr(t, resp)
	if !strings.Contains(body, "/notes/page-a") {
		t.Error("expected rendered link to /notes/page-a in reader")
	}
}

func TestNotesAutocompleteGET_ReturnsMatches(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Meeting Notes"}, "body": {""}})
	app.postForm(t, "/notes", url.Values{"title": {"Shopping List"}, "body": {""}})

	req, _ := http.NewRequest("GET", app.url("/notes/autocomplete?q=meet"), nil)
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body := bodyStr(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(body, "Meeting Notes") {
		t.Errorf("expected Meeting Notes in autocomplete results: %s", body)
	}
	if strings.Contains(body, "Shopping List") {
		t.Error("unexpected Shopping List in autocomplete results for 'meet'")
	}
}

func TestNotesAutocompleteGET_EmptyQuery(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Note One"}, "body": {""}})

	req, _ := http.NewRequest("GET", app.url("/notes/autocomplete?q="), nil)
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body := bodyStr(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(body, "Note One") {
		t.Errorf("expected all notes for empty query: %s", body)
	}
}

func TestNotesAutocompleteGET_RequiresAuth(t *testing.T) {
	app := newTestApp(t)
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return nil }}
	resp, err := client.Get(app.server.URL + "/notes/autocomplete?q=test")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if !strings.HasSuffix(resp.Request.URL.Path, "/login") {
		t.Errorf("expected redirect to /login, got %s", resp.Request.URL.Path)
	}
}

// ── Todo tests ──────────────────────────────────────────────────────────────

func TestTodoTogglePUT(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	// Create a note with todos
	app.postForm(t, "/notes", url.Values{"title": {"Todo Test"}})

	body := "# Todo Test\n\n- [ ] Buy groceries @due(2026-04-20)\n- [ ] Send email\n- [x] Done task\n"
	app.postForm(t, "/notes/todo-test", url.Values{
		"title": {"Todo Test"},
		"body":  {body},
	})

	// Toggle first todo to checked
	req, _ := http.NewRequest("PUT", app.url("/notes/todo-test/todo"),
		strings.NewReader(`{"line":3,"checked":true}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, b)
	}

	// Verify the markdown was updated
	resp2, err2 := app.client.Get(app.url("/notes/todo-test"))
	if err2 != nil {
		t.Fatal(err2)
	}
	defer resp2.Body.Close()
	b, _ := io.ReadAll(resp2.Body)
	html := string(b)
	if !strings.Contains(html, "todo-checked") {
		t.Error("expected todo-checked class in rendered HTML after toggle")
	}
}

func TestTodoTogglePUT_InvalidLine(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	app.postForm(t, "/notes", url.Values{"title": {"Todo Test2"}})
	app.postForm(t, "/notes/todo-test2", url.Values{
		"title": {"Todo Test2"},
		"body":  {"Just text, no todos\n"},
	})

	req, _ := http.NewRequest("PUT", app.url("/notes/todo-test2/todo"),
		strings.NewReader(`{"line":1,"checked":true}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Errorf("expected 400 for non-todo line, got %d", resp.StatusCode)
	}
}

func TestTodosListGET(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	// Create notes with todos
	app.postForm(t, "/notes", url.Values{"title": {"Note A"}})
	app.postForm(t, "/notes/note-a", url.Values{
		"title": {"Note A"},
		"body":  {"- [ ] Task one @due(2026-04-20)\n- [x] Done task\n"},
	})

	app.postForm(t, "/notes", url.Values{"title": {"Note B"}})
	app.postForm(t, "/notes/note-b", url.Values{
		"title": {"Note B"},
		"body":  {"- [ ] Task two\n"},
	})

	resp, err := app.client.Get(app.url("/todos"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	b, _ := io.ReadAll(resp.Body)
	html := string(b)

	// Should contain both pending tasks
	if !strings.Contains(html, "Task one") {
		t.Error("expected 'Task one' in todos view")
	}
	if !strings.Contains(html, "Task two") {
		t.Error("expected 'Task two' in todos view")
	}
	// Should NOT contain completed task
	if strings.Contains(html, "Done task") {
		t.Error("should not contain completed task in todos view")
	}
}

// ── Public note sharing tests ──────────────────────────────────────────────

func publishNote(t *testing.T, app *testApp, slug string) string {
	t.Helper()
	req, _ := http.NewRequest("PUT", app.url("/notes/"+slug+"/publish"), nil)
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("publish failed: %d %s", resp.StatusCode, b)
	}
	var out struct {
		Token     string `json:"token"`
		PublicURL string `json:"public_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Token == "" {
		t.Fatal("expected token in publish response")
	}
	return out.Token
}

func unauthClient(t *testing.T) *http.Client {
	t.Helper()
	jar := newCookieJar()
	return &http.Client{Jar: jar}
}

func TestPublishPUT_GeneratesToken(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"My Public"}})
	token := publishNote(t, app, "my-public")
	if len(token) < 20 {
		t.Errorf("token too short: %q", token)
	}
}

func TestPublishPUT_ReusesTokenOnRepublish(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"My Public"}})

	token1 := publishNote(t, app, "my-public")

	// Unpublish
	req, _ := http.NewRequest("PUT", app.url("/notes/my-public/unpublish"), nil)
	resp, _ := app.client.Do(req)
	resp.Body.Close()

	// Republish — should return same token
	token2 := publishNote(t, app, "my-public")
	if token1 != token2 {
		t.Errorf("token changed across toggle: %q → %q", token1, token2)
	}
}

func TestPublishPUT_NonOwnerReturns404(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Alice Note"}})

	// Switch to bob (new session)
	auth.SessionManager = newSessionManager()
	app2 := newTestApp(t)
	app2.login(t, "bob")

	req, _ := http.NewRequest("PUT", app2.url("/notes/alice-note/publish"), nil)
	resp, err := app2.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Errorf("expected 404 for non-owner, got %d", resp.StatusCode)
	}
}

func TestPublicNoteGET_Accessible(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Public Article"}})
	app.postForm(t, "/notes/public-article", url.Values{
		"title": {"Public Article"},
		"body":  {"# Public Article\n\nThis is public content."},
	})
	token := publishNote(t, app, "public-article")

	// Visit with unauthenticated client
	ua := unauthClient(t)
	resp, err := ua.Get(app.url("/p/" + token))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	b, _ := io.ReadAll(resp.Body)
	html := string(b)
	if !strings.Contains(html, "Public Article") {
		t.Error("expected title in response")
	}
	if !strings.Contains(html, "This is public content") {
		t.Error("expected body in response")
	}
}

func TestPublicNoteGET_UnpublishedReturns404(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Private"}})
	token := publishNote(t, app, "private")

	// Unpublish
	req, _ := http.NewRequest("PUT", app.url("/notes/private/unpublish"), nil)
	resp, _ := app.client.Do(req)
	resp.Body.Close()

	ua := unauthClient(t)
	resp2, err := ua.Get(app.url("/p/" + token))
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != 404 {
		t.Errorf("expected 404 for unpublished, got %d", resp2.StatusCode)
	}
}

func TestPublicNoteGET_UnknownTokenReturns404(t *testing.T) {
	app := newTestApp(t)
	ua := unauthClient(t)
	resp, err := ua.Get(app.url("/p/nonexistent"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
	b, _ := io.ReadAll(resp.Body)
	// The 404 page should NOT reveal any owner data
	html := strings.ToLower(string(b))
	if strings.Contains(html, "sidebar") || strings.Contains(html, "sign out") || strings.Contains(html, "tags") {
		t.Errorf("404 page should not leak owner UI: %s", html)
	}
}

func TestPublicNoteGET_ArchivedReturns404(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"ToArchive"}})
	token := publishNote(t, app, "toarchive")

	// Archive it
	req, _ := http.NewRequest("PUT", app.url("/notes/toarchive/archive"), nil)
	resp, _ := app.client.Do(req)
	resp.Body.Close()

	ua := unauthClient(t)
	resp2, err := ua.Get(app.url("/p/" + token))
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != 404 {
		t.Errorf("expected 404 after archive, got %d", resp2.StatusCode)
	}
}

func TestPublicNoteGET_WikiLinkToPrivateIsPlainText(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	// Create a private target
	app.postForm(t, "/notes", url.Values{"title": {"Secret Page"}})

	// Create a public note linking to it
	app.postForm(t, "/notes", url.Values{"title": {"Public With Link"}})
	app.postForm(t, "/notes/public-with-link", url.Values{
		"title": {"Public With Link"},
		"body":  {"See [[Secret Page]] for details."},
	})
	token := publishNote(t, app, "public-with-link")

	ua := unauthClient(t)
	resp, err := ua.Get(app.url("/p/" + token))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	html := string(b)

	// Should contain "Secret Page" as plain text, NOT as a hyperlink
	if !strings.Contains(html, "Secret Page") {
		t.Error("expected 'Secret Page' in output as plain text")
	}
	// Must NOT link to /notes/secret-page or /p/<other-token>
	if strings.Contains(html, "/notes/secret-page") || strings.Contains(html, `href="/p/`) {
		t.Errorf("wiki-link to private note must not be a hyperlink: %s", html)
	}
}

func TestPublicNotesListGET(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	app.postForm(t, "/notes", url.Values{"title": {"Shared One"}})
	app.postForm(t, "/notes", url.Values{"title": {"Shared Two"}})
	app.postForm(t, "/notes", url.Values{"title": {"Private Three"}})

	_ = publishNote(t, app, "shared-one")
	_ = publishNote(t, app, "shared-two")
	// shared-three stays private

	resp, err := app.client.Get(app.url("/public"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	b, _ := io.ReadAll(resp.Body)
	html := string(b)

	if !strings.Contains(html, "Shared One") {
		t.Error("expected 'Shared One' in public list")
	}
	if !strings.Contains(html, "Shared Two") {
		t.Error("expected 'Shared Two' in public list")
	}
	if strings.Contains(html, "Private Three") {
		t.Error("private notes should not appear in public list")
	}
}

// ── Note-sharing tests (016-note-sharing) ──────────────────────────────────

func shareNote(t *testing.T, app *testApp, slug, username, permission string) *http.Response {
	t.Helper()
	body := `{"username":"` + username + `","permission":"` + permission + `"}`
	req, _ := http.NewRequest("PUT", app.url("/notes/"+slug+"/share"), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func revokeShare(t *testing.T, app *testApp, slug, username string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest("DELETE", app.url("/notes/"+slug+"/share/"+username), nil)
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func loginAs(t *testing.T, app *testApp, username string) {
	t.Helper()
	resp := app.login(t, username)
	resp.Body.Close()
}

// secondClient returns a new http.Client with its own cookie jar, sharing the
// same test server and session manager as the original app. Used for
// cross-user scenarios where two users need concurrent sessions.
func secondClient(t *testing.T, app *testApp, username string) *http.Client {
	t.Helper()
	jar := newCookieJar()
	c := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil
		},
	}
	resp, err := c.PostForm(app.url("/login"), url.Values{"username": {username}})
	if err != nil {
		t.Fatalf("secondClient login %s: %v", username, err)
	}
	resp.Body.Close()
	return c
}

func TestShareCreatePUT_Valid(t *testing.T) {
	app := newTestApp(t)
	// Pre-create bob so the username resolves
	_, _ = models.GetOrCreateUser(app.db, "bob")

	loginAs(t, app, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Team Doc"}})

	resp := shareNote(t, app, "team-doc", "bob", "edit")
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, b)
	}
}

func TestShareCreatePUT_UnknownUsername(t *testing.T) {
	app := newTestApp(t)
	loginAs(t, app, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Doc"}})

	resp := shareNote(t, app, "doc", "ghost", "read")
	defer resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Errorf("expected 404 for unknown user, got %d", resp.StatusCode)
	}
}

func TestShareCreatePUT_SelfShareRejected(t *testing.T) {
	app := newTestApp(t)
	loginAs(t, app, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Doc"}})

	resp := shareNote(t, app, "doc", "alice", "read")
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Errorf("expected 400 for self-share, got %d", resp.StatusCode)
	}
}

func TestShareCreatePUT_InvalidPermission(t *testing.T) {
	app := newTestApp(t)
	_, _ = models.GetOrCreateUser(app.db, "bob")
	loginAs(t, app, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Doc"}})

	resp := shareNote(t, app, "doc", "bob", "admin")
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Errorf("expected 400 for invalid permission, got %d", resp.StatusCode)
	}
}

func TestShareCreatePUT_UpsertOnReshare(t *testing.T) {
	app := newTestApp(t)
	_, _ = models.GetOrCreateUser(app.db, "bob")
	loginAs(t, app, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Doc"}})

	resp1 := shareNote(t, app, "doc", "bob", "read")
	resp1.Body.Close()
	resp2 := shareNote(t, app, "doc", "bob", "edit")
	resp2.Body.Close()

	// Verify only one row via the list endpoint
	resp, err := app.client.Get(app.url("/notes/doc/shares"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	// JSON should contain bob exactly once with "edit"
	if strings.Count(string(b), `"bob"`) != 1 {
		t.Errorf("expected bob to appear once, got: %s", b)
	}
	if !strings.Contains(string(b), `"edit"`) {
		t.Errorf("expected permission edit in output, got: %s", b)
	}
}

func TestShareDeletePUT_Revokes(t *testing.T) {
	app := newTestApp(t)
	_, _ = models.GetOrCreateUser(app.db, "bob")
	loginAs(t, app, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Doc"}})
	shareNote(t, app, "doc", "bob", "read").Body.Close()

	resp := revokeShare(t, app, "doc", "bob")
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Verify collaborator list is empty
	r2, err := app.client.Get(app.url("/notes/doc/shares"))
	if err != nil {
		t.Fatal(err)
	}
	defer r2.Body.Close()
	b, _ := io.ReadAll(r2.Body)
	if strings.Contains(string(b), `"bob"`) {
		t.Errorf("bob should be revoked, but appeared in list: %s", b)
	}
}

func TestShareCreatePUT_NonOwnerReturns404(t *testing.T) {
	app := newTestApp(t)
	_, _ = models.GetOrCreateUser(app.db, "carol")
	loginAs(t, app, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Alice Note"}})

	// Bob (different cookie jar) tries to share Alice's note
	bob := secondClient(t, app, "bob")
	body := `{"username":"carol","permission":"edit"}`
	req, _ := http.NewRequest("PUT", app.url("/notes/alice-note/share"), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := bob.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Errorf("expected 404 (non-owner cannot share), got %d", resp.StatusCode)
	}
}

func TestSharedNotesListGET(t *testing.T) {
	app := newTestApp(t)
	_, _ = models.GetOrCreateUser(app.db, "bob")

	loginAs(t, app, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Shared Alpha"}})
	app.postForm(t, "/notes", url.Values{"title": {"Private Beta"}})
	shareNote(t, app, "shared-alpha", "bob", "read").Body.Close()

	bob := secondClient(t, app, "bob")
	resp, err := bob.Get(app.url("/shared"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	html := string(b)
	if !strings.Contains(html, "Shared Alpha") {
		t.Errorf("expected 'Shared Alpha' in bob's shared list, got: %s", html)
	}
	if strings.Contains(html, "Private Beta") {
		t.Error("'Private Beta' must not appear — not shared with bob")
	}
}

func TestSharedNoteReaderGET_GrantedAllowed(t *testing.T) {
	app := newTestApp(t)
	_, _ = models.GetOrCreateUser(app.db, "bob")
	loginAs(t, app, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Doc"}})
	app.postForm(t, "/notes/doc", url.Values{"title": {"Doc"}, "body": {"Hello world"}})
	shareNote(t, app, "doc", "bob", "read").Body.Close()

	bob := secondClient(t, app, "bob")
	resp, err := bob.Get(app.url("/shared/alice/doc"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 for granted share, got %d", resp.StatusCode)
	}
	b, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(b), "Hello world") {
		t.Errorf("expected body content in response: %s", b)
	}
}

func TestSharedNoteReaderGET_NotGrantedReturns404(t *testing.T) {
	app := newTestApp(t)
	_, _ = models.GetOrCreateUser(app.db, "bob")
	loginAs(t, app, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Private"}})

	bob := secondClient(t, app, "bob")
	resp, err := bob.Get(app.url("/shared/alice/private"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Errorf("expected 404 for ungranted shared access, got %d", resp.StatusCode)
	}
}

func TestSharedNoteEditorGET_ReadPermissionReturns403(t *testing.T) {
	app := newTestApp(t)
	_, _ = models.GetOrCreateUser(app.db, "bob")
	loginAs(t, app, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Doc"}})
	shareNote(t, app, "doc", "bob", "read").Body.Close()

	bob := secondClient(t, app, "bob")
	resp, err := bob.Get(app.url("/shared/alice/doc/edit"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 403 {
		t.Errorf("expected 403 for read-permission edit access, got %d", resp.StatusCode)
	}
}

func TestSharedNoteEditorGET_EditPermissionReturns200(t *testing.T) {
	app := newTestApp(t)
	_, _ = models.GetOrCreateUser(app.db, "bob")
	loginAs(t, app, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Doc"}})
	shareNote(t, app, "doc", "bob", "edit").Body.Close()

	bob := secondClient(t, app, "bob")
	resp, err := bob.Get(app.url("/shared/alice/doc/edit"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("expected 200 for edit-permission, got %d", resp.StatusCode)
	}
}

func TestSharedNoteUpdate_EditorWrites(t *testing.T) {
	app := newTestApp(t)
	_, _ = models.GetOrCreateUser(app.db, "bob")
	loginAs(t, app, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Doc"}})
	app.postForm(t, "/notes/doc", url.Values{"title": {"Doc"}, "body": {"original"}})
	shareNote(t, app, "doc", "bob", "edit").Body.Close()

	bob := secondClient(t, app, "bob")
	form := url.Values{"title": {"Doc"}, "body": {"edited by bob"}}
	resp, err := bob.PostForm(app.url("/shared/alice/doc"), form)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	// Alice reads — content should reflect bob's edit
	r, err := app.client.Get(app.url("/notes/doc"))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	b, _ := io.ReadAll(r.Body)
	if !strings.Contains(string(b), "edited by bob") {
		t.Errorf("expected bob's edit to persist, got: %s", b)
	}
}

func TestSharedNoteUpdate_ReadPermissionBlocked(t *testing.T) {
	app := newTestApp(t)
	_, _ = models.GetOrCreateUser(app.db, "bob")
	loginAs(t, app, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Doc"}})
	shareNote(t, app, "doc", "bob", "read").Body.Close()

	bob := secondClient(t, app, "bob")
	form := url.Values{"title": {"Hacked"}, "body": {"hack"}}
	resp, err := bob.PostForm(app.url("/shared/alice/doc"), form)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 403 {
		t.Errorf("expected 403 for read-only collaborator save, got %d", resp.StatusCode)
	}
}

func TestSharedNotesListGET_ArchivedExcluded(t *testing.T) {
	app := newTestApp(t)
	_, _ = models.GetOrCreateUser(app.db, "bob")
	loginAs(t, app, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Will Archive"}})
	shareNote(t, app, "will-archive", "bob", "read").Body.Close()

	// Alice archives the note
	req, _ := http.NewRequest("PUT", app.url("/notes/will-archive/archive"), nil)
	if r, err := app.client.Do(req); err == nil {
		r.Body.Close()
	}

	bob := secondClient(t, app, "bob")
	resp, err := bob.Get(app.url("/shared"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if strings.Contains(string(b), "Will Archive") {
		t.Error("archived note should not appear in shared list")
	}

	r2, err := bob.Get(app.url("/shared/alice/will-archive"))
	if err != nil {
		t.Fatal(err)
	}
	defer r2.Body.Close()
	if r2.StatusCode != 404 {
		t.Errorf("expected 404 for archived shared note, got %d", r2.StatusCode)
	}
}

func TestSharedNoteEdit_GitAttribution(t *testing.T) {
	app := newTestApp(t)
	_, _ = models.GetOrCreateUser(app.db, "bob")
	loginAs(t, app, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Collab Doc"}})
	app.postForm(t, "/notes/collab-doc", url.Values{"title": {"Collab Doc"}, "body": {"by alice"}})
	shareNote(t, app, "collab-doc", "bob", "edit").Body.Close()

	bob := secondClient(t, app, "bob")
	form := url.Values{"title": {"Collab Doc"}, "body": {"edited by bob"}}
	r, err := bob.PostForm(app.url("/shared/alice/collab-doc"), form)
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()

	out, err := exec.Command("git", "-C", app.notesDir, "log", "--format=%an").Output()
	if err != nil {
		t.Fatalf("git log: %v", err)
	}
	if !strings.Contains(string(out), "bob") {
		t.Errorf("expected 'bob' in git authors, got:\n%s", out)
	}
}
