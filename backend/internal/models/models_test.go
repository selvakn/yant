package models_test

import (
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"
	"unicode"

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

	n, err := models.CreateNote(db, u.ID, "My Note", "my-note", 0, true)
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
	n, _ := models.CreateNote(db, u.ID, "", "untitled-note", 0, true)
	if n.Title != "Untitled Note" {
		t.Errorf("expected 'Untitled Note', got %q", n.Title)
	}
}

func TestGetNote_ReturnsCorrectNoteForOwner(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	models.CreateNote(db, u.ID, "Test", "test", 0, true) //nolint:errcheck

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
	models.CreateNote(db, u.ID, "Secret", "secret", 0, true) //nolint:errcheck

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

	models.CreateNote(db, alice.ID, "Alice Note", "alice-note", 0, true) //nolint:errcheck
	models.CreateNote(db, bob.ID, "Bob Note", "bob-note", 0, true)       //nolint:errcheck

	aliceNotes, _ := models.ListNotes(db, alice.ID, "", false)
	if len(aliceNotes) != 1 || aliceNotes[0].Title != "Alice Note" {
		t.Errorf("expected only Alice Note, got: %+v", aliceNotes)
	}
}

func TestUpdateNote_ChangesTitleAndUpdatedAt(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	models.CreateNote(db, u.ID, "Original", "original", 0, true) //nolint:errcheck

	time.Sleep(time.Millisecond) // ensure updated_at differs
	n, err := models.UpdateNote(db, u.ID, "original", "Updated", 0)
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
	models.CreateNote(db, u.ID, "Bye", "bye", 0, true) //nolint:errcheck

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
	n, _ := models.CreateNote(db, u.ID, "Tagged", "tagged", 0, true)

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

	n1, _ := models.CreateNote(db, u.ID, "Note1", "note1", 0, true)
	n2, _ := models.CreateNote(db, u.ID, "Note2", "note2", 0, true)
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

	na, _ := models.CreateNote(db, alice.ID, "A", "a", 0, true)
	nb, _ := models.CreateNote(db, bob.ID, "B", "b", 0, true)
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
	n, _ := models.CreateNote(db, u.ID, "With Image", "with-image", 0, true)

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
	models.CreateNote(db, u.ID, "Hello", "hello", 0, true) //nolint:errcheck

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
	n, _ := models.CreateNote(db, u.ID, "Test", "test", 0, true)
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
	models.CreateNote(db, u.ID, "To Archive", "to-archive", 0, true) //nolint:errcheck

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
	models.CreateNote(db, u.ID, "Archived Note", "archived-note", 0, true) //nolint:errcheck
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
	models.CreateNote(db, u.ID, "Active Note", "active-note", 0, true)     //nolint:errcheck
	models.CreateNote(db, u.ID, "Archived Note", "archived-note", 0, true) //nolint:errcheck
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
	models.CreateNote(db, u.ID, "Active Note", "active-note", 0, true)     //nolint:errcheck
	models.CreateNote(db, u.ID, "Archived Note", "archived-note", 0, true) //nolint:errcheck
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
	n1, _ := models.CreateNote(db, u.ID, "Active", "active", 0, true)
	n2, _ := models.CreateNote(db, u.ID, "Archived", "archived", 0, true)
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
	_, _ = models.CreateNote(db, u.ID, "Test", "test", 0, true)
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

// ── Note Links tests ──────────────────────────────────────────────────────────

func TestParseNoteLinks_ExtractsLinks(t *testing.T) {
	body := "Check out [[My First Note]] and also [[Another Note]]."
	links := models.ParseNoteLinks(body)
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d: %v", len(links), links)
	}
	if links[0] != "My First Note" || links[1] != "Another Note" {
		t.Errorf("unexpected links: %v", links)
	}
}

func TestParseNoteLinks_DeduplicatesCaseInsensitive(t *testing.T) {
	body := "[[Note One]] and [[note one]] again"
	links := models.ParseNoteLinks(body)
	if len(links) != 1 {
		t.Errorf("expected 1 unique link, got %d: %v", len(links), links)
	}
}

func TestParseNoteLinks_EmptyBody(t *testing.T) {
	links := models.ParseNoteLinks("")
	if len(links) != 0 {
		t.Errorf("expected 0 links, got %d", len(links))
	}
}

func TestParseNoteLinks_NoLinks(t *testing.T) {
	links := models.ParseNoteLinks("No wiki links here, just [regular](links)")
	if len(links) != 0 {
		t.Errorf("expected 0 links, got %d", len(links))
	}
}

func TestSyncLinks_CreatesAndRemovesLinks(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	slug1, _ := models.GenerateSlug(db, u.ID, "Note A")
	noteA, _ := models.CreateNote(db, u.ID, "Note A", slug1, 0, true)
	slug2, _ := models.GenerateSlug(db, u.ID, "Note B")
	noteB, _ := models.CreateNote(db, u.ID, "Note B", slug2, 0, true)

	// noteB links to noteA
	err := models.SyncLinks(db, noteB.ID, u.ID, []string{"Note A"})
	if err != nil {
		t.Fatalf("SyncLinks: %v", err)
	}

	backlinks, err := models.GetBacklinks(db, noteA.ID)
	if err != nil {
		t.Fatalf("GetBacklinks: %v", err)
	}
	if len(backlinks) != 1 || backlinks[0].Slug != "note-b" {
		t.Errorf("expected backlink from note-b, got: %v", backlinks)
	}

	// Remove the link
	err = models.SyncLinks(db, noteB.ID, u.ID, []string{})
	if err != nil {
		t.Fatalf("SyncLinks (remove): %v", err)
	}
	backlinks, _ = models.GetBacklinks(db, noteA.ID)
	if len(backlinks) != 0 {
		t.Errorf("expected 0 backlinks after removal, got %d", len(backlinks))
	}
}

func TestSyncLinks_SkipsSelfLink(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	slug, _ := models.GenerateSlug(db, u.ID, "Self Ref")
	note, _ := models.CreateNote(db, u.ID, "Self Ref", slug, 0, true)

	err := models.SyncLinks(db, note.ID, u.ID, []string{"Self Ref"})
	if err != nil {
		t.Fatalf("SyncLinks: %v", err)
	}

	backlinks, _ := models.GetBacklinks(db, note.ID)
	if len(backlinks) != 0 {
		t.Errorf("expected 0 backlinks (self-link skipped), got %d", len(backlinks))
	}
}

func TestSyncLinks_CaseInsensitiveMatch(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	slugA, _ := models.GenerateSlug(db, u.ID, "My Note")
	noteA, _ := models.CreateNote(db, u.ID, "My Note", slugA, 0, true)
	slugB, _ := models.GenerateSlug(db, u.ID, "Linker")
	noteB, _ := models.CreateNote(db, u.ID, "Linker", slugB, 0, true)

	err := models.SyncLinks(db, noteB.ID, u.ID, []string{"my note"})
	if err != nil {
		t.Fatalf("SyncLinks: %v", err)
	}

	backlinks, _ := models.GetBacklinks(db, noteA.ID)
	if len(backlinks) != 1 {
		t.Errorf("expected case-insensitive match, got %d backlinks", len(backlinks))
	}
}

func TestGetBacklinks_EmptyWhenNoLinks(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	slug, _ := models.GenerateSlug(db, u.ID, "Lonely Note")
	note, _ := models.CreateNote(db, u.ID, "Lonely Note", slug, 0, true)

	backlinks, err := models.GetBacklinks(db, note.ID)
	if err != nil {
		t.Fatalf("GetBacklinks: %v", err)
	}
	if len(backlinks) != 0 {
		t.Errorf("expected 0 backlinks, got %d", len(backlinks))
	}
}

func TestResolveWikiLinks_ResolvesExistingNote(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	slug, _ := models.GenerateSlug(db, u.ID, "Target Note")
	models.CreateNote(db, u.ID, "Target Note", slug, 0, true)

	body := "See [[Target Note]] for details."
	resolved := models.ResolveWikiLinks(db, u.ID, body)
	expected := "See [Target Note](/notes/target-note) for details."
	if resolved != expected {
		t.Errorf("expected %q, got %q", expected, resolved)
	}
}

func TestResolveWikiLinks_LeavesUnresolved(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")

	body := "See [[Nonexistent Note]] here."
	resolved := models.ResolveWikiLinks(db, u.ID, body)
	expected := "See Nonexistent Note here."
	if resolved != expected {
		t.Errorf("expected %q, got %q", expected, resolved)
	}
}

func TestResolveWikiLinks_PreservesDrawingMarker(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")

	body := "Diagram:\n\n![[draw:abc12xyz]]\n"
	resolved := models.ResolveWikiLinks(db, u.ID, body)
	if resolved != body {
		t.Errorf("drawing marker must stay intact for markdown; got %q", resolved)
	}
}

func TestParseNoteLinks_IgnoresDrawingMarkers(t *testing.T) {
	body := "Link [[Real Note]] and ![[draw:abc12xyz]]"
	got := models.ParseNoteLinks(body)
	if len(got) != 1 || got[0] != "Real Note" {
		t.Fatalf("ParseNoteLinks: want [Real Note], got %#v", got)
	}
}

func TestSearchNotesByTitle_FindsMatches(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	s1, _ := models.GenerateSlug(db, u.ID, "Meeting Notes")
	models.CreateNote(db, u.ID, "Meeting Notes", s1, 0, true)
	s2, _ := models.GenerateSlug(db, u.ID, "Shopping List")
	models.CreateNote(db, u.ID, "Shopping List", s2, 0, true)

	results, err := models.SearchNotesByTitle(db, u.ID, "meet")
	if err != nil {
		t.Fatalf("SearchNotesByTitle: %v", err)
	}
	if len(results) != 1 || results[0].Title != "Meeting Notes" {
		t.Errorf("expected Meeting Notes, got %v", results)
	}
}

func TestSearchNotesByTitle_EmptyQuery(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	s1, _ := models.GenerateSlug(db, u.ID, "A")
	models.CreateNote(db, u.ID, "A", s1, 0, true)
	s2, _ := models.GenerateSlug(db, u.ID, "B")
	models.CreateNote(db, u.ID, "B", s2, 0, true)

	results, err := models.SearchNotesByTitle(db, u.ID, "")
	if err != nil {
		t.Fatalf("SearchNotesByTitle: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results for empty query, got %d", len(results))
	}
}

// ── Note drawings tests ─────────────────────────────────────────────────────

func TestGenerateDrawingID(t *testing.T) {
	for range 200 {
		id := models.GenerateDrawingID()
		if len(id) != 8 {
			t.Fatalf("expected length 8, got %d for %q", len(id), id)
		}
		for _, r := range id {
			if !unicode.IsDigit(r) && (r < 'a' || r > 'z') {
				t.Fatalf("invalid character %q in %q", r, id)
			}
		}
	}
}

func TestCreateDrawing_valid(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	n, _ := models.CreateNote(db, u.ID, "Sketch Note", "sketch-note", 0, true)

	d, err := models.CreateDrawing(db, n.ID, "Diagram A", "tldraw")
	if err != nil {
		t.Fatalf("CreateDrawing: %v", err)
	}
	if d.DrawingID == "" || len(d.DrawingID) != 8 {
		t.Errorf("expected 8-char drawing id, got %q", d.DrawingID)
	}
	if d.NoteID != n.ID || d.DisplayName != "Diagram A" || d.ToolType != "tldraw" {
		t.Errorf("unexpected struct: %+v", d)
	}
	if d.CreatedAt.IsZero() || d.UpdatedAt.IsZero() {
		t.Error("expected non-zero timestamps")
	}
}

func TestCreateDrawing_empty_name(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	n, _ := models.CreateNote(db, u.ID, "N", "n", 0, true)

	_, err := models.CreateDrawing(db, n.ID, "", "tldraw")
	if err == nil {
		t.Fatal("expected error for empty display name")
	}
}

func TestCreateDrawing_invalid_tool(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	n, _ := models.CreateNote(db, u.ID, "N", "n2", 0, true)

	_, err := models.CreateDrawing(db, n.ID, "Ok", "paint")
	if err == nil {
		t.Fatal("expected error for invalid tool type")
	}
}

func TestListDrawings_empty(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	n, _ := models.CreateNote(db, u.ID, "Empty", "empty", 0, true)

	list, err := models.ListDrawings(db, n.ID)
	if err != nil {
		t.Fatalf("ListDrawings: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected no drawings, got %#v", list)
	}
}

func TestListDrawings_multiple(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	n, _ := models.CreateNote(db, u.ID, "Multi", "multi", 0, true)

	d1, err := models.CreateDrawing(db, n.ID, "First", "tldraw")
	if err != nil {
		t.Fatalf("CreateDrawing 1: %v", err)
	}
	time.Sleep(5 * time.Millisecond)
	d2, err := models.CreateDrawing(db, n.ID, "Second", "excalidraw")
	if err != nil {
		t.Fatalf("CreateDrawing 2: %v", err)
	}

	list, err := models.ListDrawings(db, n.ID)
	if err != nil {
		t.Fatalf("ListDrawings: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 drawings, got %d", len(list))
	}
	if list[0].DrawingID != d1.DrawingID || list[0].DisplayName != "First" {
		t.Errorf("first drawing mismatch: %+v want drawing_id=%s First", list[0], d1.DrawingID)
	}
	if list[1].DrawingID != d2.DrawingID || list[1].DisplayName != "Second" {
		t.Errorf("second drawing mismatch: %+v want drawing_id=%s Second", list[1], d2.DrawingID)
	}
}

func TestGetDrawing_found(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	n, _ := models.CreateNote(db, u.ID, "G", "g", 0, true)

	created, err := models.CreateDrawing(db, n.ID, "Found Me", "excalidraw")
	if err != nil {
		t.Fatalf("CreateDrawing: %v", err)
	}

	got, err := models.GetDrawing(db, n.ID, created.DrawingID)
	if err != nil || got == nil {
		t.Fatalf("GetDrawing: %v %v", got, err)
	}
	if got.DisplayName != "Found Me" || got.ToolType != "excalidraw" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestGetDrawing_not_found(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	n, _ := models.CreateNote(db, u.ID, "G", "g-missing", 0, true)

	_, err := models.GetDrawing(db, n.ID, "xxxxxxxx")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestRenameDrawing_valid(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	n, _ := models.CreateNote(db, u.ID, "R", "r", 0, true)

	d, _ := models.CreateDrawing(db, n.ID, "Old", "tldraw")

	if err := models.RenameDrawing(db, n.ID, d.DrawingID, "New Title"); err != nil {
		t.Fatalf("RenameDrawing: %v", err)
	}
	got, err := models.GetDrawing(db, n.ID, d.DrawingID)
	if err != nil || got.DisplayName != "New Title" {
		t.Fatalf("after rename: %+v err=%v", got, err)
	}
}

func TestDeleteDrawingRecord(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	n, _ := models.CreateNote(db, u.ID, "D", "d", 0, true)

	d, _ := models.CreateDrawing(db, n.ID, "X", "tldraw")

	if err := models.DeleteDrawingRecord(db, n.ID, d.DrawingID); err != nil {
		t.Fatalf("DeleteDrawingRecord: %v", err)
	}
	_, err := models.GetDrawing(db, n.ID, d.DrawingID)
	if err == nil || !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected gone, err=%v", err)
	}
}

func TestDeleteDrawingRecord_cascade(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	n, _ := models.CreateNote(db, u.ID, "Cascade", "cascade", 0, true)

	_, _ = models.CreateDrawing(db, n.ID, "D1", "tldraw")

	if err := models.DeleteNote(db, u.ID, "cascade"); err != nil {
		t.Fatalf("DeleteNote: %v", err)
	}

	list, err := models.ListDrawings(db, n.ID)
	if err != nil {
		t.Fatalf("ListDrawings: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected cascade removed drawings, got len=%d", len(list))
	}
}

// ── Blog tests ───────────────────────────────────────────────────────────────

func TestPublishBlogPost_creates_row(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	n, _ := models.CreateNote(db, u.ID, "Blog Post", "blog-post", 0, true)

	if models.IsBlogPost(db, n.ID) {
		t.Fatal("note should not be a blog post initially")
	}

	if err := models.PublishBlogPost(db, n.ID); err != nil {
		t.Fatalf("PublishBlogPost: %v", err)
	}

	if !models.IsBlogPost(db, n.ID) {
		t.Error("expected note to be a blog post after publish")
	}
}

func TestPublishBlogPost_idempotent(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	n, _ := models.CreateNote(db, u.ID, "Blog", "blog", 0, true)

	models.PublishBlogPost(db, n.ID) //nolint:errcheck
	models.PublishBlogPost(db, n.ID) //nolint:errcheck

	if !models.IsBlogPost(db, n.ID) {
		t.Error("expected note to remain a blog post")
	}
}

func TestUnpublishBlogPost_removes_row(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	n, _ := models.CreateNote(db, u.ID, "Blog", "blog-unpub", 0, true)

	models.PublishBlogPost(db, n.ID) //nolint:errcheck
	models.UnpublishBlogPost(db, n.ID) //nolint:errcheck

	if models.IsBlogPost(db, n.ID) {
		t.Error("expected note to no longer be a blog post after unpublish")
	}
}

func TestSyncTags_publishes_blog_on_blog_tag(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	n, _ := models.CreateNote(db, u.ID, "Tagged Blog", "tagged-blog", 0, true)

	models.SyncTags(db, n.ID, []string{"blog", "golang"}) //nolint:errcheck

	if !models.IsBlogPost(db, n.ID) {
		t.Error("expected SyncTags with 'blog' to publish the note")
	}
}

func TestSyncTags_unpublishes_blog_when_tag_removed(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	n, _ := models.CreateNote(db, u.ID, "Was Blog", "was-blog", 0, true)

	models.SyncTags(db, n.ID, []string{"blog"}) //nolint:errcheck
	if !models.IsBlogPost(db, n.ID) {
		t.Fatal("should be published")
	}

	models.SyncTags(db, n.ID, []string{"golang"}) //nolint:errcheck
	if models.IsBlogPost(db, n.ID) {
		t.Error("expected removal of 'blog' tag to unpublish")
	}
}

func TestGetBlogPost_found(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	n, _ := models.CreateNote(db, u.ID, "My Post", "my-post", 0, true)
	models.SyncTags(db, n.ID, []string{"blog", "golang"}) //nolint:errcheck

	bp, err := models.GetBlogPost(db, "my-post")
	if err != nil {
		t.Fatalf("GetBlogPost: %v", err)
	}
	if bp.Note.Title != "My Post" {
		t.Errorf("expected title 'My Post', got %q", bp.Note.Title)
	}
	if bp.Username != "alice" {
		t.Errorf("expected username 'alice', got %q", bp.Username)
	}
	if bp.PublishedAt.IsZero() {
		t.Error("expected non-zero PublishedAt")
	}
	if len(bp.Tags) != 1 || bp.Tags[0] != "golang" {
		t.Errorf("expected tags [golang] (blog filtered), got %v", bp.Tags)
	}
}

func TestGetBlogPost_not_found_returns_error(t *testing.T) {
	db := openTestDB(t)
	models.GetOrCreateUser(db, "alice") //nolint:errcheck

	_, err := models.GetBlogPost(db, "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent blog post")
	}
}

func TestGetBlogPost_archived_returns_error(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	n, _ := models.CreateNote(db, u.ID, "Archived Blog", "archived-blog", 0, true)
	models.SyncTags(db, n.ID, []string{"blog"}) //nolint:errcheck
	models.ArchiveNote(db, u.ID, "archived-blog") //nolint:errcheck

	_, err := models.GetBlogPost(db, "archived-blog")
	if err == nil {
		t.Error("expected error for archived blog post")
	}
}

func TestListBlogPosts_ordered_by_published_at(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")

	n1, _ := models.CreateNote(db, u.ID, "First", "first", 0, true)
	n2, _ := models.CreateNote(db, u.ID, "Second", "second", 0, true)
	n3, _ := models.CreateNote(db, u.ID, "Third", "third", 0, true)

	// Insert blog_posts directly with explicit timestamps for deterministic ordering
	db.Exec(`INSERT INTO blog_posts(note_id, published_at) VALUES(?, '2026-01-01T00:00:00Z')`, n1.ID) //nolint:errcheck
	db.Exec(`INSERT INTO blog_posts(note_id, published_at) VALUES(?, '2026-02-01T00:00:00Z')`, n2.ID) //nolint:errcheck
	db.Exec(`INSERT INTO blog_posts(note_id, published_at) VALUES(?, '2026-03-01T00:00:00Z')`, n3.ID) //nolint:errcheck
	// Also need the blog tag in note_tags for consistency
	models.SyncTags(db, n1.ID, []string{"blog"}) //nolint:errcheck
	models.SyncTags(db, n2.ID, []string{"blog"}) //nolint:errcheck
	models.SyncTags(db, n3.ID, []string{"blog"}) //nolint:errcheck

	posts, err := models.ListBlogPosts(db, 1, 10)
	if err != nil {
		t.Fatalf("ListBlogPosts: %v", err)
	}
	if len(posts) != 3 {
		t.Fatalf("expected 3 posts, got %d", len(posts))
	}
	if posts[0].Note.Title != "Third" {
		t.Errorf("expected newest first, got %q", posts[0].Note.Title)
	}
	if posts[2].Note.Title != "First" {
		t.Errorf("expected oldest last, got %q", posts[2].Note.Title)
	}
}

func TestListBlogPosts_excludes_archived(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")

	n1, _ := models.CreateNote(db, u.ID, "Active", "active", 0, true)
	models.SyncTags(db, n1.ID, []string{"blog"}) //nolint:errcheck

	n2, _ := models.CreateNote(db, u.ID, "Archived", "archived", 0, true)
	models.SyncTags(db, n2.ID, []string{"blog"}) //nolint:errcheck
	models.ArchiveNote(db, u.ID, "archived") //nolint:errcheck

	posts, _ := models.ListBlogPosts(db, 1, 10)
	if len(posts) != 1 || posts[0].Note.Title != "Active" {
		t.Errorf("expected only Active, got %v", posts)
	}
}

func TestListBlogPosts_cross_user(t *testing.T) {
	db := openTestDB(t)
	alice, _ := models.GetOrCreateUser(db, "alice")
	bob, _ := models.GetOrCreateUser(db, "bob")

	n1, _ := models.CreateNote(db, alice.ID, "Alice Post", "alice-post", 0, true)
	models.SyncTags(db, n1.ID, []string{"blog"}) //nolint:errcheck

	n2, _ := models.CreateNote(db, bob.ID, "Bob Post", "bob-post", 0, true)
	models.SyncTags(db, n2.ID, []string{"blog"}) //nolint:errcheck

	posts, _ := models.ListBlogPosts(db, 1, 10)
	if len(posts) != 2 {
		t.Fatalf("expected 2 cross-user posts, got %d", len(posts))
	}
}

func TestCountBlogPosts(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")

	n1, _ := models.CreateNote(db, u.ID, "P1", "p1", 0, true)
	n2, _ := models.CreateNote(db, u.ID, "P2", "p2", 0, true)
	models.SyncTags(db, n1.ID, []string{"blog"}) //nolint:errcheck
	models.SyncTags(db, n2.ID, []string{"blog"}) //nolint:errcheck

	if got := models.CountBlogPosts(db); got != 2 {
		t.Errorf("expected 2, got %d", got)
	}
}

func TestListBlogPostsByTag(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")

	n1, _ := models.CreateNote(db, u.ID, "Go Post", "go-post", 0, true)
	models.SyncTags(db, n1.ID, []string{"blog", "golang"}) //nolint:errcheck

	n2, _ := models.CreateNote(db, u.ID, "Rust Post", "rust-post", 0, true)
	models.SyncTags(db, n2.ID, []string{"blog", "rust"}) //nolint:errcheck

	posts, err := models.ListBlogPostsByTag(db, "golang", 1, 10)
	if err != nil {
		t.Fatalf("ListBlogPostsByTag: %v", err)
	}
	if len(posts) != 1 || posts[0].Note.Title != "Go Post" {
		t.Errorf("expected [Go Post], got %v", posts)
	}
}

func TestCountBlogPostsByTag(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")

	n1, _ := models.CreateNote(db, u.ID, "A", "a", 0, true)
	n2, _ := models.CreateNote(db, u.ID, "B", "b", 0, true)
	models.SyncTags(db, n1.ID, []string{"blog", "golang"}) //nolint:errcheck
	models.SyncTags(db, n2.ID, []string{"blog", "golang"}) //nolint:errcheck

	if got := models.CountBlogPostsByTag(db, "golang"); got != 2 {
		t.Errorf("expected 2, got %d", got)
	}
	if got := models.CountBlogPostsByTag(db, "rust"); got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

func TestListBlogTags(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")

	n1, _ := models.CreateNote(db, u.ID, "P1", "p1", 0, true)
	n2, _ := models.CreateNote(db, u.ID, "P2", "p2", 0, true)
	models.SyncTags(db, n1.ID, []string{"blog", "golang", "tutorial"}) //nolint:errcheck
	models.SyncTags(db, n2.ID, []string{"blog", "golang"}) //nolint:errcheck

	tags := models.ListBlogTags(db)
	if len(tags) < 1 {
		t.Fatal("expected at least 1 tag")
	}
	for _, tc := range tags {
		if tc.Name == "blog" {
			t.Error("blog tag should be excluded from ListBlogTags")
		}
	}
	found := false
	for _, tc := range tags {
		if tc.Name == "golang" && tc.Count == 2 {
			found = true
		}
	}
	if !found {
		t.Errorf("expected golang with count 2, got %v", tags)
	}
}

func TestResolveWikiLinksForBlog_blog_target_linked(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")

	target, _ := models.CreateNote(db, u.ID, "Target", "target", 0, true)
	models.SyncTags(db, target.ID, []string{"blog"}) //nolint:errcheck

	body := "See [[Target]] for more."
	resolved := models.ResolveWikiLinksForBlog(db, u.ID, body)
	if !strings.Contains(resolved, `[Target](/blog/target)`) {
		t.Errorf("expected markdown blog link, got %q", resolved)
	}
}

func TestResolveWikiLinksForBlog_non_blog_target_plain(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	models.CreateNote(db, u.ID, "Private", "private", 0, true) //nolint:errcheck

	body := "See [[Private]] notes."
	resolved := models.ResolveWikiLinksForBlog(db, u.ID, body)
	if strings.Contains(resolved, `/blog/`) {
		t.Errorf("non-blog target should not be a link, got %q", resolved)
	}
	if !strings.Contains(resolved, "Private") {
		t.Errorf("expected plain text title preserved, got %q", resolved)
	}
}

func TestResolveWikiLinksForBlog_unknown_target_plain(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")

	body := "See [[Nonexistent]] here."
	resolved := models.ResolveWikiLinksForBlog(db, u.ID, body)
	if strings.Contains(resolved, `/blog/`) {
		t.Errorf("unknown target should not be a link, got %q", resolved)
	}
	if !strings.Contains(resolved, "Nonexistent") {
		t.Errorf("expected plain text title preserved, got %q", resolved)
	}
}

func TestGetAdjacentBlogPosts(t *testing.T) {
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")

	n1, _ := models.CreateNote(db, u.ID, "First", "first", 0, true)
	n2, _ := models.CreateNote(db, u.ID, "Middle", "middle", 0, true)
	n3, _ := models.CreateNote(db, u.ID, "Last", "last", 0, true)

	db.Exec(`INSERT INTO blog_posts(note_id, published_at) VALUES(?, '2026-01-01T00:00:00Z')`, n1.ID) //nolint:errcheck
	db.Exec(`INSERT INTO blog_posts(note_id, published_at) VALUES(?, '2026-02-01T00:00:00Z')`, n2.ID) //nolint:errcheck
	db.Exec(`INSERT INTO blog_posts(note_id, published_at) VALUES(?, '2026-03-01T00:00:00Z')`, n3.ID) //nolint:errcheck

	middle, _ := models.GetBlogPost(db, "middle")

	prev, next := models.GetAdjacentBlogPosts(db, middle.PublishedAt)
	if prev == nil || prev.Note.Title != "First" {
		t.Errorf("expected prev=First, got %v", prev)
	}
	if next == nil || next.Note.Title != "Last" {
		t.Errorf("expected next=Last, got %v", next)
	}
}
