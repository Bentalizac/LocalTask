package browse

import (
	"os"
	"path/filepath"
	"testing"
)

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func TestListFindsSubdirsAndFlagsTaskboard(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "dance-note"))
	mkdir(t, filepath.Join(root, "job-tracker"))
	mkdir(t, filepath.Join(root, ".git")) // hidden — should be skipped
	if err := os.WriteFile(filepath.Join(root, "dance-note", "taskboard.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("write taskboard.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# hi"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	listing, err := List(root)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if listing.Path != root {
		t.Fatalf("expected path %q, got %q", root, listing.Path)
	}
	if len(listing.Entries) != 2 {
		t.Fatalf("expected 2 entries (dirs only, hidden skipped), got %d: %+v", len(listing.Entries), listing.Entries)
	}

	byName := map[string]Entry{}
	for _, e := range listing.Entries {
		byName[e.Name] = e
	}
	if !byName["dance-note"].HasTaskboard {
		t.Fatal("expected dance-note to be flagged as having taskboard.json")
	}
	if byName["job-tracker"].HasTaskboard {
		t.Fatal("expected job-tracker to not have taskboard.json")
	}
}

func TestListParentAtFilesystemRoot(t *testing.T) {
	listing, err := List("/")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if listing.Parent != "" {
		t.Fatalf("expected no parent at filesystem root, got %q", listing.Parent)
	}
}

func TestListParentOfSubdir(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	mkdir(t, child)

	listing, err := List(child)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if listing.Parent != root {
		t.Fatalf("expected parent %q, got %q", root, listing.Parent)
	}
}

func TestListRejectsRelativePath(t *testing.T) {
	if _, err := List("relative/path"); err == nil {
		t.Fatal("expected error for relative path")
	}
}

func TestListRejectsNonexistentPath(t *testing.T) {
	if _, err := List(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}

func TestListDefaultsToHomeDir(t *testing.T) {
	listing, err := List("")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	home, _ := os.UserHomeDir()
	if home != "" && listing.Path != home {
		t.Fatalf("expected default path to be home dir %q, got %q", home, listing.Path)
	}
}
