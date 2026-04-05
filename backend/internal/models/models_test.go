package models_test

import (
	"testing"
	"time"

	"github.com/selvakn/my-notes/internal/models"
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

	aliceNotes, _ := models.ListNotes(db, alice.ID, "")
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

	tags, err := models.ListTagsForUser(db, u.ID)
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

	tags, _ := models.ListTagsForUser(db, alice.ID)
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
