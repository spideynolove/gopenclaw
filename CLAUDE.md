# GOpenClaw — Claude Code Guide

## Module

```
module github.com/spideynolove/gopenclaw
go 1.22
```

## Commands

```bash
go build ./...
go run ./cmd/gopenclaw/
go test ./...
go vet ./...
```

No Makefile yet. Run migrations manually or via startup hook in `main.go`.

## Directory Layout

```
core/
  types.go        — Message, Event, Session, Memory, ToolCall, ToolResult, Policy
  interfaces.go   — all interface definitions
  gateway.go      — event loop, orchestrates interfaces only
internal/
  providers/openai/
  providers/anthropic/       (week 5)
  channels/telegram/
  channels/discord/          (week 3)
  store/postgres/
  memory/postgres/           (week 2)
  tools/executor/            (week 4)
cmd/gopenclaw/
  main.go         — ONLY place that wires concrete types to interfaces
migrations/
```

**Dependency rule:** `internal/` depends on `core/`. `core/` has zero external imports. Never import `internal/` from `core/`.

## Core Interfaces (by week)

```go
// Week 1
type Provider interface {
    Complete(ctx context.Context, session Session) (string, error)
}

type ChannelAdapter interface {
    Start(ctx context.Context, out chan<- Event) error
    Send(ctx context.Context, evt Event, text string) error
}

type SessionStore interface {
    Load(ctx context.Context, sessionID string) (*Session, error)
    Append(ctx context.Context, sessionID string, msg Message) error
    SetSystemPrompt(ctx context.Context, sessionID, prompt string) error
}

// Week 2
type MemoryBackend interface {
    Search(ctx context.Context, tenantID, agentID, query string) ([]Memory, error)
    Store(ctx context.Context, m Memory) error
    Flush(ctx context.Context, sessionID string) error
}

// Week 4
type ToolExecutor interface {
    Execute(ctx context.Context, call ToolCall, policy Policy) (ToolResult, error)
}
```

**Do not add new interfaces to `core/` outside their scheduled week.**

## Week-by-Week Interface Schedule

| Week | New Interfaces | Concrete Implementations |
|------|---------------|-------------------------|
| 1    | Provider, ChannelAdapter, SessionStore | openai, telegram, postgres |
| 2    | MemoryBackend | postgres memory |
| 3    | — | discord, tenant middleware |
| 4    | ToolExecutor | docker sandbox, web_search, memory_write |
| 5    | — | anthropic, failover chain |

If you are implementing Week N, only the interfaces from weeks ≤ N exist. Do not forward-reference interfaces that haven't been added yet.

## Database

- Driver: `pgx/v5/stdlib` registered into `database/sql`
- Query layer: `sqlx` (struct scanning, named queries)
- No ORM. No code generation. Raw SQL only.
- Extension: `pgvector` (vector(1536) columns in `memories`)

## Session ID Format

- Single-tenant: `{channel}:{chatID}:{userID}` — e.g. `telegram:123456:789012`
- Multi-tenant (week 3+): `{tenantID}:{channel}:{chatID}:{userID}`

## Env Vars

| Var | Week | Purpose |
|-----|------|---------|
| `DATABASE_URL` | 1 | Postgres connection string |
| `TELEGRAM_TOKEN` | 1 | Telegram bot token |
| `OPENAI_API_KEY` | 1 | OpenAI API key |
| `DEFAULT_SYSTEM_PROMPT` | 1 | Default SOUL for new sessions |
| `DISCORD_TOKEN` | 3 | Discord bot token |
| `ANTHROPIC_API_KEY` | 5 | Anthropic fallback |
| `GATEWAY_BIND` | 5 | Default: 127.0.0.1:8080 |
| `ENCRYPTION_KEY` | 5 | AES-256 key for secrets at rest |

## Logging

Use `log/slog` (stdlib). No external logging libraries (no zap, logrus, zerolog). Add `traceID` to every log line in the gateway loop.

## Code Rules

- No docstrings or comments in code
- Error handling: explicit `if err != nil`, return wrapped errors with `fmt.Errorf("...: %w", err)`
- Concurrency: prefer channels over shared mutexes where possible; use `golang.org/x/sync/semaphore` for per-tenant goroutine pools (week 3)
- All context-cancellable operations must accept and propagate `ctx context.Context`
- Graceful shutdown: signal handler → cancel root context → wait for goroutines

## Design Docs

- `docs/plans/2026-03-03-gopenclaw-design.md` — architecture, schema, interface specs
- `docs/plans/2026-03-03-gopenclaw-implementation.md` — week-by-week task breakdown
