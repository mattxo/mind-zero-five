package mind

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// ClaudeResult holds the output from a Claude Code CLI invocation.
type ClaudeResult struct {
	Result   string        `json:"result"`
	Duration time.Duration `json:"duration"`
	ExitCode int           `json:"exit_code"`
}

// InvokeClaude runs the Claude Code CLI with the given prompt and returns the result.
func InvokeClaude(ctx context.Context, workDir, prompt string) (*ClaudeResult, error) {
	start := time.Now()

	cmd := exec.CommandContext(ctx, "claude", "-p", prompt, "--output-format", "json")
	cmd.Dir = workDir

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

	// Claude --output-format json wraps the result in a JSON object with a "result" field.
	var parsed struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		// If JSON parsing fails, use raw stdout as the result.
		return &ClaudeResult{
			Result:   stdout.String(),
			Duration: duration,
			ExitCode: exitCode,
		}, nil
	}

	return &ClaudeResult{
		Result:   parsed.Result,
		Duration: duration,
		ExitCode: exitCode,
	}, nil
}
