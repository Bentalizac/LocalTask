package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestReadNotARepo(t *testing.T) {
	root := t.TempDir()
	info, err := Read(context.Background(), root, 10)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if info.IsRepo {
		t.Fatal("expected IsRepo=false for a plain directory")
	}
}

func TestReadCleanRepoWithCommits(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGit(t, root, "add", "README.md")
	runGit(t, root, "commit", "-q", "-m", "first commit")
	runGit(t, root, "commit", "--allow-empty", "-q", "-m", "second commit")

	info, err := Read(context.Background(), root, 10)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !info.IsRepo {
		t.Fatal("expected IsRepo=true")
	}
	if info.Branch != "main" {
		t.Fatalf("expected branch main, got %q", info.Branch)
	}
	if !info.Clean || info.ChangedFiles != 0 {
		t.Fatalf("expected clean tree, got clean=%v changed=%d", info.Clean, info.ChangedFiles)
	}
	if len(info.RecentCommits) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(info.RecentCommits))
	}
	if info.RecentCommits[0].Subject != "second commit" {
		t.Fatalf("expected most recent commit first, got %q", info.RecentCommits[0].Subject)
	}
}

func TestReadDirtyRepo(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGit(t, root, "add", "README.md")
	runGit(t, root, "commit", "-q", "-m", "first commit")

	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatalf("modify file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "untracked.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatalf("write untracked: %v", err)
	}

	info, err := Read(context.Background(), root, 10)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if info.Clean {
		t.Fatal("expected dirty working tree")
	}
	if info.ChangedFiles != 2 {
		t.Fatalf("expected 2 changed files, got %d", info.ChangedFiles)
	}
}

func TestReadRepoWithNoCommits(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init", "-q", "-b", "main")

	info, err := Read(context.Background(), root, 10)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !info.IsRepo {
		t.Fatal("expected IsRepo=true")
	}
	if len(info.RecentCommits) != 0 {
		t.Fatalf("expected no commits, got %d", len(info.RecentCommits))
	}
}
