package mind

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// Proposal is a self-improvement proposal produced by assessment.
type Proposal struct {
	Subject     string `json:"subject"`
	Description string `json:"description"`
	Model       string `json:"model"`
}

const assessSystemPrompt = `You are the mind of mind-zero-five — an autonomous system running on a Fly machine.
You are assessing your own operational health.

Below are diagnostics showing your current state — git health, task anomalies, error patterns, system metrics.
Your code is in the working directory. Read CLAUDE.md for your architecture and principles. Read pkg/mind/ for what checks exist today.

YOUR CORE QUESTION: For every problem you see, ask:
1. What role or check SHOULD have caught this before it became a problem?
2. Does that check exist in my codebase? (Look in pkg/mind/ — poll loop, assess, deploy)
3. If it doesn't exist: propose building it. That's your highest-priority improvement.

PRIORITY ORDER:
1. BROKEN OPERATIONS — Things failing right now (push broken, tasks stuck, errors repeating)
2. MISSING WATCHERS — Failure modes that nothing monitors. If a failure has no detector, it will recur silently.
3. MISSING FIXERS — Problems detected but with no automated response.
4. CODE IMPROVEMENTS — Only after operational coverage is solid.

The system needs a watcher and a fixer for every failure mode. Gaps in that coverage are more important than any code improvement.

Propose the SINGLE most impactful improvement. Not the easiest — the most impactful.

Output format — exactly one tag:
[IMPROVE:subject|description|model]

Where:
- subject = concise task title (include file paths if relevant)
- description = what to change and why (2-3 sentences)
- model = "haiku" for trivial, "sonnet" for moderate, "opus" for complex

If the system is healthy, all failure modes have watchers, and no improvements are needed, output:
[OK]

Output ONLY the [IMPROVE:...] or [OK] tag, nothing else.`

// Assess invokes Claude to analyze the codebase and propose an improvement.
func (m *Mind) Assess(ctx context.Context) (*Proposal, error) {
	// Gather operational context: diagnostics + event/task history
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

// gatherAssessmentContext builds operational diagnostics and event/task history for the assessment prompt.
func (m *Mind) gatherAssessmentContext(ctx context.Context) string {
	var sb strings.Builder

	// --- Operational Diagnostics ---
	sb.WriteString("## Operational Diagnostics\n\n")
	m.diagGitHealth(ctx, &sb)
	m.diagTaskHealth(ctx, &sb)
	m.diagErrorPatterns(ctx, &sb)

	// --- Recent events ---
	events, err := m.events.Recent(ctx, 30)
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

	// --- Completed tasks ---
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

	// --- Blocked tasks with reasons ---
	blocked, err := m.tasks.List(ctx, "blocked", 10)
	if err != nil {
		log.Printf("mind: assess gather blocked tasks: %v", err)
	} else if len(blocked) > 0 {
		sb.WriteString("## Blocked Tasks\n\n")
		for _, t := range blocked {
			reason := ""
			if t.Metadata != nil {
				if r, ok := t.Metadata["blocked_reason"].(string); ok {
					reason = r
				}
			}
			if reason != "" {
				sb.WriteString(fmt.Sprintf("- %s (source=%s, assignee=%s) — REASON: %s\n", t.Subject, t.Source, t.Assignee, reason))
			} else {
				sb.WriteString(fmt.Sprintf("- %s (source=%s, assignee=%s)\n", t.Subject, t.Source, t.Assignee))
			}
		}
		sb.WriteString("\n")
	}

	if sb.Len() == 0 {
		sb.WriteString("No diagnostic data available.\n")
	}

	return sb.String()
}

// diagGitHealth checks git push capability and unpushed commit count.
func (m *Mind) diagGitHealth(ctx context.Context, sb *strings.Builder) {
	sb.WriteString("### Git Health\n")

	// Check push capability
	cmd := exec.CommandContext(ctx, "git", "push", "--dry-run", "origin", "main")
	cmd.Dir = m.repoDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		sb.WriteString(fmt.Sprintf("- **PUSH BROKEN**: git push --dry-run failed: %s %s\n", err, strings.TrimSpace(string(out))))
	} else {
		sb.WriteString("- Push: OK\n")
	}

	// Count unpushed commits
	cmd = exec.CommandContext(ctx, "git", "log", "origin/main..HEAD", "--oneline")
	cmd.Dir = m.repoDir
	out, err = cmd.CombinedOutput()
	if err != nil {
		sb.WriteString(fmt.Sprintf("- Unpushed commits: unknown (%v)\n", err))
	} else {
		lines := strings.TrimSpace(string(out))
		if lines == "" {
			sb.WriteString("- Unpushed commits: 0\n")
		} else {
			count := len(strings.Split(lines, "\n"))
			sb.WriteString(fmt.Sprintf("- **Unpushed commits: %d**\n", count))
		}
	}

	// Check for uncommitted changes
	cmd = exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = m.repoDir
	out, err = cmd.CombinedOutput()
	if err == nil && len(strings.TrimSpace(string(out))) > 0 {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		sb.WriteString(fmt.Sprintf("- Uncommitted changes: %d files\n", len(lines)))
	}

	sb.WriteString("\n")
}

// diagTaskHealth checks for orphaned and stale tasks.
func (m *Mind) diagTaskHealth(ctx context.Context, sb *strings.Builder) {
	sb.WriteString("### Task Health\n")

	// Counts by status
	for _, status := range []string{"pending", "in_progress", "blocked", "completed"} {
		tasks, err := m.tasks.List(ctx, status, 100)
		if err != nil {
			sb.WriteString(fmt.Sprintf("- %s: error (%v)\n", status, err))
			continue
		}
		sb.WriteString(fmt.Sprintf("- %s: %d\n", status, len(tasks)))

		// Check for anomalies within in_progress tasks
		if status == "in_progress" {
			for _, t := range tasks {
				age := time.Since(t.UpdatedAt)
				if age > 1*time.Hour {
					sb.WriteString(fmt.Sprintf("  - **STALE**: %q (in_progress for %s, assignee=%s)\n", t.Subject, age.Round(time.Minute), t.Assignee))
				}

				// Check if parent is already completed (orphaned subtask)
				if t.ParentID != "" {
					parent, err := m.tasks.Get(ctx, t.ParentID)
					if err == nil && parent.Status == "completed" {
						sb.WriteString(fmt.Sprintf("  - **ORPHANED**: %q — parent task already completed\n", t.Subject))
					}
				}
			}
		}

		// Check for abandoned tasks (blocked, retries exhausted)
		if status == "blocked" {
			for _, t := range tasks {
				retryCount := 0
				if t.Metadata != nil {
					switch v := t.Metadata["retry_count"].(type) {
					case float64:
						retryCount = int(v)
					case int:
						retryCount = v
					}
				}
				if retryCount >= 3 {
					reason := ""
					if t.Metadata != nil {
						if r, ok := t.Metadata["blocked_reason"].(string); ok {
							reason = r
						}
					}
					sb.WriteString(fmt.Sprintf("  - **ABANDONED**: %q (retries exhausted, reason: %s)\n", t.Subject, reason))
				}
			}
		}
	}

	sb.WriteString("\n")
}

// diagErrorPatterns counts recent failure events by type.
func (m *Mind) diagErrorPatterns(ctx context.Context, sb *strings.Builder) {
	sb.WriteString("### Error Patterns (recent events)\n")

	// Look for *.failed events
	failTypes := []string{
		"build.failed",
		"git.commit_push.failed",
		"mind.claude.failed",
		"mind.plan.failed",
		"mind.review.failed",
		"mind.assess.failed",
		"deploy.failed",
		"build.deploy.failed",
		"mind.recovery.failed",
		"mind.recovery.succeeded",
		"mind.error",
	}

	found := false
	for _, ft := range failTypes {
		events, err := m.events.ByType(ctx, ft, 10)
		if err != nil || len(events) == 0 {
			continue
		}
		found = true
		// Count recent ones (last hour)
		recent := 0
		for _, e := range events {
			if time.Since(e.Timestamp) < 1*time.Hour {
				recent++
			}
		}
		sb.WriteString(fmt.Sprintf("- %s: %d total, %d in last hour\n", ft, len(events), recent))
	}

	// Also check for task.blocked events
	blockedEvents, err := m.events.ByType(ctx, "task.blocked", 20)
	if err == nil && len(blockedEvents) > 0 {
		found = true
		recent := 0
		for _, e := range blockedEvents {
			if time.Since(e.Timestamp) < 1*time.Hour {
				recent++
			}
		}
		sb.WriteString(fmt.Sprintf("- task.blocked: %d total, %d in last hour\n", len(blockedEvents), recent))
	}

	if !found {
		sb.WriteString("- No failure events found\n")
	}

	sb.WriteString("\n")
}

// improveTagRe matches [IMPROVE:subject|description|model] tags.
var improveTagRe = regexp.MustCompile(`\[IMPROVE:([^|]+)\|([^|]+)\|([^]]+)\]`)

func parseAssessment(response string) *Proposal {
	if strings.Contains(response, "[OK]") {
		return nil
	}

	matches := improveTagRe.FindStringSubmatch(response)
	if matches == nil {
		// Assessment produced output that wasn't [OK] and didn't match [IMPROVE:...].
		// Log so diagnostics can detect assessment parsing failures.
		log.Printf("mind: assessment output unparseable (len=%d): %s", len(response), truncate(response, 200))
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
