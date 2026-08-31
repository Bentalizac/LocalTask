package document

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestReadWithinRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "docs", "arch.md"), "# Architecture\n")

	doc, err := Read(root, "docs/arch.md")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if doc.Content != "# Architecture\n" {
		t.Fatalf("unexpected content: %q", doc.Content)
	}
}

func TestReadRejectsEscape(t *testing.T) {
	root := t.TempDir()
	if _, err := Read(root, "../../etc/passwd"); err == nil {
		t.Fatal("expected error escaping root")
	}
}

func TestListFindsMarkdownAndSkipsNoise(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "README.md"), "# Readme\n")
	writeFile(t, filepath.Join(root, "docs", "arch.md"), "# Arch\n")
	writeFile(t, filepath.Join(root, "docs", "api.MD"), "# API\n")
	writeFile(t, filepath.Join(root, "src", "main.go"), "package main\n")
	writeFile(t, filepath.Join(root, "node_modules", "pkg", "readme.md"), "noise\n")
	writeFile(t, filepath.Join(root, ".git", "COMMIT_EDITMSG.md"), "noise\n")
	writeFile(t, filepath.Join(root, "archive", "old-design.md"), "noise\n")
	writeFile(t, filepath.Join(root, "docs", "Archive", "old-api.md"), "noise\n")

	docs, err := List(root)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	want := []string{"README.md", "docs/api.MD", "docs/arch.md"}
	if !reflect.DeepEqual(docs, want) {
		t.Fatalf("got %v, want %v", docs, want)
	}
}

func TestReadStillReachesDocsUnderArchive(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "archive", "old-design.md"), "# Old Design\n")

	// List prunes archive/ from discovery, but an explicit reference (e.g.
	// a ticket's related_docs) should still resolve.
	doc, err := Read(root, "archive/old-design.md")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if doc.Content != "# Old Design\n" {
		t.Fatalf("unexpected content: %q", doc.Content)
	}
}
