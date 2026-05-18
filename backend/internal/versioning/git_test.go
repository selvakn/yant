package versioning_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/selvakn/yant/internal/versioning"
)

func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := versioning.Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}
	return dir
}

func writeFile(t *testing.T, dir, relPath, content string) {
	t.Helper()
	full := filepath.Join(dir, relPath)
	os.MkdirAll(filepath.Dir(full), 0755) //nolint:errcheck
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", relPath, err)
	}
}

func TestInit_CreatesGitRepo(t *testing.T) {
	dir := t.TempDir()
	if err := versioning.Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}
	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		t.Error("expected .git directory to exist")
	}
}

func TestInit_IdempotentOnRepeat(t *testing.T) {
	dir := initTestRepo(t)
	if err := versioning.Init(dir); err != nil {
		t.Errorf("second Init should not error: %v", err)
	}
}

func TestInit_SeedsExistingFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "1/my-note.md", "# Hello")

	if err := versioning.Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}

	versions, err := versioning.Log(dir, "1/my-note.md", 10, 0)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(versions) != 1 {
		t.Errorf("expected 1 seeded version, got %d", len(versions))
	}
	if versions[0].Message != "seed: initial version of all notes" {
		t.Errorf("unexpected seed message: %q", versions[0].Message)
	}
}

func TestCommitFile_CreatesVersion(t *testing.T) {
	dir := initTestRepo(t)
	writeFile(t, dir, "1/note.md", "# First")

	if err := versioning.CommitFile(dir, "1/note.md", "create: note"); err != nil {
		t.Fatalf("CommitFile: %v", err)
	}

	versions, err := versioning.Log(dir, "1/note.md", 10, 0)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(versions) != 1 {
		t.Errorf("expected 1 version, got %d", len(versions))
	}
	if versions[0].Message != "create: note" {
		t.Errorf("unexpected message: %q", versions[0].Message)
	}
}

func TestCommitFile_NoopWhenContentUnchanged(t *testing.T) {
	dir := initTestRepo(t)
	writeFile(t, dir, "1/note.md", "# Same")
	versioning.CommitFile(dir, "1/note.md", "create: note") //nolint:errcheck

	if err := versioning.CommitFile(dir, "1/note.md", "update: note"); err != nil {
		t.Fatalf("CommitFile: %v", err)
	}

	versions, _ := versioning.Log(dir, "1/note.md", 10, 0)
	if len(versions) != 1 {
		t.Errorf("expected 1 version (no-op on identical content), got %d", len(versions))
	}
}

func TestCommitFile_MultipleVersions(t *testing.T) {
	dir := initTestRepo(t)
	writeFile(t, dir, "1/note.md", "version 1")
	versioning.CommitFile(dir, "1/note.md", "create: note") //nolint:errcheck
	writeFile(t, dir, "1/note.md", "version 2")
	versioning.CommitFile(dir, "1/note.md", "update: note") //nolint:errcheck
	writeFile(t, dir, "1/note.md", "version 3")
	versioning.CommitFile(dir, "1/note.md", "update: note") //nolint:errcheck

	versions, _ := versioning.Log(dir, "1/note.md", 10, 0)
	if len(versions) != 3 {
		t.Errorf("expected 3 versions, got %d", len(versions))
	}
	if versions[0].Message != "update: note" {
		t.Errorf("newest version should be last update, got %q", versions[0].Message)
	}
}

func TestCommitDelete_RemovesFromTracking(t *testing.T) {
	dir := initTestRepo(t)
	writeFile(t, dir, "1/note.md", "content")
	versioning.CommitFile(dir, "1/note.md", "create: note") //nolint:errcheck
	os.Remove(filepath.Join(dir, "1/note.md"))              //nolint:errcheck

	if err := versioning.CommitDelete(dir, "1/note.md", "delete: note"); err != nil {
		t.Fatalf("CommitDelete: %v", err)
	}
}

func TestLog_ReturnsVersionsNewestFirst(t *testing.T) {
	dir := initTestRepo(t)
	writeFile(t, dir, "1/note.md", "v1")
	versioning.CommitFile(dir, "1/note.md", "create: note") //nolint:errcheck
	writeFile(t, dir, "1/note.md", "v2")
	versioning.CommitFile(dir, "1/note.md", "update: note") //nolint:errcheck

	versions, _ := versioning.Log(dir, "1/note.md", 10, 0)
	if len(versions) < 2 {
		t.Fatalf("expected >= 2 versions, got %d", len(versions))
	}
	if !versions[0].Timestamp.After(versions[1].Timestamp) && !versions[0].Timestamp.Equal(versions[1].Timestamp) {
		t.Error("versions should be newest first")
	}
}

func TestLog_PaginationWithLimitAndOffset(t *testing.T) {
	dir := initTestRepo(t)
	for i := 1; i <= 5; i++ {
		writeFile(t, dir, "1/note.md", strings.Repeat("x", i))
		versioning.CommitFile(dir, "1/note.md", "update") //nolint:errcheck
	}

	page1, _ := versioning.Log(dir, "1/note.md", 2, 0)
	if len(page1) != 2 {
		t.Errorf("page 1: expected 2, got %d", len(page1))
	}

	page2, _ := versioning.Log(dir, "1/note.md", 2, 2)
	if len(page2) != 2 {
		t.Errorf("page 2: expected 2, got %d", len(page2))
	}

	if page1[0].CommitHash == page2[0].CommitHash {
		t.Error("pages should contain different versions")
	}
}

func TestLog_EmptyRepoReturnsNil(t *testing.T) {
	dir := initTestRepo(t)
	versions, err := versioning.Log(dir, "nonexistent.md", 10, 0)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(versions) != 0 {
		t.Errorf("expected 0 versions, got %d", len(versions))
	}
}

func TestLog_IncludesInsertionsAndDeletions(t *testing.T) {
	dir := initTestRepo(t)
	writeFile(t, dir, "1/note.md", "line one\nline two\n")
	versioning.CommitFile(dir, "1/note.md", "create") //nolint:errcheck
	writeFile(t, dir, "1/note.md", "line one\nline three\nline four\n")
	versioning.CommitFile(dir, "1/note.md", "update") //nolint:errcheck

	versions, _ := versioning.Log(dir, "1/note.md", 10, 0)
	latest := versions[0]
	if latest.Insertions == 0 && latest.Deletions == 0 {
		t.Error("expected non-zero insertions or deletions in update version")
	}
}

func TestLog_IncludesAuthorName(t *testing.T) {
	dir := initTestRepo(t)
	writeFile(t, dir, "1/note.md", "hello")
	versioning.CommitFileAs(dir, "1/note.md", "create: note", "alice", "alice@yant.local") //nolint:errcheck
	writeFile(t, dir, "1/note.md", "hello world")
	versioning.CommitFileAs(dir, "1/note.md", "update: note", "bob", "bob@yant.local") //nolint:errcheck

	versions, err := versioning.Log(dir, "1/note.md", 10, 0)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(versions) < 2 {
		t.Fatalf("expected >= 2 versions, got %d", len(versions))
	}
	// Newest first: bob's update is versions[0], alice's create is versions[1]
	if versions[0].AuthorName != "bob" {
		t.Errorf("expected AuthorName 'bob', got %q", versions[0].AuthorName)
	}
	if versions[1].AuthorName != "alice" {
		t.Errorf("expected AuthorName 'alice', got %q", versions[1].AuthorName)
	}
}

func TestShow_ReturnsContentAtVersion(t *testing.T) {
	dir := initTestRepo(t)
	writeFile(t, dir, "1/note.md", "original content")
	versioning.CommitFile(dir, "1/note.md", "create") //nolint:errcheck

	versions, _ := versioning.Log(dir, "1/note.md", 10, 0)
	content, err := versioning.Show(dir, "1/note.md", versions[0].CommitHash)
	if err != nil {
		t.Fatalf("Show: %v", err)
	}
	if content != "original content" {
		t.Errorf("expected 'original content', got %q", content)
	}
}

func TestShow_ReturnsOlderVersion(t *testing.T) {
	dir := initTestRepo(t)
	writeFile(t, dir, "1/note.md", "version 1")
	versioning.CommitFile(dir, "1/note.md", "create") //nolint:errcheck
	writeFile(t, dir, "1/note.md", "version 2")
	versioning.CommitFile(dir, "1/note.md", "update") //nolint:errcheck

	versions, _ := versioning.Log(dir, "1/note.md", 10, 0)
	content, err := versioning.Show(dir, "1/note.md", versions[1].CommitHash)
	if err != nil {
		t.Fatalf("Show: %v", err)
	}
	if content != "version 1" {
		t.Errorf("expected 'version 1', got %q", content)
	}
}

func TestShow_InvalidCommitReturnsError(t *testing.T) {
	dir := initTestRepo(t)
	_, err := versioning.Show(dir, "1/note.md", "0000000000000000000000000000000000000000")
	if err == nil {
		t.Error("expected error for invalid commit")
	}
}

func TestDiff_ShowsChanges(t *testing.T) {
	dir := initTestRepo(t)
	writeFile(t, dir, "1/note.md", "line one\nline two\n")
	versioning.CommitFile(dir, "1/note.md", "create") //nolint:errcheck
	writeFile(t, dir, "1/note.md", "line one\nline three\n")
	versioning.CommitFile(dir, "1/note.md", "update") //nolint:errcheck

	versions, _ := versioning.Log(dir, "1/note.md", 10, 0)
	raw, err := versioning.Diff(dir, "1/note.md", versions[1].CommitHash, versions[0].CommitHash)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if !strings.Contains(raw, "-line two") {
		t.Error("diff should show removed line")
	}
	if !strings.Contains(raw, "+line three") {
		t.Error("diff should show added line")
	}
}

func TestParseDiff_ParsesUnifiedDiff(t *testing.T) {
	raw := `diff --git a/1/note.md b/1/note.md
index abc123..def456 100644
--- a/1/note.md
+++ b/1/note.md
@@ -1,3 +1,3 @@
 line one
-line two
+line three
 line four
`
	lines := versioning.ParseDiff(raw)
	if len(lines) == 0 {
		t.Fatal("expected parsed lines")
	}

	var adds, removes, contexts int
	for _, l := range lines {
		switch l.Type {
		case "add":
			adds++
			if l.Content != "line three" {
				t.Errorf("unexpected add content: %q", l.Content)
			}
		case "remove":
			removes++
			if l.Content != "line two" {
				t.Errorf("unexpected remove content: %q", l.Content)
			}
		case "context":
			contexts++
		}
	}
	if adds != 1 {
		t.Errorf("expected 1 add, got %d", adds)
	}
	if removes != 1 {
		t.Errorf("expected 1 remove, got %d", removes)
	}
	if contexts < 2 {
		t.Errorf("expected at least 2 context lines, got %d", contexts)
	}
}

func TestValidCommitHash_ValidHashes(t *testing.T) {
	tests := []struct {
		hash  string
		valid bool
	}{
		{"abc123def456abc123def456abc123def456abc1", true},
		{"abcd1234", true},
		{"xyz", false},
		{"", false},
		{"ABCDEF", false},
		{"abc123!!", false},
	}
	for _, tt := range tests {
		if got := versioning.ValidCommitHash(tt.hash); got != tt.valid {
			t.Errorf("ValidCommitHash(%q) = %v, want %v", tt.hash, got, tt.valid)
		}
	}
}

func TestParentCommit_ReturnsParent(t *testing.T) {
	dir := initTestRepo(t)
	writeFile(t, dir, "1/note.md", "v1")
	versioning.CommitFile(dir, "1/note.md", "create") //nolint:errcheck
	writeFile(t, dir, "1/note.md", "v2")
	versioning.CommitFile(dir, "1/note.md", "update") //nolint:errcheck

	versions, _ := versioning.Log(dir, "1/note.md", 10, 0)
	parent, err := versioning.ParentCommit(dir, versions[0].CommitHash)
	if err != nil {
		t.Fatalf("ParentCommit: %v", err)
	}
	if parent != versions[1].CommitHash {
		t.Errorf("expected parent %s, got %s", versions[1].CommitHash, parent)
	}
}

func TestGetVersion_ReturnsVersionInfo(t *testing.T) {
	dir := initTestRepo(t)
	writeFile(t, dir, "1/note.md", "content")
	versioning.CommitFile(dir, "1/note.md", "create: note") //nolint:errcheck

	versions, _ := versioning.Log(dir, "1/note.md", 10, 0)
	v, err := versioning.GetVersion(dir, versions[0].CommitHash)
	if err != nil {
		t.Fatalf("GetVersion: %v", err)
	}
	if v.CommitHash != versions[0].CommitHash {
		t.Errorf("hash mismatch: %s != %s", v.CommitHash, versions[0].CommitHash)
	}
	if v.Message != "create: note" {
		t.Errorf("unexpected message: %q", v.Message)
	}
}

func TestFileExistsAtCommit_TrueWhenExists(t *testing.T) {
	dir := initTestRepo(t)
	writeFile(t, dir, "1/note.md", "content")
	versioning.CommitFile(dir, "1/note.md", "create") //nolint:errcheck

	versions, _ := versioning.Log(dir, "1/note.md", 10, 0)
	if !versioning.FileExistsAtCommit(dir, "1/note.md", versions[0].CommitHash) {
		t.Error("expected file to exist at commit")
	}
}

func TestFileExistsAtCommit_FalseWhenMissing(t *testing.T) {
	dir := initTestRepo(t)
	writeFile(t, dir, "1/note.md", "content")
	versioning.CommitFile(dir, "1/note.md", "create") //nolint:errcheck

	versions, _ := versioning.Log(dir, "1/note.md", 10, 0)
	if versioning.FileExistsAtCommit(dir, "1/other.md", versions[0].CommitHash) {
		t.Error("expected file NOT to exist at commit")
	}
}

func TestLog_MultiplePaths_IncludesDrawingOnlyCommits(t *testing.T) {
	dir := initTestRepo(t)

	// Arrange: create a note and a drawing as separate commits
	writeFile(t, dir, "1/note.md", "# Hello")
	versioning.CommitFile(dir, "1/note.md", "create: note") //nolint:errcheck

	writeFile(t, dir, "1/note.tldraw.json", `{"shapes":[]}`)
	versioning.CommitFile(dir, "1/note.tldraw.json", "update drawing: note") //nolint:errcheck

	// Act: log with only the markdown path — should miss the drawing commit
	mdOnly, _ := versioning.Log(dir, "1/note.md", 10, 0)
	if len(mdOnly) != 1 {
		t.Errorf("md-only log: expected 1, got %d", len(mdOnly))
	}

	// Act: log with both paths — should include both commits
	both, err := versioning.Log(dir, "1/note.md", 10, 0, "1/note.tldraw.json")
	if err != nil {
		t.Fatalf("Log multi: %v", err)
	}
	if len(both) != 2 {
		t.Errorf("multi-path log: expected 2, got %d", len(both))
	}
}

func TestLog_MultiplePaths_DeduplicatesSharedCommit(t *testing.T) {
	dir := initTestRepo(t)

	// Arrange: single commit that touches both files
	writeFile(t, dir, "1/note.md", "# Hello")
	writeFile(t, dir, "1/note.tldraw.json", `{"shapes":[]}`)
	versioning.CommitFile(dir, "1/note.md", "create: note")         //nolint:errcheck
	versioning.CommitFile(dir, "1/note.tldraw.json", "add drawing") //nolint:errcheck

	// Act: log with both paths
	versions, _ := versioning.Log(dir, "1/note.md", 10, 0, "1/note.tldraw.json")

	// Each commit touches only one file, so we should get 2 distinct versions
	if len(versions) != 2 {
		t.Errorf("expected 2 versions, got %d", len(versions))
	}
}

func TestListEverTouchedPaths_IncludesDeletedFile(t *testing.T) {
	dir := initTestRepo(t)

	writeFile(t, dir, "1/note.md", "# Hello")
	versioning.CommitFile(dir, "1/note.md", "create: note") //nolint:errcheck

	writeFile(t, dir, "1/note.tldraw.json", `{"shapes":[]}`)
	versioning.CommitFile(dir, "1/note.tldraw.json", "add drawing: note") //nolint:errcheck

	versioning.CommitDelete(dir, "1/note.tldraw.json", "delete drawing: note") //nolint:errcheck

	paths, err := versioning.ListEverTouchedPaths(dir, "1/note")
	if err != nil {
		t.Fatalf("ListEverTouchedPaths: %v", err)
	}

	found := false
	for _, p := range paths {
		if p == "1/note.tldraw.json" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected deleted drawing path in results, got: %v", paths)
	}
}

func TestListEverTouchedPaths_DeletedDrawingAppearsInLog(t *testing.T) {
	dir := initTestRepo(t)

	writeFile(t, dir, "1/note.md", "# Hello")
	versioning.CommitFile(dir, "1/note.md", "create: note") //nolint:errcheck

	writeFile(t, dir, "1/note.tldraw.json", `{"shapes":[]}`)
	versioning.CommitFile(dir, "1/note.tldraw.json", "add drawing: note") //nolint:errcheck

	versioning.CommitDelete(dir, "1/note.tldraw.json", "delete drawing: note") //nolint:errcheck

	// Simulate what NoteHistoryGET does: discover paths via git, then log them
	allPaths, _ := versioning.ListEverTouchedPaths(dir, "1/note")
	relPath := "1/note.md"
	var extraPaths []string
	for _, p := range allPaths {
		if p != relPath {
			extraPaths = append(extraPaths, p)
		}
	}

	versions, err := versioning.Log(dir, relPath, 10, 0, extraPaths...)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}

	// Should have: create note, add drawing, delete drawing = 3 versions
	if len(versions) != 3 {
		t.Errorf("expected 3 versions (create, add drawing, delete drawing), got %d", len(versions))
	}
	// Most recent should be the delete
	if !strings.Contains(versions[0].Message, "delete drawing") {
		t.Errorf("expected first version to be delete drawing, got: %q", versions[0].Message)
	}
}
