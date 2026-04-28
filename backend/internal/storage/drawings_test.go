package storage_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/selvakn/yant/internal/storage"
)

func TestWriteDrawing_Tldraw_CreatesFileAtCorrectPath(t *testing.T) {
	root := t.TempDir()
	data := []byte(`{"document":{"store":{}}}`)
	if err := storage.WriteDrawing(root, 7, "my-note", storage.DrawingTldraw, data); err != nil {
		t.Fatalf("WriteDrawing: %v", err)
	}
	path := filepath.Join(root, "7", "my-note.tldraw.json")
	read, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("file not found at %s: %v", path, err)
	}
	if string(read) != string(data) {
		t.Errorf("expected %q, got %q", string(data), string(read))
	}
}

func TestWriteDrawing_Excalidraw_CreatesFileAtCorrectPath(t *testing.T) {
	root := t.TempDir()
	data := []byte(`{"type":"excalidraw","elements":[]}`)
	if err := storage.WriteDrawing(root, 7, "my-note", storage.DrawingExcalidraw, data); err != nil {
		t.Fatalf("WriteDrawing: %v", err)
	}
	path := filepath.Join(root, "7", "my-note.excalidraw.json")
	read, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("file not found at %s: %v", path, err)
	}
	if string(read) != string(data) {
		t.Errorf("expected %q, got %q", string(data), string(read))
	}
}

func TestWriteDrawing_OverwritesExistingFile(t *testing.T) {
	root := t.TempDir()
	storage.WriteDrawing(root, 1, "note", storage.DrawingTldraw, []byte("first"))  //nolint:errcheck
	storage.WriteDrawing(root, 1, "note", storage.DrawingTldraw, []byte("second")) //nolint:errcheck
	path := filepath.Join(root, "1", "note.tldraw.json")
	data, _ := os.ReadFile(path)
	if string(data) != "second" {
		t.Errorf("expected 'second', got %q", string(data))
	}
}

func TestReadDrawing_ReturnsWrittenContent_Tldraw(t *testing.T) {
	root := t.TempDir()
	expected := []byte(`{"test":true}`)
	storage.WriteDrawing(root, 3, "hello", storage.DrawingTldraw, expected) //nolint:errcheck
	data, dt, err := storage.ReadDrawing(root, 3, "hello")
	if err != nil {
		t.Fatalf("ReadDrawing: %v", err)
	}
	if dt != storage.DrawingTldraw {
		t.Errorf("expected DrawingTldraw, got %q", dt)
	}
	if string(data) != string(expected) {
		t.Errorf("expected %q, got %q", string(expected), string(data))
	}
}

func TestReadDrawing_ReturnsWrittenContent_Excalidraw(t *testing.T) {
	root := t.TempDir()
	expected := []byte(`{"type":"excalidraw","elements":[]}`)
	storage.WriteDrawing(root, 3, "hello", storage.DrawingExcalidraw, expected) //nolint:errcheck
	data, dt, err := storage.ReadDrawing(root, 3, "hello")
	if err != nil {
		t.Fatalf("ReadDrawing: %v", err)
	}
	if dt != storage.DrawingExcalidraw {
		t.Errorf("expected DrawingExcalidraw, got %q", dt)
	}
	if string(data) != string(expected) {
		t.Errorf("expected %q, got %q", string(expected), string(data))
	}
}

func TestReadDrawing_MissingFileReturnsError(t *testing.T) {
	root := t.TempDir()
	_, _, err := storage.ReadDrawing(root, 99, "nonexistent")
	if err == nil {
		t.Error("expected error for missing file")
	}
	if !os.IsNotExist(err) {
		t.Errorf("expected os.ErrNotExist, got: %v", err)
	}
}

func TestDeleteDrawing_RemovesTldrawFile(t *testing.T) {
	root := t.TempDir()
	storage.WriteDrawing(root, 2, "to-delete", storage.DrawingTldraw, []byte("gone")) //nolint:errcheck
	if err := storage.DeleteDrawing(root, 2, "to-delete"); err != nil {
		t.Fatalf("DeleteDrawing: %v", err)
	}
	path := filepath.Join(root, "2", "to-delete.tldraw.json")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected file to be removed")
	}
}

func TestDeleteDrawing_RemovesExcalidrawFile(t *testing.T) {
	root := t.TempDir()
	storage.WriteDrawing(root, 2, "to-delete", storage.DrawingExcalidraw, []byte("gone")) //nolint:errcheck
	if err := storage.DeleteDrawing(root, 2, "to-delete"); err != nil {
		t.Fatalf("DeleteDrawing: %v", err)
	}
	path := filepath.Join(root, "2", "to-delete.excalidraw.json")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected file to be removed")
	}
}

func TestDeleteDrawing_MissingFileNoError(t *testing.T) {
	root := t.TempDir()
	if err := storage.DeleteDrawing(root, 5, "ghost"); err != nil {
		t.Errorf("expected no error for missing file, got: %v", err)
	}
}

func TestDrawingExists_ReturnsTrueForTldraw(t *testing.T) {
	root := t.TempDir()
	storage.WriteDrawing(root, 1, "exists", storage.DrawingTldraw, []byte("{}")) //nolint:errcheck
	if !storage.DrawingExists(root, 1, "exists") {
		t.Error("expected DrawingExists to return true")
	}
}

func TestDrawingExists_ReturnsTrueForExcalidraw(t *testing.T) {
	root := t.TempDir()
	storage.WriteDrawing(root, 1, "exists", storage.DrawingExcalidraw, []byte("{}")) //nolint:errcheck
	if !storage.DrawingExists(root, 1, "exists") {
		t.Error("expected DrawingExists to return true")
	}
}

func TestDrawingExists_ReturnsFalseWhenMissing(t *testing.T) {
	root := t.TempDir()
	if storage.DrawingExists(root, 1, "missing") {
		t.Error("expected DrawingExists to return false")
	}
}

func TestDetectDrawingType_Tldraw(t *testing.T) {
	root := t.TempDir()
	storage.WriteDrawing(root, 1, "note", storage.DrawingTldraw, []byte("{}")) //nolint:errcheck
	dt := storage.DetectDrawingType(root, 1, "note")
	if dt != storage.DrawingTldraw {
		t.Errorf("expected tldraw, got %q", dt)
	}
}

func TestDetectDrawingType_Excalidraw(t *testing.T) {
	root := t.TempDir()
	storage.WriteDrawing(root, 1, "note", storage.DrawingExcalidraw, []byte("{}")) //nolint:errcheck
	dt := storage.DetectDrawingType(root, 1, "note")
	if dt != storage.DrawingExcalidraw {
		t.Errorf("expected excalidraw, got %q", dt)
	}
}

func TestDetectDrawingType_None(t *testing.T) {
	root := t.TempDir()
	dt := storage.DetectDrawingType(root, 1, "note")
	if dt != storage.DrawingNone {
		t.Errorf("expected empty, got %q", dt)
	}
}

func TestDrawingRelPath_Tldraw(t *testing.T) {
	got := storage.DrawingRelPath(42, "my-note", storage.DrawingTldraw)
	want := "42/my-note.tldraw.json"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestDrawingRelPath_Excalidraw(t *testing.T) {
	got := storage.DrawingRelPath(42, "my-note", storage.DrawingExcalidraw)
	want := "42/my-note.excalidraw.json"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}
