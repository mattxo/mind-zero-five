package task

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgStore is a PostgreSQL-backed task store.
type PgStore struct {
	pool *pgxpool.Pool
}

// NewPgStore creates a PgStore.
func NewPgStore(pool *pgxpool.Pool) *PgStore {
	return &PgStore{pool: pool}
}

// EnsureTable creates the tasks table if it doesn't exist.
func (s *PgStore) EnsureTable(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS tasks (
			id           TEXT PRIMARY KEY,
			subject      TEXT NOT NULL,
			description  TEXT NOT NULL DEFAULT '',
			status       TEXT NOT NULL DEFAULT 'pending',
			priority     INTEGER NOT NULL DEFAULT 0,
			source       TEXT NOT NULL DEFAULT '',
			assignee     TEXT NOT NULL DEFAULT '',
			parent_id    TEXT NOT NULL DEFAULT '',
			blocked_by   TEXT[] DEFAULT '{}',
			metadata     JSONB NOT NULL DEFAULT '{}',
			created_at   TIMESTAMPTZ DEFAULT NOW(),
			updated_at   TIMESTAMPTZ DEFAULT NOW(),
			completed_at TIMESTAMPTZ
		)`)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status)`)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_tasks_parent ON tasks(parent_id) WHERE parent_id != ''`)
	return err
}

// Create inserts a new task.
func (s *PgStore) Create(ctx context.Context, t *Task) (*Task, error) {
	t.ID = uuid.Must(uuid.NewV7()).String()
	now := time.Now().Truncate(time.Microsecond)
	t.CreatedAt = now
	t.UpdatedAt = now
	if t.Status == "" {
		t.Status = "pending"
	}
	if t.BlockedBy == nil {
		t.BlockedBy = []string{}
	}
	if t.Metadata == nil {
		t.Metadata = map[string]any{}
	}

	metaJSON, err := json.Marshal(t.Metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO tasks (id, subject, description, status, priority, source, assignee, parent_id, blocked_by, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11, $12)`,
		t.ID, t.Subject, t.Description, t.Status, t.Priority, t.Source, t.Assignee, t.ParentID, t.BlockedBy, string(metaJSON), t.CreatedAt, t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	return t, nil
}

// Get retrieves a single task by ID.
func (s *PgStore) Get(ctx context.Context, id string) (*Task, error) {
	var t Task
	var metaJSON []byte
	err := s.pool.QueryRow(ctx, `
		SELECT id, subject, description, status, priority, source, assignee, parent_id, blocked_by, metadata, created_at, updated_at, completed_at
		FROM tasks WHERE id = $1`, id).
		Scan(&t.ID, &t.Subject, &t.Description, &t.Status, &t.Priority, &t.Source, &t.Assignee, &t.ParentID, &t.BlockedBy, &metaJSON, &t.CreatedAt, &t.UpdatedAt, &t.CompletedAt)
	if err != nil {
		return nil, fmt.Errorf("get task %s: %w", id, err)
	}
	if err := json.Unmarshal(metaJSON, &t.Metadata); err != nil {
		t.Metadata = map[string]any{}
	}
	return &t, nil
}

// Update modifies task fields. Supported keys: status, subject, description, assignee, priority, blocked_by, metadata.
func (s *PgStore) Update(ctx context.Context, id string, updates map[string]any) (*Task, error) {
	now := time.Now().Truncate(time.Microsecond)

	// Build SET clause dynamically
	setClauses := "updated_at = $1"
	args := []any{now}
	argIdx := 2

	for k, v := range updates {
		switch k {
		case "status":
			setClauses += fmt.Sprintf(", status = $%d", argIdx)
			args = append(args, v)
			argIdx++
		case "subject":
			setClauses += fmt.Sprintf(", subject = $%d", argIdx)
			args = append(args, v)
			argIdx++
		case "description":
			setClauses += fmt.Sprintf(", description = $%d", argIdx)
			args = append(args, v)
			argIdx++
		case "assignee":
			setClauses += fmt.Sprintf(", assignee = $%d", argIdx)
			args = append(args, v)
			argIdx++
		case "priority":
			setClauses += fmt.Sprintf(", priority = $%d", argIdx)
			args = append(args, v)
			argIdx++
		case "blocked_by":
			setClauses += fmt.Sprintf(", blocked_by = $%d", argIdx)
			args = append(args, v)
			argIdx++
		case "metadata":
			metaJSON, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("marshal metadata: %w", err)
			}
			setClauses += fmt.Sprintf(", metadata = $%d::jsonb", argIdx)
			args = append(args, string(metaJSON))
			argIdx++
		}
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE tasks SET %s WHERE id = $%d RETURNING id, subject, description, status, priority, source, assignee, parent_id, blocked_by, metadata, created_at, updated_at, completed_at", setClauses, argIdx)

	var t Task
	var metaJSON []byte
	err := s.pool.QueryRow(ctx, query, args...).
		Scan(&t.ID, &t.Subject, &t.Description, &t.Status, &t.Priority, &t.Source, &t.Assignee, &t.ParentID, &t.BlockedBy, &metaJSON, &t.CreatedAt, &t.UpdatedAt, &t.CompletedAt)
	if err != nil {
		return nil, fmt.Errorf("update task %s: %w", id, err)
	}
	if err := json.Unmarshal(metaJSON, &t.Metadata); err != nil {
		t.Metadata = map[string]any{}
	}
	return &t, nil
}

// Complete marks a task as completed.
func (s *PgStore) Complete(ctx context.Context, id string) (*Task, error) {
	now := time.Now().Truncate(time.Microsecond)
	var t Task
	var metaJSON []byte
	err := s.pool.QueryRow(ctx, `
		UPDATE tasks SET status = 'completed', updated_at = $1, completed_at = $1
		WHERE id = $2
		RETURNING id, subject, description, status, priority, source, assignee, parent_id, blocked_by, metadata, created_at, updated_at, completed_at`,
		now, id).
		Scan(&t.ID, &t.Subject, &t.Description, &t.Status, &t.Priority, &t.Source, &t.Assignee, &t.ParentID, &t.BlockedBy, &metaJSON, &t.CreatedAt, &t.UpdatedAt, &t.CompletedAt)
	if err != nil {
		return nil, fmt.Errorf("complete task %s: %w", id, err)
	}
	if err := json.Unmarshal(metaJSON, &t.Metadata); err != nil {
		t.Metadata = map[string]any{}
	}
	return &t, nil
}

// List returns tasks filtered by status (empty = all), ordered by priority desc then created_at asc.
func (s *PgStore) List(ctx context.Context, status string, limit int) ([]Task, error) {
	var query string
	var args []any
	if status != "" {
		query = `SELECT id, subject, description, status, priority, source, assignee, parent_id, blocked_by, metadata, created_at, updated_at, completed_at
			FROM tasks WHERE status = $1 ORDER BY priority DESC, created_at ASC LIMIT $2`
		args = []any{status, limit}
	} else {
		query = `SELECT id, subject, description, status, priority, source, assignee, parent_id, blocked_by, metadata, created_at, updated_at, completed_at
			FROM tasks ORDER BY priority DESC, created_at ASC LIMIT $1`
		args = []any{limit}
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()
	return scanTaskRows(rows)
}

// ByParent returns all subtasks of a parent task.
func (s *PgStore) ByParent(ctx context.Context, parentID string) ([]Task, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, subject, description, status, priority, source, assignee, parent_id, blocked_by, metadata, created_at, updated_at, completed_at
		FROM tasks WHERE parent_id = $1 ORDER BY priority DESC, created_at ASC`, parentID)
	if err != nil {
		return nil, fmt.Errorf("tasks by parent: %w", err)
	}
	defer rows.Close()
	return scanTaskRows(rows)
}

// Count returns total task count.
func (s *PgStore) Count(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM tasks`).Scan(&n)
	return n, err
}

// PendingCount returns count of pending tasks.
func (s *PgStore) PendingCount(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM tasks WHERE status = 'pending'`).Scan(&n)
	return n, err
}

func scanTaskRows(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]Task, error) {
	var tasks []Task
	for rows.Next() {
		var t Task
		var metaJSON []byte
		if err := rows.Scan(&t.ID, &t.Subject, &t.Description, &t.Status, &t.Priority, &t.Source, &t.Assignee, &t.ParentID, &t.BlockedBy, &metaJSON, &t.CreatedAt, &t.UpdatedAt, &t.CompletedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(metaJSON, &t.Metadata); err != nil {
			t.Metadata = map[string]any{}
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration: %w", err)
	}
	return tasks, nil
}
