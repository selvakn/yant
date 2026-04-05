package models_test

import (
	"testing"

	"github.com/selvakn/my-notes/internal/models"
)

// openClosedDB opens a DB, initializes it, closes it, and returns the closed DB
// to test error paths on all operations.
func openClosedDB(t *testing.T) *models.DB {
	t.Helper()
	db := openTestDB(t)
	db.Close()
	return db
}

func TestCreateUser_ClosedDBReturnsError(t *testing.T) {
	db := openClosedDB(t)
	_, err := models.CreateUser(db, "alice")
	if err == nil {
		t.Error("expected error on closed DB")
	}
}

func TestGetOrCreateUser_ClosedDBReturnsError(t *testing.T) {
	db := openClosedDB(t)
	_, err := models.GetOrCreateUser(db, "alice")
	if err == nil {
		t.Error("expected error on closed DB")
	}
}

func TestCreateNote_ClosedDBReturnsError(t *testing.T) {
	db := openClosedDB(t)
	_, err := models.CreateNote(db, 1, "Note", "note")
	if err == nil {
		t.Error("expected error on closed DB")
	}
}

func TestListNotes_ClosedDBReturnsError(t *testing.T) {
	db := openClosedDB(t)
	_, err := models.ListNotes(db, 1, "")
	if err == nil {
		t.Error("expected error on closed DB")
	}
}

func TestListNotes_ClosedDBWithTagReturnsError(t *testing.T) {
	db := openClosedDB(t)
	_, err := models.ListNotes(db, 1, "work")
	if err == nil {
		t.Error("expected error on closed DB with tag filter")
	}
}

func TestUpdateNote_ClosedDBReturnsError(t *testing.T) {
	db := openClosedDB(t)
	_, err := models.UpdateNote(db, 1, "slug", "Title")
	if err == nil {
		t.Error("expected error on closed DB")
	}
}

func TestDeleteNote_ClosedDBReturnsError(t *testing.T) {
	db := openClosedDB(t)
	err := models.DeleteNote(db, 1, "slug")
	if err == nil {
		t.Error("expected error on closed DB")
	}
}

func TestSyncTags_ClosedDBReturnsError(t *testing.T) {
	db := openClosedDB(t)
	err := models.SyncTags(db, 1, []string{"tag"})
	if err == nil {
		t.Error("expected error on closed DB")
	}
}

func TestListTagsForUser_ClosedDBReturnsError(t *testing.T) {
	db := openClosedDB(t)
	_, err := models.ListTagsForUser(db, 1)
	if err == nil {
		t.Error("expected error on closed DB")
	}
}

func TestCreateImage_ClosedDBReturnsError(t *testing.T) {
	db := openClosedDB(t)
	_, err := models.CreateImage(db, 1, "file.png", "orig.png", "image/png", 100)
	if err == nil {
		t.Error("expected error on closed DB")
	}
}

func TestGetImagesForNote_ClosedDBReturnsError(t *testing.T) {
	db := openClosedDB(t)
	_, err := models.GetImagesForNote(db, 1)
	if err == nil {
		t.Error("expected error on closed DB")
	}
}

func TestDeleteImagesForNote_ClosedDBReturnsError(t *testing.T) {
	db := openClosedDB(t)
	_, err := models.DeleteImagesForNote(db, 1)
	if err == nil {
		t.Error("expected error on closed DB")
	}
}

func TestGenerateSlug_ClosedDBReturnsError(t *testing.T) {
	db := openClosedDB(t)
	_, err := models.GenerateSlug(db, 1, "My Note")
	if err == nil {
		t.Error("expected error on closed DB")
	}
}

func TestSyncTags_ForeignKeyViolationReturnsError(t *testing.T) {
	db := openTestDB(t)
	// Use a non-existent noteID to trigger FK constraint on INSERT
	err := models.SyncTags(db, 99999, []string{"mytag"})
	if err == nil {
		t.Error("expected FK constraint error for non-existent noteID")
	}
}
