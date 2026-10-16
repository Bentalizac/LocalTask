package registry

import (
	"os"
	"path/filepath"
	"testing"

	"localtask/internal/config"
	"localtask/internal/ticket"
)

func writeConfig(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "taskboard.config.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("writing config: %v", err)
	}
	return path
}

func TestAddCreatesTaskboardWhenMissing(t *testing.T) {
	configPath := writeConfig(t, "projects: []\n")
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	reg := NewRegistry(cfg, nil, nil)

	projectDir := t.TempDir()
	p, err := reg.Add("JobTracker", projectDir, config.StorageRepo)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if p.Name != "JobTracker" {
		t.Fatalf("unexpected project name: %s", p.Name)
	}
	if _, err := os.Stat(filepath.Join(projectDir, "taskboard.json")); err != nil {
		t.Fatalf("expected taskboard.json to be created: %v", err)
	}

	got, ok := reg.Get("JobTracker")
	if !ok || got != p {
		t.Fatal("expected Get to return the added project")
	}
	if len(reg.List()) != 1 {
		t.Fatalf("expected 1 project in List, got %d", len(reg.List()))
	}
}

func TestAddUsesExistingTaskboard(t *testing.T) {
	configPath := writeConfig(t, "projects: []\n")
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	reg := NewRegistry(cfg, nil, nil)

	projectDir := t.TempDir()
	existing := `{"version":1,"project":"DanceNote","tickets":[{"id":"DN-1","title":"x","type":"task","status":"backlog","priority":"low"}]}`
	if err := os.WriteFile(filepath.Join(projectDir, "taskboard.json"), []byte(existing), 0o644); err != nil {
		t.Fatalf("writing taskboard: %v", err)
	}

	p, err := reg.Add("DanceNote", projectDir, config.StorageRepo)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if len(p.Snapshot().Tickets) != 1 {
		t.Fatalf("expected the existing ticket to be preserved, got %d tickets", len(p.Snapshot().Tickets))
	}
}

func TestAddPersistsToConfig(t *testing.T) {
	configPath := writeConfig(t, "projects: []\n")
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	reg := NewRegistry(cfg, nil, nil)

	projectDir := t.TempDir()
	if _, err := reg.Add("JobTracker", projectDir, ""); err != nil {
		t.Fatalf("Add: %v", err)
	}

	reloaded, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	pc, err := reloaded.FindProject("JobTracker")
	if err != nil {
		t.Fatalf("FindProject: %v", err)
	}
	if pc.Path != projectDir {
		t.Fatalf("expected persisted path %q, got %q", projectDir, pc.Path)
	}
}

func TestAddRejectsDuplicateName(t *testing.T) {
	configPath := writeConfig(t, "projects: []\n")
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	reg := NewRegistry(cfg, nil, nil)

	dirA, dirB := t.TempDir(), t.TempDir()
	if _, err := reg.Add("DanceNote", dirA, ""); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := reg.Add("DanceNote", dirB, ""); err == nil {
		t.Fatal("expected error for duplicate project name")
	}
}

func TestAddRejectsRelativePath(t *testing.T) {
	configPath := writeConfig(t, "projects: []\n")
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	reg := NewRegistry(cfg, nil, nil)

	if _, err := reg.Add("X", "relative/path", ""); err == nil {
		t.Fatal("expected error for relative path")
	}
}

func TestAddRejectsNonexistentPath(t *testing.T) {
	configPath := writeConfig(t, "projects: []\n")
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	reg := NewRegistry(cfg, nil, nil)

	if _, err := reg.Add("X", filepath.Join(t.TempDir(), "nope"), ""); err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}

func TestAddDefaultsToExternalStorage(t *testing.T) {
	configPath := writeConfig(t, "projects: []\n")
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	reg := NewRegistry(cfg, nil, nil)

	projectDir := t.TempDir()
	p, err := reg.Add("JobTracker", projectDir, "")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	wantBoardDir := filepath.Join(filepath.Dir(configPath), config.BoardsDirName, "JobTracker")
	if p.BoardDir != wantBoardDir || p.Root != projectDir {
		t.Fatalf("unexpected dirs: root %q, board %q", p.Root, p.BoardDir)
	}
	if _, err := p.CreateTicket(ticket.Ticket{Title: "x"}, "test"); err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}
	if entries, _ := os.ReadDir(projectDir); len(entries) != 0 {
		t.Fatalf("expected project directory untouched, found %d entries", len(entries))
	}
}

func TestAddRejectsUnsafeName(t *testing.T) {
	configPath := writeConfig(t, "projects: []\n")
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	reg := NewRegistry(cfg, nil, nil)

	if _, err := reg.Add("../escape", t.TempDir(), ""); err == nil {
		t.Fatal("expected error for a name containing a path separator")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(configPath), "escape")); !os.IsNotExist(err) {
		t.Fatalf("expected nothing created outside the boards directory, stat err: %v", err)
	}
}

func TestLoadCreatesMissingExternalBoardOnly(t *testing.T) {
	externalDir, repoDir := t.TempDir(), t.TempDir()
	configPath := writeConfig(t, "projects:\n"+
		"  - name: Work\n    path: "+externalDir+"\n"+
		"  - name: Personal\n    path: "+repoDir+"\n    storage: repo\n")
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	projects, errs := Load(cfg)
	if _, ok := projects["Work"]; !ok {
		t.Fatalf("expected external project to load with a fresh board, errs: %v", errs)
	}
	if _, ok := projects["Personal"]; ok || len(errs) != 1 {
		t.Fatalf("expected repo project with no taskboard.json to fail, errs: %v", errs)
	}
	for _, dir := range []string{externalDir, repoDir} {
		if entries, _ := os.ReadDir(dir); len(entries) != 0 {
			t.Fatalf("expected %s untouched, found %d entries", dir, len(entries))
		}
	}
}

func TestRefreshLoadsProjectsAddedToConfig(t *testing.T) {
	configPath := writeConfig(t, "projects: []\n")
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	reg := NewRegistry(cfg, nil, nil)

	// Another process adds a project to the same config file.
	other, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if _, err := NewRegistry(other, nil, nil).Add("JobTracker", t.TempDir(), ""); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, ok := reg.Get("JobTracker"); ok {
		t.Fatal("expected JobTracker to be unknown before Refresh")
	}

	if errs := reg.Refresh(); len(errs) != 0 {
		t.Fatalf("Refresh: %v", errs)
	}
	before, ok := reg.Get("JobTracker")
	if !ok {
		t.Fatal("expected Refresh to load JobTracker")
	}

	// A second Refresh keeps the already-loaded project as is.
	if errs := reg.Refresh(); len(errs) != 0 {
		t.Fatalf("second Refresh: %v", errs)
	}
	if after, _ := reg.Get("JobTracker"); after != before {
		t.Fatal("expected Refresh to keep the existing *Project")
	}
}

func TestRefreshWithoutConfigIsNoop(t *testing.T) {
	reg := NewRegistry(nil, nil, nil)
	if errs := reg.Refresh(); len(errs) != 0 {
		t.Fatalf("Refresh: %v", errs)
	}
	if len(reg.List()) != 0 {
		t.Fatal("expected no projects")
	}
}
