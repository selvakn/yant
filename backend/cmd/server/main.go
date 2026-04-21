package main

import (
	"bufio"
	"flag"
	"fmt"
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
	onnxLibPath := flag.String("onnx-lib", envOrDefault("ONNXRUNTIME_LIB_PATH", ""), "path to libonnxruntime.so (empty = default search)")
	modelPath := flag.String("model-path", envOrDefault("MODEL_PATH", "models/model.onnx"), "path to ONNX model file")
	tokenizerPath := flag.String("tokenizer-path", envOrDefault("TOKENIZER_PATH", "models/tokenizer.json"), "path to tokenizer.json")
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

	// Initialize embedding model
	var emb *embedding.Embedder
	emb, err = embedding.New(*onnxLibPath, *modelPath, *tokenizerPath)
	if err != nil {
		log.Printf("WARNING: Embedding model not available: %v", err)
		log.Println("Semantic search will be disabled; text-based search will be used.")
	} else {
		log.Println("Embedding model loaded successfully")
		// Backfill embeddings for notes that don't have them
		go backfillEmbeddings(db, *notesDir, emb)
	}

	tmplDir := filepath.Join(frontendDir, "templates")
	h := handlers.New(db, tmplDir, *notesDir, *uploadsDir, github, emb, *semanticSearch, *searchDebounceMS)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
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

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireLogin)

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
		r.Get("/notes/{slug}/drawing", h.DrawingGET)
		r.Put("/notes/{slug}/drawing", h.DrawingPUT)
		r.Delete("/notes/{slug}/drawing", h.DrawingDELETE)

		r.Get("/notes/{slug}/history", h.NoteHistoryGET)
		r.Get("/notes/{slug}/history/{commit}", h.NoteVersionGET)
		r.Get("/notes/{slug}/history/{commit}/diff", h.NoteVersionDiffGET)
		r.Get("/notes/{slug}/history/{commit}/drawing", h.NoteVersionDrawingGET)
		r.Post("/notes/{slug}/history/{commit}/revert", h.NoteVersionRevertPOST)

		r.Get("/todos", h.TodosListGET)
		r.Put("/notes/{slug}/todo", h.TodoTogglePUT)

		r.Put("/notes/{slug}/publish", h.PublishPUT)
		r.Put("/notes/{slug}/unpublish", h.UnpublishPUT)
		r.Get("/public", h.PublicNotesListGET)

		r.Get("/tags", h.TagsListGET)
		r.Put("/tags/{name}/color", h.TagColorPUT)
		r.Get("/uploads/{username}/{filename}", h.ImageServeGET)

		r.Get("/archive", h.ArchiveListGET)
		r.Get("/archive/search", h.ArchiveSearchGET)
		r.Get("/archive/tags", h.ArchiveTagsGET)
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

