//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/spideynolove/gopenclaw/core"
	"github.com/spideynolove/gopenclaw/internal/memory/postgres"
)

func testDB(t *testing.T) *sqlx.DB {
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

func TestMemoryStoreAndSearch(t *testing.T) {
	db := testDB(t)
	m := postgres.New(db)
	ctx := context.Background()

	vec := make([]float32, 1536)
	vec[0] = 0.9

	err := m.Store(ctx, core.Memory{
		SessionID: "telegram:1:1",
		Content:   "the user loves Go programming",
		Embedding: vec,
	})
	if err != nil {
		t.Fatalf("store: %v", err)
	}

	results, err := m.Search(ctx, "telegram:1:1", "Go programming", vec)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least 1 result")
	}
	if results[0].Content != "the user loves Go programming" {
		t.Errorf("unexpected content: %q", results[0].Content)
	}
}

func TestMemoryFlush(t *testing.T) {
	db := testDB(t)
	m := postgres.New(db)
	ctx := context.Background()

	vec := make([]float32, 1536)
	vec[0] = 0.5

	sessionID := "telegram:flush:test"
	err := m.Store(ctx, core.Memory{
		SessionID: sessionID,
		Content:   "test memory 1",
		Embedding: vec,
	})
	if err != nil {
		t.Fatalf("store: %v", err)
	}

	err = m.Flush(ctx, sessionID)
	if err != nil {
		t.Fatalf("flush: %v", err)
	}

	results, err := m.Search(ctx, sessionID, "test", vec)
	if err != nil {
		t.Fatalf("search after flush: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results after flush, got %d", len(results))
	}
}
