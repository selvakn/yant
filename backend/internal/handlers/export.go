package handlers

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/selvakn/yant/internal/models"
	"github.com/selvakn/yant/internal/storage"
)

var (
	unsafeChars    = regexp.MustCompile(`[/\\:?*|<>"]+`)
	repeatedDashes = regexp.MustCompile(`-{2,}`)
)

func sanitizeFilename(title string) string {
	name := strings.ToLower(title)
	name = unsafeChars.ReplaceAllString(name, "-")
	name = repeatedDashes.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	if name == "" {
		return "untitled-note"
	}
	return name
}

// NoteExportZIP assembles a ZIP containing the note's markdown and all sketch files.
func (h *Handler) NoteExportZIP(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")

	note, err := models.GetNote(h.db, userID, slug)
	if err != nil || note == nil {
		http.NotFound(w, r)
		return
	}

	body, err := storage.ReadNote(h.notesDir, userID, slug)
	if err != nil {
		http.Error(w, "read error", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// note.md
	if f, err := zw.Create("note.md"); err == nil {
		f.Write([]byte(body)) //nolint:errcheck
	}

	// Sketches
	drawings := storage.ListDrawingFiles(h.notesDir, userID, slug)
	for i, d := range drawings {
		n := i + 1
		if d.IsLegacy {
			data, _, rerr := storage.ReadDrawing(h.notesDir, userID, slug)
			if rerr == nil {
				ext := fmt.Sprintf("sketch-%d.%s.json", n, string(d.Type))
				if f, err := zw.Create(ext); err == nil {
					f.Write(data) //nolint:errcheck
				}
			}
		} else {
			data, rerr := storage.ReadDrawingByID(h.notesDir, userID, slug, d.DrawingID, d.Type)
			if rerr == nil {
				src := fmt.Sprintf("sketch-%d.%s.json", n, string(d.Type))
				if f, err := zw.Create(src); err == nil {
					f.Write(data) //nolint:errcheck
				}
			}
			svg, rerr := storage.ReadDrawingSVG(h.notesDir, userID, slug, d.DrawingID)
			if rerr == nil {
				svgName := fmt.Sprintf("sketch-%d.svg", n)
				if f, err := zw.Create(svgName); err == nil {
					f.Write(svg) //nolint:errcheck
				}
			}
		}
	}

	if err := zw.Close(); err != nil {
		http.Error(w, "zip error", http.StatusInternalServerError)
		return
	}

	filename := sanitizeFilename(note.Title) + ".zip"
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", buf.Len()))
	w.Write(buf.Bytes()) //nolint:errcheck
}
