package models

import (
	"html/template"
	"os"
	"path/filepath"
	"testing"

	"github.com/selvakn/my-notes/internal/storage"
)

func TestScoreNote_TitleMatch(t *testing.T) {
	note := &Note{Title: "Meeting Notes", Tags: []string{}}
	score := ScoreNote("meeting", note, "some body content")

	if score == 0 {
		t.Error("expected non-zero score for title match")
	}
}

func TestScoreNote_TagMatch(t *testing.T) {
	note := &Note{Title: "Random Title", Tags: []string{"work", "important"}}
	score := ScoreNote("work", note, "some body content")

	if score == 0 {
		t.Error("expected non-zero score for tag match")
	}
}

func TestScoreNote_BodyMatch(t *testing.T) {
	note := &Note{Title: "Random Title", Tags: []string{}}
	score := ScoreNote("chocolate", note, "make chocolate cake")

	if score == 0 {
		t.Error("expected non-zero score for body match")
	}
}

func TestScoreNote_NoMatch(t *testing.T) {
	note := &Note{Title: "Random Title", Tags: []string{"tag1"}}
	score := ScoreNote("xyz123", note, "some content")

	if score != 0 {
		t.Errorf("expected zero score for no match, got %d", score)
	}
}

func TestScoreNote_TitleWeightHigherThanBody(t *testing.T) {
	note1 := &Note{Title: "Meeting Notes", Tags: []string{}}
	note2 := &Note{Title: "Random", Tags: []string{}}

	titleScore := ScoreNote("meeting", note1, "nothing here")
	bodyScore := ScoreNote("meeting", note2, "meeting content")

	if titleScore <= bodyScore {
		t.Errorf("title match (%d) should score higher than body match (%d)", titleScore, bodyScore)
	}
}

func TestHighlightMatch_NoQuery(t *testing.T) {
	result := HighlightMatch("Hello World", "")
	expected := template.HTML("Hello World")
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestHighlightMatch_ExactMatch(t *testing.T) {
	result := HighlightMatch("Hello World", "World")
	expected := template.HTML("Hello <mark>World</mark>")
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestHighlightMatch_CaseInsensitive(t *testing.T) {
	result := HighlightMatch("Hello World", "world")
	expected := template.HTML("Hello <mark>World</mark>")
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestHighlightMatch_NoMatch(t *testing.T) {
	result := HighlightMatch("Hello World", "xyz")
	expected := template.HTML("Hello World")
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestHighlightMatch_HTMLEscaping(t *testing.T) {
	result := HighlightMatch("<script>alert('xss')</script>", "script")
	if string(result) == "<script>alert('xss')</script>" {
		t.Error("expected HTML to be escaped")
	}
	if !containsString(string(result), "&lt;") {
		t.Error("expected HTML entities in output")
	}
}

func TestBodySnippet_Empty(t *testing.T) {
	result := BodySnippet("", "test", 100)
	if result != "" {
		t.Errorf("expected empty, got %q", result)
	}
}

func TestBodySnippet_NoQuery(t *testing.T) {
	result := BodySnippet("Hello World", "", 100)
	expected := template.HTML("Hello World")
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestBodySnippet_TruncatesLongText(t *testing.T) {
	longText := "This is a very long text that should be truncated"
	result := BodySnippet(longText, "", 20)
	if len(string(result)) >= len(longText) {
		t.Error("expected truncated result")
	}
	if !containsString(string(result), "…") {
		t.Error("expected ellipsis in truncated result")
	}
}

func TestBodySnippet_IncludesHighlight(t *testing.T) {
	result := BodySnippet("Some text with meeting in it", "meeting", 100)
	if !containsString(string(result), "<mark>") {
		t.Error("expected highlight mark in snippet")
	}
}

func TestSearchNotes_EmptyQuery_ReturnsAll(t *testing.T) {
	db, notesDir := setupSearchTest(t)

	user, _ := CreateUser(db, "testuser")
	createTestNote(t, db, notesDir, user.ID, "note1", "First Note", "Content 1")
	createTestNote(t, db, notesDir, user.ID, "note2", "Second Note", "Content 2")

	results, err := SearchNotes(db, notesDir, user.ID, "")
	if err != nil {
		t.Fatalf("SearchNotes error: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestSearchNotes_FiltersByTitle(t *testing.T) {
	db, notesDir := setupSearchTest(t)

	user, _ := CreateUser(db, "testuser")
	createTestNote(t, db, notesDir, user.ID, "meeting", "Meeting Notes", "agenda items")
	createTestNote(t, db, notesDir, user.ID, "shopping", "Shopping List", "milk eggs bread")

	results, err := SearchNotes(db, notesDir, user.ID, "meeting")
	if err != nil {
		t.Fatalf("SearchNotes error: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	if len(results) > 0 && results[0].Note.Title != "Meeting Notes" {
		t.Errorf("expected Meeting Notes, got %s", results[0].Note.Title)
	}
}

func TestSearchNotes_SortsByScore(t *testing.T) {
	db, notesDir := setupSearchTest(t)

	user, _ := CreateUser(db, "testuser")
	createTestNote(t, db, notesDir, user.ID, "body-match", "Random Title", "meeting content here")
	createTestNote(t, db, notesDir, user.ID, "title-match", "Meeting Notes", "other stuff")

	results, err := SearchNotes(db, notesDir, user.ID, "meeting")
	if err != nil {
		t.Fatalf("SearchNotes error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Title match should come first (higher score)
	if results[0].Note.Slug != "title-match" {
		t.Errorf("expected title match first, got %s", results[0].Note.Slug)
	}
}

func TestSearchNotes_LongQueryTruncated(t *testing.T) {
	db, notesDir := setupSearchTest(t)

	user, _ := CreateUser(db, "testuser")
	createTestNote(t, db, notesDir, user.ID, "note1", "Test Note", "content")

	// Very long query should not cause errors
	longQuery := make([]byte, 300)
	for i := range longQuery {
		longQuery[i] = 'a'
	}

	_, err := SearchNotes(db, notesDir, user.ID, string(longQuery))
	if err != nil {
		t.Fatalf("SearchNotes with long query error: %v", err)
	}
}

func TestHighlightTags(t *testing.T) {
	tags := []string{"work", "meeting", "important"}
	result := HighlightTags(tags, "meet")

	if len(result) != 3 {
		t.Fatalf("expected 3 tags, got %d", len(result))
	}

	// "meeting" should have highlight
	if !containsString(string(result[1]), "<mark>") {
		t.Error("expected highlight in 'meeting' tag")
	}
}

// ── Test helpers ──────────────────────────────────────────────────────────────

func setupSearchTest(t *testing.T) (*DB, string) {
	t.Helper()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := InitSchema(db); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	notesDir := t.TempDir()
	return db, notesDir
}

func createTestNote(t *testing.T, db *DB, notesDir string, userID int64, slug, title, body string) {
	t.Helper()
	note, err := CreateNote(db, userID, title, slug)
	if err != nil {
		t.Fatalf("create note: %v", err)
	}

	userDir := filepath.Join(notesDir, "1")
	os.MkdirAll(userDir, 0755) //nolint:errcheck

	if err := storage.WriteNote(notesDir, userID, slug, body); err != nil {
		t.Fatalf("write note: %v", err)
	}

	tags := ParseTags(body)
	if len(tags) > 0 {
		_ = SyncTags(db, note.ID, tags)
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
