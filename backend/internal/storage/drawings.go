package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

// ReadDrawing reads the tldraw JSON for a note.
// Returns os.ErrNotExist if no drawing exists.
func ReadDrawing(root string, userID int64, slug string) ([]byte, error) {
	return os.ReadFile(drawingPath(root, userID, slug))
}

// WriteDrawing writes (or overwrites) the tldraw JSON for a note.
func WriteDrawing(root string, userID int64, slug string, data []byte) error {
	if err := EnsureUserDir(root, userID); err != nil {
		return err
	}
	return os.WriteFile(drawingPath(root, userID, slug), data, 0644)
}

// DeleteDrawing removes the tldraw JSON file for a note.
// Returns nil if the file does not exist.
func DeleteDrawing(root string, userID int64, slug string) error {
	err := os.Remove(drawingPath(root, userID, slug))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// DrawingExists checks if a drawing file exists for a note.
func DrawingExists(root string, userID int64, slug string) bool {
	_, err := os.Stat(drawingPath(root, userID, slug))
	return err == nil
}

func drawingPath(root string, userID int64, slug string) string {
	return filepath.Join(root, fmt.Sprintf("%d", userID), slug+".tldraw.json")
}
