package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/selvakn/yant/internal/models"
	"github.com/selvakn/yant/internal/storage"
	"github.com/selvakn/yant/internal/versioning"
)

const defaultPerPage = 20
const maxPerPage = 100

func (h *Handler) NoteHistoryGET(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")

	note, err := models.GetNote(h.db, userID, slug)
	if err != nil || note == nil {
		http.NotFound(w, r)
		return
	}

	page, perPage := parsePagination(r)
	relPath := fmt.Sprintf("%d/%s.md", userID, slug)
	tldrawRelPath := storage.DrawingRelPath(userID, slug, storage.DrawingTldraw)
	excalidrawRelPath := storage.DrawingRelPath(userID, slug, storage.DrawingExcalidraw)

	versions, err := versioning.Log(h.notesDir, relPath, perPage+1, (page-1)*perPage, tldrawRelPath, excalidrawRelPath)
	if err != nil {
		http.Error(w, "version history error", http.StatusInternalServerError)
		return
	}

	hasMore := len(versions) > perPage
	if hasMore {
		versions = versions[:perPage]
	}

	data := map[string]any{
		"Note":     note,
		"Versions": versions,
		"Page":     page,
		"PerPage":  perPage,
		"HasMore":  hasMore,
		"PrevPage": page - 1,
		"NextPage": page + 1,
	}
	h.render(w, r, "notes/history.html", data)
}

func (h *Handler) NoteVersionGET(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")
	commit := chi.URLParam(r, "commit")

	if !versioning.ValidCommitHash(commit) {
		http.Error(w, "invalid version identifier", http.StatusBadRequest)
		return
	}

	note, err := models.GetNote(h.db, userID, slug)
	if err != nil || note == nil {
		http.NotFound(w, r)
		return
	}

	relPath := fmt.Sprintf("%d/%s.md", userID, slug)
	content, err := versioning.Show(h.notesDir, relPath, commit)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	v, err := versioning.GetVersion(h.notesDir, commit)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var buf bytes.Buffer
	if err := h.md.Convert([]byte(content), &buf); err != nil {
		buf.WriteString("<p>Error rendering markdown</p>")
	}

	hasLegacyDrawing, legacyDT := detectDrawingAtCommit(h.notesDir, userID, slug, commit)
	legacyDrawingType := ""
	if hasLegacyDrawing {
		legacyDrawingType = string(legacyDT)
	}

	data := map[string]any{
		"Note":              note,
		"Version":           v,
		"BodyHTML":          template.HTML(buf.String()), //nolint:gosec
		"HasLegacyDrawing":  hasLegacyDrawing,
		"LegacyDrawingType": legacyDrawingType,
		"IsHistorical":      true,
	}
	h.render(w, r, "notes/version.html", data)
}

func (h *Handler) NoteVersionDiffGET(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")
	commit := chi.URLParam(r, "commit")

	if !versioning.ValidCommitHash(commit) {
		http.Error(w, "invalid version identifier", http.StatusBadRequest)
		return
	}

	note, err := models.GetNote(h.db, userID, slug)
	if err != nil || note == nil {
		http.NotFound(w, r)
		return
	}

	against := r.URL.Query().Get("against")
	if against == "" {
		parent, err := versioning.ParentCommit(h.notesDir, commit)
		if err != nil {
			http.Error(w, "no previous version to compare against", http.StatusBadRequest)
			return
		}
		against = parent
	} else if !versioning.ValidCommitHash(against) {
		http.Error(w, "invalid comparison version", http.StatusBadRequest)
		return
	}

	relPath := fmt.Sprintf("%d/%s.md", userID, slug)
	rawDiff, err := versioning.Diff(h.notesDir, relPath, against, commit)
	if err != nil {
		http.Error(w, "diff error", http.StatusInternalServerError)
		return
	}

	newVersion, err := versioning.GetVersion(h.notesDir, commit)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	oldVersion, err := versioning.GetVersion(h.notesDir, against)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	oldHasDrawing, oldDrawingType := detectDrawingAtCommit(h.notesDir, userID, slug, against)
	newHasDrawing, newDrawingType := detectDrawingAtCommit(h.notesDir, userID, slug, commit)

	diff := versioning.DiffResult{
		OldCommit:        against,
		NewCommit:        commit,
		OldDate:          oldVersion.Timestamp,
		NewDate:          newVersion.Timestamp,
		Lines:            versioning.ParseDiff(rawDiff),
		HasDrawingChange: oldHasDrawing || newHasDrawing,
		OldDrawingType:   string(oldDrawingType),
		NewDrawingType:   string(newDrawingType),
	}

	data := map[string]any{
		"Note":       note,
		"Diff":       diff,
		"OldVersion": oldVersion,
		"NewVersion": newVersion,
	}
	h.render(w, r, "notes/diff.html", data)
}

func (h *Handler) NoteVersionDrawingGET(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")
	commit := chi.URLParam(r, "commit")

	if !versioning.ValidCommitHash(commit) {
		http.Error(w, "invalid version identifier", http.StatusBadRequest)
		return
	}

	note, err := models.GetNote(h.db, userID, slug)
	if err != nil || note == nil {
		http.NotFound(w, r)
		return
	}

	_, dt := detectDrawingAtCommit(h.notesDir, userID, slug, commit)
	if dt == storage.DrawingNone {
		http.NotFound(w, r)
		return
	}

	relPath := storage.DrawingRelPath(userID, slug, dt)
	content, err := versioning.Show(h.notesDir, relPath, commit)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	writeDrawingResponse(w, []byte(content), dt)
}

// NoteVersionDrawingsListGET returns drawings referenced in markdown at a commit
// with files present in git at that revision.
// GET /notes/{slug}/history/{commit}/drawings
func (h *Handler) NoteVersionDrawingsListGET(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")
	commit := chi.URLParam(r, "commit")

	if !versioning.ValidCommitHash(commit) {
		http.Error(w, "invalid version identifier", http.StatusBadRequest)
		return
	}

	note, err := models.GetNote(h.db, userID, slug)
	if err != nil || note == nil {
		http.NotFound(w, r)
		return
	}

	relPath := fmt.Sprintf("%d/%s.md", userID, slug)
	mdContent, err := versioning.Show(h.notesDir, relPath, commit)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var ids []string
	seen := map[string]bool{}
	for _, m := range versionDrawMarkerRe.FindAllStringSubmatch(mdContent, -1) {
		if len(m) > 1 && !seen[m[1]] {
			seen[m[1]] = true
			ids = append(ids, m[1])
		}
	}

	type drawingJSON struct {
		DrawingID   string `json:"drawing_id"`
		DisplayName string `json:"display_name"`
		ToolType    string `json:"tool_type"`
	}
	var result []drawingJSON
	for _, id := range ids {
		_, dt, ok := drawingBytesAtCommit(h.notesDir, userID, slug, id, commit)
		if !ok {
			continue
		}
		displayName := "Drawing"
		if d, err := models.GetDrawing(h.db, note.ID, id); err == nil && d != nil {
			displayName = d.DisplayName
		}
		result = append(result, drawingJSON{
			DrawingID:   id,
			DisplayName: displayName,
			ToolType:    string(dt),
		})
	}
	if result == nil {
		result = []drawingJSON{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"drawings": result}) //nolint:errcheck
}

// NoteVersionDrawingByIDGET serves drawing JSON at a specific commit by drawing ID.
// GET /notes/{slug}/history/{commit}/drawings/{drawingID}
func (h *Handler) NoteVersionDrawingByIDGET(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")
	commit := chi.URLParam(r, "commit")
	drawingID := chi.URLParam(r, "drawingID")

	if !versioning.ValidCommitHash(commit) {
		http.Error(w, "invalid version identifier", http.StatusBadRequest)
		return
	}

	if n, err := models.GetNote(h.db, userID, slug); err != nil || n == nil {
		http.NotFound(w, r)
		return
	}

	data, dt, ok := drawingBytesAtCommit(h.notesDir, userID, slug, drawingID, commit)
	if !ok {
		http.NotFound(w, r)
		return
	}

	writeDrawingResponse(w, data, dt)
}

func (h *Handler) NoteVersionRevertPOST(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	slug := chi.URLParam(r, "slug")
	commit := chi.URLParam(r, "commit")

	if !versioning.ValidCommitHash(commit) {
		http.Error(w, "invalid version identifier", http.StatusBadRequest)
		return
	}

	note, err := models.GetNote(h.db, userID, slug)
	if err != nil || note == nil {
		http.NotFound(w, r)
		return
	}

	relPath := fmt.Sprintf("%d/%s.md", userID, slug)
	oldContent, err := versioning.Show(h.notesDir, relPath, commit)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	currentContent, _ := storage.ReadNote(h.notesDir, userID, slug)
	if currentContent == oldContent {
		w.Header().Set("HX-Redirect", fmt.Sprintf("/notes/%s", slug))
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := storage.WriteNote(h.notesDir, userID, slug, oldContent); err != nil {
		http.Error(w, "write error", http.StatusInternalServerError)
		return
	}

	v, _ := versioning.GetVersion(h.notesDir, commit)
	shortHash := commit[:8]
	if v != nil {
		shortHash = v.ShortHash
	}
	commitMsg := fmt.Sprintf("revert: %s to %s", slug, shortHash)
	if err := versioning.CommitFile(h.notesDir, relPath, commitMsg); err != nil {
		log.Printf("versioning: commit revert %s: %v", slug, err)
	}

	_, drawingDT := detectDrawingAtCommit(h.notesDir, userID, slug, commit)
	if drawingDT != storage.DrawingNone {
		drawingRelPath := storage.DrawingRelPath(userID, slug, drawingDT)
		drawingContent, err := versioning.Show(h.notesDir, drawingRelPath, commit)
		if err == nil {
			_ = storage.DeleteDrawing(h.notesDir, userID, slug)
			_ = storage.WriteDrawing(h.notesDir, userID, slug, drawingDT, []byte(drawingContent))
			_ = versioning.CommitFile(h.notesDir, drawingRelPath, "revert drawing: "+slug+" to "+shortHash)
		}
	}

	tags := models.ParseTags(oldContent)
	_ = models.SyncTags(h.db, note.ID, tags)
	linkedTitles := models.ParseNoteLinks(oldContent)
	_ = models.SyncLinks(h.db, note.ID, userID, linkedTitles)
	todos := models.ParseTodos(oldContent)
	_ = models.SyncTodos(h.db, note.ID, todos)
	h.generateEmbedding(note.ID, note.Title, oldContent)

	w.Header().Set("HX-Redirect", fmt.Sprintf("/notes/%s", slug))
	w.WriteHeader(http.StatusOK)
}

var versionDrawMarkerRe = regexp.MustCompile(`\!\[\[draw:([a-z0-9]+)\]\]`)

func drawingBytesAtCommit(notesDir string, userID int64, slug, drawingID, commit string) ([]byte, storage.DrawingType, bool) {
	for _, dt := range []storage.DrawingType{storage.DrawingExcalidraw, storage.DrawingTldraw} {
		rel := storage.DrawingRelPathByID(userID, slug, drawingID, dt)
		if versioning.FileExistsAtCommit(notesDir, rel, commit) {
			content, err := versioning.Show(notesDir, rel, commit)
			if err != nil {
				return nil, storage.DrawingNone, false
			}
			return []byte(content), dt, true
		}
	}
	return nil, storage.DrawingNone, false
}

func detectDrawingAtCommit(notesDir string, userID int64, slug, commit string) (bool, storage.DrawingType) {
	excalidrawRelPath := storage.DrawingRelPath(userID, slug, storage.DrawingExcalidraw)
	if versioning.FileExistsAtCommit(notesDir, excalidrawRelPath, commit) {
		return true, storage.DrawingExcalidraw
	}
	tldrawRelPath := storage.DrawingRelPath(userID, slug, storage.DrawingTldraw)
	if versioning.FileExistsAtCommit(notesDir, tldrawRelPath, commit) {
		return true, storage.DrawingTldraw
	}
	return false, storage.DrawingNone
}

func parsePagination(r *http.Request) (page, perPage int) {
	page = 1
	perPage = defaultPerPage

	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if pp := r.URL.Query().Get("per_page"); pp != "" {
		if v, err := strconv.Atoi(pp); err == nil && v > 0 && v <= maxPerPage {
			perPage = v
		}
	}
	return
}
