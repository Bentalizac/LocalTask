// Package pathutil provides the project-root containment check used
// wherever a relative path from taskboard.json or an HTTP request is
// resolved against a project's filesystem root (the local-only security
// model requires that these can never escape the configured root).
package pathutil

import (
	"errors"
	"path/filepath"
	"strings"
)

// WithinRoot resolves rel against root and returns the resulting absolute
// path. It errors if rel is absolute, or if the resolved path escapes root
// (e.g. via "..").
func WithinRoot(root, rel string) (string, error) {
	if filepath.IsAbs(rel) {
		return "", errors.New("must be a relative path")
	}
	rootClean := filepath.Clean(root)
	joined := filepath.Clean(filepath.Join(rootClean, rel))
	if joined != rootClean && !strings.HasPrefix(joined, rootClean+string(filepath.Separator)) {
		return "", errors.New("escapes project root")
	}
	return joined, nil
}
