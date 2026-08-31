package watch

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"localtask/internal/project"
)

func writeTaskboard(t *testing.T, root, description string) {
	t.Helper()
	writeTaskboardNamed(t, root, "DanceNote", description)
}

func writeTaskboardNamed(t *testing.T, root, name, description string) {
	t.Helper()
	content := `{"version":1,"project":"` + name + `","description":"` + description + `","tickets":[]}`
	if err := os.WriteFile(filepath.Join(root, project.TaskboardFileName), []byte(content), 0o644); err != nil {
		t.Fatalf("writing taskboard: %v", err)
	}
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}

func TestWatcherReloadsOnExternalChange(t *testing.T) {
	root := t.TempDir()
	writeTaskboard(t, root, "original")

	p, err := project.LoadProject("DanceNote", root)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}

	w, err := Start(map[string]*project.Project{"DanceNote": p}, nil)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { w.Close() })

	// Simulate an external process (not this Project's Save) editing the file.
	writeTaskboard(t, root, "updated externally")

	waitFor(t, 3*time.Second, func() bool {
		return p.Snapshot().Description == "updated externally"
	})
}

func TestWatcherIgnoresInvalidExternalEditThenRecovers(t *testing.T) {
	root := t.TempDir()
	writeTaskboard(t, root, "original")

	p, err := project.LoadProject("DanceNote", root)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}

	w, err := Start(map[string]*project.Project{"DanceNote": p}, nil)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { w.Close() })

	if err := os.WriteFile(filepath.Join(root, project.TaskboardFileName), []byte("not json"), 0o644); err != nil {
		t.Fatalf("writing invalid taskboard: %v", err)
	}
	// Give the watcher time to observe and reject the bad write.
	time.Sleep(200 * time.Millisecond)
	if got := p.Snapshot().Description; got != "original" {
		t.Fatalf("in-memory state changed on invalid external edit: %q", got)
	}

	writeTaskboard(t, root, "recovered")
	waitFor(t, 3*time.Second, func() bool {
		return p.Snapshot().Description == "recovered"
	})
}

func TestWatcherAddAfterStart(t *testing.T) {
	firstRoot := t.TempDir()
	writeTaskboard(t, firstRoot, "original")
	first, err := project.LoadProject("DanceNote", firstRoot)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}

	w, err := Start(map[string]*project.Project{"DanceNote": first}, nil)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { w.Close() })

	secondRoot := t.TempDir()
	writeTaskboardNamed(t, secondRoot, "JobTracker", "original")
	second, err := project.LoadProject("JobTracker", secondRoot)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	if err := w.Add(second); err != nil {
		t.Fatalf("Add: %v", err)
	}

	writeTaskboardNamed(t, secondRoot, "JobTracker", "updated externally")
	waitFor(t, 3*time.Second, func() bool {
		return second.Snapshot().Description == "updated externally"
	})

	// The originally-watched project should still be watched too.
	writeTaskboard(t, firstRoot, "also updated")
	waitFor(t, 3*time.Second, func() bool {
		return first.Snapshot().Description == "also updated"
	})
}
