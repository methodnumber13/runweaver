package foundation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepositoryAndFileHelpers(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "nested", "value.json")

	if err := WriteJSON(nested, map[string]any{"ok": true}); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	if !Exists(nested) {
		t.Fatalf("Exists(%q) = false", nested)
	}
	var parsed map[string]bool
	if err := ReadJSON(nested, &parsed); err != nil {
		t.Fatalf("ReadJSON: %v", err)
	}
	if !parsed["ok"] {
		t.Fatalf("parsed = %#v", parsed)
	}
	if got := Rel(root, nested); got != "nested/value.json" {
		t.Fatalf("Rel() = %q", got)
	}
	if got, err := RepoRoot(root); err != nil || got != root {
		t.Fatalf("RepoRoot() = %q, %v", got, err)
	}
}

func TestListAndSkipHelpers(t *testing.T) {
	if got := Unique([]string{"b", "a", "b", "", "a"}); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("Unique() = %#v", got)
	}
	if got := Limit([]string{"a", "b", "c"}, 2); len(got) != 2 || got[1] != "b" {
		t.Fatalf("Limit() = %#v", got)
	}
	if !ShouldSkipDir("node_modules") || !ShouldSkipDir(".cache-v1") || ShouldSkipDir("src") {
		t.Fatalf("unexpected ShouldSkipDir decisions")
	}
}

func TestRepositoryAndJSONErrorPaths(t *testing.T) {
	root := t.TempDir()
	badJSON := filepath.Join(root, "bad.json")
	if err := os.WriteFile(badJSON, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := ReadJSON(badJSON, &parsed); err == nil || !strings.Contains(err.Error(), "parse JSON") {
		t.Fatalf("ReadJSON err = %v, want parse JSON error", err)
	}
	if _, err := SafeJSON(make(chan int)); err == nil || !strings.Contains(err.Error(), "marshal JSON") {
		t.Fatalf("SafeJSON err = %v, want marshal JSON error", err)
	}
	missing := filepath.Join(root, "missing")
	if _, err := RepoRoot(missing); err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("RepoRoot(missing) err = %v, want does not exist", err)
	}
	filePath := filepath.Join(root, "file")
	if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := RepoRoot(filePath); err == nil || !strings.Contains(err.Error(), "must be a directory") {
		t.Fatalf("RepoRoot(file) err = %v, want directory error", err)
	}
}
