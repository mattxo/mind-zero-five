package eventgraph

import (
	"context"
	"time"
)

// Event is a single node in the hash-chained, append-only causal event log.
type Event struct {
	ID             string         `json:"id"`              // UUID v7 (time-ordered)
	Type           string         `json:"type"`            // e.g. "trust.updated", "goal.created"
	Timestamp      time.Time      `json:"timestamp"`       // when the event occurred
	Source         string         `json:"source"`          // primitive or mind that emitted
	Content        map[string]any `json:"content"`         // event payload
	Causes         []string       `json:"causes"`          // IDs of causing events
	ConversationID string         `json:"conversation_id"` // groups related events into a conversation
	Hash           string         `json:"hash"`            // SHA-256 of canonical form
	PrevHash       string         `json:"prev_hash"`       // hash chain link
}

// EventStore is the contract for event persistence.
type EventStore interface {
	Append(ctx context.Context, eventType, source string, content map[string]any, causes []string, conversationID string) (*Event, error)
	Get(ctx context.Context, id string) (*Event, error)
	Recent(ctx context.Context, limit int) ([]Event, error)
	ByType(ctx context.Context, eventType string, limit int) ([]Event, error)
	BySource(ctx context.Context, source string, limit int) ([]Event, error)
	ByConversation(ctx context.Context, conversationID string, limit int) ([]Event, error)
	Since(ctx context.Context, afterID string, limit int) ([]Event, error)
	Count(ctx context.Context) (int, error)
	VerifyChain(ctx context.Context) error
	EnsureTable(ctx context.Context) error

	// Causal traversal
	Ancestors(ctx context.Context, id string, maxDepth int) ([]Event, error)
	Descendants(ctx context.Context, id string, maxDepth int) ([]Event, error)
	Search(ctx context.Context, query string, limit int) ([]Event, error)
	DistinctTypes(ctx context.Context) ([]string, error)
	DistinctSources(ctx context.Context) ([]string, error)
}
