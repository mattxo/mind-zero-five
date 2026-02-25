package actor

import (
	"context"
	"time"
)

// Actor represents an identified entity in the system.
type Actor struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`       // "human", "mind", "system"
	Name      string    `json:"name"`       // "matt", "mind", "api"
	Email     string    `json:"email"`      // for humans
	CreatedAt time.Time `json:"created_at"`
}

// Store is the contract for actor persistence.
type Store interface {
	// Register creates or returns an existing actor. Idempotent:
	// matches on (type, name) or email.
	Register(ctx context.Context, actorType, name, email string) (*Actor, error)

	// Get returns an actor by ID.
	Get(ctx context.Context, id string) (*Actor, error)

	// ByName returns an actor by name.
	ByName(ctx context.Context, name string) (*Actor, error)

	// List returns all actors.
	List(ctx context.Context) ([]Actor, error)

	// EnsureTable creates the actors table if it doesn't exist.
	EnsureTable(ctx context.Context) error
}
