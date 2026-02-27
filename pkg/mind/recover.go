package mind

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"mind-zero-five/pkg/task"
)

const recoveryPrompt = `You are the mind's error recovery system. A task just failed. Your job is to diagnose the root cause and fix it.

## Failed Task
Subject: %s
Description: %s

## Error
Type: %s
Reason: %s

## Causal Chain (what led to this failure)
%s

## Recent Git Changes
%s

## Rules
1. Fix the ROOT CAUSE — not a workaround.
2. Do NOT weaken tests to make them pass.
3. Do NOT remove error checking or validation.
4. If the error is environmental (network down, auth expired, external service unavailable), report what's wrong but do NOT change code.
5. If you cannot identify the root cause, make NO changes. Just explain what you found.
6. After making changes, verify with: go build ./cmd/server ./cmd/mind ./cmd/eg && go test ./pkg/... ./internal/...
7. Do NOT commit — just fix and verify.`

// attemptRecovery tries to diagnose and fix the root cause of a task failure.
// Uses the eventgraph causal chain to provide full context to Claude.
// Returns true if recovery succeeded (build/test pass after fix).
func (m *Mind) attemptRecovery(ctx context.Context, t *task.Task, errorType, reason string, causes []string) bool {
	// Guard: one recovery attempt per task per failure cycle.
	if t.Metadata != nil {
		if attempted, ok := t.Metadata["recovery_attempted"].(bool); ok && attempted {
			return false
		}
	}

	// Mark that we're attempting recovery (prevents loops on retry).
	meta := copyMeta(t.Metadata)
	meta["recovery_attempted"] = true
	m.tasks.Update(ctx, t.ID, map[string]any{"metadata": meta})

	startEvent, _ := m.logEvent(ctx, "mind.recovery.started", map[string]any{
		"task_id":    t.ID,
		"error_type": errorType,
		"reason":     truncate(reason, 500),
	}, causes)

	recoveryCauses := causes
	if startEvent != nil {
		recoveryCauses = []string{startEvent.ID}
	}

	log.Printf("mind: attempting recovery for task %s (error: %s)", t.ID, errorType)

	// Gather causal context from the eventgraph.
	causalContext := m.gatherCausalContext(ctx, causes)
	gitContext := m.gatherGitContext(ctx)

	prompt := fmt.Sprintf(recoveryPrompt,
		t.Subject, t.Description,
		errorType, reason,
		causalContext, gitContext)

	result, err := InvokeClaude(ctx, m.repoDir, prompt, "sonnet")
	if err != nil {
		log.Printf("mind: recovery invocation failed for task %s: %v", t.ID, err)
		m.logEvent(ctx, "mind.recovery.failed", map[string]any{
			"task_id": t.ID,
			"stage":   "invocation",
			"error":   err.Error(),
		}, recoveryCauses)
		return false
	}

	if result.ExitCode != 0 {
		log.Printf("mind: recovery claude exited %d for task %s", result.ExitCode, t.ID)
		m.logEvent(ctx, "mind.recovery.failed", map[string]any{
			"task_id":   t.ID,
			"stage":     "claude_exit",
			"exit_code": result.ExitCode,
			"result":    truncate(result.Result, 500),
		}, recoveryCauses)
		return false
	}

	// Verify the fix compiles and tests pass.
	if err := BuildAndTest(ctx, m.repoDir); err != nil {
		log.Printf("mind: recovery build/test failed for task %s: %v", t.ID, err)
		m.logEvent(ctx, "mind.recovery.failed", map[string]any{
			"task_id": t.ID,
			"stage":   "build_test",
			"error":   truncate(err.Error(), 500),
		}, recoveryCauses)
		return false
	}

	log.Printf("mind: recovery succeeded for task %s", t.ID)
	m.logEvent(ctx, "mind.recovery.succeeded", map[string]any{
		"task_id":    t.ID,
		"error_type": errorType,
		"result":     truncate(result.Result, 500),
	}, recoveryCauses)

	return true
}

// gatherCausalContext walks the eventgraph ancestors of the given cause events
// to build a narrative of what led to the failure.
func (m *Mind) gatherCausalContext(ctx context.Context, causes []string) string {
	if len(causes) == 0 {
		return "(no causal chain available)"
	}

	var sb strings.Builder
	seen := make(map[string]bool)

	for _, causeID := range causes {
		ancestors, err := m.events.Ancestors(ctx, causeID, 10)
		if err != nil {
			log.Printf("mind: gatherCausalContext ancestors(%s): %v", causeID, err)
			continue
		}

		for _, e := range ancestors {
			if seen[e.ID] {
				continue
			}
			seen[e.ID] = true
			contentJSON, _ := json.Marshal(e.Content)
			sb.WriteString(fmt.Sprintf("- [%s] %s (source=%s): %s\n",
				e.Timestamp.Format("15:04:05"),
				e.Type, e.Source, string(contentJSON)))
		}
	}

	if sb.Len() == 0 {
		return "(causal chain empty)"
	}
	return sb.String()
}

// gatherGitContext returns recent git log and diff for recovery context.
func (m *Mind) gatherGitContext(ctx context.Context) string {
	var sb strings.Builder

	// Recent commits
	cmd := exec.CommandContext(ctx, "git", "log", "--oneline", "-5")
	cmd.Dir = m.repoDir
	if out, err := cmd.Output(); err == nil {
		sb.WriteString("Recent commits:\n")
		sb.WriteString(string(out))
		sb.WriteString("\n")
	}

	// Uncommitted changes
	cmd = exec.CommandContext(ctx, "git", "diff", "--stat")
	cmd.Dir = m.repoDir
	if out, err := cmd.Output(); err == nil && len(out) > 0 {
		sb.WriteString("Uncommitted changes:\n")
		sb.WriteString(string(out))
	}

	return sb.String()
}

// handleFailure attempts recovery before marking a task blocked.
// This is the main entry point — replaces direct markBlocked calls
// in task execution paths.
func (m *Mind) handleFailure(ctx context.Context, t *task.Task, errorType, reason string, causes []string) {
	if m.attemptRecovery(ctx, t, errorType, reason, causes) {
		// Recovery fixed the root cause — requeue for another attempt.
		m.requeueAfterRecovery(ctx, t, causes)
		return
	}
	m.markBlocked(ctx, t.ID, reason, causes)
}

// requeueAfterRecovery resets a task to pending so the mind picks it up again.
func (m *Mind) requeueAfterRecovery(ctx context.Context, t *task.Task, causes []string) {
	meta := copyMeta(t.Metadata)
	// Clear recovery flag so next attempt starts fresh.
	delete(meta, "recovery_attempted")
	// Preserve retry_count — recovery doesn't count as a retry.
	meta["recovered"] = true
	meta["recovered_at"] = time.Now().Format(time.RFC3339)

	if _, err := m.tasks.Update(ctx, t.ID, map[string]any{
		"status":   "pending",
		"assignee": "",
		"metadata": meta,
	}); err != nil {
		log.Printf("mind: requeue after recovery %s: %v", t.ID, err)
		m.logEvent(ctx, "mind.error", map[string]any{
			"operation": "requeue_after_recovery",
			"task_id":   t.ID,
			"error":     err.Error(),
		}, causes)
	}

	m.logEvent(ctx, "task.recovered", map[string]any{
		"task_id": t.ID,
		"subject": t.Subject,
	}, causes)

	log.Printf("mind: task %s requeued after recovery", t.ID)
}

// copyMeta creates a shallow copy of task metadata, or an empty map if nil.
func copyMeta(meta map[string]any) map[string]any {
	m := make(map[string]any)
	for k, v := range meta {
		m[k] = v
	}
	return m
}
