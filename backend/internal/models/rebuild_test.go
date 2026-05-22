package models_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/selvakn/yant/internal/models"
)

func TestRebuildDB_EmptyNotesDir(t *testing.T) {
	db := openTestDB(t)
	tmp := t.TempDir()
	notesDir := filepath.Join(tmp, "notes")
	uploadsDir := filepath.Join(tmp, "uploads")
	os.MkdirAll(notesDir, 0755)   //nolint:errcheck
	os.MkdirAll(uploadsDir, 0755) //nolint:errcheck

	if err := models.RebuildDB(db, notesDir, uploadsDir); err != nil {
		t.Fatalf("RebuildDB with empty dir: %v", err)
	}
}

func TestRebuildDB_NonExistentNotesDir(t *testing.T) {
	db := openTestDB(t)
	tmp := t.TempDir()
	// Pass non-existent notes dir — should succeed (no notes to import)
	notesDir := filepath.Join(tmp, "nonexistent")
	uploadsDir := filepath.Join(tmp, "uploads")

	if err := models.RebuildDB(db, notesDir, uploadsDir); err != nil {
		t.Fatalf("RebuildDB with missing dir: %v", err)
	}
}

func TestRebuildDB_RebuildsFromFiles(t *testing.T) {
	db := openTestDB(t)
	tmp := t.TempDir()
	notesDir := filepath.Join(tmp, "notes")
	uploadsDir := filepath.Join(tmp, "uploads")

	// Create alice's notes directory with two files
	aliceDir := filepath.Join(notesDir, "alice")
	os.MkdirAll(aliceDir, 0755) //nolint:errcheck

	os.WriteFile(filepath.Join(aliceDir, "my-note.md"), []byte("# My Note\n\nHello #work #ideas"), 0644) //nolint:errcheck
	os.WriteFile(filepath.Join(aliceDir, "other-note.md"), []byte("# Other Note\n\nNo tags here"), 0644) //nolint:errcheck

	// Create bob's notes directory
	bobDir := filepath.Join(notesDir, "bob")
	os.MkdirAll(bobDir, 0755)                                                                   //nolint:errcheck
	os.WriteFile(filepath.Join(bobDir, "bob-note.md"), []byte("# Bob Note\n\n#personal"), 0644) //nolint:errcheck

	if err := models.RebuildDB(db, notesDir, uploadsDir); err != nil {
		t.Fatalf("RebuildDB: %v", err)
	}

	// Check alice was created
	alice, err := models.GetUserByUsername(db, "alice")
	if err != nil {
		t.Fatalf("alice not found after rebuild: %v", err)
	}

	// Check alice's notes
	aliceNotes, err := models.ListNotes(db, alice.ID, "", false)
	if err != nil {
		t.Fatalf("ListNotes for alice: %v", err)
	}
	if len(aliceNotes) != 2 {
		t.Errorf("expected 2 notes for alice, got %d", len(aliceNotes))
	}

	// Check note with tags
	tagged, err := models.ListNotes(db, alice.ID, "work", false)
	if err != nil {
		t.Fatalf("ListNotes with tag: %v", err)
	}
	if len(tagged) != 1 || tagged[0].Title != "My Note" {
		t.Errorf("expected 1 note with #work tag named 'My Note', got: %v", tagged)
	}

	// Check bob was created
	bob, err := models.GetUserByUsername(db, "bob")
	if err != nil {
		t.Fatalf("bob not found after rebuild: %v", err)
	}
	bobNotes, _ := models.ListNotes(db, bob.ID, "", false)
	if len(bobNotes) != 1 {
		t.Errorf("expected 1 note for bob, got %d", len(bobNotes))
	}
}

func TestRebuildDB_noteDrawings_legacy(t *testing.T) {
	db := openTestDB(t)
	tmp := t.TempDir()
	notesDir := filepath.Join(tmp, "notes")
	aliceDir := filepath.Join(notesDir, "alice")
	os.MkdirAll(aliceDir, 0755)                                                       //nolint:errcheck
	os.WriteFile(filepath.Join(aliceDir, "doc.md"), []byte("# Doc\n"), 0644)          //nolint:errcheck
	os.WriteFile(filepath.Join(aliceDir, "doc.tldraw.json"), []byte(`{"x":1}`), 0644) //nolint:errcheck

	if err := models.RebuildDB(db, notesDir, ""); err != nil {
		t.Fatalf("RebuildDB: %v", err)
	}

	alice, err := models.GetUserByUsername(db, "alice")
	if err != nil {
		t.Fatalf("alice: %v", err)
	}
	notes, err := models.ListNotes(db, alice.ID, "", false)
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("expected 1 note, got %d", len(notes))
	}

	list, err := models.ListDrawings(db, notes[0].ID)
	if err != nil {
		t.Fatalf("ListDrawings: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 drawing row, got %d", len(list))
	}
	if list[0].DisplayName != "Drawing 1" || list[0].ToolType != "tldraw" {
		t.Errorf("drawing: %+v", list[0])
	}
}

func TestRebuildDB_noteDrawings_multiFormat(t *testing.T) {
	db := openTestDB(t)
	tmp := t.TempDir()
	notesDir := filepath.Join(tmp, "notes")
	aliceDir := filepath.Join(notesDir, "alice")
	os.MkdirAll(aliceDir, 0755)                                                                //nolint:errcheck
	os.WriteFile(filepath.Join(aliceDir, "pic.md"), []byte("# Pic\n"), 0644)                   //nolint:errcheck
	os.WriteFile(filepath.Join(aliceDir, "pic--abc12345.excalidraw.json"), []byte(`{}`), 0644) //nolint:errcheck

	if err := models.RebuildDB(db, notesDir, ""); err != nil {
		t.Fatalf("RebuildDB: %v", err)
	}

	alice, err := models.GetUserByUsername(db, "alice")
	if err != nil {
		t.Fatal(err)
	}
	notes, err := models.ListNotes(db, alice.ID, "", false)
	if err != nil {
		t.Fatal(err)
	}
	list, err := models.ListDrawings(db, notes[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 drawing, got %d %+v", len(list), list)
	}
	if list[0].DrawingID != "abc12345" || list[0].ToolType != "excalidraw" {
		t.Errorf("drawing: %+v", list[0])
	}
}

func TestRebuildDB_FileWithNoH1UsesSlugAsTitle(t *testing.T) {
	db := openTestDB(t)
	tmp := t.TempDir()
	notesDir := filepath.Join(tmp, "notes")

	aliceDir := filepath.Join(notesDir, "alice")
	os.MkdirAll(aliceDir, 0755)                                                          //nolint:errcheck
	os.WriteFile(filepath.Join(aliceDir, "my-slug.md"), []byte("No heading here"), 0644) //nolint:errcheck

	if err := models.RebuildDB(db, notesDir, ""); err != nil {
		t.Fatalf("RebuildDB: %v", err)
	}

	alice, _ := models.GetUserByUsername(db, "alice")
	notes, _ := models.ListNotes(db, alice.ID, "", false)
	if len(notes) == 0 {
		t.Fatal("expected at least one note")
	}
	if notes[0].Title != "my-slug" {
		t.Errorf("expected title 'my-slug' (fallback to slug), got %q", notes[0].Title)
	}
}

func TestRebuildDB_ClearsExistingData(t *testing.T) {
	db := openTestDB(t)
	// Pre-populate some data
	u, _ := models.GetOrCreateUser(db, "stale")
	models.CreateNote(db, u.ID, "Stale Note", "stale", 0, true) //nolint:errcheck

	tmp := t.TempDir()
	notesDir := filepath.Join(tmp, "notes")
	os.MkdirAll(notesDir, 0755) //nolint:errcheck

	// RebuildDB should clear existing data
	if err := models.RebuildDB(db, notesDir, ""); err != nil {
		t.Fatalf("RebuildDB: %v", err)
	}

	// stale user should no longer exist
	_, err := models.GetUserByUsername(db, "stale")
	if err == nil {
		t.Error("expected stale user to be removed after rebuild")
	}
}

func TestListNotes_FilterByTag(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")

	n1, _ := models.CreateNote(db, u.ID, "Work Note", "work-note", 0, true)
	n2, _ := models.CreateNote(db, u.ID, "Personal", "personal", 0, true)
	models.SyncTags(db, n1.ID, []string{"work"})     //nolint:errcheck
	models.SyncTags(db, n2.ID, []string{"personal"}) //nolint:errcheck

	workNotes, err := models.ListNotes(db, u.ID, "work", false)
	if err != nil {
		t.Fatalf("ListNotes with tag: %v", err)
	}
	if len(workNotes) != 1 || workNotes[0].Title != "Work Note" {
		t.Errorf("expected [Work Note], got: %v", workNotes)
	}
}

func TestUpdateNote_ReturnsNilForNonExistent(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")

	n, err := models.UpdateNote(db, u.ID, "nonexistent-slug", "New Title", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != nil {
		t.Error("expected nil for non-existent note")
	}
}

func TestDeleteNote_NoErrorForNonExistent(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")

	// Should not return an error when note doesn't exist
	if err := models.DeleteNote(db, u.ID, "ghost"); err != nil {
		t.Errorf("expected no error deleting non-existent note, got: %v", err)
	}
}

func TestGenerateSlug_MultipleCollisions(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	models.CreateNote(db, u.ID, "Hello", "hello", 0, true)   //nolint:errcheck
	models.CreateNote(db, u.ID, "Hello", "hello-2", 0, true) //nolint:errcheck

	slug, err := models.GenerateSlug(db, u.ID, "Hello")
	if err != nil {
		t.Fatal(err)
	}
	if slug != "hello-3" {
		t.Errorf("expected 'hello-3', got %q", slug)
	}
}
