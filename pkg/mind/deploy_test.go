package mind

import (
	"context"
	"errors"
	"os/exec"
	"testing"
)

// setupWorkRepo creates a working git repo with origin pointing to a bare repo,
// an initial empty commit already pushed, and a clean working tree.
func setupWorkRepo(t *testing.T) (workDir string) {
	t.Helper()

	remote := t.TempDir()
	work := t.TempDir()

	// Bare "remote"
	if out, err := exec.Command("git", "init", "--bare", remote).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v\n%s", err, out)
	}

	// Working repo
	cmds := [][]string{
		{"git", "init", "-b", "main"},
		{"git", "config", "user.email", "test@example.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "remote", "add", "origin", remote},
		{"git", "commit", "--allow-empty", "-m", "initial"},
		{"git", "push", "origin", "main"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = work
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("setup %v: %v\n%s", args, err, out)
		}
	}
	return work
}

func TestGitCommitAndPushErrNothingToPush(t *testing.T) {
	work := setupWorkRepo(t)

	// Tree is clean, remote is up-to-date → must return ErrNothingToPush
	err := GitCommitAndPush(context.Background(), work, "test commit")
	if !errors.Is(err, ErrNothingToPush) {
		t.Errorf("expected ErrNothingToPush, got: %v", err)
	}
}

func TestGitCommitAndPushBadRemote(t *testing.T) {
	work := t.TempDir()

	cmds := [][]string{
		{"git", "init", "-b", "main"},
		{"git", "config", "user.email", "test@example.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "remote", "add", "origin", "git@bad-host.invalid:bad/repo.git"},
		{"git", "commit", "--allow-empty", "-m", "initial"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = work
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("setup %v: %v\n%s", args, err, out)
		}
	}

	// Push will fail because the remote is unreachable
	err := GitCommitAndPush(context.Background(), work, "test commit")
	if err == nil {
		t.Fatal("expected error for bad remote, got nil")
	}
	if errors.Is(err, ErrNothingToPush) {
		t.Errorf("expected non-ErrNothingToPush error for bad remote, got ErrNothingToPush")
	}
}
