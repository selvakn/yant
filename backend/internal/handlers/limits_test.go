package handlers_test

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/selvakn/yant/internal/models"
)

// ── Note size limit ───────────────────────────────────────────────────────────

func TestNoteSizeLimitCreate(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "sizelimituser")

	// Body just over 5 MB
	bigBody := strings.Repeat("a", 5*1024*1024+1)
	form := url.Values{"title": {"Big Note"}, "body": {bigBody}}
	resp, err := app.client.PostForm(app.url("/notes"), form)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413, got %d", resp.StatusCode)
	}

	// Verify the note was not stored
	count, err := models.CountNotesForUser(app.db, userIDFor(t, app, "sizelimituser"))
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 notes, got %d", count)
	}
}

func TestNoteSizeLimitUpdate(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "sizeupdateuser")
	userID := userIDFor(t, app, "sizeupdateuser")

	// Create a small note first
	slug := createNoteViaAPI(t, app, "Update Size Test", "hello")

	// Now try to update with content over 5 MB
	bigBody := strings.Repeat("b", 5*1024*1024+1)
	form := url.Values{"title": {"Update Size Test"}, "body": {bigBody}}
	req, _ := http.NewRequest(http.MethodPost, app.url("/notes/"+slug), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413, got %d", resp.StatusCode)
	}

	// size_bytes should still reflect original small content
	n, err := models.GetNote(app.db, userID, slug)
	if err != nil || n == nil {
		t.Fatalf("get note: %v", err)
	}
}

func TestNoteSizeWithinLimitSucceeds(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "sizeokuser")

	body := strings.Repeat("x", 100) // tiny body
	form := url.Values{"title": {"Small Note"}, "body": {body}}
	resp, err := app.client.PostForm(app.url("/notes"), form)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		t.Errorf("expected success, got %d", resp.StatusCode)
	}
}

// ── Note count limit ─────────────────────────────────────────────────────────

func TestNoteCountLimitRegularUser(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "countlimituser")
	userID := userIDFor(t, app, "countlimituser")

	// Pre-insert 25 notes directly in DB (faster than 25 HTTP calls)
	for i := 0; i < models.MaxNotesPerUser; i++ {
		slug := fmt.Sprintf("note-%d", i)
		if _, err := models.CreateNote(app.db, userID, fmt.Sprintf("Note %d", i), slug, 0, false); err != nil {
			t.Fatalf("pre-insert note %d: %v", i, err)
		}
	}

	// 26th note should be rejected
	form := url.Values{"title": {"One Too Many"}, "body": {""}}
	resp, err := app.client.PostForm(app.url("/notes"), form)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", resp.StatusCode)
	}

	count, err := models.CountNotesForUser(app.db, userID)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != models.MaxNotesPerUser {
		t.Errorf("expected exactly %d notes, got %d", models.MaxNotesPerUser, count)
	}
}

func TestNoteCountLimitAdminExempt(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "adminunlimited")
	userID := userIDFor(t, app, "adminunlimited")

	// Make user admin
	if _, err := app.db.Exec(`UPDATE users SET is_admin = 1 WHERE id = ?`, userID); err != nil {
		t.Fatalf("promote admin: %v", err)
	}

	// Pre-insert MaxNotesPerUser notes via DB (bypassing the limit by using isAdmin=true)
	for i := 0; i < models.MaxNotesPerUser; i++ {
		slug := fmt.Sprintf("admin-note-%d", i)
		if _, err := models.CreateNote(app.db, userID, fmt.Sprintf("Admin Note %d", i), slug, 0, true); err != nil {
			t.Fatalf("pre-insert: %v", err)
		}
	}

	// Admin should be able to create a 26th note
	form := url.Values{"title": {"Admin Extra"}, "body": {""}}
	resp, err := app.client.PostForm(app.url("/notes"), form)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		t.Errorf("admin should be exempt from note count limit, got %d", resp.StatusCode)
	}
}

func TestNoteCountLimitBelowLimitSucceeds(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "belowlimituser")

	form := url.Values{"title": {"First Note"}, "body": {""}}
	resp, err := app.client.PostForm(app.url("/notes"), form)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		t.Errorf("expected success for first note, got %d", resp.StatusCode)
	}
}

func TestNoteCountLimitAtomic(t *testing.T) {
	// Two simultaneous create requests at the limit must not both succeed.
	app := newTestApp(t)
	app.login(t, "atomicuser")
	userID := userIDFor(t, app, "atomicuser")

	// Pre-insert 24 notes (one below limit)
	for i := 0; i < models.MaxNotesPerUser-1; i++ {
		slug := fmt.Sprintf("atomic-note-%d", i)
		if _, err := models.CreateNote(app.db, userID, fmt.Sprintf("Note %d", i), slug, 0, false); err != nil {
			t.Fatalf("pre-insert: %v", err)
		}
	}

	// Fire two concurrent create requests
	var wg sync.WaitGroup
	statusCodes := make([]int, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			// Each goroutine needs its own client with its own cookie jar to share the session
			// Re-use the shared client (cookie jar is goroutine-safe for reads)
			form := url.Values{"title": {fmt.Sprintf("Race Note %d", idx)}, "body": {""}}
			resp, err := app.client.PostForm(app.url("/notes"), form)
			if err != nil {
				statusCodes[idx] = -1
				return
			}
			resp.Body.Close()
			statusCodes[idx] = resp.StatusCode
		}(i)
	}
	wg.Wait()

	finalCount, err := models.CountNotesForUser(app.db, userID)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	// Must not exceed the limit
	if finalCount > models.MaxNotesPerUser {
		t.Errorf("atomic enforcement failed: got %d notes (limit %d)", finalCount, models.MaxNotesPerUser)
	}
}

// ── Image upload limits ───────────────────────────────────────────────────────

func TestImageUploadSizeLimit(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "imagesizeuser")
	slug := createNoteViaAPI(t, app, "Image Size Test", "")

	// Build a multipart body with a fake PNG > 1 MB
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("image", "big.png")
	// Fake PNG header + padding > 1 MB
	pngHeader := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x02\x00\x00\x00\x90wS\xde")
	part.Write(pngHeader)                             //nolint:errcheck
	part.Write(bytes.Repeat([]byte{0}, 1024*1024+1)) //nolint:errcheck
	w.Close()

	req, _ := http.NewRequest(http.MethodPost, app.url("/notes/"+slug+"/images"), &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413, got %d", resp.StatusCode)
	}
}

func TestImageUploadCountLimit(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "imagecountuser")
	userID := userIDFor(t, app, "imagecountuser")
	slug := createNoteViaAPI(t, app, "Image Count Test", "")
	note, _ := models.GetNote(app.db, userID, slug)

	// Pre-insert 10 image records in DB
	for i := 0; i < 10; i++ {
		models.CreateImage(app.db, note.ID, fmt.Sprintf("img%d.png", i), "orig.png", "image/png", 100) //nolint:errcheck
	}

	// 11th upload attempt should be rejected
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, _ := mw.CreateFormFile("image", "small.png")
	// Tiny valid PNG (1x1 pixel)
	part.Write([]byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x02\x00\x00\x00\x90wS\xde\x00\x00\x00\x0cIDATx\x9cc\xf8\x0f\x00\x00\x01\x01\x00\x05\x18\xd8N\x00\x00\x00\x00IEND\xaeB`\x82")) //nolint:errcheck
	mw.Close()

	req, _ := http.NewRequest(http.MethodPost, app.url("/notes/"+slug+"/images"), &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", resp.StatusCode)
	}
}

// ── Admin storage display ─────────────────────────────────────────────────────

func TestAdminUsersStorageDisplay(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "storagedisplayadmin")
	adminID := userIDFor(t, app, "storagedisplayadmin")

	// Promote to admin
	if _, err := app.db.Exec(`UPDATE users SET is_admin = 1 WHERE id = ?`, adminID); err != nil {
		t.Fatalf("promote: %v", err)
	}

	// Create another user with a note of known size
	otherUser, _ := models.GetOrCreateUser(app.db, "storageotheruser")
	models.CreateNote(app.db, otherUser.ID, "Big Note", "big-note", 2048, true) //nolint:errcheck

	resp, err := app.client.Get(app.url("/admin/users"))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var body bytes.Buffer
	body.ReadFrom(resp.Body) //nolint:errcheck
	if !strings.Contains(body.String(), "Storage") {
		t.Error("admin users page should contain 'Storage' column header")
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func userIDFor(t *testing.T, app *testApp, username string) int64 {
	t.Helper()
	u, err := models.GetUserByUsername(app.db, username)
	if err != nil || u == nil {
		t.Fatalf("userIDFor %s: %v", username, err)
	}
	return u.ID
}

func createNoteViaAPI(t *testing.T, app *testApp, title, body string) string {
	t.Helper()
	form := url.Values{"title": {title}, "body": {body}}
	resp, err := app.client.PostForm(app.url("/notes"), form)
	if err != nil {
		t.Fatalf("create note: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		t.Fatalf("create note failed with %d", resp.StatusCode)
	}
	// Extract slug from redirect URL
	finalURL := resp.Request.URL.Path
	parts := strings.Split(finalURL, "/")
	if len(parts) < 3 {
		t.Fatalf("unexpected redirect URL: %s", finalURL)
	}
	// URL is /notes/{slug}/edit
	return parts[len(parts)-2]
}
