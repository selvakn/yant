package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/selvakn/yant/internal/models"
	"github.com/selvakn/yant/internal/storage"
)

const maxImageSize    = 1 << 20 // 1 MB per image
const maxImagesPerNote = 10     // lifetime upload count per note

var allowedMIME = map[string]string{
	"image/png":  "png",
	"image/jpeg": "jpg",
	"image/gif":  "gif",
	"image/webp": "webp",
}

// ImageUploadPOST handles multipart image uploads from EasyMDE.
func (h *Handler) ImageUploadPOST(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromSession(r)
	username := usernameFromSession(r)
	slug := chi.URLParam(r, "slug")

	// Enforce max size before parsing
	r.Body = http.MaxBytesReader(w, r.Body, maxImageSize+1024)
	if err := r.ParseMultipartForm(maxImageSize); err != nil {
		if strings.Contains(err.Error(), "request body too large") || strings.Contains(err.Error(), "too large") {
			http.Error(w, `{"error":"file too large"}`, http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, `{"error":"missing image field"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Enforce per-file size limit precisely (MaxBytesReader above only catches extreme overages)
	if header.Size > maxImageSize {
		http.Error(w, `{"error":"file too large"}`, http.StatusRequestEntityTooLarge)
		return
	}

	// Detect MIME type
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	mimeType := http.DetectContentType(buf[:n])
	ext, ok := allowedMIME[mimeType]
	if !ok {
		http.Error(w, `{"error":"unsupported file type"}`, http.StatusBadRequest)
		return
	}
	// Seek back to start
	if seeker, ok := file.(io.Seeker); ok {
		seeker.Seek(0, io.SeekStart) //nolint:errcheck
	}

	// Check note ownership
	note, err := models.GetNote(h.db, userID, slug)
	if err != nil || note == nil {
		http.NotFound(w, r)
		return
	}

	// Enforce lifetime image count limit per note
	imgCount, err := models.CountImagesForNote(h.db, note.ID)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	if imgCount >= maxImagesPerNote {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprintf(w, `{"error":"This note has reached the maximum of %d images."}`, maxImagesPerNote) //nolint:errcheck
		return
	}

	// Save file
	filename := uuid.New().String() + "." + ext
	if err := storage.EnsureUploadsDir(h.uploadsDir, userID); err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	dest := storage.UploadPath(h.uploadsDir, userID, filename)
	out, err := os.Create(dest)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	defer out.Close()
	written, err := io.Copy(out, file)
	if err != nil {
		http.Error(w, `{"error":"write error"}`, http.StatusInternalServerError)
		return
	}

	// Record in DB
	if _, err := models.CreateImage(h.db, note.ID, filename, header.Filename, mimeType, written); err != nil {
		http.Error(w, `{"error":"db error"}`, http.StatusInternalServerError)
		return
	}

	url := fmt.Sprintf("/uploads/%s/%s", username, filename)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": url}) //nolint:errcheck
}

// ImageServeGET serves an uploaded image, enforcing ownership.
func (h *Handler) ImageServeGET(w http.ResponseWriter, r *http.Request) {
	username := usernameFromSession(r)
	targetUser := chi.URLParam(r, "username")
	filename := chi.URLParam(r, "filename")

	if username != targetUser {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Prevent path traversal
	if strings.Contains(filename, "/") || strings.Contains(filename, "..") {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Find user ID from username
	user, err := models.GetUserByUsername(h.db, username)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	path := storage.UploadPath(h.uploadsDir, user.ID, filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	// Detect content type from extension
	ext := strings.ToLower(filepath.Ext(filename))
	ct := map[string]string{
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
		".webp": "image/webp",
	}[ext]
	if ct == "" {
		ct = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ct)
	http.ServeFile(w, r, path)
}
