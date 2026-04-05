package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/selvakn/my-notes/internal/auth"
	"github.com/selvakn/my-notes/internal/handlers"
	"github.com/selvakn/my-notes/internal/models"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dbPath := flag.String("db", "notes.db", "SQLite database path")
	notesDir := flag.String("notes", "notes", "markdown storage root")
	uploadsDir := flag.String("uploads", "uploads", "image storage root")
	rebuildDB := flag.Bool("rebuild-db", false, "rebuild SQLite from markdown files and exit")
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

	tmplDir := filepath.Join(frontendDir, "templates")
	h := handlers.New(db, tmplDir, *notesDir, *uploadsDir)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(auth.SessionManager.LoadAndSave)

	// Static files
	staticDir := filepath.Join(frontendDir, "static")
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	// Auth routes (public)
	r.Get("/login", h.LoginGET)
	r.Post("/login", h.LoginPOST)
	r.Post("/logout", h.LogoutPOST)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/notes", http.StatusFound)
	})

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireLogin)

		r.Get("/notes", h.NotesListGET)
		r.Post("/notes", h.NotesCreatePOST)
		r.Get("/notes/{slug}", h.NoteReaderGET)
		r.Get("/notes/{slug}/edit", h.NoteEditorGET)
		r.Post("/notes/{slug}", h.NoteUpdateOrDelete) // X-HTTP-Method-Override dispatch
		r.Post("/notes/{slug}/images", h.ImageUploadPOST)
		r.Get("/notes/{slug}/drawing", h.DrawingGET)
		r.Put("/notes/{slug}/drawing", h.DrawingPUT)
		r.Delete("/notes/{slug}/drawing", h.DrawingDELETE)

		r.Get("/tags", h.TagsListGET)
		r.Get("/uploads/{username}/{filename}", h.ImageServeGET)
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

