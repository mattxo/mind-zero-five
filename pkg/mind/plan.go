package mind

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"mind-zero-five/pkg/task"
)

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

Example:
[TASK:Add validation to pkg/api/handler.go|Add input validation for the Create endpoint, checking required fields and returning 400 on invalid input|sonnet]
[TASK:Add test for validation in pkg/api/handler_test.go|Test that invalid input returns 400 with appropriate error message|sonnet]

Output ONLY the [TASK:...] tags, nothing else.`

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

	result, err := InvokeClaude(ctx, m.repoDir, fullPrompt, "opus")
	if err != nil {
		return nil, fmt.Errorf("plan invocation: %w", err)
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("plan failed (exit %d): %s", result.ExitCode, truncate(result.Result, 500))
	}

	subtasks := parseSubtasks(result.Result)
	if len(subtasks) == 0 {
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
