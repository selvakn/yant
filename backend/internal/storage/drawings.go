package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

type DrawingType string

const (
	DrawingTldraw    DrawingType = "tldraw"
	DrawingExcalidraw DrawingType = "excalidraw"
	DrawingNone      DrawingType = ""
)

func drawingExtension(dt DrawingType) string {
	switch dt {
	case DrawingExcalidraw:
		return ".excalidraw.json"
	default:
		return ".tldraw.json"
	}
}

func drawingPathTyped(root string, userID int64, slug string, dt DrawingType) string {
	return filepath.Join(root, fmt.Sprintf("%d", userID), slug+drawingExtension(dt))
}

// DetectDrawingType returns the DrawingType for a note by checking which file exists.
// Returns DrawingNone if no drawing file exists.
func DetectDrawingType(root string, userID int64, slug string) DrawingType {
	if fileExists(drawingPathTyped(root, userID, slug, DrawingExcalidraw)) {
		return DrawingExcalidraw
	}
	if fileExists(drawingPathTyped(root, userID, slug, DrawingTldraw)) {
		return DrawingTldraw
	}
	return DrawingNone
}

// ReadDrawing reads the drawing JSON for a note, auto-detecting the type.
// Returns os.ErrNotExist if no drawing exists.
func ReadDrawing(root string, userID int64, slug string) ([]byte, DrawingType, error) {
	dt := DetectDrawingType(root, userID, slug)
	if dt == DrawingNone {
		return nil, DrawingNone, os.ErrNotExist
	}
	data, err := os.ReadFile(drawingPathTyped(root, userID, slug, dt))
	return data, dt, err
}

// WriteDrawing writes (or overwrites) the drawing JSON for a note with the specified type.
func WriteDrawing(root string, userID int64, slug string, dt DrawingType, data []byte) error {
	if err := EnsureUserDir(root, userID); err != nil {
		return err
	}
	return os.WriteFile(drawingPathTyped(root, userID, slug, dt), data, 0644)
}

// DeleteDrawing removes all drawing files for a note (both types).
// Returns nil if no drawing file exists.
func DeleteDrawing(root string, userID int64, slug string) error {
	for _, dt := range []DrawingType{DrawingTldraw, DrawingExcalidraw} {
		p := drawingPathTyped(root, userID, slug, dt)
		err := os.Remove(p)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// DrawingExists checks if any drawing file exists for a note.
func DrawingExists(root string, userID int64, slug string) bool {
	return DetectDrawingType(root, userID, slug) != DrawingNone
}

// DrawingPath returns the file path for a drawing of the given type.
func DrawingPath(root string, userID int64, slug string, dt DrawingType) string {
	return drawingPathTyped(root, userID, slug, dt)
}

// DrawingRelPath returns the relative path (from notes root) for version control.
func DrawingRelPath(userID int64, slug string, dt DrawingType) string {
	return fmt.Sprintf("%d/%s%s", userID, slug, drawingExtension(dt))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
