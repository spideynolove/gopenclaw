package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/pgvector/pgvector-go"
	"github.com/spideynolove/gopenclaw/core"
)

type Memory struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Memory {
	return &Memory{db: db}
}

func (m *Memory) Store(ctx context.Context, mem core.Memory) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO memories (session_id, content, embedding) VALUES ($1, $2, $3)`,
		mem.SessionID, mem.Content, pgvector.NewVector(mem.Embedding))
	if err != nil {
		return fmt.Errorf("store memory: %w", err)
	}
	return nil
}

func (m *Memory) Search(ctx context.Context, sessionID, query string, embedding []float32) ([]core.Memory, error) {
	type row struct {
		ID        int64  `db:"id"`
		SessionID string `db:"session_id"`
		Content   string `db:"content"`
	}
	var rows []row
	err := m.db.SelectContext(ctx, &rows, `
		SELECT id, session_id, content
		FROM memories
		WHERE session_id = $1
		ORDER BY (
			0.7 * (1 - (embedding <=> $2)) +
			0.3 * ts_rank(tsv, plainto_tsquery('english', $3))
		) DESC
		LIMIT 5`,
		sessionID, pgvector.NewVector(embedding), query)
	if err != nil {
		return nil, fmt.Errorf("search memory: %w", err)
	}
	results := make([]core.Memory, len(rows))
	for i, r := range rows {
		results[i] = core.Memory{ID: r.ID, SessionID: r.SessionID, Content: r.Content}
	}
	return results, nil
}

func (m *Memory) Flush(ctx context.Context, sessionID string) error {
	_, err := m.db.ExecContext(ctx,
		`DELETE FROM memories WHERE session_id = $1`,
		sessionID)
	if err != nil {
		return fmt.Errorf("flush memory: %w", err)
	}
	return nil
}
