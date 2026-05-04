package storage_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/selvakn/yant/internal/storage"
)

func TestWriteDrawing_Tldraw_CreatesFileAtCorrectPath(t *testing.T) {
	root := t.TempDir()
	data := []byte(`{"document":{"store":{}}}`)
	if err := storage.WriteDrawing(root, 7, "my-note", storage.DrawingTldraw, data); err != nil {
		t.Fatalf("WriteDrawing: %v", err)
	}
	path := filepath.Join(root, "7", "my-note.tldraw.json")
	read, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("file not found at %s: %v", path, err)
	}
	if string(read) != string(data) {
		t.Errorf("expected %q, got %q", string(data), string(read))
	}
}

func TestWriteDrawing_Excalidraw_CreatesFileAtCorrectPath(t *testing.T) {
	root := t.TempDir()
	data := []byte(`{"type":"excalidraw","elements":[]}`)
	if err := storage.WriteDrawing(root, 7, "my-note", storage.DrawingExcalidraw, data); err != nil {
		t.Fatalf("WriteDrawing: %v", err)
	}
	path := filepath.Join(root, "7", "my-note.excalidraw.json")
	read, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("file not found at %s: %v", path, err)
	}
	if string(read) != string(data) {
		t.Errorf("expected %q, got %q", string(data), string(read))
	}
}

func TestWriteDrawing_OverwritesExistingFile(t *testing.T) {
	root := t.TempDir()
	storage.WriteDrawing(root, 1, "note", storage.DrawingTldraw, []byte("first"))  //nolint:errcheck
	storage.WriteDrawing(root, 1, "note", storage.DrawingTldraw, []byte("second")) //nolint:errcheck
	path := filepath.Join(root, "1", "note.tldraw.json")
	data, _ := os.ReadFile(path)
	if string(data) != "second" {
		t.Errorf("expected 'second', got %q", string(data))
	}
}

func TestReadDrawing_ReturnsWrittenContent_Tldraw(t *testing.T) {
	root := t.TempDir()
	expected := []byte(`{"test":true}`)
	storage.WriteDrawing(root, 3, "hello", storage.DrawingTldraw, expected) //nolint:errcheck
	data, dt, err := storage.ReadDrawing(root, 3, "hello")
	if err != nil {
		t.Fatalf("ReadDrawing: %v", err)
	}
	if dt != storage.DrawingTldraw {
		t.Errorf("expected DrawingTldraw, got %q", dt)
	}
	if string(data) != string(expected) {
		t.Errorf("expected %q, got %q", string(expected), string(data))
	}
}

func TestReadDrawing_ReturnsWrittenContent_Excalidraw(t *testing.T) {
	root := t.TempDir()
	expected := []byte(`{"type":"excalidraw","elements":[]}`)
	storage.WriteDrawing(root, 3, "hello", storage.DrawingExcalidraw, expected) //nolint:errcheck
	data, dt, err := storage.ReadDrawing(root, 3, "hello")
	if err != nil {
		t.Fatalf("ReadDrawing: %v", err)
	}
	if dt != storage.DrawingExcalidraw {
		t.Errorf("expected DrawingExcalidraw, got %q", dt)
	}
	if string(data) != string(expected) {
		t.Errorf("expected %q, got %q", string(expected), string(data))
	}
}

func TestReadDrawing_MissingFileReturnsError(t *testing.T) {
	root := t.TempDir()
	_, _, err := storage.ReadDrawing(root, 99, "nonexistent")
	if err == nil {
		t.Error("expected error for missing file")
	}
	if !os.IsNotExist(err) {
		t.Errorf("expected os.ErrNotExist, got: %v", err)
	}
}

func TestDeleteDrawing_RemovesTldrawFile(t *testing.T) {
	root := t.TempDir()
	storage.WriteDrawing(root, 2, "to-delete", storage.DrawingTldraw, []byte("gone")) //nolint:errcheck
	if err := storage.DeleteDrawing(root, 2, "to-delete"); err != nil {
		t.Fatalf("DeleteDrawing: %v", err)
	}
	path := filepath.Join(root, "2", "to-delete.tldraw.json")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected file to be removed")
	}
}

func TestDeleteDrawing_RemovesExcalidrawFile(t *testing.T) {
	root := t.TempDir()
	storage.WriteDrawing(root, 2, "to-delete", storage.DrawingExcalidraw, []byte("gone")) //nolint:errcheck
	if err := storage.DeleteDrawing(root, 2, "to-delete"); err != nil {
		t.Fatalf("DeleteDrawing: %v", err)
	}
	path := filepath.Join(root, "2", "to-delete.excalidraw.json")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected file to be removed")
	}
}

func TestDeleteDrawing_MissingFileNoError(t *testing.T) {
	root := t.TempDir()
	if err := storage.DeleteDrawing(root, 5, "ghost"); err != nil {
		t.Errorf("expected no error for missing file, got: %v", err)
	}
}

func TestDrawingExists_ReturnsTrueForTldraw(t *testing.T) {
	root := t.TempDir()
	storage.WriteDrawing(root, 1, "exists", storage.DrawingTldraw, []byte("{}")) //nolint:errcheck
	if !storage.DrawingExists(root, 1, "exists") {
		t.Error("expected DrawingExists to return true")
	}
}

func TestDrawingExists_ReturnsTrueForExcalidraw(t *testing.T) {
	root := t.TempDir()
	storage.WriteDrawing(root, 1, "exists", storage.DrawingExcalidraw, []byte("{}")) //nolint:errcheck
	if !storage.DrawingExists(root, 1, "exists") {
		t.Error("expected DrawingExists to return true")
	}
}

func TestDrawingExists_ReturnsFalseWhenMissing(t *testing.T) {
	root := t.TempDir()
	if storage.DrawingExists(root, 1, "missing") {
		t.Error("expected DrawingExists to return false")
	}
}

func TestDetectDrawingType_Tldraw(t *testing.T) {
	root := t.TempDir()
	storage.WriteDrawing(root, 1, "note", storage.DrawingTldraw, []byte("{}")) //nolint:errcheck
	dt := storage.DetectDrawingType(root, 1, "note")
	if dt != storage.DrawingTldraw {
		t.Errorf("expected tldraw, got %q", dt)
	}
}

func TestDetectDrawingType_Excalidraw(t *testing.T) {
	root := t.TempDir()
	storage.WriteDrawing(root, 1, "note", storage.DrawingExcalidraw, []byte("{}")) //nolint:errcheck
	dt := storage.DetectDrawingType(root, 1, "note")
	if dt != storage.DrawingExcalidraw {
		t.Errorf("expected excalidraw, got %q", dt)
	}
}

func TestDetectDrawingType_None(t *testing.T) {
	root := t.TempDir()
	dt := storage.DetectDrawingType(root, 1, "note")
	if dt != storage.DrawingNone {
		t.Errorf("expected empty, got %q", dt)
	}
}

func TestDrawingRelPath_Tldraw(t *testing.T) {
	got := storage.DrawingRelPath(42, "my-note", storage.DrawingTldraw)
	want := "42/my-note.tldraw.json"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestDrawingRelPath_Excalidraw(t *testing.T) {
	got := storage.DrawingRelPath(42, "my-note", storage.DrawingExcalidraw)
	want := "42/my-note.excalidraw.json"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func assertHasDrawingFile(t *testing.T, files []storage.DrawingFile, want storage.DrawingFile) {
	t.Helper()
	for _, f := range files {
		if f == want {
			return
		}
	}
	t.Fatalf("missing %#v in %#v", want, files)
}

func TestListDrawingFiles_empty(t *testing.T) {
	root := t.TempDir()
	files := storage.ListDrawingFiles(root, 1, "note")
	if len(files) != 0 {
		t.Fatalf("expected no files, got %#v", files)
	}
}

func TestListDrawingFiles_legacy(t *testing.T) {
	root := t.TempDir()
	storage.WriteDrawing(root, 9, "legacy-note", storage.DrawingTldraw, []byte("{}")) //nolint:errcheck
	files := storage.ListDrawingFiles(root, 9, "legacy-note")
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %#v", files)
	}
	assertHasDrawingFile(t, files, storage.DrawingFile{DrawingID: "", Type: storage.DrawingTldraw, IsLegacy: true})
}

func TestListDrawingFiles_new_format(t *testing.T) {
	root := t.TempDir()
	idTl := "tl12abcd"
	idEx := "ex34efgh"
	storage.WriteDrawingByID(root, 3, "multi", idTl, storage.DrawingTldraw, []byte(`{"tl":1}`))     //nolint:errcheck
	storage.WriteDrawingByID(root, 3, "multi", idEx, storage.DrawingExcalidraw, []byte(`{"ex":1}`)) //nolint:errcheck
	files := storage.ListDrawingFiles(root, 3, "multi")
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %#v", files)
	}
	assertHasDrawingFile(t, files, storage.DrawingFile{DrawingID: idTl, Type: storage.DrawingTldraw, IsLegacy: false})
	assertHasDrawingFile(t, files, storage.DrawingFile{DrawingID: idEx, Type: storage.DrawingExcalidraw, IsLegacy: false})
}

func TestListDrawingFiles_mixed(t *testing.T) {
	root := t.TempDir()
	storage.WriteDrawing(root, 4, "mixed", storage.DrawingTldraw, []byte(`legacy`))                 //nolint:errcheck
	storage.WriteDrawingByID(root, 4, "mixed", "new99aa", storage.DrawingExcalidraw, []byte(`new`)) //nolint:errcheck
	files := storage.ListDrawingFiles(root, 4, "mixed")
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %#v", files)
	}
	assertHasDrawingFile(t, files, storage.DrawingFile{DrawingID: "", Type: storage.DrawingTldraw, IsLegacy: true})
	assertHasDrawingFile(t, files, storage.DrawingFile{DrawingID: "new99aa", Type: storage.DrawingExcalidraw, IsLegacy: false})
}

func TestReadWriteDrawingByID(t *testing.T) {
	root := t.TempDir()
	want := []byte(`{"roundtrip":true}`)
	id := "rw12abcd"
	if err := storage.WriteDrawingByID(root, 2, "rw-note", id, storage.DrawingTldraw, want); err != nil {
		t.Fatalf("WriteDrawingByID: %v", err)
	}
	got, err := storage.ReadDrawingByID(root, 2, "rw-note", id, storage.DrawingTldraw)
	if err != nil {
		t.Fatalf("ReadDrawingByID: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("expected %q, got %q", string(want), string(got))
	}
}

func TestDeleteDrawingByID(t *testing.T) {
	root := t.TempDir()
	id := "delabcde"
	path := filepath.Join(root, "6", "gone-note--"+id+".tldraw.json")
	storage.WriteDrawingByID(root, 6, "gone-note", id, storage.DrawingTldraw, []byte(`{}`)) //nolint:errcheck
	if err := storage.DeleteDrawingByID(root, 6, "gone-note", id, storage.DrawingTldraw); err != nil {
		t.Fatalf("DeleteDrawingByID: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected file removed")
	}
	if _, err := storage.ReadDrawingByID(root, 6, "gone-note", id, storage.DrawingTldraw); err == nil {
		t.Error("expected read error after delete")
	}
}

func TestDetectLegacyDrawing_withFile(t *testing.T) {
	root := t.TempDir()
	storage.WriteDrawing(root, 8, "leg", storage.DrawingExcalidraw, []byte(`{}`)) //nolint:errcheck
	dt, ok := storage.DetectLegacyDrawing(root, 8, "leg")
	if !ok || dt != storage.DrawingExcalidraw {
		t.Fatalf("expected excalidraw legacy, got ok=%v dt=%q", ok, dt)
	}
}

func TestDetectLegacyDrawing_withoutFile(t *testing.T) {
	root := t.TempDir()
	dt, ok := storage.DetectLegacyDrawing(root, 8, "none")
	if ok || dt != storage.DrawingNone {
		t.Fatalf("expected no legacy, got ok=%v dt=%q", ok, dt)
	}
}

func TestMigrateLegacyDrawing(t *testing.T) {
	root := t.TempDir()
	slug := "migrate-me"
	newID := "mig12345"
	storage.WriteDrawing(root, 5, slug, storage.DrawingTldraw, []byte(`{"migrated":true}`)) //nolint:errcheck
	oldPath := filepath.Join(root, "5", slug+".tldraw.json")
	newPath := filepath.Join(root, "5", slug+"--"+newID+".tldraw.json")

	dt, err := storage.MigrateLegacyDrawing(root, 5, slug, newID)
	if err != nil {
		t.Fatalf("MigrateLegacyDrawing: %v", err)
	}
	if dt != storage.DrawingTldraw {
		t.Errorf("expected DrawingTldraw, got %q", dt)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Error("expected legacy file gone")
	}
	data, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("read new path: %v", err)
	}
	if string(data) != `{"migrated":true}` {
		t.Errorf("unexpected content %q", string(data))
	}
}

func TestWriteReadDrawingSVG(t *testing.T) {
	root := t.TempDir()
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><rect width="100" height="100"/></svg>`)
	if err := storage.WriteDrawingSVG(root, 2, "note", "abc12345", svg); err != nil {
		t.Fatalf("WriteDrawingSVG: %v", err)
	}
	got, err := storage.ReadDrawingSVG(root, 2, "note", "abc12345")
	if err != nil {
		t.Fatalf("ReadDrawingSVG: %v", err)
	}
	if string(got) != string(svg) {
		t.Errorf("expected %q, got %q", string(svg), string(got))
	}
}

func TestReadDrawingSVG_missing_returns_error(t *testing.T) {
	root := t.TempDir()
	_, err := storage.ReadDrawingSVG(root, 1, "nope", "missing1")
	if err == nil {
		t.Error("expected error for missing SVG")
	}
	if !os.IsNotExist(err) {
		t.Errorf("expected os.ErrNotExist, got: %v", err)
	}
}

func TestDeleteDrawingSVG_removes_file(t *testing.T) {
	root := t.TempDir()
	storage.WriteDrawingSVG(root, 3, "note", "del12345", []byte(`<svg/>`)) //nolint:errcheck
	if err := storage.DeleteDrawingSVG(root, 3, "note", "del12345"); err != nil {
		t.Fatalf("DeleteDrawingSVG: %v", err)
	}
	_, err := storage.ReadDrawingSVG(root, 3, "note", "del12345")
	if !os.IsNotExist(err) {
		t.Error("expected SVG file to be removed")
	}
}

func TestDeleteDrawingSVG_missing_is_noop(t *testing.T) {
	root := t.TempDir()
	if err := storage.DeleteDrawingSVG(root, 1, "note", "ghost123"); err != nil {
		t.Errorf("expected no error for missing SVG, got: %v", err)
	}
}

func TestDeleteAllDrawingsBySlug(t *testing.T) {
	root := t.TempDir()
	u := int64(11)
	slug := "purge"
	storage.WriteDrawing(root, u, slug, storage.DrawingExcalidraw, []byte(`legacy`))        //nolint:errcheck
	storage.WriteDrawingByID(root, u, slug, "aa11bb22", storage.DrawingTldraw, []byte(`a`)) //nolint:errcheck
	storage.WriteDrawingSVG(root, u, slug, "aa11bb22", []byte(`<svg/>`))                    //nolint:errcheck
	storage.WriteDrawingByID(root, u, slug, "cc33dd44", storage.DrawingTldraw, []byte(`b`)) //nolint:errcheck
	if err := storage.DeleteAllDrawingsBySlug(root, u, slug); err != nil {
		t.Fatalf("DeleteAllDrawingsBySlug: %v", err)
	}
	if files := storage.ListDrawingFiles(root, u, slug); len(files) != 0 {
		t.Fatalf("expected no drawings left, got %#v", files)
	}
	if _, err := storage.ReadDrawingSVG(root, u, slug, "aa11bb22"); !os.IsNotExist(err) {
		t.Error("expected SVG file removed during purge")
	}
}
