package httpapi

import (
	"net/http"
	"os"
)

// spaFileSystem wraps an http.FileSystem so that any path with no matching
// file falls back to index.html — required for a client-side-routed SPA
// (e.g. GET /projects/DanceNote/board must serve the app shell, not 404).
// http.Dir already rejects ".." path traversal, so this inherits that
// safety.
type spaFileSystem struct {
	root http.FileSystem
}

func (fs spaFileSystem) Open(name string) (http.File, error) {
	f, err := fs.root.Open(name)
	if os.IsNotExist(err) {
		return fs.root.Open("index.html")
	}
	return f, err
}
