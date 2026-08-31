package httpapi

import (
	"net/http"

	"localtask/internal/browse"
)

// handleBrowse powers the "Add Project" directory navigator: it lists
// subdirectories of the given path (or the server's home directory if
// path is omitted), so the frontend can render a folder picker without
// ever needing a real OS path from the browser (which browsers refuse to
// hand over, by design).
func (s *Server) handleBrowse(w http.ResponseWriter, r *http.Request) {
	listing, err := browse.List(r.URL.Query().Get("path"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, listing)
}
