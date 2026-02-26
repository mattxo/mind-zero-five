package mind

// deploy provides git, build, and process restart helpers for the mind loop.

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

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

// GitCommitAndPush stages all changes, commits with the given message, and pushes.
func GitCommitAndPush(ctx context.Context, repoDir, message string) error {
	cmds := []struct {
		name string
		args []string
	}{
		{"git", []string{"add", "-A"}},
		{"git", []string{"commit", "-m", message}},
		{"git", []string{"push", "origin", "main"}},
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

// RestartSelf replaces the running mind process with the new mind binary.
func RestartSelf() error {
	binary := "/usr/local/bin/mind"
	args := []string{"mind"}
	env := os.Environ()
	return syscall.Exec(binary, args, env)
}
