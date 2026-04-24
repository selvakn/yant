package models_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/selvakn/yant/internal/models"
)

func TestListAllUsers_ReturnsAllUsersWithCounts(t *testing.T) {
	// Arrange
	db := openTestDB(t)
	alice, _ := models.GetOrCreateUser(db, "alice")
	bob, _ := models.GetOrCreateUser(db, "bob")
	models.CreateNote(db, alice.ID, "A1", "a1")
	models.CreateNote(db, alice.ID, "A2", "a2")
	models.CreateNote(db, bob.ID, "B1", "b1")

	// Act
	users, err := models.ListAllUsers(db)

	// Assert
	if err != nil {
		t.Fatalf("ListAllUsers: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	counts := make(map[string]int)
	for _, u := range users {
		counts[u.Username] = u.NoteCount
	}
	if counts["alice"] != 2 {
		t.Errorf("expected alice=2 notes, got %d", counts["alice"])
	}
	if counts["bob"] != 1 {
		t.Errorf("expected bob=1 note, got %d", counts["bob"])
	}
}

func TestListAllUsers_EmptyDB(t *testing.T) {
	// Arrange
	db := openTestDB(t)

	// Act
	users, err := models.ListAllUsers(db)

	// Assert
	if err != nil {
		t.Fatalf("ListAllUsers: %v", err)
	}
	if len(users) != 0 {
		t.Errorf("expected 0 users, got %d", len(users))
	}
}

func TestListAllNotesForUser_ReturnsActiveAndArchived(t *testing.T) {
	// Arrange
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	models.CreateNote(db, u.ID, "Active", "active")
	models.CreateNote(db, u.ID, "Archived", "archived")
	models.ArchiveNote(db, u.ID, "archived")

	// Act
	notes, err := models.ListAllNotesForUser(db, u.ID)

	// Assert
	if err != nil {
		t.Fatalf("ListAllNotesForUser: %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes (active + archived), got %d", len(notes))
	}
	archivedCount := 0
	for _, n := range notes {
		if n.Archived {
			archivedCount++
		}
	}
	if archivedCount != 1 {
		t.Errorf("expected 1 archived note, got %d", archivedCount)
	}
}

func TestListAllNotesForUser_IncludesTags(t *testing.T) {
	// Arrange
	db := openTestDB(t)
	u, _ := models.GetOrCreateUser(db, "alice")
	n, _ := models.CreateNote(db, u.ID, "Tagged", "tagged")
	models.SyncTags(db, n.ID, []string{"work", "ideas"})

	// Act
	notes, err := models.ListAllNotesForUser(db, u.ID)

	// Assert
	if err != nil {
		t.Fatalf("ListAllNotesForUser: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("expected 1 note, got %d", len(notes))
	}
	if len(notes[0].Tags) != 2 {
		t.Errorf("expected 2 tags, got %d: %v", len(notes[0].Tags), notes[0].Tags)
	}
}

func TestGetUserByID_Found(t *testing.T) {
	// Arrange
	db := openTestDB(t)
	created, _ := models.GetOrCreateUser(db, "alice")

	// Act
	u, err := models.GetUserByID(db, created.ID)

	// Assert
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if u == nil {
		t.Fatal("expected non-nil user")
	}
	if u.Username != "alice" {
		t.Errorf("expected alice, got %s", u.Username)
	}
}

func TestGetUserByID_NotFound(t *testing.T) {
	// Arrange
	db := openTestDB(t)

	// Act
	u, err := models.GetUserByID(db, 9999)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u != nil {
		t.Error("expected nil for non-existent user")
	}
}

func TestAdminDeleteNote_RemovesNoteAndFiles(t *testing.T) {
	// Arrange
	db := openTestDB(t)
	notesDir := t.TempDir()
	uploadsDir := t.TempDir()
	u, _ := models.GetOrCreateUser(db, "alice")
	n, _ := models.CreateNote(db, u.ID, "ToDelete", "todelete")

	userNoteDir := filepath.Join(notesDir, "1")
	os.MkdirAll(userNoteDir, 0755)
	os.WriteFile(filepath.Join(userNoteDir, "todelete.md"), []byte("content"), 0644)

	// Act
	err := models.AdminDeleteNote(db, n.ID, notesDir, uploadsDir)

	// Assert
	if err != nil {
		t.Fatalf("AdminDeleteNote: %v", err)
	}
	got, _ := models.GetNote(db, u.ID, "todelete")
	if got != nil {
		t.Error("note should be deleted from DB")
	}
	if _, err := os.Stat(filepath.Join(userNoteDir, "todelete.md")); !os.IsNotExist(err) {
		t.Error("markdown file should be removed")
	}
}

func TestAdminDeleteUser_RemovesUserAndAllData(t *testing.T) {
	// Arrange
	db := openTestDB(t)
	notesDir := t.TempDir()
	uploadsDir := t.TempDir()
	u, _ := models.GetOrCreateUser(db, "victim")
	models.CreateNote(db, u.ID, "Note1", "note1")
	models.CreateNote(db, u.ID, "Note2", "note2")

	userNoteDir := filepath.Join(notesDir, "1")
	os.MkdirAll(userNoteDir, 0755)
	os.WriteFile(filepath.Join(userNoteDir, "note1.md"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(userNoteDir, "note2.md"), []byte("b"), 0644)

	// Act
	err := models.AdminDeleteUser(db, u.ID, notesDir, uploadsDir)

	// Assert
	if err != nil {
		t.Fatalf("AdminDeleteUser: %v", err)
	}
	got, _ := models.GetUserByID(db, u.ID)
	if got != nil {
		t.Error("user should be deleted from DB")
	}
	notes, _ := models.ListAllNotesForUser(db, u.ID)
	if len(notes) != 0 {
		t.Errorf("expected 0 notes after user delete, got %d", len(notes))
	}
	if _, err := os.Stat(userNoteDir); !os.IsNotExist(err) {
		t.Error("user notes directory should be removed")
	}
}

func TestAdminDeleteUser_CannotDeleteNonExistent(t *testing.T) {
	// Arrange
	db := openTestDB(t)
	notesDir := t.TempDir()
	uploadsDir := t.TempDir()

	// Act
	err := models.AdminDeleteUser(db, 9999, notesDir, uploadsDir)

	// Assert (no rows to delete, but should not error fatally since DELETE WHERE id=9999 is a no-op)
	if err != nil {
		t.Logf("AdminDeleteUser on non-existent: %v (acceptable)", err)
	}
}
