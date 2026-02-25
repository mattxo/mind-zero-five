package authority

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgStore is a PostgreSQL-backed authority store.
type PgStore struct {
	pool *pgxpool.Pool
}

// NewPgStore creates a PgStore.
func NewPgStore(pool *pgxpool.Pool) *PgStore {
	return &PgStore{pool: pool}
}

// EnsureTable creates the approval_requests table if it doesn't exist.
func (s *PgStore) EnsureTable(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS approval_requests (
			id          TEXT PRIMARY KEY,
			action      TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			level       TEXT NOT NULL,
			source      TEXT NOT NULL DEFAULT '',
			status      TEXT NOT NULL DEFAULT 'pending',
			created_at  TIMESTAMPTZ DEFAULT NOW(),
			resolved_at TIMESTAMPTZ
		)`)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_approval_status ON approval_requests(status, created_at)`)
	if err != nil {
		return err
	}

	// Policies table
	_, err = s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS authority_policies (
			id          TEXT PRIMARY KEY,
			action      TEXT NOT NULL,
			approver_id TEXT NOT NULL,
			level       TEXT NOT NULL,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
		)`)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_policy_action ON authority_policies(action)`)
	return err
}

// Create inserts a new approval request. Notification level auto-approves immediately.
func (s *PgStore) Create(ctx context.Context, action, description, source string, level Level) (*Request, error) {
	id := uuid.Must(uuid.NewV7()).String()
	now := time.Now().Truncate(time.Microsecond)

	r := &Request{
		ID:          id,
		Action:      action,
		Description: description,
		Level:       level,
		Source:      source,
		Status:      "pending",
		CreatedAt:   now,
	}

	if level == Notification {
		r.Status = "approved"
		r.ResolvedAt = &now
		_, err := s.pool.Exec(ctx, `
			INSERT INTO approval_requests (id, action, description, level, source, status, created_at, resolved_at)
			VALUES ($1, $2, $3, $4, $5, 'approved', $6, $7)`,
			r.ID, r.Action, r.Description, string(r.Level), r.Source, r.CreatedAt, r.ResolvedAt)
		if err != nil {
			return nil, fmt.Errorf("create notification: %w", err)
		}
		return r, nil
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO approval_requests (id, action, description, level, source, status, created_at)
		VALUES ($1, $2, $3, $4, $5, 'pending', $6)`,
		r.ID, r.Action, r.Description, string(r.Level), r.Source, r.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	return r, nil
}

// Resolve approves or rejects a pending request.
func (s *PgStore) Resolve(ctx context.Context, id string, approved bool) (*Request, error) {
	status := "rejected"
	if approved {
		status = "approved"
	}
	now := time.Now().Truncate(time.Microsecond)

	var r Request
	err := s.pool.QueryRow(ctx, `
		UPDATE approval_requests SET status = $1, resolved_at = $2
		WHERE id = $3 AND status = 'pending'
		RETURNING id, action, description, level, source, status, created_at, resolved_at`,
		status, now, id).
		Scan(&r.ID, &r.Action, &r.Description, &r.Level, &r.Source, &r.Status, &r.CreatedAt, &r.ResolvedAt)
	if err != nil {
		return nil, fmt.Errorf("resolve request %s: %w", id, err)
	}
	return &r, nil
}

// Get retrieves a single request by ID.
func (s *PgStore) Get(ctx context.Context, id string) (*Request, error) {
	var r Request
	err := s.pool.QueryRow(ctx, `
		SELECT id, action, description, level, source, status, created_at, resolved_at
		FROM approval_requests WHERE id = $1`, id).
		Scan(&r.ID, &r.Action, &r.Description, &r.Level, &r.Source, &r.Status, &r.CreatedAt, &r.ResolvedAt)
	if err != nil {
		return nil, fmt.Errorf("get request %s: %w", id, err)
	}
	return &r, nil
}

// Pending returns all pending approval requests.
func (s *PgStore) Pending(ctx context.Context) ([]Request, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, action, description, level, source, status, created_at, resolved_at
		FROM approval_requests WHERE status = 'pending'
		ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("pending requests: %w", err)
	}
	defer rows.Close()
	return scanRequestRows(rows)
}

// Recent returns the most recently resolved approval requests.
func (s *PgStore) Recent(ctx context.Context, limit int) ([]Request, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, action, description, level, source, status, created_at, resolved_at
		FROM approval_requests
		ORDER BY created_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("recent requests: %w", err)
	}
	defer rows.Close()
	return scanRequestRows(rows)
}

// PendingCount returns the number of pending requests.
func (s *PgStore) PendingCount(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM approval_requests WHERE status = 'pending'`).Scan(&n)
	return n, err
}

// CreatePolicy creates or updates a policy for an action. Idempotent on action.
func (s *PgStore) CreatePolicy(ctx context.Context, action, approverID string, level Level) (*Policy, error) {
	id := uuid.Must(uuid.NewV7()).String()
	now := time.Now().Truncate(time.Microsecond)

	var p Policy
	err := s.pool.QueryRow(ctx, `
		INSERT INTO authority_policies (id, action, approver_id, level, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (action) DO UPDATE SET approver_id = EXCLUDED.approver_id, level = EXCLUDED.level
		RETURNING id, action, approver_id, level, created_at`,
		id, action, approverID, string(level), now).
		Scan(&p.ID, &p.Action, &p.ApproverID, &p.Level, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create policy for %s: %w", action, err)
	}
	return &p, nil
}

// MatchPolicy finds the policy for an action. Tries exact match first, then "*" fallback.
func (s *PgStore) MatchPolicy(ctx context.Context, action string) (*Policy, error) {
	// Exact match first
	var p Policy
	err := s.pool.QueryRow(ctx, `
		SELECT id, action, approver_id, level, created_at
		FROM authority_policies WHERE action = $1`, action).
		Scan(&p.ID, &p.Action, &p.ApproverID, &p.Level, &p.CreatedAt)
	if err == nil {
		return &p, nil
	}

	// Wildcard fallback
	err = s.pool.QueryRow(ctx, `
		SELECT id, action, approver_id, level, created_at
		FROM authority_policies WHERE action = '*'`).
		Scan(&p.ID, &p.Action, &p.ApproverID, &p.Level, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("no policy for action %s: %w", action, err)
	}
	return &p, nil
}

// ListPolicies returns all policies.
func (s *PgStore) ListPolicies(ctx context.Context) ([]Policy, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, action, approver_id, level, created_at
		FROM authority_policies ORDER BY action`)
	if err != nil {
		return nil, fmt.Errorf("list policies: %w", err)
	}
	defer rows.Close()

	var policies []Policy
	for rows.Next() {
		var p Policy
		if err := rows.Scan(&p.ID, &p.Action, &p.ApproverID, &p.Level, &p.CreatedAt); err != nil {
			return nil, err
		}
		policies = append(policies, p)
	}
	return policies, rows.Err()
}

func scanRequestRows(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]Request, error) {
	var reqs []Request
	for rows.Next() {
		var r Request
		if err := rows.Scan(&r.ID, &r.Action, &r.Description, &r.Level, &r.Source, &r.Status, &r.CreatedAt, &r.ResolvedAt); err != nil {
			return nil, err
		}
		reqs = append(reqs, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration: %w", err)
	}
	return reqs, nil
}
