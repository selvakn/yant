package models_test

import (
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/selvakn/yant/internal/models"
)

// Helper: create user, returns User.
func createUser(t *testing.T, db *models.DB, username string) *models.User {
	t.Helper()
	u, err := models.GetOrCreateUser(db, username)
	if err != nil {
		t.Fatalf("create user %s: %v", username, err)
	}
	return u
}

// Helper: create a note owned by a user, returns noteID.
func createNote(t *testing.T, db *models.DB, ownerID int64, title string) int64 {
	t.Helper()
	slug, err := models.GenerateSlug(db, ownerID, title)
	if err != nil {
		t.Fatalf("generate slug: %v", err)
	}
	n, err := models.CreateNote(db, ownerID, title, slug)
	if err != nil {
		t.Fatalf("create note: %v", err)
	}
	return n.ID
}

func TestGrantShare_InsertsRow(t *testing.T) {
	db := openTestDB(t)
	alice := createUser(t, db, "alice")
	bob := createUser(t, db, "bob")
	noteID := createNote(t, db, alice.ID, "My Note")

	if err := models.GrantShare(db, noteID, bob.ID, alice.ID, models.PermissionEdit); err != nil {
		t.Fatalf("grant: %v", err)
	}
	perm, ok := models.GetShareByViewerAndNote(db, noteID, bob.ID)
	if !ok || perm != "edit" {
		t.Errorf("expected edit permission, got %q (ok=%v)", perm, ok)
	}
}

func TestGrantShare_UpsertOnRegrant(t *testing.T) {
	db := openTestDB(t)
	alice := createUser(t, db, "alice")
	bob := createUser(t, db, "bob")
	noteID := createNote(t, db, alice.ID, "My Note")

	_ = models.GrantShare(db, noteID, bob.ID, alice.ID, models.PermissionRead)
	_ = models.GrantShare(db, noteID, bob.ID, alice.ID, models.PermissionEdit)

	collaborators, _ := models.ListSharesForNote(db, noteID)
	if len(collaborators) != 1 {
		t.Errorf("expected 1 collaborator after upsert, got %d", len(collaborators))
	}
	if len(collaborators) > 0 && collaborators[0].Permission != "edit" {
		t.Errorf("expected upserted permission edit, got %q", collaborators[0].Permission)
	}
}

func TestGrantShare_RejectsSelfShare(t *testing.T) {
	db := openTestDB(t)
	alice := createUser(t, db, "alice")
	noteID := createNote(t, db, alice.ID, "My Note")

	err := models.GrantShare(db, noteID, alice.ID, alice.ID, models.PermissionRead)
	if !errors.Is(err, models.ErrSelfShare) {
		t.Errorf("expected ErrSelfShare, got %v", err)
	}
}

func TestGrantShare_RejectsInvalidPermission(t *testing.T) {
	db := openTestDB(t)
	alice := createUser(t, db, "alice")
	bob := createUser(t, db, "bob")
	noteID := createNote(t, db, alice.ID, "My Note")

	if err := models.GrantShare(db, noteID, bob.ID, alice.ID, "admin"); err == nil {
		t.Error("expected error for invalid permission")
	}
}

func TestRevokeShare(t *testing.T) {
	db := openTestDB(t)
	alice := createUser(t, db, "alice")
	bob := createUser(t, db, "bob")
	noteID := createNote(t, db, alice.ID, "My Note")

	_ = models.GrantShare(db, noteID, bob.ID, alice.ID, "edit")
	if err := models.RevokeShare(db, noteID, bob.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	_, ok := models.GetShareByViewerAndNote(db, noteID, bob.ID)
	if ok {
		t.Error("expected no share after revoke")
	}
}

func TestListSharedNotesForUser(t *testing.T) {
	db := openTestDB(t)
	alice := createUser(t, db, "alice")
	bob := createUser(t, db, "bob")

	note1 := createNote(t, db, alice.ID, "Shared One")
	note2 := createNote(t, db, alice.ID, "Shared Two")
	_ = createNote(t, db, alice.ID, "Private Three")

	_ = models.GrantShare(db, note1, bob.ID, alice.ID, "read")
	_ = models.GrantShare(db, note2, bob.ID, alice.ID, "edit")

	list, err := models.ListSharedNotesForUser(db, bob.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 shared, got %d", len(list))
	}
	for _, s := range list {
		if s.OwnerUsername != "alice" {
			t.Errorf("expected owner alice, got %q", s.OwnerUsername)
		}
	}
}

func TestCountSharedNotesForUser(t *testing.T) {
	db := openTestDB(t)
	alice := createUser(t, db, "alice")
	bob := createUser(t, db, "bob")

	if n := models.CountSharedNotesForUser(db, bob.ID); n != 0 {
		t.Errorf("expected 0, got %d", n)
	}

	note1 := createNote(t, db, alice.ID, "Shared One")
	_ = models.GrantShare(db, note1, bob.ID, alice.ID, "read")

	if n := models.CountSharedNotesForUser(db, bob.ID); n != 1 {
		t.Errorf("expected 1, got %d", n)
	}
}

func TestGetNoteForViewer_Owner(t *testing.T) {
	db := openTestDB(t)
	alice := createUser(t, db, "alice")
	_ = createNote(t, db, alice.ID, "My Note")

	note, role, err := models.GetNoteForViewer(db, alice.ID, "alice", "my-note")
	if err != nil {
		t.Fatal(err)
	}
	if note == nil {
		t.Fatal("expected note")
	}
	if role != models.RoleOwner {
		t.Errorf("expected role owner, got %q", role)
	}
}

func TestGetNoteForViewer_EditorShare(t *testing.T) {
	db := openTestDB(t)
	alice := createUser(t, db, "alice")
	bob := createUser(t, db, "bob")
	noteID := createNote(t, db, alice.ID, "Team Note")
	_ = models.GrantShare(db, noteID, bob.ID, alice.ID, "edit")

	note, role, err := models.GetNoteForViewer(db, bob.ID, "alice", "team-note")
	if err != nil {
		t.Fatal(err)
	}
	if note == nil {
		t.Fatal("expected note")
	}
	if role != models.RoleEditor {
		t.Errorf("expected editor, got %q", role)
	}
}

func TestGetNoteForViewer_ReaderShare(t *testing.T) {
	db := openTestDB(t)
	alice := createUser(t, db, "alice")
	bob := createUser(t, db, "bob")
	noteID := createNote(t, db, alice.ID, "Doc")
	_ = models.GrantShare(db, noteID, bob.ID, alice.ID, "read")

	_, role, err := models.GetNoteForViewer(db, bob.ID, "alice", "doc")
	if err != nil {
		t.Fatal(err)
	}
	if role != models.RoleReader {
		t.Errorf("expected reader, got %q", role)
	}
}

func TestGetNoteForViewer_NoAccess(t *testing.T) {
	db := openTestDB(t)
	alice := createUser(t, db, "alice")
	_ = createUser(t, db, "bob")
	_ = createNote(t, db, alice.ID, "Private")

	bob, _ := models.GetUserByUsername(db, "bob")
	_, _, err := models.GetNoteForViewer(db, bob.ID, "alice", "private")
	if !errors.Is(err, models.ErrNoAccess) {
		t.Errorf("expected ErrNoAccess, got %v", err)
	}
}

func TestGetNoteForViewer_NotFound(t *testing.T) {
	db := openTestDB(t)
	alice := createUser(t, db, "alice")

	_, _, err := models.GetNoteForViewer(db, alice.ID, "alice", "nonexistent")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestResolveWikiLinksForViewer_SharedTargetsAsLinks(t *testing.T) {
	db := openTestDB(t)
	alice := createUser(t, db, "alice")
	bob := createUser(t, db, "bob")
	targetID := createNote(t, db, alice.ID, "Shared Target")
	_ = models.GrantShare(db, targetID, bob.ID, alice.ID, "read")

	body := "See [[Shared Target]] for details."
	result := models.ResolveWikiLinksForViewer(db, bob.ID, alice.ID, "alice", body)
	expected := "[Shared Target](/shared/alice/shared-target)"
	if !strings.Contains(result, expected) {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestResolveWikiLinksForViewer_PrivateTargetsAsPlainText(t *testing.T) {
	db := openTestDB(t)
	alice := createUser(t, db, "alice")
	bob := createUser(t, db, "bob")
	_ = createNote(t, db, alice.ID, "Private Target") // NOT shared with bob

	body := "See [[Private Target]] for details."
	result := models.ResolveWikiLinksForViewer(db, bob.ID, alice.ID, "alice", body)
	if strings.Contains(result, "/shared/") || strings.Contains(result, "/notes/") {
		t.Errorf("private target must render as plain text: %q", result)
	}
	if !strings.Contains(result, "Private Target") {
		t.Errorf("expected 'Private Target' as plain text, got %q", result)
	}
}

func TestListShareCountsForOwner(t *testing.T) {
	db := openTestDB(t)
	alice := createUser(t, db, "alice")
	bob := createUser(t, db, "bob")
	carol := createUser(t, db, "carol")

	note1 := createNote(t, db, alice.ID, "Note One")
	note2 := createNote(t, db, alice.ID, "Note Two")
	note3 := createNote(t, db, alice.ID, "Note Three") // not shared

	_ = models.GrantShare(db, note1, bob.ID, alice.ID, "edit")
	_ = models.GrantShare(db, note1, carol.ID, alice.ID, "read")
	_ = models.GrantShare(db, note2, bob.ID, alice.ID, "read")

	counts, err := models.ListShareCountsForOwner(db, alice.ID)
	if err != nil {
		t.Fatalf("ListShareCountsForOwner: %v", err)
	}
	if counts[note1] != 2 {
		t.Errorf("note1: expected 2 collaborators, got %d", counts[note1])
	}
	if counts[note2] != 1 {
		t.Errorf("note2: expected 1 collaborator, got %d", counts[note2])
	}
	if _, exists := counts[note3]; exists {
		t.Errorf("unshared note3 should not appear in counts")
	}
}

func TestCascadeOnNoteDelete(t *testing.T) {
	db := openTestDB(t)
	alice := createUser(t, db, "alice")
	bob := createUser(t, db, "bob")
	noteID := createNote(t, db, alice.ID, "To Delete")
	_ = models.GrantShare(db, noteID, bob.ID, alice.ID, "read")

	_, err := db.Exec("DELETE FROM notes WHERE id = ?", noteID)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := models.GetShareByViewerAndNote(db, noteID, bob.ID); ok {
		t.Error("share row should cascade-delete with parent note")
	}
}
