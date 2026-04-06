package handlers_test

import (
	"bytes"
	"encoding/json"
	"io"
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

func TestDrawingPUT_CreatesDrawing(t *testing.T) {
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

	// Verify file was created
	drawingPath := filepath.Join(app.notesDir, "1", slug+".tldraw.json")
	data, err := os.ReadFile(drawingPath)
	if err != nil {
		t.Fatalf("drawing file not created: %v", err)
	}
	if string(data) != drawingData {
		t.Errorf("expected %q, got %q", drawingData, string(data))
	}
}

func TestDrawingGET_ReturnsExistingDrawing(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Test"}, "body": {"content"}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	// Create drawing via PUT
	drawingData := `{"document":{"shapes":[]}}`
	req, _ := http.NewRequest("PUT", app.url("/notes/"+slug+"/drawing"), bytes.NewBufferString(drawingData))
	req.Header.Set("Content-Type", "application/json")
	app.client.Do(req) //nolint:errcheck

	// GET drawing
	req2, _ := http.NewRequest("GET", app.url("/notes/"+slug+"/drawing"), nil)
	resp, err := app.client.Do(req2)
	if err != nil {
		t.Fatalf("GET drawing: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != drawingData {
		t.Errorf("expected %q, got %q", drawingData, string(body))
	}
}

func TestDrawingDELETE_RemovesDrawing(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Test"}, "body": {"content"}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	// Create drawing
	req, _ := http.NewRequest("PUT", app.url("/notes/"+slug+"/drawing"), bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	app.client.Do(req) //nolint:errcheck

	// Delete drawing
	req2, _ := http.NewRequest("DELETE", app.url("/notes/"+slug+"/drawing"), nil)
	resp, err := app.client.Do(req2)
	if err != nil {
		t.Fatalf("DELETE drawing: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Verify file was removed
	drawingPath := filepath.Join(app.notesDir, "1", slug+".tldraw.json")
	if _, err := os.Stat(drawingPath); !os.IsNotExist(err) {
		t.Error("expected drawing file to be removed")
	}
}

func TestNoteDelete_CascadesDrawing(t *testing.T) {
	app := newTestApp(t)
	app.login(t, "alice")
	app.postForm(t, "/notes", url.Values{"title": {"Test"}, "body": {"content"}})

	u, _ := models.GetUserByUsername(app.db, "alice")
	notes, _ := models.ListNotes(app.db, u.ID, "", false)
	slug := notes[0].Slug

	// Create drawing
	req, _ := http.NewRequest("PUT", app.url("/notes/"+slug+"/drawing"), bytes.NewBufferString(`{"test":1}`))
	req.Header.Set("Content-Type", "application/json")
	app.client.Do(req) //nolint:errcheck

	// Verify drawing exists
	drawingPath := filepath.Join(app.notesDir, "1", slug+".tldraw.json")
	if _, err := os.Stat(drawingPath); os.IsNotExist(err) {
		t.Fatal("drawing should exist before delete")
	}

	// Delete note via POST with X-HTTP-Method-Override
	req2, _ := http.NewRequest("POST", app.url("/notes/"+slug), nil)
	req2.Header.Set("X-HTTP-Method-Override", "DELETE")
	resp, err := app.client.Do(req2)
	if err != nil {
		t.Fatalf("DELETE note: %v", err)
	}
	resp.Body.Close()

	// Verify drawing was cascade-deleted
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
