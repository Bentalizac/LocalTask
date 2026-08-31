package httpapi

import (
	"net/http"
	"os"

	"localtask/internal/document"
)

func (s *Server) handleListDocs(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	docs, err := document.List(p.Root)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if docs == nil {
		docs = []string{}
	}
	writeJSON(w, http.StatusOK, docs)
}

func (s *Server) handleGetDoc(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	doc, err := document.Read(p.Root, r.PathValue("path"))
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}
