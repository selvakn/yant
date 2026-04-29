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

	queryEmb := make([]float32, 384)

	results, err := models.SemanticSearch(db, notesDir, u.ID, "golang", queryEmb, false, 0.3, 20)
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	found := false
	for _, r := range results {
		if r.Note.Slug == "golang-tutorial" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected text search to find 'Golang Tutorial'")
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
	writeTestNote(t, notesDir, u.ID, "vector-note", "This note has a vector embedding that is meaningful and long enough to avoid any penalties from content length adjustment")

	models.CreateNote(db, u.ID, "Text Note", "text-note")
	writeTestNote(t, notesDir, u.ID, "text-note", "This note vector embedding search")

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

func TestSemanticSearch_ShortContentPenalized(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	notesDir := t.TempDir()

	// Short-content note with embedding that technically "matches" the query
	shortNote, _ := models.CreateNote(db, u.ID, "Untitled", "short-note")
	writeTestNote(t, notesDir, u.ID, "short-note", "")

	// Longer note that has the actual query word in its body
	models.CreateNote(db, u.ID, "Research", "long-note")
	writeTestNote(t, notesDir, u.ID, "long-note", "This is a detailed note about linking notes together and cross-referencing between different topics")

	// Give the short note an embedding that is identical to the query vector
	// (would normally rank first without the length penalty)
	queryEmb := make([]float32, 384)
	queryEmb[0] = 1.0
	shortEmb := make([]float32, 384)
	shortEmb[0] = 1.0
	_ = models.UpsertEmbedding(db, shortNote.ID, shortEmb, "hash-short")

	results, err := models.SemanticSearch(db, notesDir, u.ID, "linking", queryEmb, false, 0.1, 20)
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}

	// The long note with actual text match should rank above the short note
	// despite the short note having a perfect vector match
	if results[0].Note.Slug != "long-note" {
		t.Errorf("expected long-note to rank first (text match beats penalized short-content vector match), got %s (score=%d)", results[0].Note.Slug, results[0].Score)
		for i, r := range results {
			t.Logf("  result[%d]: slug=%s score=%d", i, r.Note.Slug, r.Score)
		}
	}
}

func TestSemanticSearch_ExactMatchBoostsTextResult(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	notesDir := t.TempDir()

	// Note with short title, no body, and an embedding that passes the
	// semantic threshold (simulates the hub-vector problem)
	hubNote, _ := models.CreateNote(db, u.ID, "Short Title", "hub-note")
	writeTestNote(t, notesDir, u.ID, "hub-note", "")

	// Note with the query word literally in its body (no embedding)
	models.CreateNote(db, u.ID, "Dev Notes", "exact-note")
	writeTestNote(t, notesDir, u.ID, "exact-note", "Detailed notes on deployment steps, rollback procedures and monitoring setup for the production cluster")

	queryEmb := make([]float32, 384)
	queryEmb[0] = 1.0
	hubEmb := make([]float32, 384)
	hubEmb[0] = 1.0
	_ = models.UpsertEmbedding(db, hubNote.ID, hubEmb, "hash-hub")

	results, err := models.SemanticSearch(db, notesDir, u.ID, "deployment", queryEmb, false, 0.1, 20)
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}

	// exact-note has "deployment" in its body → exact-match bonus should
	// push it above hub-note whose semantic score is penalized by short content.
	if results[0].Note.Slug != "exact-note" {
		t.Errorf("expected exact-note first (exact text match beats penalized hub vector), got %s", results[0].Note.Slug)
		for i, r := range results {
			t.Logf("  result[%d]: slug=%s score=%d", i, r.Note.Slug, r.Score)
		}
	}
}

func TestSemanticSearch_MergesWithoutDuplicates(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	notesDir := t.TempDir()

	// Note that matches both semantically and via text
	note, _ := models.CreateNote(db, u.ID, "Vector Search Guide", "dual-note")
	writeTestNote(t, notesDir, u.ID, "dual-note", "A comprehensive guide to vector search and embedding-based retrieval systems for semantic matching")

	queryEmb := make([]float32, 384)
	queryEmb[0] = 1.0
	noteEmb := make([]float32, 384)
	noteEmb[0] = 1.0
	_ = models.UpsertEmbedding(db, note.ID, noteEmb, "hash-dual")

	results, err := models.SemanticSearch(db, notesDir, u.ID, "vector", queryEmb, false, 0.1, 20)
	if err != nil {
		t.Fatalf("search: %v", err)
	}

	count := 0
	for _, r := range results {
		if r.Note.Slug == "dual-note" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected dual-note exactly once, found %d times", count)
	}
}
