// Package watch keeps in-memory Project state in sync with external edits
// to taskboard.json (a hand edit, a git pull, another process) — spec §11:
// "When taskboard.json changes externally: reload it, validate it, update
// in-memory state."
package watch

import (
	"fmt"
	"log"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"

	"localtask/internal/project"
)

// Watcher reloads each project's in-memory state when its taskboard.json
// changes on disk outside of this process.
type Watcher struct {
	fsw *fsnotify.Watcher

	mu           sync.RWMutex
	dirToProject map[string]*project.Project
	logger       *log.Logger
}

// Start watches every project's root directory (not recursively — only
// taskboard.json itself is cached in memory and needs invalidation;
// Markdown docs are always read live from disk) and reloads a project
// whenever its taskboard.json is written or replaced. It runs until Close
// is called. Logs go to logger; pass nil to use the standard logger
// (stderr), which is safe for both the HTTP and MCP binaries.
//
// More projects can be watched later via Add — this is how a project
// added at runtime (e.g. through the HTTP API's add-project endpoint)
// gets picked up without restarting the process.
func Start(projects map[string]*project.Project, logger *log.Logger) (*Watcher, error) {
	if logger == nil {
		logger = log.Default()
	}
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{fsw: fsw, dirToProject: make(map[string]*project.Project, len(projects)), logger: logger}
	for _, p := range projects {
		if err := w.addLocked(p); err != nil {
			fsw.Close()
			return nil, err
		}
	}

	go w.run()
	return w, nil
}

// Add starts watching an additional project's root directory. Safe to call
// while the watcher is already running.
func (w *Watcher) Add(p *project.Project) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.addLocked(p)
}

// addLocked adds p to dirToProject and registers the fsnotify watch.
// Callers must hold w.mu.
func (w *Watcher) addLocked(p *project.Project) error {
	dir := filepath.Clean(p.Root)
	if err := w.fsw.Add(dir); err != nil {
		return fmt.Errorf("watching %s: %w", dir, err)
	}
	w.dirToProject[dir] = p
	return nil
}

func (w *Watcher) run() {
	for {
		select {
		case ev, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			if filepath.Base(ev.Name) != project.TaskboardFileName {
				continue
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
				continue
			}
			dir := filepath.Clean(filepath.Dir(ev.Name))
			w.mu.RLock()
			p, ok := w.dirToProject[dir]
			w.mu.RUnlock()
			if !ok {
				continue
			}
			if err := p.Reload(); err != nil {
				w.logger.Printf("watch: reload %q failed: %v", p.Name, err)
			} else {
				w.logger.Printf("watch: reloaded %q (external change)", p.Name)
			}
		case err, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			w.logger.Printf("watch: error: %v", err)
		}
	}
}

// Close stops watching. Safe to call once; further Reload calls will no
// longer happen automatically after Close returns.
func (w *Watcher) Close() error {
	return w.fsw.Close()
}
