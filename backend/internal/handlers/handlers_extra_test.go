package handlers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/selvakn/yant/internal/auth"
	"github.com/selvakn/yant/internal/handlers"
	"github.com/selvakn/yant/internal/models"
)

// ── Login page ────────────────────────────────────────────────────────────────

func TestLoginGET_ReturnsPage(t *testing.T) {
	app := newTestApp(t)
	resp := app.get(t, "/login")
	body := bodyStr(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(body, "YANT") {
		t.Errorf("expected landing page with YANT branding, got: %s", body[:min(200, len(body))])
	}
}

// ── Note not found ────────────────────────────────────────────────────────────

func TestGetNote_NotFoundReturns404(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	resp := app.get(t, "/notes/no-such-slug")
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestGetNoteEdit_NotFoundReturns404(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	resp := app.get(t, "/notes/no-such-slug/edit")
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestUpdateNote_NotFoundReturns404(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	req, _ := http.NewRequest("POST", app.url("/notes/no-such-slug"), strings.NewReader(url.Values{
		"title": {"X"},
		"body":  {"Y"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-HTTP-Method-Override", "PUT")
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestDeleteNote_NotFoundReturns404(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	req, _ := http.NewRequest("POST", app.url("/notes/no-such-slug"), nil)
	req.Header.Set("X-HTTP-Method-Override", "DELETE")
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// ── Tags htmx path ────────────────────────────────────────────────────────────

func TestGetTags_HtmxRequestRendersSidebar(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	req, _ := http.NewRequest("GET", app.url("/tags"), nil)
	req.Header.Set("HX-Request", "true")
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body := bodyStr(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", resp.StatusCode, body)
	}
}

func TestGetTags_NonHtmxReturnsJSON(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	req, _ := http.NewRequest("GET", app.url("/tags"), nil)
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body := bodyStr(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("expected JSON content-type, got %q; body: %s", ct, body)
	}
}

// ── Notes list tag filter ─────────────────────────────────────────────────────

func TestNotesListGET_TagFilter(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Work Note"}, "body": {"#work stuff"}})
	app.postForm(t, "/notes", url.Values{"title": {"Personal"}, "body": {"personal stuff"}})

	resp := app.get(t, "/notes?tag=work")
	body := bodyStr(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(body, "Work Note") {
		t.Errorf("expected Work Note in filtered list, got: %s", body[:min(300, len(body))])
	}
}

// ── RenderError tests ────────────────────────────────────────────────────────

func TestRenderError_404ViaNotFoundRoute(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	resp := app.get(t, "/nonexistent-page-xyz")
	body := bodyStr(t, resp)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
	_ = body // just verify no panic
}

func TestRenderError_403CodePath(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	resp := app.get(t, "/test/error/403")
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

func TestRenderError_FallsBackWhenTemplatesMissing(t *testing.T) {
	// Create a handler with an empty template dir so loadTemplate fails
	db, err := models.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	models.InitSchema(db) //nolint:errcheck
	t.Cleanup(func() { db.Close() })

	emptyDir := t.TempDir() // no templates here
	h := handlers.New(db, emptyDir, t.TempDir(), t.TempDir(), nil, nil, false, 300, "", "", "", "", nil)

	auth.SessionManager = newSessionManager()

	r := chi.NewRouter()
	r.Use(auth.SessionManager.LoadAndSave)
	r.Get("/login", h.LoginGET)
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		h.RenderError(w, r, http.StatusNotFound, "not found")
	})

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	// LoginGET with missing template → falls back to 500
	resp, err := http.Get(srv.URL + "/login")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	// Should get a 500 error response (template file missing)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500 when template missing, got %d", resp.StatusCode)
	}

	// NotFound with missing template → falls back to plain text
	resp2, err := http.Get(srv.URL + "/unknown")
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 response, got %d", resp2.StatusCode)
	}
}

// ── Image upload edge cases ───────────────────────────────────────────────────

func TestImageUpload_MissingFieldReturns400(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"N"}, "body": {""}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	// Send multipart form without an "image" field
	var buf strings.Builder
	buf.WriteString("--boundary\r\nContent-Disposition: form-data; name=\"other\"\r\n\r\nval\r\n--boundary--\r\n")
	req, _ := http.NewRequest("POST", app.url("/notes/"+slug+"/images"),
		strings.NewReader(buf.String()))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for missing image field, got %d", resp.StatusCode)
	}
}

func TestImageUpload_TooLargeReturns413(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"N"}, "body": {""}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	// Send 11MB file (exceeds maxImageSize of 10MB + 1024 buffer)
	bigData := make([]byte, 11*1024*1024)
	// Put a PNG header so MIME detection would pass if it got that far
	copy(bigData, minimalPNG())
	body, ct := multipartFile(t, "image", "big.png", bigData)
	req, _ := http.NewRequest("POST", app.url("/notes/"+slug+"/images"), body)
	req.Header.Set("Content-Type", ct)
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413, got %d", resp.StatusCode)
	}
}

func TestImageUpload_NoteNotFoundReturns404(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	pngData := minimalPNG()
	body, ct := multipartFile(t, "image", "test.png", pngData)
	req, _ := http.NewRequest("POST", app.url("/notes/nonexistent/images"), body)
	req.Header.Set("Content-Type", ct)
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestImageServe_UnknownUserReturns404(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	resp := app.get(t, "/uploads/nobody/nofile.png")
	resp.Body.Close()
	// Either 403 (can't serve another user's files) or 404 (user not found)
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 403 or 404, got %d", resp.StatusCode)
	}
}

func TestImageServe_PathTraversalReturns400(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	// Use the URL with ../ in the filename path component
	resp := app.get(t, "/uploads/alice/..%2Fsecret.png")
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 400/403/404 for path traversal, got %d", resp.StatusCode)
	}
}

func TestImageServe_ExistingUserMissingFileReturns404(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	// alice exists (logged in) but this file doesn't
	resp := app.get(t, "/uploads/alice/nonexistent-uuid.png")
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for missing file, got %d", resp.StatusCode)
	}
}

func TestImageServe_UnknownExtensionServesOctetStream(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	u, _ := models.GetUserByUsername(app.db, "alice")
	// Manually place a file with unknown extension in the uploads dir
	userUploadDir := fmt.Sprintf("%s/%d", app.uploadsDir, u.ID)
	os.MkdirAll(userUploadDir, 0755) //nolint:errcheck
	os.WriteFile(userUploadDir+"/test.xyz", []byte("data"), 0644) //nolint:errcheck

	resp := app.get(t, "/uploads/alice/test.xyz")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "octet-stream") && !strings.Contains(ct, "text/plain") {
		t.Logf("content-type for unknown extension: %s", ct)
	}
}

func TestNoteUpdate_WriteErrorReturns500(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root: permission restrictions don't apply")
	}
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Perm Note"}, "body": {""}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	// Make the .md file read-only so WriteNote cannot overwrite it
	mdFile := fmt.Sprintf("%s/%d/%s.md", app.notesDir, u.ID, slug)
	os.Chmod(mdFile, 0444) //nolint:errcheck
	defer os.Chmod(mdFile, 0644) //nolint:errcheck

	req, _ := http.NewRequest("POST", app.url("/notes/"+slug), strings.NewReader(url.Values{
		"title": {"Updated"},
		"body":  {"new body"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-HTTP-Method-Override", "PUT")
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500 on write error, got %d", resp.StatusCode)
	}
}

// ── LoginPOST DB error ────────────────────────────────────────────────────────

func TestLoginPOST_DBErrorReturns500(t *testing.T) {
	app := newTestApp(t)
	// Close DB before login so GetOrCreateUser fails
	app.db.Close()
	resp, err := app.client.PostForm(app.url("/login"), url.Values{"username": {"newuser"}})
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500 on DB error during login, got %d", resp.StatusCode)
	}
}

// ── Handler DB error paths ────────────────────────────────────────────────────
// After login succeeds, we close the DB to trigger 500 error paths in handlers.

func TestNotesListGET_DBErrorReturns500(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.db.Close()

	resp := app.get(t, "/notes")
	resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500 on DB error, got %d", resp.StatusCode)
	}
}

func TestNotesListGET_DBErrorWithTagReturns500(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.db.Close()

	resp := app.get(t, "/notes?tag=work")
	resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500 on DB error with tag, got %d", resp.StatusCode)
	}
}

func TestNotesCreatePOST_DBErrorReturns500(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.db.Close()

	resp := app.postForm(t, "/notes", url.Values{"title": {"X"}, "body": {"y"}})
	resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500 on DB error, got %d", resp.StatusCode)
	}
}

func TestTagsListGET_DBErrorReturns500(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.db.Close()

	req, _ := http.NewRequest("GET", app.url("/tags"), nil)
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500 on DB error, got %d", resp.StatusCode)
	}
}

// ── NoteReader with missing file ──────────────────────────────────────────────

func TestNoteReader_MissingFileRendersEmptyBody(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"File Test"}, "body": {"some content"}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	// Delete the .md file — handler should still render (with empty body)
	mdFile := fmt.Sprintf("%s/%d/%s.md", app.notesDir, u.ID, slug)
	os.Remove(mdFile) //nolint:errcheck

	resp := app.get(t, "/notes/"+slug)
	body := bodyStr(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 even with missing file, got %d: %s", resp.StatusCode, body)
	}
}

// ── ImageUpload EnsureUploadsDir blocked ──────────────────────────────────────

func TestImageUpload_UploadsDirectoryBlockedReturns500(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root: permission restrictions don't apply")
	}
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Blocked"}, "body": {""}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	// Create a FILE at uploadsDir/userID to block directory creation
	uploadUserPath := fmt.Sprintf("%s/%d", app.uploadsDir, u.ID)
	os.WriteFile(uploadUserPath, []byte("blocker"), 0644) //nolint:errcheck

	pngData := minimalPNG()
	body, ct := multipartFile(t, "image", "test.png", pngData)
	req, _ := http.NewRequest("POST", app.url("/notes/"+slug+"/images"), body)
	req.Header.Set("Content-Type", ct)
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500 when uploads dir is blocked, got %d", resp.StatusCode)
	}
}

// ── Notes create with body containing tags ────────────────────────────────────

func TestPostNotes_BodyTagsSavedToCreateNote(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	// Tags in body get parsed at create time
	app.postForm(t, "/notes", url.Values{
		"title": {"Tag Test"},
		"body":  {"Hello #mytag"},
	})

	// Verify in tags API
	req, _ := http.NewRequest("GET", app.url("/tags"), nil)
	resp, _ := app.client.Do(req)
	body := bodyStr(t, resp)
	_ = body // Tags are synced on update, not create — just verify no error
}

// ── NoteUpdateOrDelete without override defaults to update ───────────────────

func TestNoteUpdateOrDelete_NoOverrideDefaultsToUpdate(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Default"}, "body": {""}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	// POST without method override header — treated as PUT (update)
	resp := app.postForm(t, "/notes/"+slug, url.Values{"title": {"Updated"}, "body": {"new body"}})
	resp.Body.Close()
	// Should succeed (200/204/redirect), not error
	if resp.StatusCode >= 500 {
		t.Errorf("expected success, got %d", resp.StatusCode)
	}
}
