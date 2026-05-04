package handlers_test

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/selvakn/yant/internal/models"
	"github.com/selvakn/yant/internal/storage"
)

func noteSlugByTitle(t *testing.T, app *testApp, username, wantedTitle string) string {
	t.Helper()
	u, err := models.GetUserByUsername(app.db, username)
	if err != nil {
		t.Fatal(err)
	}
	notes, err := models.ListNotes(app.db, u.ID, "", false)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range notes {
		if n.Title == wantedTitle {
			return n.Slug
		}
	}
	t.Fatalf("no note with title %q for user %q", wantedTitle, username)
	return ""
}

func createBlogPost(t *testing.T, app *testApp, username, title, body string) {
	t.Helper()
	app.login(t, username)
	resp := app.postForm(t, "/notes", url.Values{"title": {title}, "body": {body}})
	io.Copy(io.Discard, resp.Body) //nolint:errcheck
	resp.Body.Close()

	slug := noteSlugByTitle(t, app, username, title)

	req, err := http.NewRequest(http.MethodPost, app.url("/notes/"+slug), strings.NewReader(url.Values{
		"title": {title},
		"body":  {body + " #blog"},
	}.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-HTTP-Method-Override", "PUT")
	resp2, err := app.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp2.Body) //nolint:errcheck
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("update note: status %d", resp2.StatusCode)
	}
}

func TestBlogIndexGET_empty(t *testing.T) {
	app := newTestApp(t)
	ua := unauthClient(t)
	resp, err := ua.Get(app.url("/blog"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	body := bodyStr(t, resp)
	if !strings.Contains(body, "No posts yet") {
		t.Fatalf("expected empty blog message, body=%q", body)
	}
}

func TestBlogIndexGET_with_posts(t *testing.T) {
	app := newTestApp(t)
	createBlogPost(t, app, "alice", "Post One", "zebrauniquealpha first body.")
	createBlogPost(t, app, "alice", "Post Two", "zebrauniquebeta second body.")

	ua := unauthClient(t)
	resp, err := ua.Get(app.url("/blog"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	html := bodyStr(t, resp)
	if !strings.Contains(html, "Post One") || !strings.Contains(html, "Post Two") {
		t.Fatalf("expected both titles in body=%q", html)
	}
	if !strings.Contains(html, "zebrauniquealpha") || !strings.Contains(html, "zebrauniquebeta") {
		t.Fatalf("expected excerpts in body=%q", html)
	}
}

func TestBlogIndexGET_excludes_archived(t *testing.T) {
	app := newTestApp(t)
	createBlogPost(t, app, "alice", "Archived Blog", "will vanish")
	slug := noteSlugByTitle(t, app, "alice", "Archived Blog")

	app.login(t, "alice")
	resp := app.put(t, "/notes/"+slug+"/archive", nil)
	io.Copy(io.Discard, resp.Body) //nolint:errcheck
	resp.Body.Close()

	ua := unauthClient(t)
	r2, err := ua.Get(app.url("/blog"))
	if err != nil {
		t.Fatal(err)
	}
	html := bodyStr(t, r2)
	if strings.Contains(html, "Archived Blog") {
		t.Fatal("archived post should not appear on blog index")
	}
}

func TestBlogIndexGET_excludes_non_blog(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	resp := app.postForm(t, "/notes", url.Values{"title": {"Secret Draft"}, "body": {"private stuff"}})
	io.Copy(io.Discard, resp.Body) //nolint:errcheck
	resp.Body.Close()

	createBlogPost(t, app, "alice", "Published Only", "hello world")

	ua := unauthClient(t)
	r2, err := ua.Get(app.url("/blog"))
	if err != nil {
		t.Fatal(err)
	}
	html := bodyStr(t, r2)
	if !strings.Contains(html, "Published Only") {
		t.Fatal("expected blog post on index")
	}
	if strings.Contains(html, "Secret Draft") {
		t.Fatal("non-blog note must not appear on blog index")
	}
}

func TestBlogIndexGET_accessible_without_auth(t *testing.T) {
	app := newTestApp(t)
	ua := unauthClient(t)
	resp, err := ua.Get(app.url("/blog"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestBlogPostGET_success(t *testing.T) {
	app := newTestApp(t)
	createBlogPost(t, app, "alice", "My Post", "Hello blog visible body.")

	ua := unauthClient(t)
	resp, err := ua.Get(app.url("/blog/alice/my-post"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	html := bodyStr(t, resp)
	if !strings.Contains(html, "My Post") {
		t.Fatalf("expected title in body=%q", html)
	}
	if !strings.Contains(html, "Hello blog visible body") {
		t.Fatalf("expected rendered body in html=%q", html)
	}
}

func TestBlogPostGET_not_found(t *testing.T) {
	app := newTestApp(t)
	ua := unauthClient(t)
	resp, err := ua.Get(app.url("/blog/alice/nonexistent-slug-xyz"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestBlogPostGET_not_blog_returns_404(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	resp := app.postForm(t, "/notes", url.Values{"title": {"Private Note"}, "body": {"no blog tag here"}})
	io.Copy(io.Discard, resp.Body) //nolint:errcheck
	resp.Body.Close()

	ua := unauthClient(t)
	r2, err := ua.Get(app.url("/blog/alice/private-note"))
	if err != nil {
		t.Fatal(err)
	}
	r2.Body.Close()
	if r2.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", r2.StatusCode)
	}
}

func TestBlogPostGET_archived_returns_404(t *testing.T) {
	app := newTestApp(t)
	createBlogPost(t, app, "alice", "Was Public", "body")
	slug := noteSlugByTitle(t, app, "alice", "Was Public")

	app.login(t, "alice")
	resp := app.put(t, "/notes/"+slug+"/archive", nil)
	io.Copy(io.Discard, resp.Body) //nolint:errcheck
	resp.Body.Close()

	ua := unauthClient(t)
	r2, err := ua.Get(app.url("/blog/alice/" + slug))
	if err != nil {
		t.Fatal(err)
	}
	r2.Body.Close()
	if r2.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", r2.StatusCode)
	}
}

func TestBlogPostGET_accessible_without_auth(t *testing.T) {
	app := newTestApp(t)
	createBlogPost(t, app, "alice", "Public Blog Post", "content")

	ua := unauthClient(t)
	resp, err := ua.Get(app.url("/blog/alice/public-blog-post"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestBlogTagGET_with_matching_posts(t *testing.T) {
	app := newTestApp(t)
	createBlogPost(t, app, "alice", "Go Tips", "intro #golang")
	createBlogPost(t, app, "alice", "Rust Tips", "intro #rust")

	ua := unauthClient(t)
	resp, err := ua.Get(app.url("/blog/tag/golang"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	html := bodyStr(t, resp)
	if !strings.Contains(html, "Go Tips") {
		t.Fatalf("expected golang post, body=%q", html)
	}
	if strings.Contains(html, "Rust Tips") {
		t.Fatal("rust post should not appear for golang tag filter")
	}
}

func TestBlogTagGET_empty_tag(t *testing.T) {
	app := newTestApp(t)
	createBlogPost(t, app, "alice", "Tagged", "x #blog #web")

	ua := unauthClient(t)
	resp, err := ua.Get(app.url("/blog/tag/nonexistent-tag-xyz"))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	html := bodyStr(t, resp)
	if !strings.Contains(html, "No posts tagged") {
		t.Fatalf("expected empty tag message, body=%q", html)
	}
}

func TestBlogDrawingSVGGET_success(t *testing.T) {
	app := newTestApp(t)
	createBlogPost(t, app, "alice", "Draw Post", "note body")
	u, err := models.GetUserByUsername(app.db, "alice")
	if err != nil {
		t.Fatal(err)
	}
	slug := noteSlugByTitle(t, app, "alice", "Draw Post")
	note, err := models.GetNote(app.db, u.ID, slug)
	if err != nil || note == nil {
		t.Fatalf("get note: %v", err)
	}
	d, err := models.CreateDrawing(app.db, note.ID, "Test Draw", "tldraw")
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.WriteDrawingSVG(app.notesDir, u.ID, slug, d.DrawingID, []byte("<svg></svg>")); err != nil {
		t.Fatal(err)
	}

	ua := unauthClient(t)
	resp, err := ua.Get(app.url("/blog/alice/" + slug + "/drawings/" + d.DrawingID + "/svg"))
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body) //nolint:errcheck
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "image/svg+xml") {
		t.Fatalf("expected image/svg+xml Content-Type, got %q", ct)
	}
}

func TestBlogDrawingSVGGET_non_blog_returns_404(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	resp := app.postForm(t, "/notes", url.Values{"title": {"Regular"}, "body": {"plain"}})
	io.Copy(io.Discard, resp.Body) //nolint:errcheck
	resp.Body.Close()

	u, err := models.GetUserByUsername(app.db, "alice")
	if err != nil {
		t.Fatal(err)
	}
	slug := noteSlugByTitle(t, app, "alice", "Regular")
	note, err := models.GetNote(app.db, u.ID, slug)
	if err != nil || note == nil {
		t.Fatalf("get note: %v", err)
	}
	d, err := models.CreateDrawing(app.db, note.ID, "X", "tldraw")
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.WriteDrawingSVG(app.notesDir, u.ID, slug, d.DrawingID, []byte("<svg/>")); err != nil {
		t.Fatal(err)
	}

	ua := unauthClient(t)
	r2, err := ua.Get(app.url("/blog/alice/" + slug + "/drawings/" + d.DrawingID + "/svg"))
	if err != nil {
		t.Fatal(err)
	}
	r2.Body.Close()
	if r2.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", r2.StatusCode)
	}
}
