package mind

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
)

// Proposal is a self-improvement proposal produced by assessment.
type Proposal struct {
	Subject     string `json:"subject"`
	Description string `json:"description"`
	Model       string `json:"model"`
}

const assessSystemPrompt = `You are the mind of mind-zero-five — an autonomous system that runs on a Fly machine.
You are analyzing your own source code to identify improvements.

Your code is in the working directory. Read CLAUDE.md to understand your architecture, principles, and invariants.

You have access to all your own source files. Read them. Understand what you are. Then identify what would make you better.

Consider:
- Code correctness and robustness
- Missing error handling or edge cases
- Missing tests (there are very few)
- Architectural improvements
- Features that would make the system more effective
- Patterns of failure visible in the event/task history below

Propose the SINGLE most impactful improvement. Not the easiest — the most impactful.

Output format — exactly one tag:
[IMPROVE:subject|description|model]

Where:
- subject = concise task title (include file paths if relevant)
- description = what to change and why (2-3 sentences)
- model = "haiku" for trivial, "sonnet" for moderate, "opus" for complex

If the codebase is in good shape and no improvements are needed, output:
[OK]

Output ONLY the [IMPROVE:...] or [OK] tag, nothing else.`

// Assess invokes Claude to analyze the codebase and propose an improvement.
func (m *Mind) Assess(ctx context.Context) (*Proposal, error) {
	// Gather context: recent events and task history
	context := m.gatherAssessmentContext(ctx)

	prompt := assessSystemPrompt + "\n\n---\n\n" + context
	prompt += "\nNow read the codebase (start with CLAUDE.md, then pkg/mind/), analyze, and propose."

	result, err := InvokeClaude(ctx, m.repoDir, prompt, "opus")
	if err != nil {
		return nil, fmt.Errorf("assess invocation: %w", err)
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("assess failed (exit %d): %s", result.ExitCode, truncate(result.Result, 500))
	}

	return parseAssessment(result.Result), nil
}

// gatherAssessmentContext builds a summary of recent events and tasks for the assessment prompt.
func (m *Mind) gatherAssessmentContext(ctx context.Context) string {
	var sb strings.Builder

	// Recent events
	events, err := m.events.Recent(ctx, 20)
	if err != nil {
		log.Printf("mind: assess gather events: %v", err)
	} else if len(events) > 0 {
		sb.WriteString("## Recent EventGraph Activity\n\n")
		for _, e := range events {
			contentJSON, _ := json.Marshal(e.Content)
			sb.WriteString(fmt.Sprintf("- [%s] %s (source=%s) %s\n",
				e.Timestamp.Format("2006-01-02 15:04"),
				e.Type, e.Source, string(contentJSON)))
		}
		sb.WriteString("\n")
	}

	// Completed tasks
	completed, err := m.tasks.List(ctx, "completed", 10)
	if err != nil {
		log.Printf("mind: assess gather completed tasks: %v", err)
	} else if len(completed) > 0 {
		sb.WriteString("## Recently Completed Tasks\n\n")
		for _, t := range completed {
			sb.WriteString(fmt.Sprintf("- %s (source=%s, assignee=%s)\n", t.Subject, t.Source, t.Assignee))
		}
		sb.WriteString("\n")
	}

	// Blocked tasks
	blocked, err := m.tasks.List(ctx, "blocked", 10)
	if err != nil {
		log.Printf("mind: assess gather blocked tasks: %v", err)
	} else if len(blocked) > 0 {
		sb.WriteString("## Blocked Tasks (potential failure patterns)\n\n")
		for _, t := range blocked {
			sb.WriteString(fmt.Sprintf("- %s (source=%s, assignee=%s)\n", t.Subject, t.Source, t.Assignee))
		}
		sb.WriteString("\n")
	}

	if sb.Len() == 0 {
		sb.WriteString("No recent event or task history available.\n")
	}

	return sb.String()
}

// improveTagRe matches [IMPROVE:subject|description|model] tags.
var improveTagRe = regexp.MustCompile(`\[IMPROVE:([^|]+)\|([^|]+)\|([^]]+)\]`)

func parseAssessment(response string) *Proposal {
	if strings.Contains(response, "[OK]") {
		return nil
	}

	matches := improveTagRe.FindStringSubmatch(response)
	if matches == nil {
		return nil
	}

	model := strings.TrimSpace(strings.ToLower(matches[3]))
	switch model {
	case "haiku", "sonnet", "opus":
	default:
		model = "sonnet"
	}

	return &Proposal{
		Subject:     strings.TrimSpace(matches[1]),
		Description: strings.TrimSpace(matches[2]),
		Model:       model,
	}
}
