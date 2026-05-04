package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type DrawingType string

const (
	DrawingTldraw     DrawingType = "tldraw"
	DrawingExcalidraw DrawingType = "excalidraw"
	DrawingNone       DrawingType = ""
)

// DrawingFile represents a drawing file found on disk.
type DrawingFile struct {
	DrawingID string
	Type      DrawingType
	IsLegacy  bool
}

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

// Multi-drawing file naming: <slug>--<drawingID>.<tool>.json
func drawingPathByID(root string, userID int64, slug, drawingID string, dt DrawingType) string {
	return filepath.Join(root, fmt.Sprintf("%d", userID), slug+"--"+drawingID+drawingExtension(dt))
}

// DrawingRelPathByID returns the relative path for version control.
func DrawingRelPathByID(userID int64, slug, drawingID string, dt DrawingType) string {
	return fmt.Sprintf("%d/%s--%s%s", userID, slug, drawingID, drawingExtension(dt))
}

// ListDrawingFiles lists all drawing files for a note (both new and legacy formats).
func ListDrawingFiles(root string, userID int64, slug string) []DrawingFile {
	dir := filepath.Join(root, fmt.Sprintf("%d", userID))
	var files []DrawingFile

	// Check legacy files first
	for _, dt := range []DrawingType{DrawingExcalidraw, DrawingTldraw} {
		legacyPath := drawingPathTyped(root, userID, slug, dt)
		if fileExists(legacyPath) {
			files = append(files, DrawingFile{DrawingID: "", Type: dt, IsLegacy: true})
		}
	}

	// Check new-format files: <slug>--<id>.<tool>.json
	prefix := slug + "--"
	entries, err := os.ReadDir(dir)
	if err != nil {
		return files
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		rest := name[len(prefix):]
		for _, dt := range []DrawingType{DrawingExcalidraw, DrawingTldraw} {
			ext := drawingExtension(dt)
			if strings.HasSuffix(rest, ext) {
				id := rest[:len(rest)-len(ext)]
				if id != "" {
					files = append(files, DrawingFile{DrawingID: id, Type: dt, IsLegacy: false})
				}
			}
		}
	}
	return files
}

// ReadDrawingByID reads a specific drawing by its ID.
func ReadDrawingByID(root string, userID int64, slug, drawingID string, dt DrawingType) ([]byte, error) {
	p := drawingPathByID(root, userID, slug, drawingID, dt)
	return os.ReadFile(p)
}

// WriteDrawingByID writes a drawing file using the new multi-drawing naming.
func WriteDrawingByID(root string, userID int64, slug, drawingID string, dt DrawingType, data []byte) error {
	if err := EnsureUserDir(root, userID); err != nil {
		return err
	}
	return os.WriteFile(drawingPathByID(root, userID, slug, drawingID, dt), data, 0644)
}

// DeleteDrawingByID removes a specific drawing file by ID.
func DeleteDrawingByID(root string, userID int64, slug, drawingID string, dt DrawingType) error {
	p := drawingPathByID(root, userID, slug, drawingID, dt)
	err := os.Remove(p)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// DetectLegacyDrawing checks if a legacy (old-format) drawing exists.
func DetectLegacyDrawing(root string, userID int64, slug string) (DrawingType, bool) {
	for _, dt := range []DrawingType{DrawingExcalidraw, DrawingTldraw} {
		if fileExists(drawingPathTyped(root, userID, slug, dt)) {
			return dt, true
		}
	}
	return DrawingNone, false
}

// MigrateLegacyDrawing renames a legacy drawing file to the new multi-drawing format.
// Returns the detected drawing type and any error.
func MigrateLegacyDrawing(root string, userID int64, slug, newDrawingID string) (DrawingType, error) {
	dt, found := DetectLegacyDrawing(root, userID, slug)
	if !found {
		return DrawingNone, fmt.Errorf("no legacy drawing found for %s", slug)
	}
	oldPath := drawingPathTyped(root, userID, slug, dt)
	newPath := drawingPathByID(root, userID, slug, newDrawingID, dt)
	if err := os.Rename(oldPath, newPath); err != nil {
		return DrawingNone, err
	}
	return dt, nil
}

// DrawingFilePathByID returns the absolute file path for a drawing by ID.
func DrawingFilePathByID(root string, userID int64, slug, drawingID string, dt DrawingType) string {
	return drawingPathByID(root, userID, slug, drawingID, dt)
}

// DeleteAllDrawingsBySlug removes all drawing files (new format) and their SVG previews for a note.
func DeleteAllDrawingsBySlug(root string, userID int64, slug string) error {
	files := ListDrawingFiles(root, userID, slug)
	for _, f := range files {
		var p string
		if f.IsLegacy {
			p = drawingPathTyped(root, userID, slug, f.Type)
		} else {
			p = drawingPathByID(root, userID, slug, f.DrawingID, f.Type)
			_ = DeleteDrawingSVG(root, userID, slug, f.DrawingID)
		}
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func svgPathByID(root string, userID int64, slug, drawingID string) string {
	return filepath.Join(root, fmt.Sprintf("%d", userID), slug+"--"+drawingID+".svg")
}

// WriteDrawingSVG stores a pre-rendered SVG preview for a drawing.
func WriteDrawingSVG(root string, userID int64, slug, drawingID string, data []byte) error {
	if err := EnsureUserDir(root, userID); err != nil {
		return err
	}
	return os.WriteFile(svgPathByID(root, userID, slug, drawingID), data, 0644)
}

// ReadDrawingSVG reads the SVG preview for a drawing. Returns os.ErrNotExist if not present.
func ReadDrawingSVG(root string, userID int64, slug, drawingID string) ([]byte, error) {
	return os.ReadFile(svgPathByID(root, userID, slug, drawingID))
}

// DeleteDrawingSVG removes the SVG preview file for a drawing.
func DeleteDrawingSVG(root string, userID int64, slug, drawingID string) error {
	err := os.Remove(svgPathByID(root, userID, slug, drawingID))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
