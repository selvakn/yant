package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/selvakn/yant/internal/models"
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
