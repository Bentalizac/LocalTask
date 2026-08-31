// Package document reads Markdown documentation files from within a
// project's root. The filesystem remains authoritative — this package does
// no rendering; that happens client-side (see spec §9).
package document

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"localtask/internal/pathutil"
)

// skipDirs are directory names never descended into during discovery —
// version control internals, dependency trees, and archived docs have no
// *current* project documentation worth surfacing. Matched
// case-insensitively (see List). A doc under one of these is still
// reachable via Read if something explicitly references it (e.g. a
// ticket's related_docs) — this only prunes passive discovery.
var skipDirs = map[string]bool{
	".git":          true,
	"node_modules":  true,
	"vendor":        true,
	"archive":       true,
	"archived_code": true,
	"site-packages": true,
}

// Doc is a single documentation file's raw contents.
type Doc struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// Read returns the contents of relPath within root. relPath must stay
// within root (see pathutil.WithinRoot).
func Read(root, relPath string) (*Doc, error) {
	abs, err := pathutil.WithinRoot(root, relPath)
	if err != nil {
		return nil, fmt.Errorf("invalid doc path %q: %w", relPath, err)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	return &Doc{Path: relPath, Content: string(data)}, nil
}

// List discovers every Markdown file within root, returning root-relative
// slash-separated paths in sorted order (see spec §9: "Discover Markdown
// files").
func List(root string) ([]string, error) {
	var docs []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && (skipDirs[strings.ToLower(d.Name())] || strings.HasPrefix(d.Name(), ".")) {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(path), ".md") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		docs = append(docs, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(docs)
	return docs, nil
}
