package mind

// deploy provides git, build, and process restart helpers for the mind loop.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// ErrNothingToPush is returned by GitCommitAndPush when the working tree is
// clean and git push confirms the remote is already up-to-date. This is a
// harmless no-op; callers can distinguish it from a real push failure.
var ErrNothingToPush = errors.New("nothing to push: working tree clean and remote up-to-date")

// goCmd creates an exec.Cmd for the go tool with PATH set correctly.
func goCmd(ctx context.Context, repoDir string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "/usr/local/go/bin/go", args...)
	cmd.Dir = repoDir
	// Ensure Go toolchain can find itself
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "PATH=") && !strings.Contains(env, "/usr/local/go/bin") {
			env = env + ":/usr/local/go/bin"
		}
		cmd.Env = append(cmd.Env, env)
	}
	return cmd
}

// CleanWorkingTree checks for uncommitted changes left by a crash and commits them
// separately as orphaned recovery. Returns the list of files that were dirty, or nil
// if the tree was clean. This prevents cross-task contamination from git add -A
// sweeping orphaned files into the next task's commit.
func CleanWorkingTree(ctx context.Context, repoDir string) ([]string, error) {
	statusCmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	statusCmd.Dir = repoDir
	statusOut, err := statusCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git status --porcelain: %w", err)
	}

	output := strings.TrimSpace(string(statusOut))
	if output == "" {
		return nil, nil // clean
	}

	// Collect dirty file paths for logging
	var files []string
	for _, line := range strings.Split(output, "\n") {
		if len(line) > 3 {
			files = append(files, strings.TrimSpace(line[2:]))
		}
	}

	log.Printf("mind: cleanWorkingTree: %d orphaned files from crash recovery", len(files))

	// Commit orphaned changes under a clear label
	for _, args := range [][]string{
		{"add", "-A"},
		{"commit", "-m", "mind: orphaned changes from crash recovery"},
		{"push", "origin", "main"},
	} {
		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Dir = repoDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			return files, fmt.Errorf("git %v: %w\n%s", args, err, string(out))
		}
	}

	return files, nil
}

// GitCommitAndPush stages all changes, commits with the given message, and pushes.
// If the working tree is clean (nothing to commit), it skips add/commit but still
// pushes to ensure any prior unpushed commits reach the remote.
// Returns nil on success (including clean no-op). Real errors mean data is at risk.
func GitCommitAndPush(ctx context.Context, repoDir, message string) error {
	// Check if there's anything to commit
	statusCmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	statusCmd.Dir = repoDir
	statusOut, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("git status --porcelain: %w", err)
	}

	treeClean := len(strings.TrimSpace(string(statusOut))) == 0

	if !treeClean {
		// Stage and commit
		for _, args := range [][]string{
			{"add", "-A"},
			{"commit", "-m", message},
		} {
			cmd := exec.CommandContext(ctx, "git", args...)
			cmd.Dir = repoDir
			out, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("git %v: %w\n%s", args, err, string(out))
			}
		}
	}

	// Always push — there may be unpushed commits from earlier subtasks
	pushCmd := exec.CommandContext(ctx, "git", "push", "origin", "main")
	pushCmd.Dir = repoDir
	pushOut, err := pushCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push origin main: %w\n%s", err, string(pushOut))
	}

	// If the tree was clean and push transferred nothing, this was a true no-op.
	if treeClean && strings.Contains(string(pushOut), "Everything up-to-date") {
		return ErrNothingToPush
	}
	return nil
}

// Build compiles the server and eg binaries.
func Build(ctx context.Context, repoDir string) error {
	targets := []struct {
		output string
		pkg    string
	}{
		{"/usr/local/bin/server", "./cmd/server"},
		{"/usr/local/bin/mind", "./cmd/mind"},
		{"/usr/local/bin/eg", "./cmd/eg"},
	}
	for _, t := range targets {
		cmd := goCmd(ctx, repoDir, "build", "-o", t.output, t.pkg)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("build %s: %w\n%s", t.pkg, err, string(out))
		}
	}
	return nil
}

// BuildAndTest runs go build ./... && go test ./... in the repo.
func BuildAndTest(ctx context.Context, repoDir string) error {
	for _, args := range [][]string{
		{"build", "./cmd/server", "./cmd/mind", "./cmd/eg"},
		{"test", "./pkg/...", "./internal/..."},
	} {
		cmd := goCmd(ctx, repoDir, args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("go %v: %w\n%s", args, err, string(out))
		}
	}
	return nil
}

// RestartSelf backs up the current binary, then replaces the running mind
// process with the new mind binary. The watchdog can restore the backup
// if the new binary crashes.
func RestartSelf() error {
	binary := "/usr/local/bin/mind"
	backup := "/usr/local/bin/mind.bak"

	// Backup current binary before replacing ourselves
	if err := copyFile(binary, backup); err != nil {
		log.Printf("mind: backup binary before restart: %v (continuing anyway)", err)
	}

	args := []string{"mind"}
	env := os.Environ()
	return syscall.Exec(binary, args, env)
}

// writeHeartbeat touches the heartbeat file so the watchdog knows we're alive.
func writeHeartbeat() {
	f, err := os.Create("/tmp/mind-heartbeat")
	if err != nil {
		return
	}
	f.Close()
}

func copyFile(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	d, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer d.Close()

	_, err = io.Copy(d, s)
	return err
}
