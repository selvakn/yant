package storage_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/selvakn/yant/internal/storage"
)

func TestWriteNote_ErrorWhenUserDirIsAFile(t *testing.T) {
	root := t.TempDir()
	// Create a regular FILE where the user directory should be
	if err := os.WriteFile(filepath.Join(root, "7"), []byte("blocker"), 0644); err != nil {
		t.Fatal(err)
	}
	// WriteNote should fail: MkdirAll cannot create a dir where a file exists
	err := storage.WriteNote(root, 7, "note", "body")
	if err == nil {
		t.Error("expected error when user dir path is occupied by a file")
	}
}

func TestDeleteNoteFile_ErrorOnPermission(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root: permission restrictions don't apply")
	}
	root := t.TempDir()
	// Write a note
	storage.WriteNote(root, 1, "perm-note", "content") //nolint:errcheck
	// Make the directory read-only
	dirPath := filepath.Join(root, "1")
	os.Chmod(dirPath, 0555) //nolint:errcheck
	defer os.Chmod(dirPath, 0755) //nolint:errcheck

	err := storage.DeleteNoteFile(root, 1, "perm-note")
	if err == nil {
		t.Error("expected error deleting from read-only directory")
	}
}
