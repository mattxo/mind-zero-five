package actor

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgStore is a PostgreSQL-backed actor store.
type PgStore struct {
	pool *pgxpool.Pool
}

// NewPgStore creates a PgStore.
func NewPgStore(pool *pgxpool.Pool) *PgStore {
	return &PgStore{pool: pool}
}

// EnsureTable creates the actors table if it doesn't exist.
func (s *PgStore) EnsureTable(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS actors (
			id         TEXT PRIMARY KEY,
			type       TEXT NOT NULL,
			name       TEXT NOT NULL,
			email      TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS actors_type_name_idx ON actors(type, name)`)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS actors_email_idx ON actors(email) WHERE email IS NOT NULL`)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS actors_name_idx ON actors(name)`)
	return err
}

// Register creates or returns an existing actor. Idempotent.
func (s *PgStore) Register(ctx context.Context, actorType, name, email string) (*Actor, error) {
	// Try to find existing by email
	if email != "" {
		a, err := s.scanOne(ctx, `SELECT id, type, name, email, created_at FROM actors WHERE email = $1`, email)
		if err == nil {
			return a, nil
		}
	}

	// Try to find by type + name
	a, err := s.scanOne(ctx, `SELECT id, type, name, email, created_at FROM actors WHERE type = $1 AND name = $2`, actorType, name)
	if err == nil {
		return a, nil
	}

	// Create new
	id := uuid.Must(uuid.NewV7()).String()
	now := time.Now().Truncate(time.Microsecond)
	emailPtr := nilIfEmpty(email)

	_, err = s.pool.Exec(ctx, `
		INSERT INTO actors (id, type, name, email, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT DO NOTHING`,
		id, actorType, name, emailPtr, now)
	if err != nil {
		return nil, fmt.Errorf("register actor %s/%s: %w", actorType, name, err)
	}

	// Re-fetch to handle race conditions (ON CONFLICT DO NOTHING)
	a, err = s.scanOne(ctx, `SELECT id, type, name, email, created_at FROM actors WHERE type = $1 AND name = $2`, actorType, name)
	if err != nil {
		return nil, fmt.Errorf("register actor %s/%s: re-fetch failed: %w", actorType, name, err)
	}
	return a, nil
}

// Get returns an actor by ID.
func (s *PgStore) Get(ctx context.Context, id string) (*Actor, error) {
	a, err := s.scanOne(ctx, `SELECT id, type, name, email, created_at FROM actors WHERE id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("get actor %s: %w", id, err)
	}
	return a, nil
}

// ByName returns an actor by name.
func (s *PgStore) ByName(ctx context.Context, name string) (*Actor, error) {
	a, err := s.scanOne(ctx, `SELECT id, type, name, email, created_at FROM actors WHERE name = $1`, name)
	if err != nil {
		return nil, fmt.Errorf("actor by name %s: %w", name, err)
	}
	return a, nil
}

// List returns all actors.
func (s *PgStore) List(ctx context.Context) ([]Actor, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, type, name, email, created_at FROM actors ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("list actors: %w", err)
	}
	defer rows.Close()

	var actors []Actor
	for rows.Next() {
		var a Actor
		var email *string
		if err := rows.Scan(&a.ID, &a.Type, &a.Name, &email, &a.CreatedAt); err != nil {
			return nil, err
		}
		if email != nil {
			a.Email = *email
		}
		actors = append(actors, a)
	}
	return actors, rows.Err()
}

func (s *PgStore) scanOne(ctx context.Context, query string, args ...any) (*Actor, error) {
	var a Actor
	var email *string
	err := s.pool.QueryRow(ctx, query, args...).Scan(&a.ID, &a.Type, &a.Name, &email, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	if email != nil {
		a.Email = *email
	}
	return &a, nil
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
