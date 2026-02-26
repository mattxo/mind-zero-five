package mind

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ClaudeResult holds the output from a Claude Code CLI invocation.
type ClaudeResult struct {
	Result   string        `json:"result"`
	Stderr   string        `json:"stderr,omitempty"`
	Duration time.Duration `json:"duration"`
	ExitCode int           `json:"exit_code"`
}

// InvokeClaude runs the Claude Code CLI with the given prompt and returns the result.
// If model is non-empty, it is passed as --model to the CLI (e.g. "opus", "sonnet", "haiku").
func InvokeClaude(ctx context.Context, workDir, prompt, model string) (*ClaudeResult, error) {
	start := time.Now()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	args := []string{"-p", prompt, "--output-format", "json", "--allowedTools", "Edit Write Read Glob Grep Bash"}
	if model != "" {
		args = append(args, "--model", model)
	}
	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = workDir
	// Clean environment: remove vars that interfere with Claude CLI auth.
	// CLAUDECODE triggers nested-session detection.
	// ANTHROPIC_API_KEY overrides OAuth credentials if set incorrectly.
	// The CLI will use OAuth creds from ~/.claude/.credentials.json.
	// Ensure Go is in PATH for build/test commands.
	hasPath := false
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "CLAUDECODE=") || strings.HasPrefix(env, "ANTHROPIC_API_KEY=") {
			continue
		}
		if strings.HasPrefix(env, "PATH=") {
			if !strings.Contains(env, "/usr/local/go/bin") {
				env = env + ":/usr/local/go/bin"
			}
			hasPath = true
		}
		cmd.Env = append(cmd.Env, env)
	}
	if !hasPath {
		cmd.Env = append(cmd.Env, "PATH=/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin")
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start)

	exitCode := 0
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return &ClaudeResult{
				Stderr:   "claude invocation timed out after 10 minutes",
				Duration: duration,
				ExitCode: -1,
			}, nil
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("run claude: %w (stderr: %s)", err, stderr.String())
		}
	}

	stderrStr := stderr.String()

	// Claude --output-format json wraps the result in a JSON object with a "result" field.
	var parsed struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		// If JSON parsing fails, use raw stdout as the result.
		return &ClaudeResult{
			Result:   stdout.String(),
			Stderr:   stderrStr,
			Duration: duration,
			ExitCode: exitCode,
		}, nil
	}

	return &ClaudeResult{
		Result:   parsed.Result,
		Stderr:   stderrStr,
		Duration: duration,
		ExitCode: exitCode,
	}, nil
}
