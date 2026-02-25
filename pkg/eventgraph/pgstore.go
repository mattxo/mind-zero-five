package eventgraph

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgStore is a PostgreSQL-backed EventStore with hash-chained integrity.
type PgStore struct {
	pool *pgxpool.Pool
}

// NewPgStore creates a PgStore.
func NewPgStore(pool *pgxpool.Pool) *PgStore {
	return &PgStore{pool: pool}
}

// EnsureTable creates the events table if it doesn't exist.
func (s *PgStore) EnsureTable(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS events (
			id              TEXT PRIMARY KEY,
			type            TEXT NOT NULL,
			timestamp       TIMESTAMPTZ NOT NULL,
			source          TEXT NOT NULL,
			content         JSONB NOT NULL DEFAULT '{}',
			causes          TEXT[] DEFAULT '{}',
			conversation_id TEXT NOT NULL DEFAULT '',
			hash            TEXT NOT NULL,
			prev_hash       TEXT NOT NULL DEFAULT ''
		)`)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_events_type ON events(type)`)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_events_source ON events(source)`)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_events_timestamp_id ON events(timestamp, id)`)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_events_causes ON events USING GIN(causes)`)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_events_conversation ON events(conversation_id) WHERE conversation_id != ''`)
	return err
}

// Append creates and stores a new event, computing the hash chain.
func (s *PgStore) Append(ctx context.Context, eventType, source string, content map[string]any, causes []string, conversationID string) (*Event, error) {
	if content == nil {
		content = map[string]any{}
	}
	if causes == nil {
		causes = []string{}
	}

	contentJSON, err := json.Marshal(content)
	if err != nil {
		return nil, fmt.Errorf("marshal content: %w", err)
	}

	now := time.Now().Truncate(time.Microsecond)
	id := uuid.Must(uuid.NewV7()).String()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var prevHash string
	err = tx.QueryRow(ctx, `SELECT hash FROM events ORDER BY timestamp DESC, id DESC LIMIT 1 FOR UPDATE`).Scan(&prevHash)
	if err != nil {
		prevHash = ""
	}

	hash := computeHash(prevHash, id, eventType, source, conversationID, now, contentJSON)

	e := &Event{
		ID:             id,
		Type:           eventType,
		Timestamp:      now,
		Source:         source,
		Content:        content,
		Causes:         causes,
		ConversationID: conversationID,
		Hash:           hash,
		PrevHash:       prevHash,
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO events (id, type, timestamp, source, content, causes, conversation_id, hash, prev_hash)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8, $9)`,
		e.ID, e.Type, e.Timestamp, e.Source, string(contentJSON), e.Causes, e.ConversationID, e.Hash, e.PrevHash)
	if err != nil {
		return nil, fmt.Errorf("insert event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit event: %w", err)
	}

	return e, nil
}

// Get retrieves a single event by ID.
func (s *PgStore) Get(ctx context.Context, id string) (*Event, error) {
	e, err := s.scanOne(ctx, `
		SELECT id, type, timestamp, source, content, causes, conversation_id, hash, prev_hash
		FROM events WHERE id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("get event %s: %w", id, err)
	}
	return e, nil
}

// Recent returns the most recent events in reverse chronological order.
func (s *PgStore) Recent(ctx context.Context, limit int) ([]Event, error) {
	return s.scanMany(ctx, `
		SELECT id, type, timestamp, source, content, causes, conversation_id, hash, prev_hash
		FROM events ORDER BY timestamp DESC, id DESC LIMIT $1`, limit)
}

// ByType returns events filtered by type (exact match or prefix with %).
func (s *PgStore) ByType(ctx context.Context, eventType string, limit int) ([]Event, error) {
	return s.scanMany(ctx, `
		SELECT id, type, timestamp, source, content, causes, conversation_id, hash, prev_hash
		FROM events WHERE type = $1 ORDER BY timestamp DESC, id DESC LIMIT $2`, eventType, limit)
}

// BySource returns events filtered by source.
func (s *PgStore) BySource(ctx context.Context, source string, limit int) ([]Event, error) {
	return s.scanMany(ctx, `
		SELECT id, type, timestamp, source, content, causes, conversation_id, hash, prev_hash
		FROM events WHERE source = $1 ORDER BY timestamp DESC, id DESC LIMIT $2`, source, limit)
}

// ByConversation returns events in a conversation in chronological order.
func (s *PgStore) ByConversation(ctx context.Context, conversationID string, limit int) ([]Event, error) {
	return s.scanMany(ctx, `
		SELECT id, type, timestamp, source, content, causes, conversation_id, hash, prev_hash
		FROM events WHERE conversation_id = $1 ORDER BY timestamp ASC, id ASC LIMIT $2`, conversationID, limit)
}

// Since returns events created after the given ID, for polling/SSE.
func (s *PgStore) Since(ctx context.Context, afterID string, limit int) ([]Event, error) {
	return s.scanMany(ctx, `
		SELECT id, type, timestamp, source, content, causes, conversation_id, hash, prev_hash
		FROM events WHERE (timestamp, id) > (SELECT timestamp, id FROM events WHERE id = $1)
		ORDER BY timestamp ASC, id ASC LIMIT $2`, afterID, limit)
}

// Count returns the total number of events.
func (s *PgStore) Count(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM events`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count events: %w", err)
	}
	return n, nil
}

// VerifyChain walks the entire chain chronologically and verifies hash integrity.
func (s *PgStore) VerifyChain(ctx context.Context) error {
	rows, err := s.pool.Query(ctx, `
		SELECT id, type, timestamp, source, content, causes, conversation_id, hash, prev_hash
		FROM events ORDER BY timestamp ASC, id ASC`)
	if err != nil {
		return fmt.Errorf("verify chain query: %w", err)
	}
	defer rows.Close()

	prevHash := ""
	i := 0
	for rows.Next() {
		var e Event
		var contentJSON []byte
		err := rows.Scan(&e.ID, &e.Type, &e.Timestamp, &e.Source, &contentJSON, &e.Causes, &e.ConversationID, &e.Hash, &e.PrevHash)
		if err != nil {
			return fmt.Errorf("verify chain scan row %d: %w", i, err)
		}
		if err := json.Unmarshal(contentJSON, &e.Content); err != nil {
			e.Content = map[string]any{"_raw": string(contentJSON)}
		}

		if e.PrevHash != prevHash {
			return fmt.Errorf("event %d (%s): prev_hash mismatch: got %s, want %s", i, e.ID, e.PrevHash, prevHash)
		}
		contentJSON2, _ := json.Marshal(e.Content)
		expected := computeHash(prevHash, e.ID, e.Type, e.Source, e.ConversationID, e.Timestamp, contentJSON2)
		if e.Hash != expected {
			expected2 := computeHash(prevHash, e.ID, e.Type, e.Source, e.ConversationID, e.Timestamp, contentJSON)
			if e.Hash != expected2 {
				return fmt.Errorf("event %d (%s): hash mismatch: got %s, want remarshal=%s or raw=%s", i, e.ID, e.Hash, expected, expected2)
			}
		}
		prevHash = e.Hash
		i++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("verify chain rows: %w", err)
	}
	return nil
}

// Ancestors walks up the causes chain recursively.
func (s *PgStore) Ancestors(ctx context.Context, id string, maxDepth int) ([]Event, error) {
	return s.scanMany(ctx, `
		WITH RECURSIVE ancestors AS (
			SELECT e.id, e.type, e.timestamp, e.source, e.content, e.causes, e.conversation_id, e.hash, e.prev_hash, 1 AS depth
			FROM events e
			WHERE e.id = ANY(SELECT unnest(causes) FROM events WHERE id = $1)
			UNION
			SELECT e.id, e.type, e.timestamp, e.source, e.content, e.causes, e.conversation_id, e.hash, e.prev_hash, a.depth + 1
			FROM events e
			JOIN ancestors a ON e.id = ANY(SELECT unnest(causes) FROM events WHERE id = a.id)
			WHERE a.depth < $2
		)
		SELECT DISTINCT id, type, timestamp, source, content, causes, conversation_id, hash, prev_hash
		FROM ancestors
		ORDER BY timestamp ASC, id ASC`, id, maxDepth)
}

// Descendants finds events that cite the given ID in their causes.
func (s *PgStore) Descendants(ctx context.Context, id string, maxDepth int) ([]Event, error) {
	return s.scanMany(ctx, `
		WITH RECURSIVE descendants AS (
			SELECT e.id, e.type, e.timestamp, e.source, e.content, e.causes, e.conversation_id, e.hash, e.prev_hash, 1 AS depth
			FROM events e
			WHERE $1 = ANY(e.causes)
			UNION
			SELECT e.id, e.type, e.timestamp, e.source, e.content, e.causes, e.conversation_id, e.hash, e.prev_hash, d.depth + 1
			FROM events e
			JOIN descendants d ON d.id = ANY(e.causes)
			WHERE d.depth < $2
		)
		SELECT DISTINCT id, type, timestamp, source, content, causes, conversation_id, hash, prev_hash
		FROM descendants
		ORDER BY timestamp ASC, id ASC`, id, maxDepth)
}

// Search performs a text search across type, source, and content fields.
func (s *PgStore) Search(ctx context.Context, query string, limit int) ([]Event, error) {
	like := "%" + query + "%"
	return s.scanMany(ctx, `
		SELECT id, type, timestamp, source, content, causes, conversation_id, hash, prev_hash
		FROM events
		WHERE type ILIKE $1 OR source ILIKE $1 OR content::text ILIKE $1
		ORDER BY timestamp DESC, id DESC LIMIT $2`, like, limit)
}

// DistinctTypes returns all unique event types.
func (s *PgStore) DistinctTypes(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT type FROM events ORDER BY type`)
	if err != nil {
		return nil, fmt.Errorf("distinct types: %w", err)
	}
	defer rows.Close()
	var types []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		types = append(types, t)
	}
	return types, rows.Err()
}

// DistinctSources returns all unique event sources.
func (s *PgStore) DistinctSources(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT source FROM events ORDER BY source`)
	if err != nil {
		return nil, fmt.Errorf("distinct sources: %w", err)
	}
	defer rows.Close()
	var sources []string
	for rows.Next() {
		var src string
		if err := rows.Scan(&src); err != nil {
			return nil, err
		}
		sources = append(sources, src)
	}
	return sources, rows.Err()
}

func (s *PgStore) scanOne(ctx context.Context, query string, args ...any) (*Event, error) {
	var e Event
	var contentJSON []byte
	err := s.pool.QueryRow(ctx, query, args...).
		Scan(&e.ID, &e.Type, &e.Timestamp, &e.Source, &contentJSON, &e.Causes, &e.ConversationID, &e.Hash, &e.PrevHash)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(contentJSON, &e.Content); err != nil {
		return nil, fmt.Errorf("unmarshal content: %w", err)
	}
	return &e, nil
}

func (s *PgStore) scanMany(ctx context.Context, query string, args ...any) ([]Event, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRows(rows)
}

func scanRows(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]Event, error) {
	var events []Event
	for rows.Next() {
		var e Event
		var contentJSON []byte
		if err := rows.Scan(&e.ID, &e.Type, &e.Timestamp, &e.Source, &contentJSON, &e.Causes, &e.ConversationID, &e.Hash, &e.PrevHash); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(contentJSON, &e.Content); err != nil {
			return nil, fmt.Errorf("unmarshal content: %w", err)
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration: %w", err)
	}
	return events, nil
}

// computeHash computes a SHA-256 hash for chain integrity.
func computeHash(prevHash, id, eventType, source, conversationID string, timestamp time.Time, contentJSON []byte) string {
	data := fmt.Sprintf("%s|%s|%s|%s|%s|%d|%s", prevHash, id, eventType, source, conversationID, timestamp.UnixNano(), string(contentJSON))
	h := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", h)
}
