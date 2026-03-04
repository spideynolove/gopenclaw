package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jmoiron/sqlx"
	"github.com/spideynolove/gopenclaw/core"
	"github.com/spideynolove/gopenclaw/internal/channels/telegram"
	"github.com/spideynolove/gopenclaw/internal/providers/openai"
	"github.com/spideynolove/gopenclaw/store/postgres"
)

func mustEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		slog.Error("missing required environment variable", "key", key)
		os.Exit(1)
	}
	return val
}

func mustEnvOrDefault(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}

func migrate(db *sqlx.DB) error {
	migration, err := os.ReadFile("migrations/001_init.sql")
	if err != nil {
		return fmt.Errorf("read migration: %w", err)
	}

	_, err = db.Exec(string(migration))
	if err != nil {
		return fmt.Errorf("execute migration: %w", err)
	}

	return nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	databaseURL := mustEnv("DATABASE_URL")
	telegramToken := mustEnv("TELEGRAM_TOKEN")
	openaiAPIKey := mustEnv("OPENAI_API_KEY")
	defaultSystemPrompt := mustEnvOrDefault("DEFAULT_SYSTEM_PROMPT", "You are a helpful assistant.")

	db, err := sqlx.Open("pgx", databaseURL)
	if err != nil {
		slog.Error("failed to open database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		slog.Error("failed to ping database", "err", err)
		os.Exit(1)
	}

	if err := migrate(db); err != nil {
		slog.Error("failed to run migrations", "err", err)
		os.Exit(1)
	}

	store := postgres.New(db)
	slog.Info("initialized session store with default system prompt", "prompt", defaultSystemPrompt)

	provider := openai.New(openaiAPIKey, "gpt-4", "")
	adapter, err := telegram.New(telegramToken)
	if err != nil {
		slog.Error("failed to create telegram adapter", "err", err)
		os.Exit(1)
	}

	gateway := core.NewGateway(provider, adapter, store)

	slog.Info("gopenclaw starting")

	errChan := make(chan error, 1)
	go func() {
		errChan <- gateway.Run(ctx)
	}()

	select {
	case <-sigChan:
		slog.Info("signal received, shutting down")
		cancel()
		<-errChan
	case err := <-errChan:
		if err != nil && err != context.Canceled {
			slog.Error("gateway error", "err", err)
			os.Exit(1)
		}
	}
}
