package handlers_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/selvakn/yant/internal/models"
)

func TestMultiDrawingLifecycle(t *testing.T) {
	// Arrange
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Lifecycle Note"}, "body": {"initial"}})

	u, err := models.GetUserByUsername(app.db, "alice")
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	notes, err := models.ListNotes(app.db, u.ID, "", false)
	if err != nil || len(notes) != 1 {
		t.Fatalf("notes: %v len=%d", err, len(notes))
	}
	slug := notes[0].Slug

	postDrawing := func(displayName, toolType string) map[string]string {
		t.Helper()
		body := `{"display_name":"` + displayName + `","tool_type":"` + toolType + `"}`
		req, _ := http.NewRequest(http.MethodPost, app.url("/notes/"+slug+"/drawings"), bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.client.Do(req)
		if err != nil {
			t.Fatalf("POST drawings: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("POST drawings: status %d: %s", resp.StatusCode, b)
		}
		var out map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatalf("decode create: %v", err)
		}
		return out
	}

	// Act / Assert — create Arch Diagram (excalidraw), Sequence Flow (tldraw)
	outArch := postDrawing("Arch Diagram", "excalidraw")
	outSeq := postDrawing("Sequence Flow", "tldraw")
	id1 := outArch["drawing_id"]
	id2 := outSeq["drawing_id"]
	if id1 == "" || id2 == "" || id1 == id2 {
		t.Fatalf("unexpected ids: %q %q", id1, id2)
	}

	reqList, _ := http.NewRequest(http.MethodGet, app.url("/notes/"+slug+"/drawings"), nil)
	respList, err := app.client.Do(reqList)
	if err != nil {
		t.Fatalf("GET drawings: %v", err)
	}
	defer respList.Body.Close()
	if respList.StatusCode != http.StatusOK {
		t.Fatalf("GET drawings list: %d", respList.StatusCode)
	}
	var list1 struct {
		Drawings []struct {
			DrawingID   string `json:"drawing_id"`
			DisplayName string `json:"display_name"`
			ToolType    string `json:"tool_type"`
		} `json:"drawings"`
	}
	if err := json.NewDecoder(respList.Body).Decode(&list1); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list1.Drawings) != 2 {
		t.Fatalf("expected 2 drawings, got %d", len(list1.Drawings))
	}

	excalPayload := `{"type":"excalidraw","elements":[{"id":"a1","type":"rectangle","x":0,"y":0,"width":10,"height":10}]}`
	reqPut, _ := http.NewRequest(http.MethodPut, app.url("/notes/"+slug+"/drawings/"+id1), bytes.NewBufferString(excalPayload))
	reqPut.Header.Set("Content-Type", "application/json")
	respPut, err := app.client.Do(reqPut)
	if err != nil {
		t.Fatalf("PUT drawing: %v", err)
	}
	defer respPut.Body.Close()
	if respPut.StatusCode != http.StatusOK {
		t.Fatalf("PUT drawing: %d", respPut.StatusCode)
	}

	reqGet, _ := http.NewRequest(http.MethodGet, app.url("/notes/"+slug+"/drawings/"+id1), nil)
	respGet, err := app.client.Do(reqGet)
	if err != nil {
		t.Fatalf("GET drawing: %v", err)
	}
	defer respGet.Body.Close()
	if respGet.StatusCode != http.StatusOK {
		t.Fatalf("GET drawing: %d", respGet.StatusCode)
	}
	var wrapped map[string]json.RawMessage
	if err := json.NewDecoder(respGet.Body).Decode(&wrapped); err != nil {
		t.Fatalf("decode get: %v", err)
	}
	if string(wrapped["type"]) != `"excalidraw"` {
		t.Fatalf("type: %s", wrapped["type"])
	}
	if wrapped["data"] == nil {
		t.Fatal("expected data key")
	}

	patchBody := `{"display_name":"Updated Name"}`
	reqPatch, _ := http.NewRequest(http.MethodPatch, app.url("/notes/"+slug+"/drawings/"+id1), bytes.NewBufferString(patchBody))
	reqPatch.Header.Set("Content-Type", "application/json")
	respPatch, err := app.client.Do(reqPatch)
	if err != nil {
		t.Fatalf("PATCH: %v", err)
	}
	defer respPatch.Body.Close()
	if respPatch.StatusCode != http.StatusOK {
		t.Fatalf("PATCH: %d", respPatch.StatusCode)
	}

	reqList2, _ := http.NewRequest(http.MethodGet, app.url("/notes/"+slug+"/drawings"), nil)
	respList2, err := app.client.Do(reqList2)
	if err != nil {
		t.Fatalf("GET drawings: %v", err)
	}
	defer respList2.Body.Close()
	var list2 struct {
		Drawings []struct {
			DrawingID   string `json:"drawing_id"`
			DisplayName string `json:"display_name"`
		} `json:"drawings"`
	}
	if err := json.NewDecoder(respList2.Body).Decode(&list2); err != nil {
		t.Fatalf("decode list2: %v", err)
	}
	if len(list2.Drawings) != 2 {
		t.Fatalf("expected 2 drawings after rename, got %d", len(list2.Drawings))
	}
	foundUpdated := false
	for _, d := range list2.Drawings {
		if d.DrawingID == id1 && d.DisplayName == "Updated Name" {
			foundUpdated = true
		}
	}
	if !foundUpdated {
		t.Fatalf("rename not reflected: %#v", list2.Drawings)
	}

	reqDel, _ := http.NewRequest(http.MethodDelete, app.url("/notes/"+slug+"/drawings/"+id2), nil)
	respDel, err := app.client.Do(reqDel)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	defer respDel.Body.Close()
	if respDel.StatusCode != http.StatusOK {
		t.Fatalf("DELETE: %d", respDel.StatusCode)
	}

	reqList3, _ := http.NewRequest(http.MethodGet, app.url("/notes/"+slug+"/drawings"), nil)
	respList3, err := app.client.Do(reqList3)
	if err != nil {
		t.Fatalf("GET drawings: %v", err)
	}
	defer respList3.Body.Close()
	var list3 struct {
		Drawings []struct {
			DrawingID   string `json:"drawing_id"`
			DisplayName string `json:"display_name"`
		} `json:"drawings"`
	}
	if err := json.NewDecoder(respList3.Body).Decode(&list3); err != nil {
		t.Fatalf("decode list3: %v", err)
	}
	if len(list3.Drawings) != 1 {
		t.Fatalf("expected 1 drawing left, got %d", len(list3.Drawings))
	}
	if list3.Drawings[0].DrawingID != id1 || list3.Drawings[0].DisplayName != "Updated Name" {
		t.Fatalf("unexpected remaining: %#v", list3.Drawings[0])
	}
}

func TestDrawingIsolationBetweenNotes(t *testing.T) {
	// Arrange
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Iso Alpha"}, "body": {""}})
	app.postForm(t, "/notes", url.Values{"title": {"Iso Beta"}, "body": {""}})

	u, err := models.GetUserByUsername(app.db, "alice")
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	notes, err := models.ListNotes(app.db, u.ID, "", false)
	if err != nil || len(notes) != 2 {
		t.Fatalf("notes: %v len=%d", err, len(notes))
	}
	var slugA, slugB string
	for _, n := range notes {
		switch n.Title {
		case "Iso Alpha":
			slugA = n.Slug
		case "Iso Beta":
			slugB = n.Slug
		}
	}
	if slugA == "" || slugB == "" {
		t.Fatalf("slugs missing: %q %q", slugA, slugB)
	}

	createOn := func(slug, name string) {
		t.Helper()
		body := `{"display_name":"` + name + `","tool_type":"tldraw"}`
		req, _ := http.NewRequest(http.MethodPost, app.url("/notes/"+slug+"/drawings"), bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.client.Do(req)
		if err != nil {
			t.Fatalf("POST drawings: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("POST drawings %s: %d %s", slug, resp.StatusCode, b)
		}
	}

	// Act
	createOn(slugA, "Drawing A")
	createOn(slugB, "Drawing B")

	countDrawings := func(slug string) int {
		t.Helper()
		req, _ := http.NewRequest(http.MethodGet, app.url("/notes/"+slug+"/drawings"), nil)
		resp, err := app.client.Do(req)
		if err != nil {
			t.Fatalf("GET drawings: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET drawings %s: %d", slug, resp.StatusCode)
		}
		var out struct {
			Drawings []any `json:"drawings"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		return len(out.Drawings)
	}

	if n := countDrawings(slugA); n != 1 {
		t.Fatalf("note A: want 1 drawing, got %d", n)
	}
	if n := countDrawings(slugB); n != 1 {
		t.Fatalf("note B: want 1 drawing, got %d", n)
	}

	delReq, _ := http.NewRequest(http.MethodPost, app.url("/notes/"+slugA), nil)
	delReq.Header.Set("X-HTTP-Method-Override", "DELETE")
	delResp, err := app.client.Do(delReq)
	if err != nil {
		t.Fatalf("DELETE note: %v", err)
	}
	delResp.Body.Close()

	// Assert — B unaffected
	if n := countDrawings(slugB); n != 1 {
		t.Fatalf("after deleting note A, note B should still have 1 drawing, got %d", n)
	}
}

func TestDrawingMarkerRendering(t *testing.T) {
	// Arrange
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Marker Note"}, "body": {"before"}})

	u, err := models.GetUserByUsername(app.db, "alice")
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	notes, err := models.ListNotes(app.db, u.ID, "", false)
	if err != nil || len(notes) != 1 {
		t.Fatalf("notes: %v", err)
	}
	slug := notes[0].Slug
	title := notes[0].Title

	createBody := `{"display_name":"Embedded","tool_type":"tldraw"}`
	reqCreate, _ := http.NewRequest(http.MethodPost, app.url("/notes/"+slug+"/drawings"), bytes.NewBufferString(createBody))
	reqCreate.Header.Set("Content-Type", "application/json")
	respCreate, err := app.client.Do(reqCreate)
	if err != nil {
		t.Fatalf("POST drawings: %v", err)
	}
	defer respCreate.Body.Close()
	if respCreate.StatusCode != http.StatusCreated {
		t.Fatalf("create: %d", respCreate.StatusCode)
	}
	var created map[string]string
	if err := json.NewDecoder(respCreate.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	id := created["drawing_id"]
	if id == "" {
		t.Fatal("empty drawing_id")
	}

	marker := "![[draw:" + id + "]]"
	form := url.Values{
		"title": {title},
		"body":  {"See diagram:\n\n" + marker},
	}
	reqUp, _ := http.NewRequest(http.MethodPost, app.url("/notes/"+slug), strings.NewReader(form.Encode()))
	reqUp.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqUp.Header.Set("X-HTTP-Method-Override", "PUT")
	respUp, err := app.client.Do(reqUp)
	if err != nil {
		t.Fatalf("PUT note: %v", err)
	}
	respUp.Body.Close()

	// Act
	respReader := app.get(t, "/notes/"+slug)
	html := bodyStr(t, respReader)

	// Assert
	want := `<div class="drawing-embed" data-drawing-id="` + id + `"></div>`
	if !strings.Contains(html, want) {
		t.Fatalf("reader HTML missing embed %q; snippet: %s", want, html[:min(800, len(html))])
	}
}

func TestMultiDrawingValidation(t *testing.T) {
	// Arrange
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Validation Note"}, "body": {""}})

	u, err := models.GetUserByUsername(app.db, "alice")
	if err != nil {
		t.Fatalf("user: %v", err)
	}
	notes, err := models.ListNotes(app.db, u.ID, "", false)
	if err != nil || len(notes) != 1 {
		t.Fatalf("notes: %v", err)
	}
	slug := notes[0].Slug

	assertStatus := func(resp *http.Response, want int) {
		t.Helper()
		if resp.StatusCode != want {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("status: want %d got %d: %s", want, resp.StatusCode, b)
		}
		resp.Body.Close()
	}

	// Empty display name
	bodyEmpty := `{"display_name":"","tool_type":"tldraw"}`
	req1, _ := http.NewRequest(http.MethodPost, app.url("/notes/"+slug+"/drawings"), bytes.NewBufferString(bodyEmpty))
	req1.Header.Set("Content-Type", "application/json")
	resp1, err := app.client.Do(req1)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	assertStatus(resp1, http.StatusBadRequest)

	// Invalid tool type
	bodyBadTool := `{"display_name":"Ok","tool_type":"paint"}`
	req2, _ := http.NewRequest(http.MethodPost, app.url("/notes/"+slug+"/drawings"), bytes.NewBufferString(bodyBadTool))
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := app.client.Do(req2)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	assertStatus(resp2, http.StatusBadRequest)

	ghost := "nonexistent"

	req3, _ := http.NewRequest(http.MethodGet, app.url("/notes/"+slug+"/drawings/"+ghost), nil)
	resp3, err := app.client.Do(req3)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	assertStatus(resp3, http.StatusNotFound)

	req4, _ := http.NewRequest(http.MethodPut, app.url("/notes/"+slug+"/drawings/"+ghost), bytes.NewBufferString(`{}`))
	req4.Header.Set("Content-Type", "application/json")
	resp4, err := app.client.Do(req4)
	if err != nil {
		t.Fatalf("PUT: %v", err)
	}
	assertStatus(resp4, http.StatusNotFound)

	req5, _ := http.NewRequest(http.MethodDelete, app.url("/notes/"+slug+"/drawings/"+ghost), nil)
	resp5, err := app.client.Do(req5)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	assertStatus(resp5, http.StatusNotFound)
}
