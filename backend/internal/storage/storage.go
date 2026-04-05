package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

// EnsureUserDir creates the per-user notes directory if it does not exist.
func EnsureUserDir(root string, userID int64) error {
	return os.MkdirAll(filepath.Join(root, fmt.Sprintf("%d", userID)), 0755)
}

// WriteNote writes (or overwrites) the markdown body for a note.
func WriteNote(root string, userID int64, slug, body string) error {
	if err := EnsureUserDir(root, userID); err != nil {
		return err
	}
	path := notePath(root, userID, slug)
	return os.WriteFile(path, []byte(body), 0644)
}

// ReadNote reads the markdown body for a note.
func ReadNote(root string, userID int64, slug string) (string, error) {
	data, err := os.ReadFile(notePath(root, userID, slug))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// DeleteNoteFile removes the markdown file for a note.
func DeleteNoteFile(root string, userID int64, slug string) error {
	err := os.Remove(notePath(root, userID, slug))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// EnsureUploadsDir creates the per-user uploads directory if it does not exist.
func EnsureUploadsDir(root string, userID int64) error {
	return os.MkdirAll(filepath.Join(root, fmt.Sprintf("%d", userID)), 0755)
}

// UploadPath returns the full path for an uploaded image file.
func UploadPath(root string, userID int64, filename string) string {
	return filepath.Join(root, fmt.Sprintf("%d", userID), filename)
}

func notePath(root string, userID int64, slug string) string {
	return filepath.Join(root, fmt.Sprintf("%d", userID), slug+".md")
}
