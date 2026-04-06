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
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/selvakn/my-notes/internal/auth"
	"github.com/selvakn/my-notes/internal/handlers"
	"github.com/selvakn/my-notes/internal/models"
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

	// Fresh session manager per test to avoid cross-test contamination
	auth.SessionManager = newSessionManager()
	tmplDir := resolveOrStubTemplateDir(t)
	h := handlers.New(db, tmplDir, notesDir, uploadsDir)

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(auth.SessionManager.LoadAndSave)

	r.Get("/login", h.LoginGET)
	r.Post("/login", h.LoginPOST)
	r.Post("/logout", h.LogoutPOST)

	r.Group(func(r chi.Router) {
		r.Use(auth.RequireLogin)
		r.Get("/notes", h.NotesListGET)
		r.Get("/notes/search", h.NotesSearchGET)
		r.Post("/notes", h.NotesCreatePOST)
		r.Get("/notes/{slug}", h.NoteReaderGET)
		r.Get("/notes/{slug}/edit", h.NoteEditorGET)
		r.Post("/notes/{slug}", h.NoteUpdateOrDelete)
		r.Post("/notes/{slug}/images", h.ImageUploadPOST)
		r.Get("/notes/{slug}/drawing", h.DrawingGET)
		r.Put("/notes/{slug}/drawing", h.DrawingPUT)
		r.Delete("/notes/{slug}/drawing", h.DrawingDELETE)
		r.Get("/tags", h.TagsListGET)
		r.Put("/tags/{name}/color", h.TagColorPUT)
		r.Get("/uploads/{username}/{filename}", h.ImageServeGET)

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
	notes, err := models.ListNotes(app.db, u.ID, "")
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
	notes, _ := models.ListNotes(app.db, u.ID, "")
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
	notes, _ := models.ListNotes(app.db, u.ID, "")
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
	notes, _ := models.ListNotes(app.db, u.ID, "")
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
	notes, _ := models.ListNotes(app.db, u.ID, "")
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
	notes, _ := models.ListNotes(app.db, u.ID, "")
	slug := notes[0].Slug
	mdPath := filepath.Join(app.notesDir, fmt.Sprintf("%d", u.ID), slug+".md")

	req, _ := http.NewRequest("POST", app.url("/notes/"+slug), nil)
	req.Header.Set("X-HTTP-Method-Override", "DELETE")
	resp, _ := app.client.Do(req)
	resp.Body.Close()

	// DB row gone
	remaining, _ := models.ListNotes(app.db, u.ID, "")
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
	notes, _ := models.ListNotes(app.db, u.ID, "")
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
	notes, _ := models.ListNotes(app.db, u.ID, "")
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
	notes, _ := models.ListNotes(app.db, u.ID, "")
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
	notes, _ := models.ListNotes(app.db, u.ID, "")
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
	notes, _ := models.ListNotes(app.db, u.ID, "")
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
	notes, _ := models.ListNotes(app.db, u.ID, "")
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
	allNotes, _ := models.ListNotes(app.db, u.ID, "")
	for _, n := range allNotes {
		tags := models.ParseTags(n.Title)
		_ = tags
	}

	tagged, _ := models.ListNotes(app.db, u.ID, "work")
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
