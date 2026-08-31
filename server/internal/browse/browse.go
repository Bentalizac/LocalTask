// Package browse lists directories on the server's local filesystem, for
// the "Add Project" UI's folder navigator. This is deliberately the one
// place in the app not constrained to inside a project root — the whole
// point is to find a new project anywhere on disk before it's configured.
// It only ever reports directory names, never file contents.
package browse

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"localtask/internal/project"
)

// Entry is one subdirectory found while listing a path.
type Entry struct {
	Name         string `json:"name"`
	Path         string `json:"path"`
	HasTaskboard bool   `json:"has_taskboard"`
}

// Listing is the contents of one directory, for rendering a folder
// navigator: the resolved current path, its parent (empty at the
// filesystem root, so the UI knows not to offer "up"), and its
// subdirectories.
type Listing struct {
	Path         string  `json:"path"`
	Parent       string  `json:"parent,omitempty"`
	HasTaskboard bool    `json:"has_taskboard"`
	Entries      []Entry `json:"entries"`
}

// List lists the subdirectories of path. If path is empty, it defaults to
// the server process's home directory (falling back to "/" if that can't
// be determined) — a reasonable starting point for browsing.
func List(path string) (*Listing, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			home = "/"
		}
		path = home
	}
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("path must be absolute: %q", path)
	}
	path = filepath.Clean(path)

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %q", path)
	}

	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	entries := make([]Entry, 0, len(dirEntries))
	for _, de := range dirEntries {
		if !de.IsDir() {
			continue
		}
		if de.Name() != "" && de.Name()[0] == '.' {
			continue // skip hidden dirs (.git, etc.) — noise, never a project root itself
		}
		childPath := filepath.Join(path, de.Name())
		entries = append(entries, Entry{
			Name:         de.Name(),
			Path:         childPath,
			HasTaskboard: hasTaskboard(childPath),
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })

	parent := filepath.Dir(path)
	if parent == path {
		parent = "" // at filesystem root
	}

	return &Listing{
		Path:         path,
		Parent:       parent,
		HasTaskboard: hasTaskboard(path),
		Entries:      entries,
	}, nil
}

func hasTaskboard(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, project.TaskboardFileName))
	return err == nil && !info.IsDir()
}
