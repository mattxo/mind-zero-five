package mind

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"mind-zero-five/pkg/task"
)

const reviewSystemPrompt = `You are a senior code reviewer examining a git diff.

Check for:
1. Correctness — does the code do what the task asked?
2. Bugs — nil pointer, off-by-one, missing error handling, race conditions
3. Style — consistent with the codebase patterns
4. Security — no injection, no leaked secrets, proper input validation

If the diff looks good, output exactly:
[OK]

If there are issues, output one tag per issue:
[ISSUE:subject|description|model]

Where:
- subject = short fix description (include file path)
- description = what's wrong and how to fix it
- model = "haiku" for trivial, "sonnet" for moderate, "opus" for complex

Output ONLY [OK] or [ISSUE:...] tags, nothing else.`

// Review asks Opus to review the diff produced since startCommit.
// Returns nil if the review passes, or a list of issue subtasks to fix.
func (m *Mind) Review(ctx context.Context, t *task.Task, startCommit string) ([]SubtaskSpec, error) {
	diff, err := gitDiff(ctx, m.repoDir, startCommit, "HEAD")
	if err != nil {
		return nil, fmt.Errorf("get diff for review: %w", err)
	}
	if diff == "" {
		return nil, nil // no changes to review
	}

	prompt := fmt.Sprintf("Review this diff for task: %s\n\n```diff\n%s\n```", t.Subject, truncate(diff, 8000))
	fullPrompt := reviewSystemPrompt + "\n\n---\n\n" + prompt

	result, err := InvokeClaude(ctx, m.repoDir, fullPrompt, "opus")
	if err != nil {
		return nil, fmt.Errorf("review invocation: %w", err)
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("review failed (exit %d): %s", result.ExitCode, truncate(result.Result, 500))
	}

	return parseReviewResult(result.Result), nil
}

// issueTagRe matches [ISSUE:subject|description|model] tags.
var issueTagRe = regexp.MustCompile(`\[ISSUE:([^|]+)\|([^|]+)\|([^]]+)\]`)

func parseReviewResult(response string) []SubtaskSpec {
	// Check for [OK] first
	if strings.Contains(response, "[OK]") {
		return nil
	}

	matches := issueTagRe.FindAllStringSubmatch(response, -1)
	var issues []SubtaskSpec
	for _, m := range matches {
		subject := strings.TrimSpace(m[1])
		desc := strings.TrimSpace(m[2])
		model := strings.TrimSpace(strings.ToLower(m[3]))

		switch model {
		case "haiku", "sonnet", "opus":
		default:
			model = "sonnet"
		}

		issues = append(issues, SubtaskSpec{
			Subject:     subject,
			Description: desc,
			Model:       model,
		})
	}
	return issues
}

// gitDiff returns the diff between two commits in a repo.
func gitDiff(ctx context.Context, repoDir, from, to string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "diff", from+".."+to)
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git diff %s..%s: %w", from, to, err)
	}
	return string(out), nil
}

// getCurrentCommit returns the current HEAD commit hash.
func getCurrentCommit(ctx context.Context, repoDir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
