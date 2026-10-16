// Package git provides read-only Git information about a project directory
// (current branch, working tree status, recent commits) by shelling out to
// the system git binary. Per spec §21, Git is never a prerequisite: any
// failure to read Git info (not a repo, git not installed, etc.) degrades
// to Info{IsRepo: false} rather than an error.
package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Commit is one entry in a project's recent commit history.
type Commit struct {
	Hash      string    `json:"hash"`
	ShortHash string    `json:"short_hash"`
	Subject   string    `json:"subject"`
	Author    string    `json:"author"`
	Date      time.Time `json:"date"`
}

// Info is a project's read-only Git snapshot.
type Info struct {
	IsRepo        bool     `json:"is_repo"`
	Branch        string   `json:"branch,omitempty"`
	Clean         bool     `json:"clean"`
	ChangedFiles  int      `json:"changed_files"`
	RecentCommits []Commit `json:"recent_commits,omitempty"`
}

const defaultTimeout = 5 * time.Second

func run(ctx context.Context, root string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = root
	// Stops `git status` from refreshing .git/index, so reading git info
	// never writes into the project's repo.
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return out.String(), nil
}

func isRepo(ctx context.Context, root string) bool {
	out, err := run(ctx, root, "rev-parse", "--is-inside-work-tree")
	return err == nil && strings.TrimSpace(out) == "true"
}

// branchName resolves the current branch. It tries symbolic-ref first,
// which works even on the unborn HEAD of a repo with no commits yet;
// rev-parse --abbrev-ref HEAD (the fallback) fails in that case.
func branchName(ctx context.Context, root string) (string, error) {
	if out, err := run(ctx, root, "symbolic-ref", "--short", "HEAD"); err == nil {
		return strings.TrimSpace(out), nil
	}
	out, err := run(ctx, root, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// Read returns root's Git info, including up to recentCommitCount recent
// commits (defaulting to 10 if <= 0). If root is not a Git repository (or
// git isn't installed), it returns Info{IsRepo: false} with a nil error —
// this is a normal, expected state, not a failure.
func Read(ctx context.Context, root string, recentCommitCount int) (*Info, error) {
	if !isRepo(ctx, root) {
		return &Info{IsRepo: false}, nil
	}

	branch, err := branchName(ctx, root)
	if err != nil {
		return nil, err
	}
	status, err := run(ctx, root, "status", "--porcelain")
	if err != nil {
		return nil, err
	}
	changedFiles := 0
	for line := range strings.SplitSeq(status, "\n") {
		if strings.TrimSpace(line) != "" {
			changedFiles++
		}
	}

	commits, err := recentCommits(ctx, root, recentCommitCount)
	if err != nil {
		return nil, err
	}

	return &Info{
		IsRepo:        true,
		Branch:        branch,
		Clean:         changedFiles == 0,
		ChangedFiles:  changedFiles,
		RecentCommits: commits,
	}, nil
}

func recentCommits(ctx context.Context, root string, n int) ([]Commit, error) {
	if n <= 0 {
		n = 10
	}
	const sep = "\x1f"
	format := strings.Join([]string{"%H", "%h", "%s", "%an", "%aI"}, sep)
	out, err := run(ctx, root, "log", fmt.Sprintf("-n%d", n), "--pretty=format:"+format)
	if err != nil {
		// A repo with no commits yet makes `git log` fail; that's not a
		// real error, just an empty history.
		if strings.Contains(err.Error(), "does not have any commits yet") {
			return []Commit{}, nil
		}
		return nil, err
	}

	trimmed := strings.TrimSpace(out)
	if trimmed == "" {
		return []Commit{}, nil
	}
	lines := strings.Split(trimmed, "\n")
	commits := make([]Commit, 0, len(lines))
	for _, line := range lines {
		parts := strings.Split(line, sep)
		if len(parts) != 5 {
			continue
		}
		date, _ := time.Parse(time.RFC3339, parts[4])
		commits = append(commits, Commit{
			Hash: parts[0], ShortHash: parts[1], Subject: parts[2], Author: parts[3], Date: date,
		})
	}
	return commits, nil
}
