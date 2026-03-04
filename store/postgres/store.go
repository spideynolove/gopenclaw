package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/spideynolove/gopenclaw/core"
)

type Store struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Load(ctx context.Context, sessionID string) (*core.Session, error) {
	session := &core.Session{ID: sessionID}

	var prompt sql.NullString
	err := s.db.GetContext(ctx, &prompt,
		`SELECT system_prompt FROM sessions WHERE id = $1`, sessionID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if prompt.Valid {
		session.SystemPrompt = prompt.String
	}

	type row struct {
		Role    string `db:"role"`
		Content string `db:"content"`
	}
	var rows []row
	err = s.db.SelectContext(ctx, &rows,
		`SELECT role, content FROM messages WHERE session_id = $1 ORDER BY created_at`, sessionID)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		session.Messages = append(session.Messages, core.Message{Role: r.Role, Content: r.Content})
	}
	return session, nil
}

func (s *Store) Append(ctx context.Context, sessionID string, msg core.Message) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (id) VALUES ($1) ON CONFLICT DO NOTHING`, sessionID)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO messages (session_id, role, content) VALUES ($1, $2, $3)`,
		sessionID, msg.Role, msg.Content)
	return err
}

func (s *Store) SetSystemPrompt(ctx context.Context, sessionID, prompt string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (id, system_prompt) VALUES ($1, $2)
		 ON CONFLICT (id) DO UPDATE SET system_prompt = EXCLUDED.system_prompt`,
		sessionID, prompt)
	return err
}
