package handlers

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
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

	versions, err := versioning.Log(h.notesDir, relPath, perPage+1, (page-1)*perPage)
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

	drawingRelPath := fmt.Sprintf("%d/%s.tldraw.json", userID, slug)
	hasDrawing := versioning.FileExistsAtCommit(h.notesDir, drawingRelPath, commit)

	data := map[string]any{
		"Note":         note,
		"Version":      v,
		"BodyHTML":     template.HTML(buf.String()), //nolint:gosec
		"HasDrawing":   hasDrawing,
		"IsHistorical": true,
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

	drawingRelPath := fmt.Sprintf("%d/%s.tldraw.json", userID, slug)
	oldHasDrawing := versioning.FileExistsAtCommit(h.notesDir, drawingRelPath, against)
	newHasDrawing := versioning.FileExistsAtCommit(h.notesDir, drawingRelPath, commit)

	diff := versioning.DiffResult{
		OldCommit:        against,
		NewCommit:        commit,
		OldDate:          oldVersion.Timestamp,
		NewDate:          newVersion.Timestamp,
		Lines:            versioning.ParseDiff(rawDiff),
		HasDrawingChange: oldHasDrawing || newHasDrawing,
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

	drawingRelPath := fmt.Sprintf("%d/%s.tldraw.json", userID, slug)
	content, err := versioning.Show(h.notesDir, drawingRelPath, commit)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(content)) //nolint:errcheck
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

	drawingRelPath := fmt.Sprintf("%d/%s.tldraw.json", userID, slug)
	if versioning.FileExistsAtCommit(h.notesDir, drawingRelPath, commit) {
		drawingContent, err := versioning.Show(h.notesDir, drawingRelPath, commit)
		if err == nil {
			_ = storage.WriteDrawing(h.notesDir, userID, slug, []byte(drawingContent))
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

