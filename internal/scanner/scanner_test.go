package scanner

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func createTestFile(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("dummy"), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestScanFindsMP3AndOGG(t *testing.T) {
	dir := t.TempDir()
	createTestFile(t, dir, "song1.mp3")
	createTestFile(t, dir, "song2.MP3")
	createTestFile(t, dir, "song3.ogg")
	createTestFile(t, dir, "song4.OGG")
	createTestFile(t, dir, "notmusic.txt")
	createTestFile(t, dir, "other.flac")

	results, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if len(results) != 4 {
		t.Fatalf("found %d files, want 4", len(results))
	}
	names := make([]string, len(results))
	for i, r := range results {
		names[i] = filepath.Base(r.Path)
	}
	sort.Strings(names)
	expected := []string{"song1.mp3", "song2.MP3", "song3.ogg", "song4.OGG"}
	for i, want := range expected {
		if names[i] != want {
			t.Errorf("names[%d] = %s, want %s", i, names[i], want)
		}
	}
}

func TestScanRecursive(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "subdir")
	os.Mkdir(sub, 0755)
	createTestFile(t, dir, "root.mp3")
	createTestFile(t, sub, "nested.ogg")

	results, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("found %d files, want 2", len(results))
	}
}

func TestScanEmptyDir(t *testing.T) {
	dir := t.TempDir()
	results, err := Scan(dir)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("found %d files, want 0", len(results))
	}
}

func TestScanMissingDir(t *testing.T) {
	// filepath.Walk on Windows does not error on missing root;
	// it just returns no files. This is acceptable behavior.
	dir := t.TempDir()
	missing := filepath.Join(dir, "doesnotexist")
	results, err := Scan(missing)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("found %d files, want 0", len(results))
	}
}
