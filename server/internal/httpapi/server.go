// Package httpapi implements the HTTP surface of the taskboard
// application. Handlers stay thin: all domain logic lives in the project
// package, which is the same service layer the MCP server will use.
package httpapi

import (
	"fmt"
	"net/http"

	"localtask/internal/project"
	"localtask/internal/registry"
)

// Server serves the taskboard HTTP API over a live project registry.
type Server struct {
	registry  *registry.Registry
	staticDir string
	mux       *http.ServeMux
}

// New builds a Server over reg. If staticDir is non-empty, the server also
// serves the built frontend from that directory (with SPA fallback to
// index.html) for any request that doesn't match an /api/ route — this is
// how the Docker image serves the whole application from one process/port.
// Pass "" for API-only mode (the mode used in local dev, where Vite serves
// the frontend separately).
func New(reg *registry.Registry, staticDir string) *Server {
	s := &Server{registry: reg, staticDir: staticDir, mux: http.NewServeMux()}
	s.routes()
	return s
}

// Handler returns the server's http.Handler.
func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/browse", s.handleBrowse)
	s.mux.HandleFunc("GET /api/projects", s.handleListProjects)
	s.mux.HandleFunc("POST /api/projects", s.handleAddProject)
	s.mux.HandleFunc("GET /api/projects/{project}", s.handleGetProject)
	s.mux.HandleFunc("GET /api/projects/{project}/tickets", s.handleListTickets)
	s.mux.HandleFunc("GET /api/projects/{project}/tickets/{ticket}", s.handleGetTicket)
	s.mux.HandleFunc("POST /api/projects/{project}/tickets", s.handleCreateTicket)
	s.mux.HandleFunc("PATCH /api/projects/{project}/tickets/{ticket}", s.handleUpdateTicket)
	s.mux.HandleFunc("GET /api/projects/{project}/git", s.handleGetGit)
	s.mux.HandleFunc("GET /api/projects/{project}/docs", s.handleListDocs)
	s.mux.HandleFunc("GET /api/projects/{project}/docs/{path...}", s.handleGetDoc)

	if s.staticDir != "" {
		s.mux.Handle("/", http.FileServer(spaFileSystem{http.Dir(s.staticDir)}))
	}
}

// project resolves the {project} path value, writing a 404 and returning
// ok=false if it isn't a configured/loaded project.
func (s *Server) project(w http.ResponseWriter, r *http.Request) (p *project.Project, ok bool) {
	name := r.PathValue("project")
	p, found := s.registry.Get(name)
	if !found {
		writeError(w, http.StatusNotFound, fmt.Errorf("project %q not found", name))
		return nil, false
	}
	return p, true
}
