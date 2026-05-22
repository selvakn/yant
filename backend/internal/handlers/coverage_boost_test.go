package handlers_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/selvakn/yant/internal/models"
	"github.com/selvakn/yant/internal/versioning"
)

// ── Public media endpoints ────────────────────────────────────────────────────

func TestPublicDrawingsListGET_ValidToken(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "pubdrawlist")
	app.postForm(t, "/notes", url.Values{"title": {"DrawList Note"}, "body": {"content"}})
	token := publishNote(t, app, "drawlist-note")

	ua := unauthClient(t)
	resp, err := ua.Get(app.url("/p/" + token + "/drawings"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var out map[string]any
	json.NewDecoder(resp.Body).Decode(&out) //nolint:errcheck
	if _, ok := out["drawings"]; !ok {
		t.Error("expected 'drawings' key in response")
	}
}

func TestPublicDrawingsListGET_InvalidToken(t *testing.T) {
	app := newTestApp(t)
	ua := unauthClient(t)
	resp, err := ua.Get(app.url("/p/badtoken/drawings"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestPublicDrawingGET_ValidTokenNoDrawing(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "pubdrawget")
	app.postForm(t, "/notes", url.Values{"title": {"DrawGet Note"}, "body": {"content"}})
	token := publishNote(t, app, "drawget-note")

	ua := unauthClient(t)
	resp, err := ua.Get(app.url("/p/" + token + "/drawing"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	// No drawing exists for this note — should return 404 JSON
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for note without drawing, got %d", resp.StatusCode)
	}
}

func TestPublicDrawingGET_InvalidToken(t *testing.T) {
	app := newTestApp(t)
	ua := unauthClient(t)
	resp, err := ua.Get(app.url("/p/badtoken/drawing"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestPublicDrawingByIDGET_InvalidToken(t *testing.T) {
	app := newTestApp(t)
	ua := unauthClient(t)
	resp, err := ua.Get(app.url("/p/badtoken/drawings/someid"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestPublicDrawingByIDGET_UnknownDrawing(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "pubdrawbyid")
	app.postForm(t, "/notes", url.Values{"title": {"ByID Note"}, "body": {"content"}})
	token := publishNote(t, app, "byid-note")

	ua := unauthClient(t)
	resp, err := ua.Get(app.url("/p/" + token + "/drawings/nosuchdrawing"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for unknown drawing, got %d", resp.StatusCode)
	}
}

func TestPublicImageServeGET_InvalidToken(t *testing.T) {
	app := newTestApp(t)
	ua := unauthClient(t)
	resp, err := ua.Get(app.url("/p/badtoken/uploads/foo.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestPublicImageServeGET_PathTraversal(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "pubimgtraverse")
	app.postForm(t, "/notes", url.Values{"title": {"Traverse Note"}, "body": {"content"}})
	token := publishNote(t, app, "traverse-note")

	ua := unauthClient(t)
	resp, err := ua.Get(app.url("/p/" + token + "/uploads/../../etc/passwd"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	// chi router will match the literal filename including ".." — handler rejects it
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for path traversal, got %d", resp.StatusCode)
	}
}

func TestPublicImageServeGET_ImageNotBelongingToNote(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "pubimgnotbelongs")
	app.postForm(t, "/notes", url.Values{"title": {"Belongs Note"}, "body": {"content"}})
	token := publishNote(t, app, "belongs-note")

	ua := unauthClient(t)
	resp, err := ua.Get(app.url("/p/" + token + "/uploads/doesnotexist.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for image not belonging to note, got %d", resp.StatusCode)
	}
}

// ── Shared note version diff/revert ─────────────────────────────────────────

func sharedSetupWithHistory(t *testing.T) (app *testApp, slug, commit1, commit2 string, bobClient *http.Client) {
	t.Helper()
	app = newTestApp(t)
	app.login(t, "alice")
	slug = createNoteAndGetSlug(t, app, "Shared History Note", "first version")
	updateNote(t, app, slug, "Shared History Note", "second version")

	aliceU, _ := models.GetUserByUsername(app.db, "alice")
	bobU, _ := models.GetOrCreateUser(app.db, "bob")

	notes, _ := models.ListNotes(app.db, aliceU.ID, "", false)
	var noteID int64
	for _, n := range notes {
		if n.Slug == slug {
			noteID = n.ID
			break
		}
	}
	if err := models.GrantShare(app.db, noteID, bobU.ID, aliceU.ID, "edit"); err != nil {
		t.Fatalf("GrantShare: %v", err)
	}

	relPath := noteRelPath(t, app, slug)
	versions, err := versioning.Log(app.notesDir, relPath, 10, 0)
	if err != nil || len(versions) < 2 {
		t.Fatalf("need at least 2 versions, got %d: %v", len(versions), err)
	}
	commit1 = versions[len(versions)-1].CommitHash // oldest
	commit2 = versions[0].CommitHash              // newest

	bobClient = &http.Client{
		Jar: newCookieJar(),
		CheckRedirect: func(req *http.Request, via []*http.Request) error { return nil },
	}
	bobClient.PostForm(app.url("/login"), url.Values{"username": {"bob"}}) //nolint:errcheck

	return
}

func TestSharedNoteVersionDiffGET_InvalidHash(t *testing.T) {
	app, slug, _, _, bobClient := sharedSetupWithHistory(t)
	resp, err := bobClient.Get(app.url("/shared/alice/" + slug + "/history/badhash/diff"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestSharedNoteVersionDiffGET_NoAccess(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	slug := createNoteAndGetSlug(t, app, "Private Diff", "v1")
	updateNote(t, app, slug, "Private Diff", "v2")

	relPath := noteRelPath(t, app, slug)
	versions, _ := versioning.Log(app.notesDir, relPath, 10, 0)
	if len(versions) < 1 {
		t.Skip("no versions")
	}
	hash := versions[0].CommitHash

	carolClient := &http.Client{
		Jar:           newCookieJar(),
		CheckRedirect: func(req *http.Request, via []*http.Request) error { return nil },
	}
	carolClient.PostForm(app.url("/login"), url.Values{"username": {"carol"}}) //nolint:errcheck

	resp, err := carolClient.Get(app.url("/shared/alice/" + slug + "/history/" + hash + "/diff"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestSharedNoteVersionDiffGET_EditorCanView(t *testing.T) {
	app, slug, commit1, commit2, bobClient := sharedSetupWithHistory(t)
	resp, err := bobClient.Get(app.url("/shared/alice/" + slug + "/history/" + commit2 + "/diff?against=" + commit1))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("expected 200, got %d: %s", resp.StatusCode, body)
	}
}

func TestSharedNoteVersionRevertPOST_InvalidHash(t *testing.T) {
	app, slug, _, _, bobClient := sharedSetupWithHistory(t)
	req, _ := http.NewRequest("POST", app.url("/shared/alice/"+slug+"/history/badhash/revert"), nil)
	resp, err := bobClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestSharedNoteVersionRevertPOST_ReadOnlyForbidden(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	slug := createNoteAndGetSlug(t, app, "ReadOnly Revert", "v1")
	updateNote(t, app, slug, "ReadOnly Revert", "v2")

	aliceU, _ := models.GetUserByUsername(app.db, "alice")
	carolU, _ := models.GetOrCreateUser(app.db, "carol")
	notes, _ := models.ListNotes(app.db, aliceU.ID, "", false)
	var noteID int64
	for _, n := range notes {
		if n.Slug == slug {
			noteID = n.ID
			break
		}
	}
	models.GrantShare(app.db, noteID, carolU.ID, aliceU.ID, "read") //nolint:errcheck

	relPath := noteRelPath(t, app, slug)
	versions, _ := versioning.Log(app.notesDir, relPath, 10, 0)
	if len(versions) < 1 {
		t.Skip("no versions")
	}
	hash := versions[len(versions)-1].CommitHash

	carolClient := &http.Client{
		Jar:           newCookieJar(),
		CheckRedirect: func(req *http.Request, via []*http.Request) error { return nil },
	}
	carolClient.PostForm(app.url("/login"), url.Values{"username": {"carol"}}) //nolint:errcheck

	req, _ := http.NewRequest("POST", app.url("/shared/alice/"+slug+"/history/"+hash+"/revert"), nil)
	resp, err := carolClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 for read-only user, got %d", resp.StatusCode)
	}
}

func TestSharedNoteVersionRevertPOST_EditorCanRevert(t *testing.T) {
	app, slug, commit1, _, bobClient := sharedSetupWithHistory(t)

	req, _ := http.NewRequest("POST", app.url("/shared/alice/"+slug+"/history/"+commit1+"/revert"),
		strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := bobClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("expected 200, got %d: %s", resp.StatusCode, body)
	}
}
