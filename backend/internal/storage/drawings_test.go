package storage_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/selvakn/my-notes/internal/storage"
)

func TestWriteDrawing_CreatesFileAtCorrectPath(t *testing.T) {
	root := t.TempDir()
	data := []byte(`{"document":{"store":{}}}`)
	if err := storage.WriteDrawing(root, 7, "my-note", data); err != nil {
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

func TestWriteDrawing_OverwritesExistingFile(t *testing.T) {
	root := t.TempDir()
	storage.WriteDrawing(root, 1, "note", []byte("first"))  //nolint:errcheck
	storage.WriteDrawing(root, 1, "note", []byte("second")) //nolint:errcheck
	path := filepath.Join(root, "1", "note.tldraw.json")
	data, _ := os.ReadFile(path)
	if string(data) != "second" {
		t.Errorf("expected 'second', got %q", string(data))
	}
}

func TestReadDrawing_ReturnsWrittenContent(t *testing.T) {
	root := t.TempDir()
	expected := []byte(`{"test":true}`)
	storage.WriteDrawing(root, 3, "hello", expected) //nolint:errcheck
	data, err := storage.ReadDrawing(root, 3, "hello")
	if err != nil {
		t.Fatalf("ReadDrawing: %v", err)
	}
	if string(data) != string(expected) {
		t.Errorf("expected %q, got %q", string(expected), string(data))
	}
}

func TestReadDrawing_MissingFileReturnsError(t *testing.T) {
	root := t.TempDir()
	_, err := storage.ReadDrawing(root, 99, "nonexistent")
	if err == nil {
		t.Error("expected error for missing file")
	}
	if !os.IsNotExist(err) {
		t.Errorf("expected os.ErrNotExist, got: %v", err)
	}
}

func TestDeleteDrawing_RemovesFile(t *testing.T) {
	root := t.TempDir()
	storage.WriteDrawing(root, 2, "to-delete", []byte("gone")) //nolint:errcheck
	if err := storage.DeleteDrawing(root, 2, "to-delete"); err != nil {
		t.Fatalf("DeleteDrawing: %v", err)
	}
	path := filepath.Join(root, "2", "to-delete.tldraw.json")
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

func TestDrawingExists_ReturnsTrueWhenExists(t *testing.T) {
	root := t.TempDir()
	storage.WriteDrawing(root, 1, "exists", []byte("{}")) //nolint:errcheck
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
