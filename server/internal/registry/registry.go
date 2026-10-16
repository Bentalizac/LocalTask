// Package registry loads every project named in a config.Config into
// memory, for the HTTP server and the MCP server to share unchanged (see
// spec §2: "Do not implement separate task-management logic for MCP and
// HTTP"). Registry additionally supports adding a project at runtime (the
// HTTP API's add-project endpoint), persisting it to the config file and
// registering it live with no restart required.
package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"maps"
	"os"
	"path/filepath"
	"sync"

	"localtask/internal/config"
	"localtask/internal/project"
	"localtask/internal/ticket"
	"localtask/internal/watch"

	"localtask/internal/storage"
)

// Load loads every configured project. A project that fails to load is
// reported in errs (identified by name) rather than aborting the load of
// the others, so one bad taskboard.json doesn't take every project down.
//
// An externally stored project with no board yet gets an empty one, since
// that directory belongs to LocalTask. A repo-stored project with no
// taskboard.json fails to load instead, so LocalTask never writes into a
// repo it wasn't explicitly asked to (see Registry.Add).
func Load(cfg *config.Config) (map[string]*project.Project, []error) {
	projects := make(map[string]*project.Project, len(cfg.Projects))
	var errs []error
	for _, pc := range cfg.Projects {
		p, err := load(cfg, pc, pc.EffectiveStorage() == config.StorageExternal)
		if err != nil {
			errs = append(errs, fmt.Errorf("project %q: %w", pc.Name, err))
			continue
		}
		projects[pc.Name] = p
	}
	return projects, errs
}

// load resolves pc's board directory and loads the project, first creating
// an empty taskboard.json there if create is set and none exists.
func load(cfg *config.Config, pc config.ProjectConfig, create bool) (*project.Project, error) {
	boardDir, err := cfg.BoardDir(pc)
	if err != nil {
		return nil, err
	}
	if create {
		if err := ensureTaskboard(boardDir, pc.Name); err != nil {
			return nil, err
		}
	}
	p, err := project.LoadProject(pc.Name, pc.Path, boardDir)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// ensureTaskboard creates boardDir and an empty taskboard.json in it,
// unless that file already exists.
func ensureTaskboard(boardDir, name string) error {
	taskboardPath := filepath.Join(boardDir, project.TaskboardFileName)
	if _, err := os.Stat(taskboardPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("checking for existing taskboard.json: %w", err)
	}
	if err := os.MkdirAll(boardDir, 0o755); err != nil {
		return fmt.Errorf("creating board directory: %w", err)
	}
	tb := project.Taskboard{Version: 1, Project: name, Tickets: []ticket.Ticket{}}
	data, err := json.MarshalIndent(tb, "", "  ")
	if err != nil {
		return err
	}
	if err := storage.WriteFileAtomic(taskboardPath, data, 0o644); err != nil {
		return fmt.Errorf("creating taskboard.json: %w", err)
	}
	return nil
}

// ErrProjectExists is returned by Add when a project with that name is
// already registered.
var ErrProjectExists = errors.New("project already exists")

// Registry is a live, mutable set of loaded projects, backed by a
// config.Config it can persist new entries to and a watch.Watcher it can
// register new projects with. Safe for concurrent use.
type Registry struct {
	mu       sync.RWMutex
	projects map[string]*project.Project
	cfg      *config.Config
	watcher  *watch.Watcher
}

// NewRegistry wraps an already-loaded project map (see Load) as a live
// Registry. cfg is where Add persists new entries; watcher is where Add
// registers a new project for filesystem watching. Either may be nil —
// Add then simply skips that step.
func NewRegistry(cfg *config.Config, projects map[string]*project.Project, watcher *watch.Watcher) *Registry {
	if projects == nil {
		projects = make(map[string]*project.Project)
	}
	return &Registry{projects: projects, cfg: cfg, watcher: watcher}
}

// Get returns the project with the given name.
func (r *Registry) Get(name string) (*project.Project, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.projects[name]
	return p, ok
}

// List returns a snapshot copy of every registered project, keyed by name.
func (r *Registry) List() map[string]*project.Project {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]*project.Project, len(r.projects))
	maps.Copy(out, r.projects)
	return out
}

// Add registers a new project at path under name with the given storage
// mode (empty means config.StorageExternal): creating an empty
// taskboard.json in its board directory if one doesn't already exist,
// loading it, persisting it to the config file, and wiring it into the
// filesystem watcher — all live, with no restart required. On failure, no
// partial state is left registered (a taskboard.json created along the
// way is left in place, though, since that's exactly what the caller
// asked for).
func (r *Registry) Add(name, path string, store config.Storage) (*project.Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if name == "" {
		return nil, fmt.Errorf("project name is required")
	}
	if _, exists := r.projects[name]; exists {
		return nil, fmt.Errorf("%w: %q", ErrProjectExists, name)
	}
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("project path must be absolute: %q", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("project path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("project path is not a directory: %q", path)
	}

	pc := config.ProjectConfig{Name: name, Path: path, Storage: store}
	if err := pc.Validate(); err != nil {
		return nil, err
	}
	cfg := r.cfg
	if cfg == nil {
		cfg = config.New("")
	}
	p, err := load(cfg, pc, true)
	if err != nil {
		return nil, err
	}

	if r.cfg != nil {
		if err := r.cfg.AddProject(pc); err != nil {
			return nil, err
		}
	}

	if r.watcher != nil {
		if err := r.watcher.Add(p); err != nil {
			log.Printf("registry: watching %q failed (external edits won't auto-reload): %v", name, err)
		}
	}

	r.projects[name] = p
	return p, nil
}
