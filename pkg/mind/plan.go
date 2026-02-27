package mind

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	"mind-zero-five/pkg/task"
)

// ErrAlreadyDone is returned by Plan when the response indicates the task is
// already complete and no changes are needed.
var ErrAlreadyDone = errors.New("task already done")

// SubtaskSpec describes a subtask produced by the planning phase.
type SubtaskSpec struct {
	Subject     string
	Description string
	Model       string // "haiku", "sonnet", "opus" — complexity-routed
}

// planningSystemPrompt instructs the planner to decompose tasks into subtasks.
const planningSystemPrompt = `You are a senior software architect decomposing a task into small, focused subtasks.

Rules:
1. Each subtask should be ONE specific change (usually one file).
2. Include the exact file path in the subject.
3. Order: schema/models first, then handlers/logic, then tests.
4. Maximum 8 subtasks. If the task needs more, it's too big — split differently.
5. Assign a model based on complexity:
   - "haiku" for trivial changes (add a comment, rename, simple one-liner)
   - "sonnet" for moderate changes (new function, modify handler, add test)
   - "opus" for complex changes (new package, architectural refactor, tricky logic)

Output format — one tag per subtask, each on its own line:
[TASK:subject|description|model]

If the task is already complete and requires NO code changes, output exactly:
[ALREADY_DONE]

Example:
[TASK:Add validation to pkg/api/handler.go|Add input validation for the Create endpoint, checking required fields and returning 400 on invalid input|sonnet]
[TASK:Add test for validation in pkg/api/handler_test.go|Test that invalid input returns 400 with appropriate error message|sonnet]

Output ONLY the [TASK:...] tags or [ALREADY_DONE], nothing else.`

// Plan asks Opus to decompose a task into subtasks.
func (m *Mind) Plan(ctx context.Context, t *task.Task) ([]SubtaskSpec, error) {
	prompt := fmt.Sprintf("Decompose this task into focused subtasks.\n\nSubject: %s\n", t.Subject)
	if t.Description != "" {
		prompt += fmt.Sprintf("\nDescription:\n%s\n", t.Description)
	}
	prompt += fmt.Sprintf("\nWorking directory: %s\n", m.repoDir)
	prompt += "\nAnalyze the codebase to understand the current structure before decomposing."

	// Use the system prompt by prepending it to the prompt (Claude CLI -p mode)
	fullPrompt := planningSystemPrompt + "\n\n---\n\n" + prompt

	result, err := invokeClaudeFn(ctx, m.repoDir, fullPrompt, "opus")
	if err != nil {
		return nil, fmt.Errorf("plan invocation: %w", err)
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("plan failed (exit %d): %s", result.ExitCode, truncate(result.Result, 500))
	}

	subtasks := parseSubtasks(result.Result)
	if len(subtasks) == 0 {
		if isAlreadyDone(result.Result) {
			return nil, ErrAlreadyDone
		}
		return nil, fmt.Errorf("plan produced no subtasks from response: %s", truncate(result.Result, 500))
	}
	if len(subtasks) > 8 {
		subtasks = subtasks[:8]
		log.Printf("mind: plan capped at 8 subtasks (had %d)", len(subtasks))
	}

	return subtasks, nil
}

// taskTagRe matches [TASK:subject|description|model] tags.
var taskTagRe = regexp.MustCompile(`\[TASK:([^|]+)\|([^|]+)\|([^]]+)\]`)

// isAlreadyDone returns true if the response contains strong signals indicating
// the task requires no changes. The explicit [ALREADY_DONE] tag (emitted by the
// planner when it determines nothing needs doing) is the primary signal.
// Natural-language phrases are kept as a fallback but "completed" is
// intentionally excluded — it appears in sentences like "needs to be completed
// before release" or "not yet completed" and would cause false positives that
// silently drop real work.
func isAlreadyDone(response string) bool {
	// Primary: explicit tag from the planner.
	if strings.Contains(response, "[ALREADY_DONE]") {
		return true
	}

	// Fallback: high-confidence phrases that are unlikely to appear in a
	// negative or conditional context.
	lower := strings.ToLower(response)
	indicators := []string{
		"already done",
		"already implemented",
		"no changes needed",
		"no changes required",
		"already exists",
		"already present",
	}
	for _, phrase := range indicators {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

func parseSubtasks(response string) []SubtaskSpec {
	matches := taskTagRe.FindAllStringSubmatch(response, -1)
	var subtasks []SubtaskSpec
	for _, m := range matches {
		subject := strings.TrimSpace(m[1])
		desc := strings.TrimSpace(m[2])
		model := strings.TrimSpace(strings.ToLower(m[3]))

		// Validate model
		switch model {
		case "haiku", "sonnet", "opus":
			// ok
		default:
			model = "sonnet" // safe default
		}

		subtasks = append(subtasks, SubtaskSpec{
			Subject:     subject,
			Description: desc,
			Model:       model,
		})
	}
	return subtasks
}
