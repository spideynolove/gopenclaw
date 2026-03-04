//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/spideynolove/gopenclaw/core"
	"github.com/spideynolove/gopenclaw/store/postgres"
)

func newDB(t *testing.T) *sqlx.DB {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = "postgres://gopenclaw:gopenclaw@localhost:5432/gopenclaw?sslmode=disable"
	}
	db, err := sqlx.Open("pgx", url)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestStoreLoadEmpty(t *testing.T) {
	db := newDB(t)
	s := postgres.New(db)
	ctx := context.Background()

	session, err := s.Load(ctx, "telegram:1:1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.ID != "telegram:1:1" {
		t.Errorf("unexpected session ID: %q", session.ID)
	}
	if len(session.Messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(session.Messages))
	}
}

func TestStoreAppendAndLoad(t *testing.T) {
	db := newDB(t)
	s := postgres.New(db)
	ctx := context.Background()

	sessionID := "telegram:test:append"
	err := s.Append(ctx, sessionID, core.Message{Role: "user", Content: "hello"})
	if err != nil {
		t.Fatalf("append: %v", err)
	}

	session, err := s.Load(ctx, sessionID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(session.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(session.Messages))
	}
	if session.Messages[0].Content != "hello" {
		t.Errorf("unexpected content: %q", session.Messages[0].Content)
	}
}

func TestStoreSetSystemPrompt(t *testing.T) {
	db := newDB(t)
	s := postgres.New(db)
	ctx := context.Background()

	sessionID := "telegram:test:soul"
	err := s.SetSystemPrompt(ctx, sessionID, "You are a pirate.")
	if err != nil {
		t.Fatalf("set system prompt: %v", err)
	}

	session, err := s.Load(ctx, sessionID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if session.SystemPrompt != "You are a pirate." {
		t.Errorf("unexpected system prompt: %q", session.SystemPrompt)
	}
}
