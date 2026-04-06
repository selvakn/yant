package storage_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/selvakn/yant/internal/storage"
)

func TestEnsureUserDir_CreatesDirectory(t *testing.T) {
	root := t.TempDir()
	if err := storage.EnsureUserDir(root, 42); err != nil {
		t.Fatalf("EnsureUserDir: %v", err)
	}
	dir := filepath.Join(root, "42")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("expected directory %s to exist", dir)
	}
}

func TestEnsureUserDir_IdempotentOnRepeat(t *testing.T) {
	root := t.TempDir()
	if err := storage.EnsureUserDir(root, 1); err != nil {
		t.Fatal(err)
	}
	if err := storage.EnsureUserDir(root, 1); err != nil {
		t.Errorf("second call should not error: %v", err)
	}
}

func TestWriteNote_CreatesFileAtCorrectPath(t *testing.T) {
	root := t.TempDir()
	if err := storage.WriteNote(root, 7, "my-note", "# Hello"); err != nil {
		t.Fatalf("WriteNote: %v", err)
	}
	path := filepath.Join(root, "7", "my-note.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("file not found at %s: %v", path, err)
	}
	if string(data) != "# Hello" {
		t.Errorf("expected '# Hello', got %q", string(data))
	}
}

func TestWriteNote_OverwritesExistingFile(t *testing.T) {
	root := t.TempDir()
	storage.WriteNote(root, 1, "note", "first") //nolint:errcheck
	storage.WriteNote(root, 1, "note", "second") //nolint:errcheck
	path := filepath.Join(root, "1", "note.md")
	data, _ := os.ReadFile(path)
	if string(data) != "second" {
		t.Errorf("expected 'second', got %q", string(data))
	}
}

func TestReadNote_ReturnsWrittenContent(t *testing.T) {
	root := t.TempDir()
	storage.WriteNote(root, 3, "hello", "world content") //nolint:errcheck
	body, err := storage.ReadNote(root, 3, "hello")
	if err != nil {
		t.Fatalf("ReadNote: %v", err)
	}
	if body != "world content" {
		t.Errorf("expected 'world content', got %q", body)
	}
}

func TestReadNote_MissingFileReturnsError(t *testing.T) {
	root := t.TempDir()
	_, err := storage.ReadNote(root, 99, "nonexistent")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestDeleteNoteFile_RemovesFile(t *testing.T) {
	root := t.TempDir()
	storage.WriteNote(root, 2, "to-delete", "gone") //nolint:errcheck
	if err := storage.DeleteNoteFile(root, 2, "to-delete"); err != nil {
		t.Fatalf("DeleteNoteFile: %v", err)
	}
	path := filepath.Join(root, "2", "to-delete.md")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected file to be removed")
	}
}

func TestDeleteNoteFile_MissingFileNoError(t *testing.T) {
	root := t.TempDir()
	if err := storage.DeleteNoteFile(root, 5, "ghost"); err != nil {
		t.Errorf("expected no error for missing file, got: %v", err)
	}
}

func TestEnsureUploadsDir_CreatesDirectory(t *testing.T) {
	root := t.TempDir()
	if err := storage.EnsureUploadsDir(root, 10); err != nil {
		t.Fatalf("EnsureUploadsDir: %v", err)
	}
	dir := filepath.Join(root, "10")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("expected upload dir %s to exist", dir)
	}
}

func TestUploadPath_ReturnsCorrectPath(t *testing.T) {
	path := storage.UploadPath("/uploads", 5, "abc.png")
	expected := "/uploads/5/abc.png"
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}
