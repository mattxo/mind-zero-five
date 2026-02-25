package mind

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// GitCommitAndPush stages all changes, commits with the given message, and pushes.
func GitCommitAndPush(ctx context.Context, repoDir, message string) error {
	cmds := []struct {
		name string
		args []string
	}{
		{"git", []string{"add", "-A"}},
		{"git", []string{"commit", "-m", message}},
		{"git", []string{"push"}},
	}
	for _, c := range cmds {
		cmd := exec.CommandContext(ctx, c.name, c.args...)
		cmd.Dir = repoDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s %v: %w\n%s", c.name, c.args, err, string(out))
		}
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
		{"/usr/local/bin/eg", "./cmd/eg"},
	}
	for _, t := range targets {
		cmd := exec.CommandContext(ctx, "go", "build", "-o", t.output, t.pkg)
		cmd.Dir = repoDir
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
		{"build", "./..."},
		{"test", "./..."},
	} {
		cmd := exec.CommandContext(ctx, "go", args...)
		cmd.Dir = repoDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("go %v: %w\n%s", args, err, string(out))
		}
	}
	return nil
}

// RestartSelf replaces the running process with the new server binary.
func RestartSelf() error {
	binary := "/usr/local/bin/server"
	args := os.Args
	env := os.Environ()
	return syscall.Exec(binary, args, env)
}
