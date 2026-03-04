package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	"golang.org/x/sync/semaphore"

	"github.com/spideynolove/gopenclaw/core"
	"github.com/spideynolove/gopenclaw/internal/channels/discord"
	"github.com/spideynolove/gopenclaw/internal/channels/telegram"
	"github.com/spideynolove/gopenclaw/internal/crypto"
	"github.com/spideynolove/gopenclaw/internal/memory/postgres"
	"github.com/spideynolove/gopenclaw/internal/providers"
	"github.com/spideynolove/gopenclaw/internal/providers/anthropic"
	"github.com/spideynolove/gopenclaw/internal/providers/openai"
	"github.com/spideynolove/gopenclaw/internal/security"
	store "github.com/spideynolove/gopenclaw/store/postgres"
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
	files := []string{
		"migrations/001_init.sql",
		"migrations/002_memory.sql",
		"migrations/003_tenants.sql",
		"migrations/004_encrypted_secrets.sql",
	}

	for _, file := range files {
		migration, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", file, err)
		}

		_, err = db.Exec(string(migration))
		if err != nil {
			return fmt.Errorf("execute migration %s: %w", file, err)
		}
	}

	return nil
}

type Server struct {
	db              *sqlx.DB
	mux             *http.ServeMux
	provider        core.Provider
	store           core.SessionStore
	memoryBackend   core.MemoryBackend
	telegramAdapter core.ChannelAdapter
	discordAdapter  core.ChannelAdapter
	tenantSems      map[string]*semaphore.Weighted
	tenantMu        sync.RWMutex
	eventChan       chan core.Event
	gatewayDone     chan error
}

func NewServer(db *sqlx.DB, provider core.Provider, store core.SessionStore, memoryBackend core.MemoryBackend, telegramAdapter core.ChannelAdapter, discordAdapter core.ChannelAdapter) *Server {
	return &Server{
		db:              db,
		mux:             http.NewServeMux(),
		provider:        provider,
		store:           store,
		memoryBackend:   memoryBackend,
		telegramAdapter: telegramAdapter,
		discordAdapter:  discordAdapter,
		tenantSems:      make(map[string]*semaphore.Weighted),
		eventChan:       make(chan core.Event, 100),
		gatewayDone:     make(chan error, 1),
	}
}

func (s *Server) getSemaphore(tenantID string) *semaphore.Weighted {
	s.tenantMu.Lock()
	defer s.tenantMu.Unlock()

	if sem, ok := s.tenantSems[tenantID]; ok {
		return sem
	}

	sem := semaphore.NewWeighted(5)
	s.tenantSems[tenantID] = sem
	return sem
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("POST /webhook/{tenantID}/telegram", s.telegramWebhook)
	s.mux.HandleFunc("POST /webhook/{tenantID}/discord", s.discordWebhook)
	s.mux.HandleFunc("GET /health", s.health)
}

func (s *Server) telegramWebhook(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenantID")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}

	var update struct {
		Message struct {
			Chat struct {
				ID int64 `json:"id"`
			} `json:"chat"`
			From struct {
				ID int64 `json:"id"`
			} `json:"from"`
			Text string `json:"text"`
		} `json:"message"`
	}

	if err := json.Unmarshal(body, &update); err != nil {
		http.Error(w, "parse json", http.StatusBadRequest)
		return
	}

	if update.Message.Text == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	sessionID := core.SessionID(tenantID, "telegram", update.Message.Chat.ID, update.Message.From.ID)
	evt := core.Event{
		SessionID: sessionID,
		ChatID:    update.Message.Chat.ID,
		UserID:    update.Message.From.ID,
		Text:      update.Message.Text,
		TenantID:  tenantID,
	}

	select {
	case s.eventChan <- evt:
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "event queue full", http.StatusServiceUnavailable)
	}
}

func (s *Server) discordWebhook(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenantID")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}

	var msg struct {
		Content   string `json:"content"`
		ChannelID string `json:"channel_id"`
		Author    struct {
			ID string `json:"id"`
		} `json:"author"`
	}

	if err := json.Unmarshal(body, &msg); err != nil {
		http.Error(w, "parse json", http.StatusBadRequest)
		return
	}

	if msg.Content == "" || msg.Author.ID == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var chatID int64
	_, err = fmt.Sscanf(msg.ChannelID, "%d", &chatID)
	if err != nil {
		http.Error(w, "parse channel id", http.StatusBadRequest)
		return
	}

	var userID int64
	_, err = fmt.Sscanf(msg.Author.ID, "%d", &userID)
	if err != nil {
		http.Error(w, "parse user id", http.StatusBadRequest)
		return
	}

	sessionID := core.SessionID(tenantID, "discord", chatID, userID)
	evt := core.Event{
		SessionID: sessionID,
		ChatID:    chatID,
		UserID:    userID,
		Text:      msg.Content,
		TenantID:  tenantID,
	}

	select {
	case s.eventChan <- evt:
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "event queue full", http.StatusServiceUnavailable)
	}
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok"}`)
}

func (s *Server) Run(ctx context.Context) error {
	s.registerRoutes()

	gateway := core.NewGateway(s.provider, s.telegramAdapter, s.store)

	go func() {
		s.gatewayDone <- gateway.Run(ctx)
	}()

	httpServer := &http.Server{
		Addr:    mustEnvOrDefault("GATEWAY_BIND", "127.0.0.1:8080"),
		Handler: s.mux,
	}

	go func() {
		slog.Info("http server starting", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.gatewayDone <- fmt.Errorf("http server: %w", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	httpServer.Shutdown(shutdownCtx)

	return <-s.gatewayDone
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	databaseURL := mustEnv("DATABASE_URL")
	telegramToken := mustEnv("TELEGRAM_TOKEN")
	openaiAPIKey := mustEnv("OPENAI_API_KEY")
	discordToken := os.Getenv("DISCORD_TOKEN")
	anthropicAPIKey := os.Getenv("ANTHROPIC_API_KEY")
	encryptionKeyStr := os.Getenv("ENCRYPTION_KEY")
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

	sessionStore := store.New(db)
	memoryBackend := postgres.New(db)
	slog.Info("initialized stores", "prompt", defaultSystemPrompt)

	_ = security.New()
	slog.Info("initialized input sanitizer")

	var secretManager *crypto.SecretManager
	if encryptionKeyStr != "" {
		keyBytes, err := base64.StdEncoding.DecodeString(encryptionKeyStr)
		if err != nil {
			slog.Warn("failed to decode encryption key, using unencrypted mode", "err", err)
		} else {
			secretManager, err = crypto.NewSecretManager(keyBytes)
			if err != nil {
				slog.Warn("failed to initialize secret manager", "err", err)
			} else {
				slog.Info("initialized secret manager with AES-256-GCM")
			}
		}
	} else {
		slog.Warn("ENCRYPTION_KEY not set, secrets will not be encrypted")
	}

	openaiProvider := openai.New(openaiAPIKey, "gpt-4", "")
	providerList := []core.Provider{openaiProvider}

	if anthropicAPIKey != "" {
		anthropicProvider := anthropic.New(anthropicAPIKey, "claude-3-5-sonnet-20241022")
		providerList = append(providerList, anthropicProvider)
		slog.Info("added anthropic provider to failover chain")
	}

	var provider core.Provider
	if len(providerList) > 1 {
		provider = providers.NewChain(providerList...)
		slog.Info("initialized provider chain", "count", len(providerList))
	} else {
		provider = providerList[0]
		slog.Info("using single provider (openai)")
	}

	telegramAdapter, err := telegram.New(telegramToken)
	if err != nil {
		slog.Error("failed to create telegram adapter", "err", err)
		os.Exit(1)
	}

	var discordAdapter core.ChannelAdapter
	if discordToken != "" {
		tenantID := os.Getenv("DEFAULT_TENANT_ID")
		if tenantID == "" {
			tenantID = "default"
		}
		discordAdapter, err = discord.New(discordToken, tenantID)
		if err != nil {
			slog.Error("failed to create discord adapter", "err", err)
			os.Exit(1)
		}
	}

	server := NewServer(db, provider, sessionStore, memoryBackend, telegramAdapter, discordAdapter)

	slog.Info("gopenclaw starting", "sanitizer_enabled", true, "encryption_enabled", secretManager != nil)

	go func() {
		<-sigChan
		slog.Info("signal received, shutting down")
		cancel()
	}()

	if err := server.Run(ctx); err != nil && err != context.Canceled {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
