package httpapi

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"localtask/internal/browse"
	"localtask/internal/config"
	"localtask/internal/registry"
)

// newRegistryTestServer builds a Server over an empty but persistable
// registry (a real config file backing it), for exercising GET /api/browse
// and POST /api/projects end-to-end, including on-disk persistence.
func newRegistryTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "taskboard.config.yaml")
	if err := os.WriteFile(configPath, []byte("projects: []\n"), 0o644); err != nil {
		t.Fatalf("writing config: %v", err)
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	reg := registry.NewRegistry(cfg, nil, nil)
	return New(reg, ""), configPath
}

func TestHandleBrowseListsSubdirs(t *testing.T) {
	s, _ := newRegistryTestServer(t)

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "dance-note"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	rec := doRequest(t, s, http.MethodGet, "/api/browse?path="+root, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body)
	}
	var listing browse.Listing
	if err := json.Unmarshal(rec.Body.Bytes(), &listing); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(listing.Entries) != 1 || listing.Entries[0].Name != "dance-note" {
		t.Fatalf("unexpected listing: %+v", listing)
	}
}

func TestHandleBrowseRejectsRelativePath(t *testing.T) {
	s, _ := newRegistryTestServer(t)
	rec := doRequest(t, s, http.MethodGet, "/api/browse?path=relative/path", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body)
	}
}

func TestHandleAddProjectCreatesTaskboardAndPersists(t *testing.T) {
	s, configPath := newRegistryTestServer(t)
	projectDir := t.TempDir()

	rec := doRequest(t, s, http.MethodPost, "/api/projects", map[string]any{
		"name": "JobTracker",
		"path": projectDir,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body)
	}
	var resp projectResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Name != "JobTracker" {
		t.Fatalf("unexpected response: %+v", resp)
	}

	// Immediately queryable, no restart.
	rec2 := doRequest(t, s, http.MethodGet, "/api/projects/JobTracker/tickets", nil)
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected new project to be immediately queryable, got %d: %s", rec2.Code, rec2.Body)
	}

	// Persisted to the config file on disk.
	reloaded, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if _, err := reloaded.FindProject("JobTracker"); err != nil {
		t.Fatalf("expected project persisted to config: %v", err)
	}

	// taskboard.json was bootstrapped.
	if _, err := os.Stat(filepath.Join(projectDir, "taskboard.json")); err != nil {
		t.Fatalf("expected taskboard.json to be created: %v", err)
	}
}

func TestHandleAddProjectDuplicateName(t *testing.T) {
	s, _ := newRegistryTestServer(t)
	dirA, dirB := t.TempDir(), t.TempDir()

	rec := doRequest(t, s, http.MethodPost, "/api/projects", map[string]any{"name": "X", "path": dirA})
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body)
	}
	rec2 := doRequest(t, s, http.MethodPost, "/api/projects", map[string]any{"name": "X", "path": dirB})
	if rec2.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec2.Code, rec2.Body)
	}
}

func TestHandleAddProjectBadPath(t *testing.T) {
	s, _ := newRegistryTestServer(t)
	rec := doRequest(t, s, http.MethodPost, "/api/projects", map[string]any{
		"name": "X",
		"path": "relative/path",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body)
	}
}

func TestHandleAddProjectMissingFields(t *testing.T) {
	s, _ := newRegistryTestServer(t)
	rec := doRequest(t, s, http.MethodPost, "/api/projects", map[string]any{"name": "X"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body)
	}
}
