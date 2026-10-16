package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"

	"localtask/internal/config"
	"localtask/internal/project"
	"localtask/internal/registry"
)

type projectResponse struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Version     int             `json:"version"`
	Metrics     project.Metrics `json:"metrics"`
}

func toProjectResponse(name string, p *project.Project) projectResponse {
	snap := p.Snapshot()
	return projectResponse{
		Name:        name,
		Description: snap.Description,
		Version:     snap.Version,
		Metrics:     p.Metrics(),
	}
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	all := s.registry.List()
	names := make([]string, 0, len(all))
	for name := range all {
		names = append(names, name)
	}
	sort.Strings(names)

	resp := make([]projectResponse, 0, len(names))
	for _, name := range names {
		resp = append(resp, toProjectResponse(name, all[name]))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	p, ok := s.project(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, toProjectResponse(r.PathValue("project"), p))
}

type addProjectRequest struct {
	Name    string         `json:"name"`
	Path    string         `json:"path"`
	Storage config.Storage `json:"storage,omitempty"`
}

// handleAddProject registers a project at an arbitrary path found via the
// GET /api/browse directory navigator, bootstrapping an empty
// taskboard.json in its board directory if one doesn't already exist. The
// board lives outside the project unless storage is "repo". Persists to the
// config file and is immediately queryable — no restart required.
func (s *Server) handleAddProject(w http.ResponseWriter, r *http.Request) {
	var req addProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Name == "" || req.Path == "" {
		writeError(w, http.StatusBadRequest, errors.New("name and path are required"))
		return
	}

	p, err := s.registry.Add(req.Name, req.Path, req.Storage)
	if err != nil {
		if errors.Is(err, registry.ErrProjectExists) {
			writeError(w, http.StatusConflict, err)
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, toProjectResponse(req.Name, p))
}
