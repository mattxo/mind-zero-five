package mind

import (
	"context"
	"fmt"
	"log"
	"strings"

	"mind-zero-five/pkg/authority"
	"mind-zero-five/pkg/eventgraph"
	"mind-zero-five/pkg/task"
)

// Mind is the autonomous event-driven loop that picks up tasks,
// invokes Claude Code CLI, builds, commits, and deploys.
type Mind struct {
	bus     *eventgraph.Bus
	tasks   task.Store
	auth    authority.Store
	repoDir string
}

// New creates a Mind.
func New(bus *eventgraph.Bus, tasks task.Store, auth authority.Store, repoDir string) *Mind {
	return &Mind{
		bus:     bus,
		tasks:   tasks,
		auth:    auth,
		repoDir: repoDir,
	}
}

// Run subscribes to the event bus and reacts to events until ctx is cancelled.
func (m *Mind) Run(ctx context.Context) {
	ch := m.bus.Subscribe()
	defer m.bus.Unsubscribe(ch)

	log.Println("mind: running, subscribed to event bus")

	for {
		select {
		case <-ctx.Done():
			log.Println("mind: shutting down")
			return
		case e, ok := <-ch:
			if !ok {
				return
			}
			m.handle(ctx, e)
		}
	}
}

func (m *Mind) handle(ctx context.Context, e *eventgraph.Event) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("mind: panic handling event %s (%s): %v", e.ID, e.Type, r)
			m.logEvent(ctx, "mind.panic", map[string]any{
				"event_id":   e.ID,
				"event_type": e.Type,
				"error":      fmt.Sprintf("%v", r),
			}, []string{e.ID})
		}
	}()

	switch e.Type {
	case "task.created":
		m.onTaskCreated(ctx, e)
	case "authority.resolved":
		m.onAuthorityResolved(ctx, e)
	}
}

func (m *Mind) onTaskCreated(ctx context.Context, e *eventgraph.Event) {
	taskID, _ := e.Content["task_id"].(string)
	if taskID == "" {
		return
	}

	t, err := m.tasks.Get(ctx, taskID)
	if err != nil {
		log.Printf("mind: get task %s: %v", taskID, err)
		return
	}
	if t.Status != "pending" {
		return
	}

	// Claim the task
	t, err = m.tasks.Update(ctx, taskID, map[string]any{
		"status":   "in_progress",
		"assignee": "mind",
	})
	if err != nil {
		log.Printf("mind: claim task %s: %v", taskID, err)
		return
	}

	claimEvent, _ := m.logEvent(ctx, "task.claimed", map[string]any{
		"task_id": taskID,
		"subject": t.Subject,
	}, []string{e.ID})

	m.executeTask(ctx, t, claimEvent)
}

func (m *Mind) executeTask(ctx context.Context, t *task.Task, causeEvent *eventgraph.Event) {
	causes := []string{}
	if causeEvent != nil {
		causes = []string{causeEvent.ID}
	}

	// Build prompt from task
	prompt := fmt.Sprintf("You are working in %s. Complete this task:\n\nSubject: %s\n", m.repoDir, t.Subject)
	if t.Description != "" {
		prompt += fmt.Sprintf("\nDescription: %s\n", t.Description)
	}
	prompt += "\nAfter making changes, verify with: go build ./... && go test ./...\n"
	prompt += "Do NOT commit — just make the code changes and verify they build.\n"

	invokeEvent, _ := m.logEvent(ctx, "mind.claude.invoked", map[string]any{
		"task_id": t.ID,
		"prompt":  truncate(prompt, 500),
	}, causes)

	invokeCauses := causes
	if invokeEvent != nil {
		invokeCauses = []string{invokeEvent.ID}
	}

	// Invoke Claude
	result, err := InvokeClaude(ctx, m.repoDir, prompt)
	if err != nil {
		log.Printf("mind: claude invocation failed for task %s: %v", t.ID, err)
		m.logEvent(ctx, "mind.claude.failed", map[string]any{
			"task_id": t.ID,
			"error":   err.Error(),
		}, invokeCauses)
		m.markBlocked(ctx, t.ID, "claude invocation failed: "+err.Error(), invokeCauses)
		return
	}

	completedEvent, _ := m.logEvent(ctx, "mind.claude.completed", map[string]any{
		"task_id":   t.ID,
		"exit_code": result.ExitCode,
		"duration":  result.Duration.String(),
		"result":    truncate(result.Result, 1000),
		"stderr":    truncate(result.Stderr, 500),
	}, invokeCauses)

	completedCauses := invokeCauses
	if completedEvent != nil {
		completedCauses = []string{completedEvent.ID}
	}

	if result.ExitCode != 0 {
		// Retry once with error context
		retryPrompt := fmt.Sprintf("The previous attempt to complete this task failed (exit code %d).\n\nOriginal task: %s\n\nOutput:\n%s\n\nPlease fix the issues and verify with: go build ./... && go test ./...",
			result.ExitCode, t.Subject, truncate(result.Result, 2000))

		m.logEvent(ctx, "mind.claude.retry", map[string]any{
			"task_id": t.ID,
		}, completedCauses)

		result, err = InvokeClaude(ctx, m.repoDir, retryPrompt)
		if err != nil || result.ExitCode != 0 {
			errMsg := "retry failed"
			if err != nil {
				errMsg = err.Error()
			}
			m.markBlocked(ctx, t.ID, errMsg, completedCauses)
			return
		}
	}

	// Build and test
	if err := BuildAndTest(ctx, m.repoDir); err != nil {
		log.Printf("mind: build/test failed for task %s: %v", t.ID, err)
		m.logEvent(ctx, "build.failed", map[string]any{
			"task_id": t.ID,
			"error":   truncate(err.Error(), 1000),
		}, completedCauses)
		m.markBlocked(ctx, t.ID, "build/test failed: "+truncate(err.Error(), 200), completedCauses)
		return
	}

	buildEvent, _ := m.logEvent(ctx, "build.passed", map[string]any{
		"task_id": t.ID,
	}, completedCauses)

	buildCauses := completedCauses
	if buildEvent != nil {
		buildCauses = []string{buildEvent.ID}
	}

	// Git commit and push
	commitMsg := fmt.Sprintf("mind: %s", t.Subject)
	if err := GitCommitAndPush(ctx, m.repoDir, commitMsg); err != nil {
		// Not fatal — may have nothing to commit
		log.Printf("mind: git commit/push for task %s: %v", t.ID, err)
	} else {
		m.logEvent(ctx, "code.committed", map[string]any{
			"task_id": t.ID,
			"message": commitMsg,
		}, buildCauses)
	}

	// Complete the task
	if _, err := m.tasks.Complete(ctx, t.ID); err != nil {
		log.Printf("mind: complete task %s: %v", t.ID, err)
	}
	m.logEvent(ctx, "task.completed", map[string]any{
		"task_id": t.ID,
		"subject": t.Subject,
	}, buildCauses)

	// Build deployment binaries
	if err := Build(ctx, m.repoDir); err != nil {
		log.Printf("mind: build deploy binaries for task %s: %v", t.ID, err)
		m.logEvent(ctx, "build.deploy.failed", map[string]any{
			"task_id": t.ID,
			"error":   truncate(err.Error(), 500),
		}, buildCauses)
		return
	}

	m.logEvent(ctx, "build.completed", map[string]any{
		"task_id": t.ID,
	}, buildCauses)

	// Request authority to restart
	req, err := m.auth.Create(ctx, "restart", fmt.Sprintf("Task completed: %s. New binaries built. Ready to restart.", t.Subject), "mind", authority.Required)
	if err != nil {
		log.Printf("mind: request restart authority: %v", err)
		return
	}
	m.logEvent(ctx, "authority.requested", map[string]any{
		"task_id":      t.ID,
		"authority_id": req.ID,
		"action":       "restart",
	}, buildCauses)
}

func (m *Mind) onAuthorityResolved(ctx context.Context, e *eventgraph.Event) {
	authID, _ := e.Content["authority_id"].(string)
	approved, _ := e.Content["approved"].(bool)
	action, _ := e.Content["action"].(string)

	if !approved || !strings.Contains(action, "restart") {
		return
	}

	log.Printf("mind: restart approved (authority %s), deploying", authID)
	m.logEvent(ctx, "deploy.started", map[string]any{
		"authority_id": authID,
	}, []string{e.ID})

	if err := RestartSelf(); err != nil {
		log.Printf("mind: restart failed: %v", err)
		m.logEvent(ctx, "deploy.failed", map[string]any{
			"authority_id": authID,
			"error":        err.Error(),
		}, []string{e.ID})
	}
}

func (m *Mind) markBlocked(ctx context.Context, taskID, reason string, causes []string) {
	if _, err := m.tasks.Update(ctx, taskID, map[string]any{
		"status": "blocked",
	}); err != nil {
		log.Printf("mind: mark task %s blocked: %v", taskID, err)
	}
	m.logEvent(ctx, "task.blocked", map[string]any{
		"task_id": taskID,
		"reason":  reason,
	}, causes)
}

func (m *Mind) logEvent(ctx context.Context, eventType string, content map[string]any, causes []string) (*eventgraph.Event, error) {
	e, err := m.bus.Append(ctx, eventType, "mind", content, causes, "")
	if err != nil {
		log.Printf("mind: log event %s: %v", eventType, err)
	}
	return e, err
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
