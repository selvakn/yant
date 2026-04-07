package models_test

import (
	"testing"

	"github.com/selvakn/yant/internal/models"
)

func TestPrepareEmbeddingText_TitleAndBody(t *testing.T) {
	text := models.PrepareEmbeddingText("Hello World", "This is the body")
	want := "Hello World\n\nThis is the body"
	if text != want {
		t.Errorf("got %q, want %q", text, want)
	}
}

func TestPrepareEmbeddingText_TitleOnly(t *testing.T) {
	text := models.PrepareEmbeddingText("Hello World", "")
	if text != "Hello World" {
		t.Errorf("got %q", text)
	}
}

func TestPrepareEmbeddingText_Truncation(t *testing.T) {
	body := make([]byte, 10000)
	for i := range body {
		body[i] = 'a'
	}
	text := models.PrepareEmbeddingText("title", string(body))
	if len(text) > 8000 {
		t.Errorf("expected text truncated to 8000, got %d", len(text))
	}
}

func TestContentHash_DifferentForDifferentContent(t *testing.T) {
	h1 := models.ContentHash("title", "body1")
	h2 := models.ContentHash("title", "body2")
	if h1 == h2 {
		t.Error("expected different hashes for different content")
	}
}

func TestContentHash_SameForSameContent(t *testing.T) {
	h1 := models.ContentHash("title", "body")
	h2 := models.ContentHash("title", "body")
	if h1 != h2 {
		t.Error("expected same hash for same content")
	}
}

func TestNeedsEmbedding_TrueWhenNoEmbedding(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	n, _ := models.CreateNote(db, u.ID, "Test", "test")

	if !models.NeedsEmbedding(db, n.ID, "somehash") {
		t.Error("expected NeedsEmbedding=true for new note")
	}
}

func TestNeedsEmbedding_FalseWhenHashMatches(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	n, _ := models.CreateNote(db, u.ID, "Test", "test")

	emb := make([]float32, 384)
	if err := models.UpsertEmbedding(db, n.ID, emb, "myhash"); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if models.NeedsEmbedding(db, n.ID, "myhash") {
		t.Error("expected NeedsEmbedding=false when hash matches")
	}
}

func TestNeedsEmbedding_TrueWhenHashDiffers(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	n, _ := models.CreateNote(db, u.ID, "Test", "test")

	emb := make([]float32, 384)
	if err := models.UpsertEmbedding(db, n.ID, emb, "oldhash"); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if !models.NeedsEmbedding(db, n.ID, "newhash") {
		t.Error("expected NeedsEmbedding=true when hash differs")
	}
}

func TestUpsertEmbedding_InsertsAndUpdates(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	n, _ := models.CreateNote(db, u.ID, "Test", "test")

	emb1 := make([]float32, 384)
	emb1[0] = 1.0
	if err := models.UpsertEmbedding(db, n.ID, emb1, "hash1"); err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	// Second upsert should update
	emb2 := make([]float32, 384)
	emb2[0] = 2.0
	if err := models.UpsertEmbedding(db, n.ID, emb2, "hash2"); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	// hash should be updated
	if models.NeedsEmbedding(db, n.ID, "hash2") {
		t.Error("expected hash2 to match after update")
	}
}

func TestDeleteEmbedding_RemovesData(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	n, _ := models.CreateNote(db, u.ID, "Test", "test")

	emb := make([]float32, 384)
	_ = models.UpsertEmbedding(db, n.ID, emb, "hash")

	if err := models.DeleteEmbedding(db, n.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if !models.NeedsEmbedding(db, n.ID, "hash") {
		t.Error("expected NeedsEmbedding=true after deletion")
	}
}

func TestNotesWithoutEmbeddings_ReturnsUnembeddedNotes(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	n1, _ := models.CreateNote(db, u.ID, "Note One", "note-one")
	n2, _ := models.CreateNote(db, u.ID, "Note Two", "note-two")

	// Embed only n1
	emb := make([]float32, 384)
	_ = models.UpsertEmbedding(db, n1.ID, emb, "hash")

	results, err := models.NotesWithoutEmbeddings(db)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 unembedded note, got %d", len(results))
	}
	if results[0].ID != n2.ID {
		t.Errorf("expected note %d, got %d", n2.ID, results[0].ID)
	}
}
