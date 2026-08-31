package config

import (
	"os"
	"path/filepath"
	"testing"
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

func TestLoadValidConfig(t *testing.T) {
	path := writeConfig(t, `
projects:
  - name: DanceNote
    path: /projects/dance-note
  - name: Job Tracker
    path: /projects/job-tracker
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(cfg.Projects))
	}
	p, err := cfg.FindProject("Job Tracker")
	if err != nil {
		t.Fatalf("FindProject: %v", err)
	}
	if p.Path != "/projects/job-tracker" {
		t.Fatalf("unexpected path: %s", p.Path)
	}
}

func TestLoadRejectsDuplicateNames(t *testing.T) {
	path := writeConfig(t, `
projects:
  - name: DanceNote
    path: /projects/a
  - name: DanceNote
    path: /projects/b
`)
	if _, err := Load(path); err == nil {
		t.Fatal("expected error for duplicate project names")
	}
}

func TestLoadRejectsMissingFields(t *testing.T) {
	path := writeConfig(t, `
projects:
  - name: DanceNote
`)
	if _, err := Load(path); err == nil {
		t.Fatal("expected error for missing path")
	}
}

func TestAddProjectPersists(t *testing.T) {
	path := writeConfig(t, `
projects:
  - name: DanceNote
    path: /projects/dance-note
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := cfg.AddProject(ProjectConfig{Name: "JobTracker", Path: "/projects/job-tracker"}); err != nil {
		t.Fatalf("AddProject: %v", err)
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(reloaded.Projects) != 2 {
		t.Fatalf("expected 2 projects on disk, got %d", len(reloaded.Projects))
	}
	p, err := reloaded.FindProject("JobTracker")
	if err != nil {
		t.Fatalf("FindProject: %v", err)
	}
	if p.Path != "/projects/job-tracker" {
		t.Fatalf("unexpected path: %s", p.Path)
	}
}

func TestAddProjectRejectsDuplicateName(t *testing.T) {
	path := writeConfig(t, `
projects:
  - name: DanceNote
    path: /projects/dance-note
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := cfg.AddProject(ProjectConfig{Name: "DanceNote", Path: "/somewhere/else"}); err == nil {
		t.Fatal("expected error for duplicate name")
	}
	if len(cfg.Projects) != 1 {
		t.Fatalf("expected rollback to leave 1 project, got %d", len(cfg.Projects))
	}
}

func TestFindProjectUnknown(t *testing.T) {
	path := writeConfig(t, `
projects:
  - name: DanceNote
    path: /projects/a
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err := cfg.FindProject("Nope"); err == nil {
		t.Fatal("expected error for unknown project")
	}
}
