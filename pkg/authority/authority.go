package authority

import (
	"context"
	"time"
)

// Level determines how an approval request is handled.
type Level string

const (
	Required     Level = "required"     // blocks until a human approves or rejects
	Recommended  Level = "recommended"  // auto-approves after timeout
	Notification Level = "notification" // auto-approves immediately
)

// RecommendedTimeout is how long a Recommended request waits before auto-approving.
const RecommendedTimeout = 15 * time.Minute

// Request is an approval request.
type Request struct {
	ID          string     `json:"id"`
	Action      string     `json:"action"`
	Description string     `json:"description"`
	Level       Level      `json:"level"`
	Source      string     `json:"source"`
	Status      string     `json:"status"` // pending, approved, rejected
	CreatedAt   time.Time  `json:"created_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

// Policy defines who can approve a given action and at what level.
type Policy struct {
	ID         string    `json:"id"`
	Action     string    `json:"action"`      // exact match or "*" wildcard
	ApproverID string    `json:"approver_id"` // actor ID of the approver
	Level      Level     `json:"level"`       // default level for this action
	CreatedAt  time.Time `json:"created_at"`
}

// Store is the contract for authority persistence.
type Store interface {
	Create(ctx context.Context, action, description, source string, level Level) (*Request, error)
	Resolve(ctx context.Context, id string, approved bool) (*Request, error)
	Get(ctx context.Context, id string) (*Request, error)
	Pending(ctx context.Context) ([]Request, error)
	Recent(ctx context.Context, limit int) ([]Request, error)
	PendingCount(ctx context.Context) (int, error)

	// Policies
	CreatePolicy(ctx context.Context, action, approverID string, level Level) (*Policy, error)
	MatchPolicy(ctx context.Context, action string) (*Policy, error)
	ListPolicies(ctx context.Context) ([]Policy, error)

	EnsureTable(ctx context.Context) error
}
