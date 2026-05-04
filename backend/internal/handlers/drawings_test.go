package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/selvakn/yant/internal/models"
	"github.com/selvakn/yant/internal/storage"
)

func TestDrawingGET_NoDrawingReturns404(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Test"}, "body": {"content"}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	req, _ := http.NewRequest("GET", app.url("/notes/"+slug+"/drawing"), nil)
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatalf("GET drawing: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result) //nolint:errcheck
	if result["error"] != "no drawing" {
		t.Errorf("expected error 'no drawing', got %v", result)
	}
}

func TestDrawingPUT_CreatesDrawing_Tldraw(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Test"}, "body": {"content"}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	drawingData := `{"document":{"store":{"test":true}}}`
	req, _ := http.NewRequest("PUT", app.url("/notes/"+slug+"/drawing"), bytes.NewBufferString(drawingData))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatalf("PUT drawing: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	drawingPath := filepath.Join(app.notesDir, "1", slug+".tldraw.json")
	data, err := os.ReadFile(drawingPath)
	if err != nil {
		t.Fatalf("drawing file not created: %v", err)
	}
	if string(data) != drawingData {
		t.Errorf("expected %q, got %q", drawingData, string(data))
	}
}

func TestDrawingPUT_CreatesDrawing_Excalidraw(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Test"}, "body": {"content"}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	drawingData := `{"type":"excalidraw","elements":[]}`
	req, _ := http.NewRequest("PUT", app.url("/notes/"+slug+"/drawing?type=excalidraw"), bytes.NewBufferString(drawingData))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatalf("PUT drawing: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	drawingPath := filepath.Join(app.notesDir, "1", slug+".excalidraw.json")
	data, err := os.ReadFile(drawingPath)
	if err != nil {
		t.Fatalf("excalidraw drawing file not created: %v", err)
	}
	if string(data) != drawingData {
		t.Errorf("expected %q, got %q", drawingData, string(data))
	}
}

func TestDrawingPUT_ConflictWhenDifferentType(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Test"}, "body": {"content"}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	// Create tldraw drawing
	req, _ := http.NewRequest("PUT", app.url("/notes/"+slug+"/drawing"), bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	app.client.Do(req) //nolint:errcheck

	// Try excalidraw on same note
	req2, _ := http.NewRequest("PUT", app.url("/notes/"+slug+"/drawing?type=excalidraw"), bytes.NewBufferString(`{}`))
	req2.Header.Set("Content-Type", "application/json")
	resp, err := app.client.Do(req2)
	if err != nil {
		t.Fatalf("PUT drawing: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409, got %d", resp.StatusCode)
	}
}

func TestDrawingGET_ReturnsTldrawWrapped(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Test"}, "body": {"content"}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	drawingData := `{"document":{"shapes":[]}}`
	req, _ := http.NewRequest("PUT", app.url("/notes/"+slug+"/drawing"), bytes.NewBufferString(drawingData))
	req.Header.Set("Content-Type", "application/json")
	app.client.Do(req) //nolint:errcheck

	req2, _ := http.NewRequest("GET", app.url("/notes/"+slug+"/drawing"), nil)
	resp, err := app.client.Do(req2)
	if err != nil {
		t.Fatalf("GET drawing: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]json.RawMessage
	json.NewDecoder(resp.Body).Decode(&result) //nolint:errcheck

	if string(result["type"]) != `"tldraw"` {
		t.Errorf("expected type tldraw, got %s", result["type"])
	}
	if result["document"] == nil {
		t.Error("expected document key in response")
	}
}

func TestDrawingGET_ReturnsExcalidrawWrapped(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Test"}, "body": {"content"}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	drawingData := `{"type":"excalidraw","elements":[]}`
	req, _ := http.NewRequest("PUT", app.url("/notes/"+slug+"/drawing?type=excalidraw"), bytes.NewBufferString(drawingData))
	req.Header.Set("Content-Type", "application/json")
	app.client.Do(req) //nolint:errcheck

	req2, _ := http.NewRequest("GET", app.url("/notes/"+slug+"/drawing"), nil)
	resp, err := app.client.Do(req2)
	if err != nil {
		t.Fatalf("GET drawing: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]json.RawMessage
	json.NewDecoder(resp.Body).Decode(&result) //nolint:errcheck

	if string(result["type"]) != `"excalidraw"` {
		t.Errorf("expected type excalidraw, got %s", result["type"])
	}
	if result["data"] == nil {
		t.Error("expected data key in response")
	}
}

func TestDrawingDELETE_RemovesTldrawDrawing(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Test"}, "body": {"content"}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	req, _ := http.NewRequest("PUT", app.url("/notes/"+slug+"/drawing"), bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	app.client.Do(req) //nolint:errcheck

	req2, _ := http.NewRequest("DELETE", app.url("/notes/"+slug+"/drawing"), nil)
	resp, err := app.client.Do(req2)
	if err != nil {
		t.Fatalf("DELETE drawing: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	drawingPath := filepath.Join(app.notesDir, "1", slug+".tldraw.json")
	if _, err := os.Stat(drawingPath); !os.IsNotExist(err) {
		t.Error("expected drawing file to be removed")
	}
}

func TestDrawingDELETE_RemovesExcalidrawDrawing(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Test"}, "body": {"content"}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	req, _ := http.NewRequest("PUT", app.url("/notes/"+slug+"/drawing?type=excalidraw"), bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	app.client.Do(req) //nolint:errcheck

	req2, _ := http.NewRequest("DELETE", app.url("/notes/"+slug+"/drawing"), nil)
	resp, err := app.client.Do(req2)
	if err != nil {
		t.Fatalf("DELETE drawing: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	drawingPath := filepath.Join(app.notesDir, "1", slug+".excalidraw.json")
	if _, err := os.Stat(drawingPath); !os.IsNotExist(err) {
		t.Error("expected excalidraw drawing file to be removed")
	}
}

func TestNoteDelete_CascadesDrawing(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Test"}, "body": {"content"}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	req, _ := http.NewRequest("PUT", app.url("/notes/"+slug+"/drawing"), bytes.NewBufferString(`{"test":1}`))
	req.Header.Set("Content-Type", "application/json")
	app.client.Do(req) //nolint:errcheck

	drawingPath := filepath.Join(app.notesDir, "1", slug+".tldraw.json")
	if _, err := os.Stat(drawingPath); os.IsNotExist(err) {
		t.Fatal("drawing should exist before delete")
	}

	req2, _ := http.NewRequest("POST", app.url("/notes/"+slug), nil)
	req2.Header.Set("X-HTTP-Method-Override", "DELETE")
	resp, err := app.client.Do(req2)
	if err != nil {
		t.Fatalf("DELETE note: %v", err)
	}
	resp.Body.Close()

	if _, err := os.Stat(drawingPath); !os.IsNotExist(err) {
		t.Error("expected drawing to be cascade-deleted with note")
	}
}

func TestDrawingGET_NonexistentNoteReturns404(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	req, _ := http.NewRequest("GET", app.url("/notes/nonexistent/drawing"), nil)
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatalf("GET drawing: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestDrawingPUT_NonexistentNoteReturns404(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")

	req, _ := http.NewRequest("PUT", app.url("/notes/nonexistent/drawing"), bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatalf("PUT drawing: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestDrawingPUT_DefaultTypeisTldraw(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Test"}, "body": {"content"}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	req, _ := http.NewRequest("PUT", app.url("/notes/"+slug+"/drawing"), bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	app.client.Do(req) //nolint:errcheck

	tldrawPath := filepath.Join(app.notesDir, "1", slug+".tldraw.json")
	if _, err := os.Stat(tldrawPath); os.IsNotExist(err) {
		t.Error("expected tldraw drawing file to exist (default type)")
	}
	excalidrawPath := filepath.Join(app.notesDir, "1", slug+".excalidraw.json")
	if _, err := os.Stat(excalidrawPath); !os.IsNotExist(err) {
		t.Error("expected excalidraw drawing file to NOT exist")
	}
}

func TestLegacyDrawingMigration(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Test"}, "body": {"content"}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	userDir := filepath.Join(app.notesDir, strconv.FormatInt(u.ID, 10))
	drawingJSON := []byte(`{"document":{"legacy":true}}`)
	if err := storage.WriteDrawing(app.notesDir, u.ID, slug, storage.DrawingTldraw, drawingJSON); err != nil {
		t.Fatalf("WriteDrawing: %v", err)
	}

	legacyPath := filepath.Join(userDir, slug+".tldraw.json")
	if _, err := os.Stat(legacyPath); err != nil {
		t.Fatalf("legacy file should exist before GET: %v", err)
	}

	req, _ := http.NewRequest("GET", app.url("/notes/"+slug+"/drawings"), nil)
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatalf("GET drawings: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Drawings []struct {
			DrawingID   string `json:"drawing_id"`
			DisplayName string `json:"display_name"`
			ToolType    string `json:"tool_type"`
		} `json:"drawings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Drawings) != 1 {
		t.Fatalf("expected 1 drawing after migration, got %d", len(result.Drawings))
	}
	d := result.Drawings[0]
	if d.DisplayName != "Drawing 1" {
		t.Errorf("display_name: want Drawing 1, got %q", d.DisplayName)
	}
	if d.ToolType != "tldraw" {
		t.Errorf("tool_type: want tldraw, got %q", d.ToolType)
	}
	if d.DrawingID == "" {
		t.Fatal("expected non-empty drawing_id")
	}

	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Error("legacy drawing file should be removed after migration")
	}

	newPath := filepath.Join(userDir, slug+"--"+d.DrawingID+".tldraw.json")
	got, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("new-format drawing file missing: %v", err)
	}
	if string(got) != string(drawingJSON) {
		t.Errorf("file content: got %q, want %q", got, drawingJSON)
	}
}

func TestDrawingsListGET_empty(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Test"}, "body": {"content"}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	req, _ := http.NewRequest("GET", app.url("/notes/"+slug+"/drawings"), nil)
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatalf("GET drawings: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result struct {
		Drawings []map[string]any `json:"drawings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Drawings) != 0 {
		t.Errorf("expected empty drawings slice, got %#v", result.Drawings)
	}
}

func TestDrawingsCreatePOST_valid(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Test"}, "body": {"content"}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	body := `{"display_name":"Diagram A","tool_type":"tldraw"}`
	req, _ := http.NewRequest("POST", app.url("/notes/"+slug+"/drawings"), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatalf("POST drawings: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var out map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["drawing_id"] == "" {
		t.Error("expected non-empty drawing_id")
	}
	if out["display_name"] != "Diagram A" {
		t.Errorf("display_name: got %q", out["display_name"])
	}
	if out["tool_type"] != "tldraw" {
		t.Errorf("tool_type: got %q", out["tool_type"])
	}
	wantMarker := "![[draw:" + out["drawing_id"] + "]]"
	if out["marker"] != wantMarker {
		t.Errorf("marker: want %q got %q", wantMarker, out["marker"])
	}
}

func TestDrawingsCreatePOST_invalid_tool(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Test"}, "body": {"content"}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	body := `{"display_name":"Ok","tool_type":"paint"}`
	req, _ := http.NewRequest("POST", app.url("/notes/"+slug+"/drawings"), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatalf("POST drawings: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	var out map[string]string
	json.NewDecoder(resp.Body).Decode(&out) //nolint:errcheck
	if !strings.Contains(out["error"], "invalid") {
		t.Errorf("expected invalid tool error, got %q", out["error"])
	}
}

func TestDrawingsCreatePOST_empty_name(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Test"}, "body": {"content"}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	body := `{"display_name":"","tool_type":"tldraw"}`
	req, _ := http.NewRequest("POST", app.url("/notes/"+slug+"/drawings"), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatalf("POST drawings: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestDrawingByIDGET_not_found(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Test"}, "body": {"content"}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	req, _ := http.NewRequest("GET", app.url("/notes/"+slug+"/drawings/does-not-exist"), nil)
	resp, err := app.client.Do(req)
	if err != nil {
		t.Fatalf("GET drawing by id: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestDrawingByIDPUT_and_GET(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Test"}, "body": {"content"}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	createBody := `{"display_name":"D","tool_type":"tldraw"}`
	reqCreate, _ := http.NewRequest("POST", app.url("/notes/"+slug+"/drawings"), bytes.NewBufferString(createBody))
	reqCreate.Header.Set("Content-Type", "application/json")
	respCreate, err := app.client.Do(reqCreate)
	if err != nil {
		t.Fatalf("POST drawings: %v", err)
	}
	defer respCreate.Body.Close()
	if respCreate.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 on create, got %d", respCreate.StatusCode)
	}
	var created map[string]string
	json.NewDecoder(respCreate.Body).Decode(&created) //nolint:errcheck
	id := created["drawing_id"]
	if id == "" {
		t.Fatal("missing drawing_id")
	}

	drawingPayload := `{"document":{"store":{"k":1}}}`
	reqPut, _ := http.NewRequest("PUT", app.url("/notes/"+slug+"/drawings/"+id), bytes.NewBufferString(drawingPayload))
	reqPut.Header.Set("Content-Type", "application/json")
	respPut, err := app.client.Do(reqPut)
	if err != nil {
		t.Fatalf("PUT drawing: %v", err)
	}
	defer respPut.Body.Close()
	if respPut.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on PUT, got %d", respPut.StatusCode)
	}

	reqGet, _ := http.NewRequest("GET", app.url("/notes/"+slug+"/drawings/"+id), nil)
	respGet, err := app.client.Do(reqGet)
	if err != nil {
		t.Fatalf("GET drawing: %v", err)
	}
	defer respGet.Body.Close()
	if respGet.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on GET, got %d", respGet.StatusCode)
	}

	var wrapped map[string]json.RawMessage
	json.NewDecoder(respGet.Body).Decode(&wrapped) //nolint:errcheck
	if string(wrapped["type"]) != `"tldraw"` {
		t.Errorf("type: got %s", wrapped["type"])
	}
	if wrapped["document"] == nil {
		t.Fatal("expected document in response")
	}
}

func TestDrawingByIDRenamePATCH(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Test"}, "body": {"content"}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	createBody := `{"display_name":"Old","tool_type":"excalidraw"}`
	reqCreate, _ := http.NewRequest("POST", app.url("/notes/"+slug+"/drawings"), bytes.NewBufferString(createBody))
	reqCreate.Header.Set("Content-Type", "application/json")
	respCreate, err := app.client.Do(reqCreate)
	if err != nil {
		t.Fatalf("POST drawings: %v", err)
	}
	defer respCreate.Body.Close()
	var created map[string]string
	json.NewDecoder(respCreate.Body).Decode(&created) //nolint:errcheck
	id := created["drawing_id"]

	patchBody := `{"display_name":"Renamed"}`
	reqPatch, _ := http.NewRequest("PATCH", app.url("/notes/"+slug+"/drawings/"+id), bytes.NewBufferString(patchBody))
	reqPatch.Header.Set("Content-Type", "application/json")
	respPatch, err := app.client.Do(reqPatch)
	if err != nil {
		t.Fatalf("PATCH drawing: %v", err)
	}
	defer respPatch.Body.Close()
	if respPatch.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on PATCH, got %d", respPatch.StatusCode)
	}

	reqList, _ := http.NewRequest("GET", app.url("/notes/"+slug+"/drawings"), nil)
	respList, err := app.client.Do(reqList)
	if err != nil {
		t.Fatalf("GET drawings list: %v", err)
	}
	defer respList.Body.Close()
	var listOut struct {
		Drawings []struct {
			DrawingID   string `json:"drawing_id"`
			DisplayName string `json:"display_name"`
		} `json:"drawings"`
	}
	json.NewDecoder(respList.Body).Decode(&listOut) //nolint:errcheck
	if len(listOut.Drawings) != 1 {
		t.Fatalf("expected 1 drawing, got %d", len(listOut.Drawings))
	}
	if listOut.Drawings[0].DisplayName != "Renamed" {
		t.Errorf("display_name after rename: got %q", listOut.Drawings[0].DisplayName)
	}
}

func TestDrawingByIDDELETE(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Test"}, "body": {"content"}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	createBody := `{"display_name":"ToDelete","tool_type":"tldraw"}`
	reqCreate, _ := http.NewRequest("POST", app.url("/notes/"+slug+"/drawings"), bytes.NewBufferString(createBody))
	reqCreate.Header.Set("Content-Type", "application/json")
	respCreate, err := app.client.Do(reqCreate)
	if err != nil {
		t.Fatalf("POST drawings: %v", err)
	}
	defer respCreate.Body.Close()
	var created map[string]string
	json.NewDecoder(respCreate.Body).Decode(&created) //nolint:errcheck
	id := created["drawing_id"]

	reqDel, _ := http.NewRequest("DELETE", app.url("/notes/"+slug+"/drawings/"+id), nil)
	respDel, err := app.client.Do(reqDel)
	if err != nil {
		t.Fatalf("DELETE drawing: %v", err)
	}
	defer respDel.Body.Close()
	if respDel.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on DELETE, got %d", respDel.StatusCode)
	}

	reqGet, _ := http.NewRequest("GET", app.url("/notes/"+slug+"/drawings/"+id), nil)
	respGet, err := app.client.Do(reqGet)
	if err != nil {
		t.Fatalf("GET after delete: %v", err)
	}
	defer respGet.Body.Close()
	if respGet.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", respGet.StatusCode)
	}

	reqList, _ := http.NewRequest("GET", app.url("/notes/"+slug+"/drawings"), nil)
	respList, err := app.client.Do(reqList)
	if err != nil {
		t.Fatalf("GET drawings list: %v", err)
	}
	defer respList.Body.Close()
	var listOut struct {
		Drawings []any `json:"drawings"`
	}
	json.NewDecoder(respList.Body).Decode(&listOut) //nolint:errcheck
	if len(listOut.Drawings) != 0 {
		t.Errorf("expected no drawings after delete, got %d", len(listOut.Drawings))
	}
}
