package registry

import (
	"os"
	"path/filepath"
	"testing"

	"localtask/internal/config"
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
	p, err := reg.Add("JobTracker", projectDir)
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

	p, err := reg.Add("DanceNote", projectDir)
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
	if _, err := reg.Add("JobTracker", projectDir); err != nil {
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
	if _, err := reg.Add("DanceNote", dirA); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := reg.Add("DanceNote", dirB); err == nil {
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

	if _, err := reg.Add("X", "relative/path"); err == nil {
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

	if _, err := reg.Add("X", filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}
