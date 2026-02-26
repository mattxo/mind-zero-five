// Package mind implements the autonomous mind loop that:
// - Polls Postgres for pending tasks every 5 seconds
// - Invokes Claude Code CLI to execute tasks (with plan, implement, review, finish cycle)
// - Handles self-assessment when idle to propose improvements
// - Manages retry and exponential backoff for blocked tasks
package mind

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"mind-zero-five/pkg/authority"
	"mind-zero-five/pkg/eventgraph"
	"mind-zero-five/pkg/task"
)

// Mind is the autonomous loop that picks up tasks, invokes Claude Code CLI,
// builds, commits, deploys, and — when idle — assesses itself for improvements.
type Mind struct {
	events  eventgraph.EventStore
	tasks   task.Store
	auth    authority.Store
	actorID string // this mind's actor ID, for policy matching
	repoDir string

	// pendingRestart holds the authority request ID when waiting for
	// human approval of a restart. Empty when not waiting.
	pendingRestart string

	// Self-improvement state
	pendingProposal string    // authority request ID for current improvement proposal
	lastAssessment  time.Time // when the last assessment ran
	assessInterval  time.Duration
}

// New creates a Mind.
func New(events eventgraph.EventStore, tasks task.Store, auth authority.Store, actorID, repoDir string) *Mind {
	return &Mind{
		events:         events,
		tasks:          tasks,
		auth:           auth,
		actorID:        actorID,
		repoDir:        repoDir,
		assessInterval: 30 * time.Minute,
	}
}

// Run polls for pending tasks and resolved authority requests until ctx is cancelled.
func (m *Mind) Run(ctx context.Context) {
	log.Println("mind: running, polling for tasks")

	// Catch up immediately on startup
	m.poll(ctx)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("mind: shutting down")
			return
		case <-ticker.C:
			m.poll(ctx)
		}
	}
}

// poll checks for work to do: pending tasks or resolved authority requests.
func (m *Mind) poll(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("mind: panic in poll: %v", r)
			m.logEvent(ctx, "mind.panic", map[string]any{
				"error": fmt.Sprintf("%v", r),
			}, nil)
		}
	}()

	// Priority order: restart > proposal > tasks > assess
	if m.pendingRestart != "" {
		m.checkRestart(ctx)
		return
	}

	if m.pendingProposal != "" {
		m.checkProposal(ctx)
		return
	}

	m.retryBlockedTasks(ctx)

	if m.checkPendingTasks(ctx) {
		return // found and started a task
	}

	// Idle — maybe assess for improvements
	m.maybeAssess(ctx)
}

// checkPendingTasks looks for pending tasks and claims the first one.
// Returns true if a task was found and executed.
func (m *Mind) checkPendingTasks(ctx context.Context) bool {
	tasks, err := m.tasks.List(ctx, "pending", 10)
	if err != nil {
		log.Printf("mind: poll pending tasks: %v", err)
		return false
	}

	for i := range tasks {
		t := &tasks[i]
		// Skip tasks already assigned to someone else
		if t.Assignee != "" && t.Assignee != "mind" {
			continue
		}

		// Claim and execute
		t, err = m.tasks.Update(ctx, t.ID, map[string]any{
			"status":   "in_progress",
			"assignee": "mind",
		})
		if err != nil {
			log.Printf("mind: claim task %s: %v", tasks[i].ID, err)
			continue
		}

		claimEvent, _ := m.logEvent(ctx, "task.claimed", map[string]any{
			"task_id": t.ID,
			"subject": t.Subject,
		}, nil)

		// Subtasks (have a parent) execute directly — no planning phase
		if t.ParentID != "" {
			m.executeSubtask(ctx, t, claimEvent)
		} else {
			m.executeTask(ctx, t, claimEvent)
		}

		// Process one task per poll cycle
		return true
	}
	return false
}

// checkRestart polls for resolution of a pending restart authority request.
func (m *Mind) checkRestart(ctx context.Context) {
	req, err := m.auth.Get(ctx, m.pendingRestart)
	if err != nil {
		log.Printf("mind: check restart authority %s: %v", m.pendingRestart, err)
		m.pendingRestart = ""
		return
	}

	if req.Status == "pending" {
		return // still waiting
	}

	if req.Status == "approved" {
		log.Printf("mind: restart approved (authority %s), deploying", req.ID)
		m.doRestart(ctx, req.ID, nil)
	} else {
		log.Printf("mind: restart rejected (authority %s)", req.ID)
	}
	m.pendingRestart = ""
}

// maybeAssess runs a self-assessment if enough time has passed since the last one.
func (m *Mind) maybeAssess(ctx context.Context) {
	if time.Since(m.lastAssessment) < m.assessInterval {
		return
	}

	m.lastAssessment = time.Now()
	log.Println("mind: idle — running self-assessment")

	assessEvent, _ := m.logEvent(ctx, "mind.assess.started", map[string]any{}, nil)

	causes := []string{}
	if assessEvent != nil {
		causes = []string{assessEvent.ID}
	}

	proposal, err := m.Assess(ctx)
	if err != nil {
		log.Printf("mind: assessment failed: %v", err)
		m.logEvent(ctx, "mind.assess.failed", map[string]any{
			"error": err.Error(),
		}, causes)
		return
	}

	if proposal == nil {
		log.Println("mind: assessment found no improvements needed")
		m.logEvent(ctx, "mind.assess.completed", map[string]any{
			"result": "ok",
		}, causes)
		return
	}

	log.Printf("mind: proposing improvement: %s", proposal.Subject)
	m.logEvent(ctx, "mind.assess.completed", map[string]any{
		"result":  "proposal",
		"subject": proposal.Subject,
	}, causes)

	// Encode proposal as JSON in the authority request description
	proposalJSON, _ := json.Marshal(proposal)
	req, err := m.auth.Create(ctx, "self-improve",
		string(proposalJSON),
		"mind", authority.Required)
	if err != nil {
		log.Printf("mind: create self-improve authority: %v", err)
		return
	}

	m.logEvent(ctx, "authority.requested", map[string]any{
		"authority_id": req.ID,
		"action":       "self-improve",
		"subject":      proposal.Subject,
	}, causes)

	m.pendingProposal = req.ID
	log.Printf("mind: improvement proposal submitted for approval (authority %s): %s", req.ID, proposal.Subject)
}

// checkProposal polls for resolution of a pending self-improvement authority request.
func (m *Mind) checkProposal(ctx context.Context) {
	req, err := m.auth.Get(ctx, m.pendingProposal)
	if err != nil {
		log.Printf("mind: check proposal authority %s: %v", m.pendingProposal, err)
		m.pendingProposal = ""
		return
	}

	if req.Status == "pending" {
		return // still waiting for Matt
	}

	if req.Status == "approved" {
		log.Printf("mind: improvement approved (authority %s)", req.ID)

		// Parse proposal from description
		var proposal Proposal
		if err := json.Unmarshal([]byte(req.Description), &proposal); err != nil {
			log.Printf("mind: parse proposal from authority %s: %v", req.ID, err)
			m.pendingProposal = ""
			return
		}

		// Create the task
		t, err := m.tasks.Create(ctx, &task.Task{
			Subject:     proposal.Subject,
			Description: proposal.Description,
			Source:      "mind",
			Metadata: map[string]any{
				"model":        proposal.Model,
				"self_improve": true,
				"authority_id": req.ID,
			},
		})
		if err != nil {
			log.Printf("mind: create improvement task: %v", err)
			m.pendingProposal = ""
			return
		}

		m.logEvent(ctx, "self-improve.task.created", map[string]any{
			"task_id":      t.ID,
			"authority_id": req.ID,
			"subject":      proposal.Subject,
		}, nil)

		log.Printf("mind: created improvement task %s: %s", t.ID, proposal.Subject)
	} else {
		log.Printf("mind: improvement rejected (authority %s)", req.ID)
		m.logEvent(ctx, "self-improve.rejected", map[string]any{
			"authority_id": req.ID,
		}, nil)
	}

	m.pendingProposal = ""
}

// executeTask orchestrates the full plan → implement → review → finish cycle.
func (m *Mind) executeTask(ctx context.Context, t *task.Task, causeEvent *eventgraph.Event) {
	causes := []string{}
	if causeEvent != nil {
		causes = []string{causeEvent.ID}
	}

	// Record the starting commit so we can diff later
	startCommit, err := getCurrentCommit(ctx, m.repoDir)
	if err != nil {
		log.Printf("mind: get start commit: %v", err)
		startCommit = "HEAD~20" // fallback: review last 20 commits
	}

	// --- PLAN ---
	m.logEvent(ctx, "mind.plan.started", map[string]any{
		"task_id": t.ID,
		"subject": t.Subject,
	}, causes)

	subtaskSpecs, err := m.Plan(ctx, t)
	if err != nil {
		log.Printf("mind: plan failed for task %s: %v", t.ID, err)
		// Fall back to direct execution (single-shot, like before)
		m.logEvent(ctx, "mind.plan.failed", map[string]any{
			"task_id": t.ID,
			"error":   err.Error(),
		}, causes)
		m.executeDirectly(ctx, t, causes)
		return
	}

	planEvent, _ := m.logEvent(ctx, "mind.plan.completed", map[string]any{
		"task_id":       t.ID,
		"subtask_count": len(subtaskSpecs),
	}, causes)

	planCauses := causes
	if planEvent != nil {
		planCauses = []string{planEvent.ID}
	}

	// Create subtasks in the task store
	subtaskIDs, err := m.createSubtasks(ctx, t.ID, subtaskSpecs, planCauses)
	if err != nil {
		log.Printf("mind: create subtasks for task %s: %v", t.ID, err)
		m.markBlocked(ctx, t.ID, "failed to create subtasks: "+err.Error(), planCauses)
		return
	}

	// --- IMPLEMENT ---
	blocked := m.implementSubtasks(ctx, t.ID, subtaskIDs, planCauses)
	if blocked {
		return // parent already marked blocked by implementSubtasks
	}

	// --- REVIEW (max 2 rounds) ---
	reviewCauses := planCauses
	for round := 0; round < 2; round++ {
		m.logEvent(ctx, "mind.review.started", map[string]any{
			"task_id": t.ID,
			"round":   round + 1,
		}, reviewCauses)

		issues, err := m.Review(ctx, t, startCommit)
		if err != nil {
			log.Printf("mind: review failed for task %s: %v", t.ID, err)
			m.logEvent(ctx, "mind.review.failed", map[string]any{
				"task_id": t.ID,
				"error":   err.Error(),
			}, reviewCauses)
			break // proceed to finish — review failure shouldn't block
		}

		reviewEvent, _ := m.logEvent(ctx, "mind.review.completed", map[string]any{
			"task_id":     t.ID,
			"round":       round + 1,
			"issue_count": len(issues),
			"clean":       len(issues) == 0,
		}, reviewCauses)

		if len(issues) == 0 {
			break // clean review
		}

		if reviewEvent != nil {
			reviewCauses = []string{reviewEvent.ID}
		}

		// Create fix subtasks from review issues
		fixIDs, err := m.createSubtasks(ctx, t.ID, issues, reviewCauses)
		if err != nil {
			log.Printf("mind: create fix subtasks: %v", err)
			break
		}

		blocked = m.implementSubtasks(ctx, t.ID, fixIDs, reviewCauses)
		if blocked {
			return
		}
	}

	// --- FINISH ---
	m.finishTask(ctx, t, reviewCauses)
}

// executeDirectly is the fallback single-shot execution (used when planning fails).
func (m *Mind) executeDirectly(ctx context.Context, t *task.Task, causes []string) {
	prompt := fmt.Sprintf("You are working in %s. Complete this task:\n\nSubject: %s\n", m.repoDir, t.Subject)
	if t.Description != "" {
		prompt += fmt.Sprintf("\nDescription: %s\n", t.Description)
	}
	prompt += "\nAfter making changes, verify with: go build ./... && go test ./...\n"
	prompt += "Do NOT commit — just make the code changes and verify they build.\n"

	if t.Metadata != nil {
		if reason, ok := t.Metadata["prev_failure_reason"].(string); ok && reason != "" {
			prompt = "IMPORTANT — Previous attempt failed with: " + reason + ". Avoid repeating this mistake.\n\n" + prompt
		}
	}

	invokeEvent, _ := m.logEvent(ctx, "mind.claude.invoked", map[string]any{
		"task_id": t.ID,
		"prompt":  truncate(prompt, 500),
		"mode":    "direct",
	}, causes)

	invokeCauses := causes
	if invokeEvent != nil {
		invokeCauses = []string{invokeEvent.ID}
	}

	result, err := InvokeClaude(ctx, m.repoDir, prompt, "")
	if err != nil {
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
	}, invokeCauses)

	completedCauses := invokeCauses
	if completedEvent != nil {
		completedCauses = []string{completedEvent.ID}
	}

	if result.ExitCode != 0 {
		m.markBlocked(ctx, t.ID, fmt.Sprintf("claude failed (exit %d)", result.ExitCode), completedCauses)
		return
	}

	if err := BuildAndTest(ctx, m.repoDir); err != nil {
		m.logEvent(ctx, "build.failed", map[string]any{
			"task_id": t.ID,
			"error":   truncate(err.Error(), 1000),
		}, completedCauses)
		m.markBlocked(ctx, t.ID, "build/test failed: "+truncate(err.Error(), 200), completedCauses)
		return
	}

	m.finishTask(ctx, t, completedCauses)
}

// executeSubtask handles a single focused subtask (already claimed).
func (m *Mind) executeSubtask(ctx context.Context, t *task.Task, causeEvent *eventgraph.Event) {
	causes := []string{}
	if causeEvent != nil {
		causes = []string{causeEvent.ID}
	}

	m.logEvent(ctx, "mind.subtask.started", map[string]any{
		"task_id":   t.ID,
		"parent_id": t.ParentID,
		"subject":   t.Subject,
	}, causes)

	// Determine model from metadata
	model := ""
	if t.Metadata != nil {
		if m, ok := t.Metadata["model"].(string); ok {
			model = m
		}
	}

	prompt := fmt.Sprintf("You are working in %s. Complete this specific subtask:\n\nSubject: %s\n", m.repoDir, t.Subject)
	if t.Description != "" {
		prompt += fmt.Sprintf("\nDescription: %s\n", t.Description)
	}
	prompt += "\nThis is a focused change — make ONLY the changes described above.\n"
	prompt += "After making changes, verify with: go build ./... && go test ./...\n"
	prompt += "Do NOT commit — just make the code changes and verify they build.\n"

	if t.Metadata != nil {
		if reason, ok := t.Metadata["prev_failure_reason"].(string); ok && reason != "" {
			prompt = "IMPORTANT — Previous attempt failed with: " + reason + ". Avoid repeating this mistake.\n\n" + prompt
		}
	}

	invokeEvent, _ := m.logEvent(ctx, "mind.claude.invoked", map[string]any{
		"task_id": t.ID,
		"model":   model,
		"prompt":  truncate(prompt, 500),
	}, causes)

	invokeCauses := causes
	if invokeEvent != nil {
		invokeCauses = []string{invokeEvent.ID}
	}

	result, err := InvokeClaude(ctx, m.repoDir, prompt, model)
	if err != nil {
		m.logEvent(ctx, "mind.claude.failed", map[string]any{
			"task_id": t.ID,
			"error":   err.Error(),
		}, invokeCauses)
		m.markBlocked(ctx, t.ID, "claude failed: "+err.Error(), invokeCauses)
		return
	}

	completedEvent, _ := m.logEvent(ctx, "mind.claude.completed", map[string]any{
		"task_id":   t.ID,
		"exit_code": result.ExitCode,
		"duration":  result.Duration.String(),
		"result":    truncate(result.Result, 1000),
	}, invokeCauses)

	completedCauses := invokeCauses
	if completedEvent != nil {
		completedCauses = []string{completedEvent.ID}
	}

	if result.ExitCode != 0 {
		// Retry once
		retryPrompt := fmt.Sprintf("The previous attempt failed (exit code %d).\n\nOriginal task: %s\n\nOutput:\n%s\n\nPlease fix and verify with: go build ./... && go test ./...",
			result.ExitCode, t.Subject, truncate(result.Result, 2000))

		m.logEvent(ctx, "mind.claude.retry", map[string]any{
			"task_id": t.ID,
		}, completedCauses)

		result, err = InvokeClaude(ctx, m.repoDir, retryPrompt, model)
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
		m.logEvent(ctx, "build.failed", map[string]any{
			"task_id": t.ID,
			"error":   truncate(err.Error(), 1000),
		}, completedCauses)
		m.markBlocked(ctx, t.ID, "build/test failed: "+truncate(err.Error(), 200), completedCauses)
		return
	}

	// Incremental commit for this subtask
	commitMsg := fmt.Sprintf("mind: %s", t.Subject)
	if err := GitCommitAndPush(ctx, m.repoDir, commitMsg); err != nil {
		log.Printf("mind: git commit for subtask %s: %v", t.ID, err)
		m.logEvent(ctx, "git.commit_push.failed", map[string]any{
			"task_id": t.ID,
			"error":   truncate(err.Error(), 500),
		}, completedCauses)
	} else {
		m.logEvent(ctx, "code.committed", map[string]any{
			"task_id": t.ID,
			"message": commitMsg,
		}, completedCauses)
	}

	// Complete the subtask
	if _, err := m.tasks.Complete(ctx, t.ID); err != nil {
		log.Printf("mind: complete subtask %s: %v", t.ID, err)
	}

	m.logEvent(ctx, "mind.subtask.completed", map[string]any{
		"task_id":   t.ID,
		"parent_id": t.ParentID,
		"subject":   t.Subject,
	}, completedCauses)
}

// createSubtasks creates subtask records in the task store.
func (m *Mind) createSubtasks(ctx context.Context, parentID string, specs []SubtaskSpec, causes []string) ([]string, error) {
	var ids []string
	for _, spec := range specs {
		st := &task.Task{
			Subject:     spec.Subject,
			Description: spec.Description,
			Source:      "mind",
			ParentID:    parentID,
			Metadata: map[string]any{
				"model": spec.Model,
			},
		}
		created, err := m.tasks.Create(ctx, st)
		if err != nil {
			return nil, fmt.Errorf("create subtask %q: %w", spec.Subject, err)
		}
		ids = append(ids, created.ID)
		log.Printf("mind: created subtask %s (%s) [%s]", created.ID, spec.Subject, spec.Model)
	}
	return ids, nil
}

// implementSubtasks executes subtasks sequentially. Returns true if any subtask blocked.
func (m *Mind) implementSubtasks(ctx context.Context, parentID string, subtaskIDs []string, causes []string) bool {
	for _, stID := range subtaskIDs {
		st, err := m.tasks.Get(ctx, stID)
		if err != nil {
			log.Printf("mind: get subtask %s: %v", stID, err)
			m.markBlocked(ctx, parentID, fmt.Sprintf("get subtask %s: %v", stID, err), causes)
			return true
		}

		// Subtask might already be completed (if re-running after a review round)
		if st.Status == "completed" {
			continue
		}

		// Claim the subtask
		st, err = m.tasks.Update(ctx, stID, map[string]any{
			"status":   "in_progress",
			"assignee": "mind",
		})
		if err != nil {
			log.Printf("mind: claim subtask %s: %v", stID, err)
			continue
		}

		claimEvent, _ := m.logEvent(ctx, "task.claimed", map[string]any{
			"task_id": stID,
			"subject": st.Subject,
		}, causes)

		m.executeSubtask(ctx, st, claimEvent)

		// Check if subtask got blocked
		st, err = m.tasks.Get(ctx, stID)
		if err != nil {
			continue
		}
		if st.Status == "blocked" {
			m.markBlocked(ctx, parentID, fmt.Sprintf("subtask %s blocked: %s", stID, st.Subject), causes)
			return true
		}
	}
	return false
}

// finishTask handles the completion flow: push, complete, build binary, request restart.
func (m *Mind) finishTask(ctx context.Context, t *task.Task, causes []string) {
	// Final push (in case subtask commits haven't been pushed yet)
	commitMsg := fmt.Sprintf("mind: %s", t.Subject)
	if err := GitCommitAndPush(ctx, m.repoDir, commitMsg); err != nil {
		log.Printf("mind: final commit/push for task %s: %v", t.ID, err)
		m.logEvent(ctx, "git.commit_push.failed", map[string]any{
			"task_id": t.ID,
			"error":   truncate(err.Error(), 500),
		}, causes)
	} else {
		m.logEvent(ctx, "code.committed", map[string]any{
			"task_id": t.ID,
			"message": commitMsg,
		}, causes)
	}

	// Complete the task
	if _, err := m.tasks.Complete(ctx, t.ID); err != nil {
		log.Printf("mind: complete task %s: %v", t.ID, err)
	}
	m.logEvent(ctx, "task.completed", map[string]any{
		"task_id": t.ID,
		"subject": t.Subject,
	}, causes)

	// Build deployment binaries
	if err := Build(ctx, m.repoDir); err != nil {
		log.Printf("mind: build deploy binaries for task %s: %v", t.ID, err)
		m.logEvent(ctx, "build.deploy.failed", map[string]any{
			"task_id": t.ID,
			"error":   truncate(err.Error(), 500),
		}, causes)
		return
	}

	deployEvent, _ := m.logEvent(ctx, "build.completed", map[string]any{
		"task_id": t.ID,
	}, causes)

	deployCauses := causes
	if deployEvent != nil {
		deployCauses = []string{deployEvent.ID}
	}

	// Request authority to restart — policy determines if self-approved or needs human
	m.requestRestart(ctx, t, deployCauses)
}

func (m *Mind) requestRestart(ctx context.Context, t *task.Task, causes []string) {
	req, err := m.auth.Create(ctx, "restart",
		fmt.Sprintf("Task completed: %s. New binaries built.", t.Subject),
		"mind", authority.Required)
	if err != nil {
		log.Printf("mind: request restart authority: %v", err)
		return
	}

	reqEvent, _ := m.logEvent(ctx, "authority.requested", map[string]any{
		"task_id":      t.ID,
		"authority_id": req.ID,
		"action":       "restart",
	}, causes)

	reqCauses := causes
	if reqEvent != nil {
		reqCauses = []string{reqEvent.ID}
	}

	// Check policy — can the mind self-approve?
	policy, err := m.auth.MatchPolicy(ctx, "restart")
	if err != nil {
		log.Printf("mind: no policy for restart, leaving pending for human: %v", err)
		m.pendingRestart = req.ID
		return
	}

	if policy.ApproverID == m.actorID {
		// Self-approve: the policy says the mind can approve its own restarts
		_, err := m.auth.Resolve(ctx, req.ID, true)
		if err != nil {
			log.Printf("mind: self-approve restart: %v", err)
			return
		}
		log.Printf("mind: self-approved restart (policy: %s)", policy.Action)
		m.logEvent(ctx, "authority.self_approved", map[string]any{
			"authority_id": req.ID,
			"policy_id":    policy.ID,
			"action":       "restart",
		}, reqCauses)

		m.doRestart(ctx, req.ID, reqCauses)
		return
	}

	// Otherwise: leave pending, poll will check for resolution
	m.pendingRestart = req.ID
}

func (m *Mind) doRestart(ctx context.Context, authID string, causes []string) {
	m.logEvent(ctx, "deploy.started", map[string]any{
		"authority_id": authID,
	}, causes)

	if err := RestartSelf(); err != nil {
		log.Printf("mind: restart failed: %v", err)
		m.logEvent(ctx, "deploy.failed", map[string]any{
			"authority_id": authID,
			"error":        err.Error(),
		}, causes)
	}
}

func (m *Mind) markBlocked(ctx context.Context, taskID, reason string, causes []string) {
	// Read existing metadata to preserve retry_count and other fields.
	meta := map[string]any{}
	if t, err := m.tasks.Get(ctx, taskID); err == nil && t.Metadata != nil {
		for k, v := range t.Metadata {
			meta[k] = v
		}
	}
	meta["blocked_reason"] = reason
	if _, exists := meta["retry_count"]; !exists {
		meta["retry_count"] = 0
	}

	if _, err := m.tasks.Update(ctx, taskID, map[string]any{
		"status":   "blocked",
		"metadata": meta,
	}); err != nil {
		log.Printf("mind: mark task %s blocked: %v", taskID, err)
	}
	m.logEvent(ctx, "task.blocked", map[string]any{
		"task_id": taskID,
		"reason":  reason,
	}, causes)
}

// retryBlockedTasks finds blocked tasks assigned to the mind that are older than
// 15 minutes and have been retried fewer than 3 times, then requeues them as
// pending. Returns true if any tasks were retried.
func (m *Mind) retryBlockedTasks(ctx context.Context) bool {
	tasks, err := m.tasks.List(ctx, "blocked", 20)
	if err != nil {
		log.Printf("mind: list blocked tasks: %v", err)
		return false
	}

	retried := false

	for i := range tasks {
		t := &tasks[i]
		if t.Assignee != "mind" {
			continue
		}

		// Read retry_count from metadata (default 0)
		retryCount := 0
		if t.Metadata != nil {
			switch v := t.Metadata["retry_count"].(type) {
			case int:
				retryCount = v
			case float64:
				retryCount = int(v)
			}
		}
		if retryCount >= 3 {
			continue
		}

		// Exponential backoff: retry 0 waits 15m, retry 1 waits 30m, retry 2 waits 60m.
		cutoff := time.Now().Add(-15 * time.Minute * time.Duration(1<<retryCount))
		if t.UpdatedAt.After(cutoff) {
			continue // updated too recently
		}

		// Build updated metadata, preserving existing fields
		meta := map[string]any{}
		if t.Metadata != nil {
			for k, v := range t.Metadata {
				meta[k] = v
			}
		}
		// Copy blocked_reason to prev_failure_reason before clearing
		if br, ok := meta["blocked_reason"].(string); ok && br != "" {
			meta["prev_failure_reason"] = br
		}
		meta["retry_count"] = retryCount + 1

		if _, err := m.tasks.Update(ctx, t.ID, map[string]any{
			"status":   "pending",
			"assignee": "",
			"metadata": meta,
		}); err != nil {
			log.Printf("mind: retry task %s: %v", t.ID, err)
			continue
		}

		m.logEvent(ctx, "task.retried", map[string]any{
			"task_id":     t.ID,
			"subject":     t.Subject,
			"retry_count": retryCount + 1,
		}, nil)

		log.Printf("mind: retried task %s (retry %d): %s", t.ID, retryCount+1, t.Subject)
		retried = true
	}

	return retried
}

func (m *Mind) logEvent(ctx context.Context, eventType string, content map[string]any, causes []string) (*eventgraph.Event, error) {
	e, err := m.events.Append(ctx, eventType, "mind", content, causes, "")
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
