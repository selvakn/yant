package main

import (
	"bufio"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/selvakn/yant/internal/auth"
	"github.com/selvakn/yant/internal/handlers"
	"github.com/selvakn/yant/internal/models"
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
	flag.Parse()

	// Resolve template + static paths relative to the binary's working dir.
	// When running from backend/, frontend/ is at ../frontend/
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
		}
		log.Println("GitHub OAuth enabled")
	} else {
		log.Println("WARNING: GitHub OAuth not configured (set GITHUB_CLIENT_ID and GITHUB_CLIENT_SECRET)")
	}

	tmplDir := filepath.Join(frontendDir, "templates")
	h := handlers.New(db, tmplDir, *notesDir, *uploadsDir, github)

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

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireLogin)

		r.Get("/notes", h.NotesListGET)
		r.Get("/notes/search", h.NotesSearchGET)
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

