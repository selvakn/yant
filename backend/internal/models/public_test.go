package models_test

import (
	"strings"
	"testing"

	"github.com/selvakn/yant/internal/models"
)

// Helper: create a user and note, returns noteID.
func createUserAndNote(t *testing.T, db *models.DB, username, title string) int64 {
	t.Helper()
	u, err := models.GetOrCreateUser(db, username)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	slug, err := models.GenerateSlug(db, u.ID, title)
	if err != nil {
		t.Fatalf("generate slug: %v", err)
	}
	n, err := models.CreateNote(db, u.ID, title, slug)
	if err != nil {
		t.Fatalf("create note: %v", err)
	}
	return n.ID
}

func TestGenerateShareToken_Unique(t *testing.T) {
	t1, err := models.GenerateShareToken()
	if err != nil {
		t.Fatal(err)
	}
	t2, err := models.GenerateShareToken()
	if err != nil {
		t.Fatal(err)
	}
	if t1 == t2 {
		t.Errorf("expected distinct tokens, both got %q", t1)
	}
	if len(t1) < 20 {
		t.Errorf("token too short: %q (len=%d)", t1, len(t1))
	}
	// URL-safe: no '+', '/', or '='
	if strings.ContainsAny(t1, "+/=") {
		t.Errorf("token should be URL-safe: %q", t1)
	}
}

func TestPublishNote_GeneratesTokenOnFirstPublish(t *testing.T) {
	db := openTestDB(t)
	noteID := createUserAndNote(t, db, "alice", "My Note")

	token, err := models.PublishNote(db, noteID)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	share, err := models.GetPublicShare(db, noteID)
	if err != nil || share == nil {
		t.Fatalf("expected share, got %v, err=%v", share, err)
	}
	if !share.Published {
		t.Error("expected published=true")
	}
	if share.Token != token {
		t.Errorf("token mismatch: got %q, want %q", share.Token, token)
	}
}

func TestPublishNote_ReusesTokenOnRepublish(t *testing.T) {
	db := openTestDB(t)
	noteID := createUserAndNote(t, db, "alice", "My Note")

	token1, _ := models.PublishNote(db, noteID)
	_ = models.UnpublishNote(db, noteID)
	token2, err := models.PublishNote(db, noteID)
	if err != nil {
		t.Fatal(err)
	}
	if token1 != token2 {
		t.Errorf("token should persist across toggles: %q != %q", token1, token2)
	}
}

func TestUnpublishNote_KeepsTokenButMarksPrivate(t *testing.T) {
	db := openTestDB(t)
	noteID := createUserAndNote(t, db, "alice", "My Note")

	token, _ := models.PublishNote(db, noteID)
	_ = models.UnpublishNote(db, noteID)

	share, err := models.GetPublicShare(db, noteID)
	if err != nil || share == nil {
		t.Fatalf("row should still exist after unpublish")
	}
	if share.Published {
		t.Error("expected published=false after unpublish")
	}
	if share.Token != token {
		t.Errorf("token should persist: got %q, want %q", share.Token, token)
	}
}

func TestGetNoteByToken_Published(t *testing.T) {
	db := openTestDB(t)
	noteID := createUserAndNote(t, db, "alice", "My Note")

	token, _ := models.PublishNote(db, noteID)

	note, err := models.GetNoteByToken(db, token)
	if err != nil {
		t.Fatal(err)
	}
	if note == nil {
		t.Fatal("expected note, got nil")
	}
	if note.ID != noteID {
		t.Errorf("wrong note ID: got %d, want %d", note.ID, noteID)
	}
}

func TestGetNoteByToken_UnpublishedReturnsNil(t *testing.T) {
	db := openTestDB(t)
	noteID := createUserAndNote(t, db, "alice", "My Note")

	token, _ := models.PublishNote(db, noteID)
	_ = models.UnpublishNote(db, noteID)

	note, err := models.GetNoteByToken(db, token)
	if err != nil {
		t.Fatal(err)
	}
	if note != nil {
		t.Errorf("expected nil for unpublished, got note ID %d", note.ID)
	}
}

func TestGetNoteByToken_ArchivedReturnsNil(t *testing.T) {
	db := openTestDB(t)
	noteID := createUserAndNote(t, db, "alice", "My Note")

	token, _ := models.PublishNote(db, noteID)
	// Archive the note
	_, err := db.Exec("UPDATE notes SET archived = 1 WHERE id = ?", noteID)
	if err != nil {
		t.Fatal(err)
	}

	note, err := models.GetNoteByToken(db, token)
	if err != nil {
		t.Fatal(err)
	}
	if note != nil {
		t.Error("expected nil for archived, got note")
	}
}

func TestGetNoteByToken_UnknownReturnsNil(t *testing.T) {
	db := openTestDB(t)
	note, err := models.GetNoteByToken(db, "nonexistent-token")
	if err != nil {
		t.Fatal(err)
	}
	if note != nil {
		t.Error("expected nil for unknown token")
	}
}

func TestListPublishedNotes(t *testing.T) {
	db := openTestDB(t)
	id1 := createUserAndNote(t, db, "alice", "Note One")
	id2 := createUserAndNote(t, db, "alice", "Note Two")
	_ = createUserAndNote(t, db, "alice", "Note Three") // stays private

	_, _ = models.PublishNote(db, id1)
	_, _ = models.PublishNote(db, id2)

	user, _ := models.GetUserByUsername(db, "alice")
	list, err := models.ListPublishedNotes(db, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 published, got %d", len(list))
	}
	// Verify id3 not in list (by title check)
	for _, s := range list {
		if s.Title == "Note Three" {
			t.Error("Note Three should not be in published list")
		}
	}
}

func TestCountPublishedNotes(t *testing.T) {
	db := openTestDB(t)
	id1 := createUserAndNote(t, db, "alice", "Note One")
	id2 := createUserAndNote(t, db, "alice", "Note Two")

	user, _ := models.GetUserByUsername(db, "alice")
	if n := models.CountPublishedNotes(db, user.ID); n != 0 {
		t.Errorf("expected 0, got %d", n)
	}

	_, _ = models.PublishNote(db, id1)
	_, _ = models.PublishNote(db, id2)

	if n := models.CountPublishedNotes(db, user.ID); n != 2 {
		t.Errorf("expected 2, got %d", n)
	}

	_ = models.UnpublishNote(db, id1)

	if n := models.CountPublishedNotes(db, user.ID); n != 1 {
		t.Errorf("expected 1 after unpublish, got %d", n)
	}
}

func TestDeleteCascadesPublicNote(t *testing.T) {
	db := openTestDB(t)
	noteID := createUserAndNote(t, db, "alice", "My Note")
	_, _ = models.PublishNote(db, noteID)

	_, err := db.Exec("DELETE FROM notes WHERE id = ?", noteID)
	if err != nil {
		t.Fatal(err)
	}

	share, err := models.GetPublicShare(db, noteID)
	if err != nil {
		t.Fatal(err)
	}
	if share != nil {
		t.Error("public_notes row should cascade-delete with parent note")
	}
}

func TestResolveWikiLinksPublic_PrivateTargetsAsPlainText(t *testing.T) {
	db := openTestDB(t)
	_ = createUserAndNote(t, db, "alice", "Private Target")
	user, _ := models.GetUserByUsername(db, "alice")

	body := "Some text [[Private Target]] here."
	result := models.ResolveWikiLinksPublic(db, user.ID, body)

	if strings.Contains(result, "(/p/") || strings.Contains(result, "(/notes/") {
		t.Errorf("private target should not produce a link: %q", result)
	}
	if !strings.Contains(result, "Private Target") {
		t.Errorf("expected plain text 'Private Target' in output, got: %q", result)
	}
}

func TestResolveWikiLinksPublic_PublicTargetsAsLinks(t *testing.T) {
	db := openTestDB(t)
	publicID := createUserAndNote(t, db, "alice", "Public Target")
	token, _ := models.PublishNote(db, publicID)
	user, _ := models.GetUserByUsername(db, "alice")

	body := "Link to [[Public Target]]!"
	result := models.ResolveWikiLinksPublic(db, user.ID, body)

	expected := "[Public Target](/p/" + token + ")"
	if !strings.Contains(result, expected) {
		t.Errorf("expected %q in result, got: %q", expected, result)
	}
}
