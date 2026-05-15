package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/selvakn/yant/internal/auth"
	"github.com/selvakn/yant/internal/embedding"
	"github.com/selvakn/yant/internal/handlers"
	"github.com/selvakn/yant/internal/models"
	"github.com/selvakn/yant/internal/storage"
	"github.com/selvakn/yant/internal/versioning"
)

func main() {
	loadDotenv()

	addr := flag.String("addr", ":8080", "listen address")
	dbPath := flag.String("db", "notes.db", "SQLite database path")
	notesDir := flag.String("notes", "notes", "markdown storage root")
	uploadsDir := flag.String("uploads", "uploads", "image storage root")
	rebuildDB := flag.Bool("rebuild-db", false, "rebuild SQLite from markdown files and exit")
	ghClientID := flag.String("github-client-id", envOrDefault("GITHUB_CLIENT_ID", ""), "GitHub OAuth client ID")
	ghClientSecret := flag.String("github-client-secret", envOrDefault("GITHUB_CLIENT_SECRET", ""), "GitHub OAuth client secret")
	baseURL := flag.String("base-url", envOrDefault("BASE_URL", ""), "external base URL for OAuth callbacks (e.g. https://notes.example.com)")
	semanticSearch := flag.Bool("semantic-search", envOrDefault("SEMANTIC_SEARCH", "true") == "true", "enable semantic search (default: true)")
	searchDebounceMS := flag.Int("search-debounce", envOrDefaultInt("SEARCH_DEBOUNCE_MS", 300), "search debounce delay in milliseconds")
	adminUser := flag.String("admin-user", envOrDefault("ADMIN_USER", ""), "GitHub username of the initial admin user")
	modelParamPath := flag.String("model-param", envOrDefault("MODEL_PATH", "/data/models/model.ncnn.param"), "path to ncnn model .param file")
	modelBinPath := flag.String("model-bin", envOrDefault("MODEL_BIN_PATH", "/data/models/model.ncnn.bin"), "path to ncnn model .bin file")
	tokenizerPath := flag.String("tokenizer-path", envOrDefault("TOKENIZER_PATH", "/data/models/tokenizer.json"), "path to tokenizer.json")
	tldrawLicenseKey := flag.String("tldraw-license-key", envOrDefault("TLDRAW_LICENSE_KEY", ""), "tldraw SDK license key (env: TLDRAW_LICENSE_KEY)")
	blogName := flag.String("blog-name", envOrDefault("BLOG_NAME", "Blog"), "public blog title (env: BLOG_NAME)")
	blogDomain := flag.String("blog-domain", envOrDefault("BLOG_DOMAIN", ""), "custom domain for blog (env: BLOG_DOMAIN)")
	giscusRepo := flag.String("giscus-repo", envOrDefault("GISCUS_REPO", ""), "GitHub repo for giscus comments (env: GISCUS_REPO)")
	giscusRepoID := flag.String("giscus-repo-id", envOrDefault("GISCUS_REPO_ID", ""), "GitHub repo ID for giscus (env: GISCUS_REPO_ID)")
	giscusCategory := flag.String("giscus-category", envOrDefault("GISCUS_CATEGORY", ""), "discussion category for giscus (env: GISCUS_CATEGORY)")
	giscusCategoryID := flag.String("giscus-category-id", envOrDefault("GISCUS_CATEGORY_ID", ""), "discussion category ID for giscus (env: GISCUS_CATEGORY_ID)")
	linkedinURL := flag.String("linkedin-url", envOrDefault("LINKEDIN_URL", ""), "author LinkedIn profile URL (env: LINKEDIN_URL)")
	flag.Parse()

	// Ensure data directories exist (required for distroless images with no shell)
	for _, dir := range []string{filepath.Dir(*dbPath), *notesDir, *uploadsDir} {
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				log.Fatalf("create directory %s: %v", dir, err)
			}
		}
	}

	if err := versioning.Init(*notesDir); err != nil {
		log.Printf("WARNING: Version control not available: %v", err)
	} else {
		log.Println("Version control initialized for notes directory")
	}

	frontendDir := resolveFrontend()

	db, err := models.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	if err := models.InitSchema(db); err != nil {
		log.Fatalf("init schema: %v", err)
	}

	if *adminUser != "" {
		promoted, err := models.BootstrapAdmin(db, *adminUser)
		if err != nil {
			log.Printf("WARNING: Failed to bootstrap admin user %q: %v", *adminUser, err)
		} else if promoted {
			log.Printf("Admin user %q activated", *adminUser)
		} else {
			log.Printf("Admin user %q not found in DB yet — will be promoted on first login", *adminUser)
		}
	}

	// Persist sessions across server restarts (30-day lifetime).
	auth.ConfigureSessionStore(db.DB)

	if *rebuildDB {
		if err := models.RebuildDB(db, *notesDir, *uploadsDir); err != nil {
			log.Fatalf("rebuild db: %v", err)
		}
		log.Println("Database rebuilt successfully")
		os.Exit(0)
	}

	var github *auth.GitHubOAuth
	if *ghClientID != "" && *ghClientSecret != "" {
		github = &auth.GitHubOAuth{
			ClientID:     *ghClientID,
			ClientSecret: *ghClientSecret,
			BaseURL:      *baseURL,
		}
		log.Println("GitHub OAuth enabled")
	} else {
		log.Println("WARNING: GitHub OAuth not configured (set GITHUB_CLIENT_ID and GITHUB_CLIENT_SECRET)")
	}

	var giscus *handlers.GiscusConfig
	if *giscusRepo != "" {
		giscus = &handlers.GiscusConfig{
			Repo:       *giscusRepo,
			RepoID:     *giscusRepoID,
			Category:   *giscusCategory,
			CategoryID: *giscusCategoryID,
		}
		log.Printf("Giscus comments enabled for %s", *giscusRepo)
	}

	tmplDir := filepath.Join(frontendDir, "templates")
	h := handlers.New(db, tmplDir, *notesDir, *uploadsDir, github, nil, *semanticSearch, *searchDebounceMS, *adminUser, *tldrawLicenseKey, *blogName, *blogDomain, *linkedinURL, giscus)

	if *semanticSearch {
		go initEmbedder(h, db, *notesDir, *modelParamPath, *modelBinPath, *tokenizerPath)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	if *blogDomain != "" {
		r.Use(handlers.BlogDomainMiddleware(*blogDomain))
		log.Printf("Blog domain routing enabled for %s", *blogDomain)
	}
	r.Use(auth.SessionManager.LoadAndSave)

	// Static files
	staticDir := filepath.Join(frontendDir, "static")
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	// Auth routes (public)
	r.Get("/login", h.LoginGET)
	r.Get("/auth/github", h.GitHubLoginGET)
	r.Get("/auth/github/callback", h.GitHubCallbackGET)
	r.Post("/logout", h.LogoutPOST)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/notes", http.StatusFound)
	})

	// Public note sharing (no auth required)
	r.Get("/p/{token}", h.PublicNoteGET)
	r.Get("/p/{token}/uploads/{filename}", h.PublicImageServeGET)
	r.Get("/p/{token}/drawing", h.PublicDrawingGET)
	r.Get("/p/{token}/drawings", h.PublicDrawingsListGET)
	r.Get("/p/{token}/drawings/{drawingID}", h.PublicDrawingByIDGET)
	r.Get("/p/{token}/drawings/{drawingID}/svg", h.PublicDrawingSVGGET)

	r.Get("/blog", h.BlogIndexGET)
	r.Get("/blog/tag/{tag}", h.BlogTagGET)
	r.Get("/blog/{slug}/drawings/{drawingID}/svg", h.BlogDrawingSVGGET)
	r.Get("/blog/{slug}", h.BlogPostGET)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireLogin)
		r.Use(auth.RequireActive(func(userID int64) bool {
			return models.IsUserDisabled(db, userID)
		}))

		r.Get("/notes", h.NotesListGET)
		r.Get("/notes/search", h.NotesSearchGET)
		r.Get("/notes/autocomplete", h.NotesAutocompleteGET)
		r.Post("/notes", h.NotesCreatePOST)
		r.Get("/notes/{slug}", h.NoteReaderGET)
		r.Get("/notes/{slug}/edit", h.NoteEditorGET)
		r.Post("/notes/{slug}", h.NoteUpdateOrDelete) // X-HTTP-Method-Override dispatch
		r.Put("/notes/{slug}/archive", h.NotesArchivePUT)
		r.Put("/notes/{slug}/restore", h.NotesRestorePUT)
		r.Post("/notes/{slug}/images", h.ImageUploadPOST)
		r.Get("/notes/{slug}/export", h.NoteExportZIP)
		r.Get("/notes/{slug}/drawing", h.DrawingGET)
		r.Put("/notes/{slug}/drawing", h.DrawingPUT)
		r.Delete("/notes/{slug}/drawing", h.DrawingDELETE)

		// Multi-drawing routes
		r.Get("/notes/{slug}/drawings", h.DrawingsListGET)
		r.Post("/notes/{slug}/drawings", h.DrawingsCreatePOST)
		r.Get("/notes/{slug}/drawings/{drawingID}", h.DrawingByIDGET)
		r.Put("/notes/{slug}/drawings/{drawingID}", h.DrawingByIDPUT)
		r.Patch("/notes/{slug}/drawings/{drawingID}", h.DrawingByIDRenamePATCH)
		r.Delete("/notes/{slug}/drawings/{drawingID}", h.DrawingByIDDELETE)
		r.Get("/notes/{slug}/drawings/{drawingID}/svg", h.DrawingSVGGET)
		r.Put("/notes/{slug}/drawings/{drawingID}/svg", h.DrawingSVGPUT)

		r.Get("/notes/{slug}/history", h.NoteHistoryGET)
		r.Get("/notes/{slug}/history/{commit}", h.NoteVersionGET)
		r.Get("/notes/{slug}/history/{commit}/diff", h.NoteVersionDiffGET)
		r.Get("/notes/{slug}/history/{commit}/drawings", h.NoteVersionDrawingsListGET)
		r.Get("/notes/{slug}/history/{commit}/drawings/{drawingID}", h.NoteVersionDrawingByIDGET)
		r.Get("/notes/{slug}/history/{commit}/drawing", h.NoteVersionDrawingGET)
		r.Post("/notes/{slug}/history/{commit}/revert", h.NoteVersionRevertPOST)

		r.Get("/todos", h.TodosListGET)
		r.Put("/notes/{slug}/todo", h.TodoTogglePUT)

		r.Put("/notes/{slug}/publish", h.PublishPUT)
		r.Put("/notes/{slug}/unpublish", h.UnpublishPUT)
		r.Get("/public", h.PublicNotesListGET)

		r.Put("/notes/{slug}/share", h.ShareCreatePUT)
		r.Delete("/notes/{slug}/share/{username}", h.ShareDeletePUT)
		r.Get("/notes/{slug}/shares", h.ShareListGET)

		r.Get("/shared", h.SharedNotesListGET)
		r.Get("/shared/{username}/{slug}", h.SharedNoteReaderGET)
		r.Get("/shared/{username}/{slug}/drawing", h.SharedDrawingGET)
		r.Get("/shared/{username}/{slug}/drawings", h.SharedDrawingsListGET)
		r.Post("/shared/{username}/{slug}/drawings", h.SharedDrawingsCreatePOST)
		r.Get("/shared/{username}/{slug}/drawings/{drawingID}", h.SharedDrawingByIDGET)
		r.Put("/shared/{username}/{slug}/drawings/{drawingID}", h.SharedDrawingByIDPUT)
		r.Patch("/shared/{username}/{slug}/drawings/{drawingID}", h.SharedDrawingByIDRenamePATCH)
		r.Delete("/shared/{username}/{slug}/drawings/{drawingID}", h.SharedDrawingByIDDELETE)
		r.Get("/shared/{username}/{slug}/drawings/{drawingID}/svg", h.SharedDrawingSVGGET)
		r.Put("/shared/{username}/{slug}/drawings/{drawingID}/svg", h.SharedDrawingSVGPUT)
		r.Get("/shared/{username}/{slug}/history", h.SharedNoteHistoryGET)
		r.Get("/shared/{username}/{slug}/edit", h.SharedNoteEditorGET)
		r.Post("/shared/{username}/{slug}", h.SharedNoteUpdate)

		r.Get("/tags", h.TagsListGET)
		r.Put("/tags/{name}/color", h.TagColorPUT)
		r.Get("/uploads/{username}/{filename}", h.ImageServeGET)

		r.Get("/archive", h.ArchiveListGET)
		r.Get("/archive/search", h.ArchiveSearchGET)
		r.Get("/archive/tags", h.ArchiveTagsGET)

		h.RegisterAdminRoutes(r)
	})

	// Custom 404
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		h.RenderError(w, r, http.StatusNotFound, "Page not found")
	})

	log.Printf("Listening on %s", *addr)
	if err := http.ListenAndServe(*addr, r); err != nil {
		log.Fatal(err)
	}
}

// loadDotenv reads a .env file from the working directory or repo root and
// sets any variables not already present in the environment. Missing file is
// silently ignored.
func loadDotenv() {
	candidates := []string{".env", "../.env"}
	for _, path := range candidates {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			k, v, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			k = strings.TrimSpace(k)
			v = strings.TrimSpace(v)
			v = strings.Trim(v, `"'`)
			if _, exists := os.LookupEnv(k); !exists {
				os.Setenv(k, v)
			}
		}
		return
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrDefaultInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			return n
		}
	}
	return fallback
}

const (
	// ncnn model files are published by the convert-model GitHub Actions workflow
	// as assets on the "model-v2" GitHub Release in this repository.
	modelParamDownloadURL = "https://github.com/selvakn/yant/releases/download/model-v2/model.ncnn.param"
	modelBinDownloadURL   = "https://github.com/selvakn/yant/releases/download/model-v2/model.ncnn.bin"
	tokenizerDownloadURL  = "https://huggingface.co/optimum/all-MiniLM-L6-v2/resolve/main/tokenizer.json"
)

// initEmbedder downloads model files if missing, initialises the ncnn embedder,
// then hot-swaps it into the handler so semantic search becomes available without restarting.
func initEmbedder(h *handlers.Handler, db *models.DB, notesDir, paramPath, binPath, tokenizerPath string) {
	for _, dl := range []struct{ path, url string }{
		{paramPath, modelParamDownloadURL},
		{binPath, modelBinDownloadURL},
		{tokenizerPath, tokenizerDownloadURL},
	} {
		if err := downloadFile(dl.path, dl.url); err != nil {
			log.Printf("WARNING: Failed to download %s: %v — semantic search unavailable", filepath.Base(dl.path), err)
			return
		}
	}

	emb, err := embedding.New(paramPath, binPath, tokenizerPath)
	if err != nil {
		log.Printf("WARNING: Embedding model not available: %v — semantic search unavailable", err)
		// Delete cached model files so the next restart re-downloads fresh ones.
		// This handles the case where cached files are stale or incompatible.
		for _, p := range []string{paramPath, binPath} {
			if rmErr := os.Remove(p); rmErr == nil {
				log.Printf("Deleted stale model file %s; will re-download on next restart", filepath.Base(p))
			}
		}
		return
	}
	log.Println("Embedding model loaded, semantic search enabled")
	h.SetEmbedder(emb)
	go backfillEmbeddings(db, notesDir, emb)
}

// downloadFile downloads url to dest if dest does not already exist.
// Downloads to a .tmp file first then renames atomically to avoid partial files.
func downloadFile(dest, url string) error {
	if _, err := os.Stat(dest); err == nil {
		return nil // already cached
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	log.Printf("Downloading %s ...", filepath.Base(dest))
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, url)
	}
	tmp := dest + ".download"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	n, copyErr := io.Copy(f, resp.Body)
	f.Close()
	if copyErr != nil {
		os.Remove(tmp)
		return copyErr
	}
	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		return err
	}
	log.Printf("Downloaded %s (%.1f MB)", filepath.Base(dest), float64(n)/(1024*1024))
	return nil
}

func backfillEmbeddings(db *models.DB, notesDir string, emb *embedding.Embedder) {
	notes, err := models.NotesWithoutEmbeddings(db)
	if err != nil {
		log.Printf("backfill: failed to query notes: %v", err)
		return
	}
	if len(notes) == 0 {
		return
	}
	log.Printf("backfill: generating embeddings for %d notes", len(notes))
	for i, n := range notes {
		body, _ := storage.ReadNote(notesDir, n.UserID, n.Slug)
		text := models.PrepareEmbeddingText(n.Title, body)
		hash := models.ContentHash(n.Title, body)
		vec, err := emb.Embed(text)
		if err != nil {
			log.Printf("backfill: failed to embed note %d (%s): %v", n.ID, n.Slug, err)
			continue
		}
		if err := models.UpsertEmbedding(db, n.ID, vec, hash); err != nil {
			log.Printf("backfill: failed to store embedding for note %d: %v", n.ID, err)
			continue
		}
		if (i+1)%10 == 0 || i+1 == len(notes) {
			log.Printf("backfill: %d/%d notes processed", i+1, len(notes))
		}
	}
	log.Println("backfill: complete")
}

func resolveFrontend() string {
	// Try ../../frontend relative to binary location, fall back to ../frontend
	candidates := []string{
		"../frontend",
		"../../frontend",
		"frontend",
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "templates")); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	log.Fatal("Cannot locate frontend/ directory. Run from backend/ or project root.")
	return ""
}
