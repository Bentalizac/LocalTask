package httpapi

import (
	"net/http"

	"localtask/internal/git"
)

func (s *Server) handleGetGit(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	info, err := git.Read(r.Context(), p.Root, 10)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, info)
}
