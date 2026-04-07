package models_test

import (
	"testing"

	"github.com/selvakn/yant/internal/models"
	"github.com/selvakn/yant/internal/storage"
)

func writeTestNote(t *testing.T, dir string, userID int64, slug, body string) {
	t.Helper()
	if err := storage.WriteNote(dir, userID, slug, body); err != nil {
		t.Fatalf("write note %s: %v", slug, err)
	}
}

func TestSemanticSearch_WithoutEmbeddings_FallsBackToText(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	notesDir := t.TempDir()

	models.CreateNote(db, u.ID, "Golang Tutorial", "golang-tutorial")
	writeTestNote(t, notesDir, u.ID, "golang-tutorial", "Learn Go programming #golang")

	models.CreateNote(db, u.ID, "Python Guide", "python-guide")
	writeTestNote(t, notesDir, u.ID, "python-guide", "Python programming #python")

	// Query embedding is all zeros (no real semantic match possible)
	queryEmb := make([]float32, 384)

	results, err := models.SemanticSearch(db, notesDir, u.ID, "golang", queryEmb, false, 0.3, 20)
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	// Should fall back to text search and find "Golang Tutorial"
	found := false
	for _, r := range results {
		if r.Note.Slug == "golang-tutorial" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected text fallback to find 'Golang Tutorial'")
	}
}

func TestSemanticSearch_EmptyQuery_ReturnsAllViaText(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	notesDir := t.TempDir()

	models.CreateNote(db, u.ID, "Note One", "note-one")
	writeTestNote(t, notesDir, u.ID, "note-one", "First note")

	models.CreateNote(db, u.ID, "Note Two", "note-two")
	writeTestNote(t, notesDir, u.ID, "note-two", "Second note")

	results, err := models.SearchNotes(db, notesDir, u.ID, "", false)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestSemanticSearch_RespectsMaxResults(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	notesDir := t.TempDir()

	for i := 0; i < 5; i++ {
		slug := "note-" + string(rune('a'+i))
		title := "Note " + string(rune('A'+i))
		models.CreateNote(db, u.ID, title, slug)
		writeTestNote(t, notesDir, u.ID, slug, "test content #test")
	}

	queryEmb := make([]float32, 384)
	results, err := models.SemanticSearch(db, notesDir, u.ID, "test", queryEmb, false, 0.0, 2)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) > 2 {
		t.Errorf("expected at most 2 results, got %d", len(results))
	}
}

func TestSemanticSearch_VectorMatchesReturnedFirst(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	notesDir := t.TempDir()

	n1, _ := models.CreateNote(db, u.ID, "Vector Note", "vector-note")
	writeTestNote(t, notesDir, u.ID, "vector-note", "This note has a vector embedding")

	models.CreateNote(db, u.ID, "Text Note", "text-note")
	writeTestNote(t, notesDir, u.ID, "text-note", "This note vector embedding search")

	// Give n1 an embedding that matches the query exactly
	queryEmb := make([]float32, 384)
	queryEmb[0] = 1.0
	noteEmb := make([]float32, 384)
	noteEmb[0] = 1.0
	_ = models.UpsertEmbedding(db, n1.ID, noteEmb, "hash1")

	results, err := models.SemanticSearch(db, notesDir, u.ID, "vector", queryEmb, false, 0.1, 20)
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
	if results[0].Note.Slug != "vector-note" {
		t.Errorf("expected vector-note first, got %s", results[0].Note.Slug)
	}
}

func TestSemanticSearch_ArchiveFilterWorks(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	notesDir := t.TempDir()

	models.CreateNote(db, u.ID, "Active Note", "active-note")
	writeTestNote(t, notesDir, u.ID, "active-note", "active content")

	models.CreateNote(db, u.ID, "Archived Note", "archived-note")
	writeTestNote(t, notesDir, u.ID, "archived-note", "archived content")
	models.ArchiveNote(db, u.ID, "archived-note")

	queryEmb := make([]float32, 384)

	activeResults, _ := models.SemanticSearch(db, notesDir, u.ID, "content", queryEmb, false, 0.0, 20)
	archivedResults, _ := models.SemanticSearch(db, notesDir, u.ID, "content", queryEmb, true, 0.0, 20)

	activeCount := 0
	for _, r := range activeResults {
		if r.Note.Slug == "active-note" {
			activeCount++
		}
	}
	archivedCount := 0
	for _, r := range archivedResults {
		if r.Note.Slug == "archived-note" {
			archivedCount++
		}
	}

	if activeCount == 0 {
		t.Error("expected active note in non-archived search")
	}
	if archivedCount == 0 {
		t.Error("expected archived note in archived search")
	}
}
