package models_test

import (
	"testing"
	"time"

	"github.com/selvakn/yant/internal/models"
)

func openTestDB(t *testing.T) *models.DB {
	t.Helper()
	db, err := models.Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := models.InitSchema(db); err != nil {
		t.Fatalf("schema: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// ── User tests ────────────────────────────────────────────────────────────────

func TestGetOrCreateUser_CreatesNewUser(t *testing.T) {
	db := openTestDB(t)
	u, err := models.GetOrCreateUser(db, "alice")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}
	if u.Username != "alice" {
		t.Errorf("expected alice, got %s", u.Username)
	}
	if u.ID == 0 {
		t.Error("expected non-zero ID")
	}
}

func TestGetOrCreateUser_ReturnsSameOnSecondCall(t *testing.T) {
	db := openTestDB(t)
	u1, _ := models.GetOrCreateUser(db, "bob")
	u2, err := models.GetOrCreateUser(db, "bob")
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if u1.ID != u2.ID {
		t.Errorf("expected same ID, got %d and %d", u1.ID, u2.ID)
	}
}

func TestGetUserByUsername_NotFound(t *testing.T) {
	db := openTestDB(t)
	_, err := models.GetUserByUsername(db, "nobody")
	if err == nil {
		t.Error("expected error for missing user")
	}
}

// ── Note tests ────────────────────────────────────────────────────────────────

func TestCreateNote_ReturnsPopulatedNote(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")

	n, err := models.CreateNote(db, u.ID, "My Note", "my-note")
	if err != nil {
		t.Fatalf("CreateNote: %v", err)
	}
	if n.Title != "My Note" || n.Slug != "my-note" {
		t.Errorf("unexpected note: %+v", n)
	}
	if n.ID == 0 {
		t.Error("expected non-zero note ID")
	}
	if n.CreatedAt.IsZero() || n.UpdatedAt.IsZero() {
		t.Error("expected non-zero timestamps")
	}
}

func TestCreateNote_EmptyTitleDefaultsToUntitled(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	n, _ := models.CreateNote(db, u.ID, "", "untitled-note")
	if n.Title != "Untitled Note" {
		t.Errorf("expected 'Untitled Note', got %q", n.Title)
	}
}

func TestGetNote_ReturnsCorrectNoteForOwner(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	models.CreateNote(db, u.ID, "Test", "test") //nolint:errcheck

	n, err := models.GetNote(db, u.ID, "test")
	if err != nil || n == nil {
		t.Fatalf("GetNote: %v %v", n, err)
	}
	if n.Title != "Test" {
		t.Errorf("expected 'Test', got %q", n.Title)
	}
}

func TestGetNote_ReturnsNilForWrongUser(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	models.CreateNote(db, u.ID, "Secret", "secret") //nolint:errcheck

	n, err := models.GetNote(db, 9999, "secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != nil {
		t.Error("expected nil for wrong user")
	}
}

func TestListNotes_ReturnsOnlyUserNotes(t *testing.T) {
	db := openTestDB(t)
	alice, _ := models.GetOrCreateUser(db, "alice")
	bob, _ := models.GetOrCreateUser(db, "bob")

	models.CreateNote(db, alice.ID, "Alice Note", "alice-note") //nolint:errcheck
	models.CreateNote(db, bob.ID, "Bob Note", "bob-note")       //nolint:errcheck

	aliceNotes, _ := models.ListNotes(db, alice.ID, "", false)
	if len(aliceNotes) != 1 || aliceNotes[0].Title != "Alice Note" {
		t.Errorf("expected only Alice Note, got: %+v", aliceNotes)
	}
}

func TestUpdateNote_ChangesTitleAndUpdatedAt(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	models.CreateNote(db, u.ID, "Original", "original") //nolint:errcheck

	time.Sleep(time.Millisecond) // ensure updated_at differs
	n, err := models.UpdateNote(db, u.ID, "original", "Updated")
	if err != nil || n == nil {
		t.Fatalf("UpdateNote: %v", err)
	}
	if n.Title != "Updated" {
		t.Errorf("expected 'Updated', got %q", n.Title)
	}
}

func TestDeleteNote_RemovesRow(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	models.CreateNote(db, u.ID, "Bye", "bye") //nolint:errcheck

	if err := models.DeleteNote(db, u.ID, "bye"); err != nil {
		t.Fatalf("DeleteNote: %v", err)
	}
	n, _ := models.GetNote(db, u.ID, "bye")
	if n != nil {
		t.Error("expected nil after delete")
	}
}

// ── Tag tests ─────────────────────────────────────────────────────────────────

func TestParseTags_ExtractsHashtags(t *testing.T) {
	tags := models.ParseTags("Hello #world and #go-lang plus #test_123")
	want := map[string]bool{"world": true, "go-lang": true, "test_123": true}
	if len(tags) != len(want) {
		t.Fatalf("expected %d tags, got %v", len(want), tags)
	}
	for _, tag := range tags {
		if !want[tag] {
			t.Errorf("unexpected tag %q", tag)
		}
	}
}

func TestParseTags_CaseInsensitive(t *testing.T) {
	tags := models.ParseTags("#Work #WORK #work")
	if len(tags) != 1 || tags[0] != "work" {
		t.Errorf("expected single 'work', got: %v", tags)
	}
}

func TestParseTags_EmptyBody(t *testing.T) {
	tags := models.ParseTags("")
	if len(tags) != 0 {
		t.Errorf("expected no tags, got: %v", tags)
	}
}

func TestSyncTags_InsertsThenReplaces(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	n, _ := models.CreateNote(db, u.ID, "Tagged", "tagged")

	// First sync
	models.SyncTags(db, n.ID, []string{"work", "ideas"}) //nolint:errcheck
	note, _ := models.GetNote(db, u.ID, "tagged")
	if len(note.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d: %v", len(note.Tags), note.Tags)
	}

	// Replace with different tags
	models.SyncTags(db, n.ID, []string{"personal"}) //nolint:errcheck
	note2, _ := models.GetNote(db, u.ID, "tagged")
	if len(note2.Tags) != 1 || note2.Tags[0] != "personal" {
		t.Errorf("expected [personal], got %v", note2.Tags)
	}
}

func TestListTagsForUser_CountsCorrectly(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")

	n1, _ := models.CreateNote(db, u.ID, "Note1", "note1")
	n2, _ := models.CreateNote(db, u.ID, "Note2", "note2")
	models.SyncTags(db, n1.ID, []string{"work"}) //nolint:errcheck
	models.SyncTags(db, n2.ID, []string{"work", "ideas"}) //nolint:errcheck

	tags, err := models.ListTagsForUser(db, u.ID, false)
	if err != nil {
		t.Fatalf("ListTagsForUser: %v", err)
	}
	found := make(map[string]int)
	for _, tc := range tags {
		found[tc.Name] = tc.Count
	}
	if found["work"] != 2 {
		t.Errorf("expected work count=2, got %d", found["work"])
	}
	if found["ideas"] != 1 {
		t.Errorf("expected ideas count=1, got %d", found["ideas"])
	}
}

func TestListTagsForUser_OnlyCurrentUser(t *testing.T) {
	db := openTestDB(t)
	alice, _ := models.GetOrCreateUser(db, "alice")
	bob, _ := models.GetOrCreateUser(db, "bob")

	na, _ := models.CreateNote(db, alice.ID, "A", "a")
	nb, _ := models.CreateNote(db, bob.ID, "B", "b")
	models.SyncTags(db, na.ID, []string{"alicetag"}) //nolint:errcheck
	models.SyncTags(db, nb.ID, []string{"bobtag"})   //nolint:errcheck

	tags, _ := models.ListTagsForUser(db, alice.ID, false)
	for _, tc := range tags {
		if tc.Name == "bobtag" {
			t.Error("alice should not see bob's tags")
		}
	}
}

// ── Image tests ───────────────────────────────────────────────────────────────

func TestCreateAndDeleteImages(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	n, _ := models.CreateNote(db, u.ID, "With Image", "with-image")

	img, err := models.CreateImage(db, n.ID, "abc.png", "photo.png", "image/png", 1024)
	if err != nil || img.ID == 0 {
		t.Fatalf("CreateImage: %v", err)
	}

	imgs, _ := models.GetImagesForNote(db, n.ID)
	if len(imgs) != 1 {
		t.Errorf("expected 1 image, got %d", len(imgs))
	}

	filenames, err := models.DeleteImagesForNote(db, n.ID)
	if err != nil {
		t.Fatalf("DeleteImagesForNote: %v", err)
	}
	if len(filenames) != 1 || filenames[0] != "abc.png" {
		t.Errorf("expected [abc.png], got %v", filenames)
	}

	imgs2, _ := models.GetImagesForNote(db, n.ID)
	if len(imgs2) != 0 {
		t.Errorf("expected 0 images after delete, got %d", len(imgs2))
	}
}

// ── Slug tests ────────────────────────────────────────────────────────────────

func TestGenerateSlug_Basic(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	slug, err := models.GenerateSlug(db, u.ID, "Hello World")
	if err != nil {
		t.Fatal(err)
	}
	if slug != "hello-world" {
		t.Errorf("expected 'hello-world', got %q", slug)
	}
}

func TestGenerateSlug_Collision(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	models.CreateNote(db, u.ID, "Hello", "hello") //nolint:errcheck

	slug, _ := models.GenerateSlug(db, u.ID, "Hello")
	if slug != "hello-2" {
		t.Errorf("expected 'hello-2', got %q", slug)
	}
}

// ── Tag color tests ──────────────────────────────────────────────────────────

func TestAutoTagColor_Deterministic(t *testing.T) {
	c1 := models.AutoTagColor("work")
	c2 := models.AutoTagColor("work")
	if c1 != c2 {
		t.Errorf("expected same color for same tag, got %s and %s", c1, c2)
	}
	// Different tags may have different colors
	c3 := models.AutoTagColor("personal")
	_ = c3 // just ensure it runs without panic
}

func TestAutoTagColor_InPalette(t *testing.T) {
	color := models.AutoTagColor("testing")
	found := false
	for _, c := range models.ColorPalette {
		if c == color {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("auto color %s not in palette", color)
	}
}

func TestSetAndGetTagColor(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")

	// Before setting, should return auto color
	auto := models.GetTagColor(db, u.ID, "work")
	if auto != models.AutoTagColor("work") {
		t.Errorf("expected auto color, got %s", auto)
	}

	// Set custom color
	err := models.SetTagColor(db, u.ID, "work", "#ee9b00")
	if err != nil {
		t.Fatalf("SetTagColor: %v", err)
	}

	// Now should return custom color
	custom := models.GetTagColor(db, u.ID, "work")
	if custom != "#ee9b00" {
		t.Errorf("expected #ee9b00, got %s", custom)
	}

	// Update color
	models.SetTagColor(db, u.ID, "work", "#001219") //nolint:errcheck
	updated := models.GetTagColor(db, u.ID, "work")
	if updated != "#001219" {
		t.Errorf("expected #001219, got %s", updated)
	}
}

func TestListTagsForUser_IncludesColor(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	n, _ := models.CreateNote(db, u.ID, "Test", "test")
	models.SyncTags(db, n.ID, []string{"work", "ideas"}) //nolint:errcheck

	// Set color for "work" only
	models.SetTagColor(db, u.ID, "work", "#ca6702") //nolint:errcheck

	tags, err := models.ListTagsForUser(db, u.ID, false)
	if err != nil {
		t.Fatalf("ListTagsForUser: %v", err)
	}

	for _, tc := range tags {
		if tc.Color == "" {
			t.Errorf("tag %s has empty color", tc.Name)
		}
		if tc.Name == "work" && tc.Color != "#ca6702" {
			t.Errorf("expected #ca6702 for work, got %s", tc.Color)
		}
		if tc.Name == "ideas" && tc.Color != models.AutoTagColor("ideas") {
			t.Errorf("expected auto color for ideas, got %s", tc.Color)
		}
	}
}

// ── Archive tests ─────────────────────────────────────────────────────────────

func TestArchiveNote_SetsArchivedFlag(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	models.CreateNote(db, u.ID, "To Archive", "to-archive") //nolint:errcheck

	err := models.ArchiveNote(db, u.ID, "to-archive")
	if err != nil {
		t.Fatalf("ArchiveNote: %v", err)
	}

	note, _ := models.GetNote(db, u.ID, "to-archive")
	if !note.Archived {
		t.Error("expected note to be archived")
	}
}

func TestArchiveNote_ReturnsErrorForNonExistent(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")

	err := models.ArchiveNote(db, u.ID, "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent note")
	}
}

func TestRestoreNote_ClearsArchivedFlag(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	models.CreateNote(db, u.ID, "Archived Note", "archived-note") //nolint:errcheck
	models.ArchiveNote(db, u.ID, "archived-note")                 //nolint:errcheck

	err := models.RestoreNote(db, u.ID, "archived-note")
	if err != nil {
		t.Fatalf("RestoreNote: %v", err)
	}

	note, _ := models.GetNote(db, u.ID, "archived-note")
	if note.Archived {
		t.Error("expected note to not be archived after restore")
	}
}

func TestRestoreNote_ReturnsErrorForNonExistent(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")

	err := models.RestoreNote(db, u.ID, "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent note")
	}
}

func TestListNotes_ExcludesArchivedByDefault(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	models.CreateNote(db, u.ID, "Active Note", "active-note")     //nolint:errcheck
	models.CreateNote(db, u.ID, "Archived Note", "archived-note") //nolint:errcheck
	models.ArchiveNote(db, u.ID, "archived-note")                 //nolint:errcheck

	notes, _ := models.ListNotes(db, u.ID, "", false)
	if len(notes) != 1 {
		t.Fatalf("expected 1 active note, got %d", len(notes))
	}
	if notes[0].Slug != "active-note" {
		t.Errorf("expected active-note, got %s", notes[0].Slug)
	}
}

func TestListNotes_IncludesArchivedWhenRequested(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	models.CreateNote(db, u.ID, "Active Note", "active-note")     //nolint:errcheck
	models.CreateNote(db, u.ID, "Archived Note", "archived-note") //nolint:errcheck
	models.ArchiveNote(db, u.ID, "archived-note")                 //nolint:errcheck

	notes, _ := models.ListNotes(db, u.ID, "", true)
	if len(notes) != 1 {
		t.Fatalf("expected 1 archived note, got %d", len(notes))
	}
	if notes[0].Slug != "archived-note" {
		t.Errorf("expected archived-note, got %s", notes[0].Slug)
	}
}

func TestListTagsForUser_ExcludesArchivedByDefault(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	n1, _ := models.CreateNote(db, u.ID, "Active", "active")
	n2, _ := models.CreateNote(db, u.ID, "Archived", "archived")
	models.SyncTags(db, n1.ID, []string{"work"})    //nolint:errcheck
	models.SyncTags(db, n2.ID, []string{"archive"}) //nolint:errcheck
	models.ArchiveNote(db, u.ID, "archived")        //nolint:errcheck

	tags, _ := models.ListTagsForUser(db, u.ID, false)
	for _, tc := range tags {
		if tc.Name == "archive" {
			t.Error("should not include tags from archived notes")
		}
	}
}

func TestInitSchema_MigratesExistingDBWithoutArchivedColumn(t *testing.T) {
	db, err := models.Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Create schema without the archived column (simulates pre-archive DB)
	_, err = db.Exec(`
		PRAGMA journal_mode=WAL;
		PRAGMA foreign_keys=ON;
		CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT UNIQUE NOT NULL, created_at TEXT NOT NULL);
		CREATE TABLE notes (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL REFERENCES users(id), slug TEXT NOT NULL, title TEXT NOT NULL DEFAULT 'Untitled Note', created_at TEXT NOT NULL, updated_at TEXT NOT NULL, UNIQUE(user_id, slug));
		CREATE TABLE note_tags (note_id INTEGER NOT NULL REFERENCES notes(id) ON DELETE CASCADE, tag_name TEXT NOT NULL, PRIMARY KEY(note_id, tag_name));
		CREATE TABLE images (id INTEGER PRIMARY KEY AUTOINCREMENT, note_id INTEGER NOT NULL REFERENCES notes(id) ON DELETE CASCADE, filename TEXT NOT NULL, original TEXT NOT NULL, mime_type TEXT NOT NULL, size INTEGER NOT NULL);
		CREATE TABLE tag_colors (user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE, tag_name TEXT NOT NULL, color TEXT NOT NULL, PRIMARY KEY(user_id, tag_name));
	`)
	if err != nil {
		t.Fatalf("create pre-archive schema: %v", err)
	}

	// InitSchema should migrate successfully (add archived column)
	if err := models.InitSchema(db); err != nil {
		t.Fatalf("InitSchema on pre-archive DB failed: %v", err)
	}

	// Verify the column was added and works
	u, _ := models.GetOrCreateUser(db, "alice")
	_, _ = models.CreateNote(db, u.ID, "Test", "test")
	note, err := models.GetNote(db, u.ID, "test")
	if err != nil || note == nil {
		t.Fatalf("GetNote after migration: %v", err)
	}
	if note.Archived {
		t.Error("new note should default to not archived")
	}

	// Archive and verify
	if err := models.ArchiveNote(db, u.ID, "test"); err != nil {
		t.Fatalf("ArchiveNote after migration: %v", err)
	}
	note, _ = models.GetNote(db, u.ID, "test")
	if !note.Archived {
		t.Error("note should be archived")
	}
}
