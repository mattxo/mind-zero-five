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
func InvokeClaude(ctx context.Context, workDir, prompt string) (*ClaudeResult, error) {
	start := time.Now()

	cmd := exec.CommandContext(ctx, "claude", "-p", prompt, "--output-format", "json")
	cmd.Dir = workDir
	// Clean environment: remove CLAUDECODE to avoid nested-session detection,
	// and ensure the subprocess inherits necessary env vars.
	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, "CLAUDECODE=") {
			cmd.Env = append(cmd.Env, env)
		}
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start)

	exitCode := 0
	if err != nil {
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
