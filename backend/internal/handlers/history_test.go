package handlers_test

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/selvakn/yant/internal/models"
	"github.com/selvakn/yant/internal/versioning"
)

func createNoteAndGetSlug(t *testing.T, app *testApp, title, body string) string {
	t.Helper()
	app.postForm(t, "/notes", url.Values{"title": {title}, "body": {body}})
	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	for _, n := range notes {
		if n.Title == title {
			return n.Slug
		}
	}
	t.Fatalf("note %q not found", title)
	return ""
}

func noteRelPath(t *testing.T, app *testApp, slug string) string {
	t.Helper()
	u, _ := models.GetUserByUsername(app.db, "alice")
	return fmt.Sprintf("%d/%s.md", u.ID, slug)
}

func updateNote(t *testing.T, app *testApp, slug, title, body string) {
	t.Helper()
	req, _ := http.NewRequest("POST", app.url("/notes/"+slug), strings.NewReader(url.Values{
		"title": {title},
		"body":  {body},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-HTTP-Method-Override", "PUT")
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatalf("update note: %v", err)
	}
	resp.Body.Close()
}

// ── History List ─────────────────────────────────────────────────────────────

func TestNoteHistoryGET_ShowsVersions(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	slug := createNoteAndGetSlug(t, app, "History Note", "version 1")
	updateNote(t, app, slug, "History Note", "version 2")

	resp := app.get(t, "/notes/"+slug+"/history")
	body := bodyStr(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body[:min(300, len(body))])
	}
	if !strings.Contains(body, "create:") || !strings.Contains(body, "update:") {
		t.Errorf("expected version entries, got: %s", body[:min(500, len(body))])
	}
}

func TestNoteHistoryGET_SingleVersion(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	slug := createNoteAndGetSlug(t, app, "Single Version", "only content")

	resp := app.get(t, "/notes/"+slug+"/history")
	body := bodyStr(t, resp)
	if !strings.Contains(body, "create:") {
		t.Errorf("expected single create version, got: %s", body[:min(300, len(body))])
	}
}

func TestNoteHistoryGET_NotFound(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	resp := app.get(t, "/notes/nonexistent/history")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestNoteHistoryGET_IncludesDrawingChanges(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	slug := createNoteAndGetSlug(t, app, "Drawing History", "some content")

	// Save a drawing via the PUT endpoint
	drawingBody := strings.NewReader(`{"shapes":[{"id":"s1"}]}`)
	req, _ := http.NewRequest("PUT", app.url("/notes/"+slug+"/drawing"), drawingBody)
	req.Header.Set("Content-Type", "application/json")
	dresp, err := app.client.Do(req)
	if err != nil {
		t.Fatalf("PUT drawing: %v", err)
	}
	dresp.Body.Close()

	// Fetch history — should include both the note create and the drawing update
	resp := app.get(t, "/notes/"+slug+"/history")
	body := bodyStr(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(body, "update drawing:") {
		t.Errorf("expected drawing commit in history, got: %s", body[:min(500, len(body))])
	}
}

func TestNoteHistoryGET_RequiresAuth(t *testing.T) {
	app := newTestApp(t)
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return nil }}
	resp, _ := client.Get(app.server.URL + "/notes/any/history")
	if !strings.HasSuffix(resp.Request.URL.Path, "/login") {
		t.Errorf("expected redirect to login, got %s", resp.Request.URL.Path)
	}
}

// ── Version View ─────────────────────────────────────────────────────────────

func TestNoteVersionGET_RendersOldContent(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	slug := createNoteAndGetSlug(t, app, "Version View", "original content")
	updateNote(t, app, slug, "Version View", "updated content")

	relPath := noteRelPath(t, app, slug)
	versions, err := versioning.Log(app.notesDir, relPath, 10, 0)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(versions) < 2 {
		t.Fatalf("expected >=2 versions, got %d", len(versions))
	}
	oldCommit := versions[len(versions)-1].CommitHash

	resp := app.get(t, "/notes/"+slug+"/history/"+oldCommit)
	body := bodyStr(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body[:min(300, len(body))])
	}
	if !strings.Contains(body, "original content") {
		t.Errorf("expected old content, got: %s", body[:min(300, len(body))])
	}
	if !strings.Contains(body, "version-banner") {
		t.Errorf("expected version-banner indicator")
	}
}

func TestNoteVersionGET_InvalidHash(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	slug := createNoteAndGetSlug(t, app, "Bad Hash", "content")
	resp := app.get(t, "/notes/"+slug+"/history/not-a-hash!!")
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestNoteVersionGET_CommitNotFound(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	slug := createNoteAndGetSlug(t, app, "Missing Commit", "content")
	resp := app.get(t, "/notes/"+slug+"/history/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// ── Diff View ────────────────────────────────────────────────────────────────

func TestNoteVersionDiffGET_ShowsChanges(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	slug := createNoteAndGetSlug(t, app, "Diff Note", "line one\nline two\n")
	updateNote(t, app, slug, "Diff Note", "line one\nline three\n")

	relPath := noteRelPath(t, app, slug)
	versions, _ := versioning.Log(app.notesDir, relPath, 10, 0)
	if len(versions) < 2 {
		t.Fatalf("expected >=2 versions, got %d", len(versions))
	}
	latestCommit := versions[0].CommitHash

	resp := app.get(t, "/notes/"+slug+"/history/"+latestCommit+"/diff")
	body := bodyStr(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body[:min(500, len(body))])
	}
	if !strings.Contains(body, "diff-remove") || !strings.Contains(body, "diff-add") {
		t.Errorf("expected diff lines with diff-remove and diff-add classes, got: %s", body[:min(500, len(body))])
	}
}

func TestNoteVersionDiffGET_InvalidHash(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	slug := createNoteAndGetSlug(t, app, "Diff Bad", "content")
	resp := app.get(t, "/notes/"+slug+"/history/INVALID/diff")
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

// ── Revert ───────────────────────────────────────────────────────────────────

func TestNoteVersionRevertPOST_RevertsContent(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	slug := createNoteAndGetSlug(t, app, "Revert Note", "original")
	updateNote(t, app, slug, "Revert Note", "changed")

	relPath := noteRelPath(t, app, slug)
	versions, _ := versioning.Log(app.notesDir, relPath, 10, 0)
	if len(versions) < 2 {
		t.Fatalf("expected >=2 versions, got %d", len(versions))
	}
	oldCommit := versions[len(versions)-1].CommitHash

	resp := app.postForm(t, "/notes/"+slug+"/history/"+oldCommit+"/revert", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	readerResp := app.get(t, "/notes/"+slug)
	readerBody := bodyStr(t, readerResp)
	if !strings.Contains(readerBody, "original") {
		t.Error("expected reverted content to contain 'original'")
	}
}

func TestNoteVersionRevertPOST_CreatesNewVersion(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	slug := createNoteAndGetSlug(t, app, "Revert Track", "v1")
	updateNote(t, app, slug, "Revert Track", "v2")

	relPath := noteRelPath(t, app, slug)
	versionsBefore, _ := versioning.Log(app.notesDir, relPath, 10, 0)
	oldCommit := versionsBefore[len(versionsBefore)-1].CommitHash

	app.postForm(t, "/notes/"+slug+"/history/"+oldCommit+"/revert", nil)

	versionsAfter, _ := versioning.Log(app.notesDir, relPath, 10, 0)
	if len(versionsAfter) <= len(versionsBefore) {
		t.Errorf("expected revert to create new version: before=%d after=%d", len(versionsBefore), len(versionsAfter))
	}
	if !strings.Contains(versionsAfter[0].Message, "revert:") {
		t.Errorf("expected revert commit message, got: %q", versionsAfter[0].Message)
	}
}

func TestNoteVersionRevertPOST_NoopWhenSameContent(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	slug := createNoteAndGetSlug(t, app, "Noop Revert", "same content")

	relPath := noteRelPath(t, app, slug)
	versions, _ := versioning.Log(app.notesDir, relPath, 10, 0)
	latestCommit := versions[0].CommitHash

	resp := app.postForm(t, "/notes/"+slug+"/history/"+latestCommit+"/revert", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for no-op revert, got %d", resp.StatusCode)
	}
}

func TestNoteVersionRevertPOST_InvalidHash(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	slug := createNoteAndGetSlug(t, app, "Bad Revert", "content")
	resp := app.postForm(t, "/notes/"+slug+"/history/INVALID/revert", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

// ── Authorship ───────────────────────────────────────────────────────────────

func TestNoteUpdate_CommitsWithAuthorName(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	slug := createNoteAndGetSlug(t, app, "Authorship Note", "first")
	updateNote(t, app, slug, "Authorship Note", "second")

	relPath := noteRelPath(t, app, slug)
	versions, err := versioning.Log(app.notesDir, relPath, 2, 0)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(versions) == 0 {
		t.Fatal("expected at least one version")
	}
	// Most recent commit should be attributed to "alice"
	if versions[0].AuthorName != "alice" {
		t.Errorf("expected AuthorName 'alice', got %q", versions[0].AuthorName)
	}
}

func TestSharedNoteHistoryGET_ShowsVersionsWithAuthors(t *testing.T) {
	app := newTestApp(t)

	// Alice creates a note
	app.login(t, "alice")
	slug := createNoteAndGetSlug(t, app, "Collab History", "alice initial")

	// Grant bob edit access directly via DB
	aliceU, _ := models.GetUserByUsername(app.db, "alice")
	bobU, err := models.GetOrCreateUser(app.db, "bob")
	if err != nil {
		t.Fatalf("create bob: %v", err)
	}
	notes, _ := models.ListNotes(app.db, aliceU.ID, "", false)
	var noteID int64
	for _, n := range notes {
		if n.Slug == slug {
			noteID = n.ID
			break
		}
	}
	if noteID == 0 {
		t.Fatalf("note %q not found", slug)
	}
	if err := models.GrantShare(app.db, noteID, bobU.ID, aliceU.ID, "edit"); err != nil {
		t.Fatalf("GrantShare: %v", err)
	}

	// Bob edits the shared note
	bobClient := &http.Client{
		Jar: newCookieJar(),
		CheckRedirect: func(req *http.Request, via []*http.Request) error { return nil },
	}
	bobClient.PostForm(app.url("/login"), url.Values{"username": {"bob"}}) //nolint:errcheck
	req, _ := http.NewRequest("POST", app.url("/shared/alice/"+slug), strings.NewReader(url.Values{
		"title": {"Collab History"},
		"body":  {"bob edited"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	bobClient.Do(req) //nolint:errcheck

	// Bob views the shared history
	resp, err := bobClient.Get(app.url("/shared/alice/" + slug + "/history"))
	if err != nil {
		t.Fatalf("GET shared history: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "Collab History") {
		t.Errorf("expected note title in history page, got: %s", bodyStr)
	}
	// The history should show bob's authorship in the author column
	if !strings.Contains(bodyStr, "bob") {
		t.Errorf("expected 'bob' as author in history, got: %s", bodyStr)
	}
}

func TestSharedNoteHistoryGET_RequiresAccess(t *testing.T) {
	app := newTestApp(t)

	// Alice creates a note (not shared with anyone)
	app.login(t, "alice")
	createNoteAndGetSlug(t, app, "Private Note", "secret")

	// Carol has no access
	carolClient := &http.Client{
		Jar: newCookieJar(),
		CheckRedirect: func(req *http.Request, via []*http.Request) error { return nil },
	}
	carolClient.PostForm(app.url("/login"), url.Values{"username": {"carol"}}) //nolint:errcheck

	resp, err := carolClient.Get(app.url("/shared/alice/private-note/history"))
	if err != nil {
		t.Fatalf("GET shared history: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// ── Last Editor on Reader ─────────────────────────────────────────────────────

func TestNoteReaderGET_ShowsLastEditor(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	slug := createNoteAndGetSlug(t, app, "Last Editor Note", "initial content")

	resp := app.get(t, "/notes/"+slug)
	body := bodyStr(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(body, "alice") {
		t.Errorf("expected 'alice' as last editor in reader, got: %s", body[:min(300, len(body))])
	}
}
