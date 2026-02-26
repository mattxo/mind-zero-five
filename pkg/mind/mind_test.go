package mind

import (
	"context"
	"testing"
	"time"

	"mind-zero-five/pkg/authority"
	"mind-zero-five/pkg/eventgraph"
	"mind-zero-five/pkg/task"
)

// --- Mock task store ---

type mockTaskStore struct {
	tasks map[string]*task.Task
}

func newMockTaskStore() *mockTaskStore {
	return &mockTaskStore{tasks: make(map[string]*task.Task)}
}

func (s *mockTaskStore) Create(_ context.Context, t *task.Task) (*task.Task, error) {
	t.ID = "task-" + t.Subject
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()
	cp := *t
	s.tasks[t.ID] = &cp
	return &cp, nil
}

func (s *mockTaskStore) Get(_ context.Context, id string) (*task.Task, error) {
	t, ok := s.tasks[id]
	if !ok {
		return nil, nil
	}
	cp := *t
	return &cp, nil
}

func (s *mockTaskStore) Update(_ context.Context, id string, updates map[string]any) (*task.Task, error) {
	t, ok := s.tasks[id]
	if !ok {
		return nil, nil
	}
	if v, ok := updates["status"]; ok {
		t.Status = v.(string)
	}
	if v, ok := updates["assignee"]; ok {
		t.Assignee = v.(string)
	}
	if v, ok := updates["metadata"]; ok {
		t.Metadata = v.(map[string]any)
	}
	t.UpdatedAt = time.Now()
	cp := *t
	return &cp, nil
}

func (s *mockTaskStore) Complete(_ context.Context, id string) (*task.Task, error) {
	t, ok := s.tasks[id]
	if !ok {
		return nil, nil
	}
	t.Status = "completed"
	now := time.Now()
	t.CompletedAt = &now
	cp := *t
	return &cp, nil
}

func (s *mockTaskStore) List(_ context.Context, status string, limit int) ([]task.Task, error) {
	var result []task.Task
	for _, t := range s.tasks {
		if t.Status == status {
			result = append(result, *t)
		}
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}

func (s *mockTaskStore) ByParent(_ context.Context, parentID string) ([]task.Task, error) {
	var result []task.Task
	for _, t := range s.tasks {
		if t.ParentID == parentID {
			result = append(result, *t)
		}
	}
	return result, nil
}

func (s *mockTaskStore) Count(_ context.Context) (int, error)        { return len(s.tasks), nil }
func (s *mockTaskStore) PendingCount(_ context.Context) (int, error) { return 0, nil }
func (s *mockTaskStore) EnsureTable(_ context.Context) error         { return nil }

// --- Mock event store ---

type mockEventStore struct{}

func (s *mockEventStore) Append(_ context.Context, eventType, source string, content map[string]any, causes []string, conversationID string) (*eventgraph.Event, error) {
	return &eventgraph.Event{ID: "evt-1", Type: eventType}, nil
}
func (s *mockEventStore) Get(_ context.Context, id string) (*eventgraph.Event, error) { return nil, nil }
func (s *mockEventStore) Recent(_ context.Context, limit int) ([]eventgraph.Event, error) {
	return nil, nil
}
func (s *mockEventStore) ByType(_ context.Context, eventType string, limit int) ([]eventgraph.Event, error) {
	return nil, nil
}
func (s *mockEventStore) BySource(_ context.Context, source string, limit int) ([]eventgraph.Event, error) {
	return nil, nil
}
func (s *mockEventStore) ByConversation(_ context.Context, conversationID string, limit int) ([]eventgraph.Event, error) {
	return nil, nil
}
func (s *mockEventStore) Since(_ context.Context, afterID string, limit int) ([]eventgraph.Event, error) {
	return nil, nil
}
func (s *mockEventStore) Count(_ context.Context) (int, error)      { return 0, nil }
func (s *mockEventStore) VerifyChain(_ context.Context) error        { return nil }
func (s *mockEventStore) EnsureTable(_ context.Context) error        { return nil }
func (s *mockEventStore) Ancestors(_ context.Context, id string, maxDepth int) ([]eventgraph.Event, error) {
	return nil, nil
}
func (s *mockEventStore) Descendants(_ context.Context, id string, maxDepth int) ([]eventgraph.Event, error) {
	return nil, nil
}
func (s *mockEventStore) Search(_ context.Context, query string, limit int) ([]eventgraph.Event, error) {
	return nil, nil
}
func (s *mockEventStore) DistinctTypes(_ context.Context) ([]string, error)   { return nil, nil }
func (s *mockEventStore) DistinctSources(_ context.Context) ([]string, error) { return nil, nil }

// --- Mock authority store ---

type mockAuthStore struct{}

func (s *mockAuthStore) Create(_ context.Context, action, description, source string, level authority.Level) (*authority.Request, error) {
	return &authority.Request{ID: "auth-1"}, nil
}
func (s *mockAuthStore) Resolve(_ context.Context, id string, approved bool) (*authority.Request, error) {
	return &authority.Request{ID: id}, nil
}
func (s *mockAuthStore) Get(_ context.Context, id string) (*authority.Request, error) {
	return nil, nil
}
func (s *mockAuthStore) Pending(_ context.Context) ([]authority.Request, error) { return nil, nil }
func (s *mockAuthStore) Recent(_ context.Context, limit int) ([]authority.Request, error) {
	return nil, nil
}
func (s *mockAuthStore) PendingCount(_ context.Context) (int, error)     { return 0, nil }
func (s *mockAuthStore) CreatePolicy(_ context.Context, action, approverID string, level authority.Level) (*authority.Policy, error) {
	return nil, nil
}
func (s *mockAuthStore) MatchPolicy(_ context.Context, action string) (*authority.Policy, error) {
	return nil, nil
}
func (s *mockAuthStore) ListPolicies(_ context.Context) ([]authority.Policy, error) { return nil, nil }
func (s *mockAuthStore) EnsureTable(_ context.Context) error                        { return nil }

// --- Helpers ---

func newTestMind(ts task.Store) *Mind {
	return New(&mockEventStore{}, ts, &mockAuthStore{}, "mind", "/tmp")
}

func addTask(ts *mockTaskStore, id, status, assignee string, updatedAt time.Time, meta map[string]any) {
	t := &task.Task{
		ID:        id,
		Subject:   id,
		Status:    status,
		Assignee:  assignee,
		UpdatedAt: updatedAt,
		Metadata:  meta,
	}
	ts.tasks[id] = t
}

// --- Tests ---

// TestMarkBlockedStoresMetadata verifies that markBlocked writes blocked_reason
// and initialises retry_count=0 in the task's metadata.
func TestMarkBlockedStoresMetadata(t *testing.T) {
	ts := newMockTaskStore()
	addTask(ts, "t1", "in_progress", "mind", time.Now(), nil)

	m := newTestMind(ts)
	m.markBlocked(context.Background(), "t1", "something went wrong", nil)

	stored := ts.tasks["t1"]
	if stored.Status != "blocked" {
		t.Fatalf("expected status=blocked, got %q", stored.Status)
	}
	if stored.Metadata == nil {
		t.Fatal("expected non-nil metadata")
	}
	reason, _ := stored.Metadata["blocked_reason"].(string)
	if reason != "something went wrong" {
		t.Errorf("blocked_reason: want %q, got %q", "something went wrong", reason)
	}
	// retry_count must be present and zero on first block
	rc, ok := stored.Metadata["retry_count"]
	if !ok {
		t.Fatal("retry_count missing from metadata")
	}
	switch v := rc.(type) {
	case int:
		if v != 0 {
			t.Errorf("retry_count: want 0, got %d", v)
		}
	case float64:
		if int(v) != 0 {
			t.Errorf("retry_count: want 0, got %v", v)
		}
	default:
		t.Errorf("retry_count unexpected type %T", rc)
	}
}

// TestMarkBlockedPreservesExistingRetryCount ensures that if retry_count is
// already set in metadata, markBlocked does not reset it.
func TestMarkBlockedPreservesExistingRetryCount(t *testing.T) {
	ts := newMockTaskStore()
	addTask(ts, "t2", "in_progress", "mind", time.Now(), map[string]any{"retry_count": 2})

	m := newTestMind(ts)
	m.markBlocked(context.Background(), "t2", "another failure", nil)

	stored := ts.tasks["t2"]
	rc, ok := stored.Metadata["retry_count"]
	if !ok {
		t.Fatal("retry_count missing")
	}
	var count int
	switch v := rc.(type) {
	case int:
		count = v
	case float64:
		count = int(v)
	}
	if count != 2 {
		t.Errorf("retry_count: want 2 (preserved), got %d", count)
	}
}

// TestRetryBlockedTasksResetsEligible verifies that a blocked task assigned to
// the mind that is older than 15 minutes and has retry_count < 3 is reset to
// pending with an incremented retry_count.
func TestRetryBlockedTasksResetsEligible(t *testing.T) {
	ts := newMockTaskStore()
	old := time.Now().Add(-20 * time.Minute)
	addTask(ts, "t3", "blocked", "mind", old, map[string]any{
		"blocked_reason": "transient error",
		"retry_count":    0,
	})

	m := newTestMind(ts)
	retried := m.retryBlockedTasks(context.Background())

	if !retried {
		t.Fatal("expected retried=true")
	}
	stored := ts.tasks["t3"]
	if stored.Status != "pending" {
		t.Errorf("status: want pending, got %q", stored.Status)
	}
	if stored.Assignee != "" {
		t.Errorf("assignee: want empty, got %q", stored.Assignee)
	}
	var rc int
	switch v := stored.Metadata["retry_count"].(type) {
	case int:
		rc = v
	case float64:
		rc = int(v)
	}
	if rc != 1 {
		t.Errorf("retry_count: want 1, got %d", rc)
	}
	// prev_failure_reason should be set from the old blocked_reason
	pfr, _ := stored.Metadata["prev_failure_reason"].(string)
	if pfr != "transient error" {
		t.Errorf("prev_failure_reason: want %q, got %q", "transient error", pfr)
	}
}

// TestRetryBlockedTasksMaxRetries verifies that tasks at retry_count=3 are not
// retried.
func TestRetryBlockedTasksMaxRetries(t *testing.T) {
	ts := newMockTaskStore()
	old := time.Now().Add(-20 * time.Minute)
	addTask(ts, "t4", "blocked", "mind", old, map[string]any{
		"retry_count": 3,
	})

	m := newTestMind(ts)
	retried := m.retryBlockedTasks(context.Background())

	if retried {
		t.Error("expected retried=false for task at max retries")
	}
	stored := ts.tasks["t4"]
	if stored.Status != "blocked" {
		t.Errorf("status: want blocked (unchanged), got %q", stored.Status)
	}
}

// TestRetryBlockedTasksTooRecent verifies that tasks updated fewer than 15
// minutes ago are not retried.
func TestRetryBlockedTasksTooRecent(t *testing.T) {
	ts := newMockTaskStore()
	recent := time.Now().Add(-5 * time.Minute)
	addTask(ts, "t5", "blocked", "mind", recent, map[string]any{
		"retry_count": 0,
	})

	m := newTestMind(ts)
	retried := m.retryBlockedTasks(context.Background())

	if retried {
		t.Error("expected retried=false for task updated too recently")
	}
	stored := ts.tasks["t5"]
	if stored.Status != "blocked" {
		t.Errorf("status: want blocked (unchanged), got %q", stored.Status)
	}
}
