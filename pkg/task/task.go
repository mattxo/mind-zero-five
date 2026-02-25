package task

import (
	"context"
	"time"
)

// Task represents a unit of work in the system.
type Task struct {
	ID          string         `json:"id"`
	Subject     string         `json:"subject"`
	Description string         `json:"description"`
	Status      string         `json:"status"`       // pending, in_progress, completed, blocked
	Priority    int            `json:"priority"`      // 0 = normal, higher = more urgent
	Source      string         `json:"source"`        // who created it
	Assignee    string         `json:"assignee"`      // who's working on it
	ParentID    string         `json:"parent_id"`     // for subtasks
	BlockedBy   []string       `json:"blocked_by"`    // task IDs blocking this
	Metadata    map[string]any `json:"metadata"`      // flexible extra data
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
}

// Store is the contract for task persistence.
type Store interface {
	Create(ctx context.Context, t *Task) (*Task, error)
	Get(ctx context.Context, id string) (*Task, error)
	Update(ctx context.Context, id string, updates map[string]any) (*Task, error)
	Complete(ctx context.Context, id string) (*Task, error)
	List(ctx context.Context, status string, limit int) ([]Task, error)
	ByParent(ctx context.Context, parentID string) ([]Task, error)
	Count(ctx context.Context) (int, error)
	PendingCount(ctx context.Context) (int, error)
	EnsureTable(ctx context.Context) error
}
