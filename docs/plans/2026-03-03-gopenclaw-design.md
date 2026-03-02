# GOpenClaw — 5-Week Implementation Design

**Date:** 2026-03-03
**Status:** Approved

## Context

GOpenClaw is a Go port of OpenClaw — a TypeScript personal AI assistant bot. It draws lessons from three existing implementations:

| Project | Lang | Key Lesson |
|---|---|---|
| OpenClaw | TypeScript | Full feature set, architecture reference |
| PicoClaw | Go | Goroutine-based concurrency, what NOT to strip |
| ZeroClaw | Rust | Trait/interface pattern, security defaults |

GOpenClaw's differentiators: multi-tenant, Postgres, clean interface architecture, prompt injection defense.

---

## Approach

**Thin interface layer + immediate implementation.**

Define only the interfaces needed for the current week. Implement concretely in parallel. No stubs, no deferred refactoring.

---

## Architecture

### Clean Architecture Layers

```
Domain (core/)
  types.go       — Message, Event, Session structs
  interfaces.go  — Provider, ChannelAdapter, SessionStore, MemoryBackend, ToolExecutor

Application (core/)
  gateway.go     — event loop, orchestrates interfaces only

Infrastructure (internal/)
  providers/openai/
  providers/anthropic/      (week 5 failover)
  channels/telegram/
  channels/discord/         (week 3)
  store/postgres/
  memory/postgres/          (week 2)
  tools/executor/           (week 4)

Composition root
  cmd/gopenclaw/main.go     — only place that wires concrete types to interfaces
```

Dependency rule: infrastructure depends on domain. Domain has zero external dependencies.

### Core Interfaces (evolution by week)

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

### Core Types

```go
type Message struct {
    Role    string
    Content string
}

type Event struct {
    SessionID string
    ChatID    int64
    UserID    int64
    Text      string
}

type Session struct {
    ID           string
    SystemPrompt string
    Messages     []Message
}

type Memory struct {
    ID        string
    TenantID  string
    AgentID   string
    Content   string
    Embedding []float32
    CreatedAt time.Time
}
```

### Gateway Loop

```
Event arrives (from any channel adapter)
  → Load session (SessionStore.Load)
  → Inject relevant memories (MemoryBackend.Search) [week 2+]
  → Append user message
  → Provider.Complete(session)
  → If tool_call in response → ToolExecutor.Execute [week 4+]
  → Append assistant message
  → SessionStore.Append
  → ChannelAdapter.Send reply
  → Check compaction threshold [week 2+]
```

---

## Database

**PostgreSQL + pgvector.**
Driver: `pgx/v5/stdlib` (registered into `database/sql`).
Query layer: `sqlx` (struct scanning, named queries, no ORM, no code generation).

### Selection Rationale

| Criterion | PostgreSQL |
|---|---|
| Relational (sessions, messages) | Native |
| Vector search (memory embeddings) | pgvector extension |
| Full-text search (BM25 hybrid) | tsvector |
| Multi-tenant isolation | Row-level security |
| ACID transactions | Native |
| Go driver | pgx (best-in-class) |
| Maturity | Excellent |

### Schema

```sql
-- Week 1
CREATE TABLE sessions (
    id            TEXT PRIMARY KEY,
    system_prompt TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE messages (
    id          BIGSERIAL PRIMARY KEY,
    session_id  TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    role        TEXT NOT NULL,
    content     TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ON messages (session_id, created_at);

-- Week 2
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE memories (
    id          BIGSERIAL PRIMARY KEY,
    tenant_id   TEXT NOT NULL,
    agent_id    TEXT NOT NULL,
    content     TEXT NOT NULL,
    embedding   vector(1536),
    tsv         TSVECTOR GENERATED ALWAYS AS (to_tsvector('english', content)) STORED,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ON memories USING ivfflat (embedding vector_cosine_ops);
CREATE INDEX ON memories USING GIN (tsv);

-- Week 3
CREATE TABLE tenants (
    id         TEXT PRIMARY KEY,
    api_key    TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE agents (
    id            TEXT PRIMARY KEY,
    tenant_id     TEXT NOT NULL REFERENCES tenants(id),
    system_prompt TEXT NOT NULL DEFAULT '',
    model         TEXT NOT NULL DEFAULT 'gpt-4o',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE sessions ADD COLUMN tenant_id TEXT REFERENCES tenants(id);
ALTER TABLE sessions ADD COLUMN agent_id  TEXT REFERENCES agents(id);
ALTER TABLE memories ADD COLUMN session_id TEXT REFERENCES sessions(id);
```

### Session ID Format

`{channel}:{chatID}:{userID}` — e.g. `telegram:123456:789012`

Encodes channel in prefix. Multi-tenant extension: `{tenantID}:{channel}:{chatID}:{userID}`.

---

## 5-Week Plan

### Week 1 — Foundation

**Goal:** Working bot end-to-end. Clean architecture established.

**Scope:**
- Project scaffold: go.mod, directory structure, all layers present
- `core/types.go` + `core/interfaces.go` (Provider, ChannelAdapter, SessionStore)
- `store/postgres`: sqlx + pgx driver, Load/Append/SetSystemPrompt
- `providers/openai`: direct HTTP to chat completions, non-streaming
- `channels/telegram`: go-telegram-bot-api/v5, Start/Send
- `core/gateway.go`: event loop
- `cmd/gopenclaw/main.go`: compose and start
- Config via env vars: `TELEGRAM_TOKEN`, `OPENAI_API_KEY`, `DATABASE_URL`
- Migration executed on startup
- Graceful shutdown via signal + context cancellation
- Structured logging with `log/slog`

**Session ID:** `telegram:{chatID}:{userID}`
**System prompt:** default from env var `DEFAULT_SYSTEM_PROMPT`

**Milestone:** Telegram DM → OpenAI reply → full session history in Postgres

**Dependencies:**
```
github.com/jackc/pgx/v5
github.com/jmoiron/sqlx
github.com/go-telegram-bot-api/telegram-bot-api/v5
```

---

### Week 2 — Memory System

**Goal:** Long-term memory that survives context window limits.

**Scope:**
- `MemoryBackend` interface added to core
- `memory/postgres`: Store, Search (hybrid cosine + BM25), Flush
- Embedding calls: OpenAI `text-embedding-3-small` via direct HTTP
- Hybrid search scoring: `0.7 * cosine_score + 0.3 * bm25_score`
- Compaction/prune: token counter → threshold → silent LLM flush turn → write to memories
- Relevant memories injected into prompt assembly before LLM call
- Token counting: tiktoken-compatible logic (no external lib, implement cl100k_base approximation)

**Compaction trigger:** `len(messages) * avg_tokens > 0.75 * context_window`

**Milestone:** Bot recalls facts from previous sessions; old context pruned without data loss

**New dependency:**
```
github.com/pgvector/pgvector-go
```

---

### Week 3 — Multi-Tenant + Discord

**Goal:** One binary serves multiple isolated tenants. Discord added as second channel.

**Scope:**
- `tenants` + `agents` tables; schema migration
- `TenantResolver` middleware: `Authorization: Bearer {api_key}` → tenantID
- All DB queries scoped by `tenant_id`
- Per-tenant goroutine pools: `golang.org/x/sync/semaphore` (default: 5 concurrent LLM calls per tenant)
- Webhook routing: `POST /webhook/{tenantID}/telegram`, `POST /webhook/{tenantID}/discord`
- `channels/discord`: discordgo library, same ChannelAdapter interface
- Per-tenant SOUL loaded from `agents` table
- Session ID extended: `{tenantID}:{channel}:{chatID}:{userID}`

**Milestone:** Two tenants running simultaneously, isolated DB rows, isolated goroutine pools, both Telegram and Discord working

**New dependencies:**
```
github.com/bwmarrin/discordgo
golang.org/x/sync
```

---

### Week 4 — Tool System

**Goal:** Bot executes actions, results feed back into conversation.

**Scope:**
- `ToolExecutor` interface + `ToolCall`, `ToolResult`, `Policy` types in core
- Tool policy chain: 5 levels (global → provider → agent → session → sandbox)
- Deny list always overrides allow list
- 2 built-in tools: `memory_write` (stores fact to MemoryBackend), `web_search` (HTTP fetch + summarize)
- Docker sandbox execution: `docker/docker` client SDK, per-tool container
- Tool call loop in gateway: LLM response → parse tool_calls → execute → inject results → re-call LLM
- Owner-only tool guard: verified sender check before execution

**Milestone:** Bot calls `web_search`, result injected into context, final answer includes fetched data

**New dependency:**
```
github.com/docker/docker/client
```

---

### Week 5 — Security + Production Hardening

**Goal:** Prompt injection defense, multi-provider failover, deployable.

**Scope:**

*Prompt injection defense:*
- Input sanitization layer: deny-list regex patterns applied before context assembly
- Role enforcement: context builder never places user content in `system` role
- SOUL immutability: system prompt always loaded from DB, never from user input
- Blast radius: `workspace_only=true` flag blocks system path access in tools

*Multi-provider failover:*
- `providers/anthropic`: Anthropic API implementation
- Provider priority chain: `openai → anthropic → (extensible)`
- Automatic retry with next provider on 429/500

*Security hardening:*
- AES-256-GCM encrypted secrets at rest (API keys in DB)
- Gateway binds to `127.0.0.1` by default; explicit flag to expose
- DM pairing: unknown senders blocked until 6-digit code exchange (per-tenant toggle)

*Observability:*
- Structured request tracing via `log/slog` with `traceID` on every log line
- `GET /health` endpoint

*Deployment:*
- `docker-compose.yml`: app + Postgres + pgvector
- `.env.example` with all required vars
- Single static binary via `CGO_ENABLED=0 GOOS=linux go build`

**Milestone:** Prompt injection attempt blocked at sanitization layer; provider failover verified; binary deploys via docker-compose

---

## Weekly Interface Summary

| Week | New Interfaces | Concrete Implementations |
|---|---|---|
| 1 | Provider, ChannelAdapter, SessionStore | openai, telegram, postgres |
| 2 | MemoryBackend | postgres memory |
| 3 | — | discord, tenant middleware |
| 4 | ToolExecutor | docker sandbox, web_search, memory_write |
| 5 | — | anthropic, failover chain |

---

## Configuration (env vars)

| Var | Week | Purpose |
|---|---|---|
| `DATABASE_URL` | 1 | Postgres connection string |
| `TELEGRAM_TOKEN` | 1 | Telegram bot token |
| `OPENAI_API_KEY` | 1 | OpenAI API key |
| `DEFAULT_SYSTEM_PROMPT` | 1 | Default SOUL for new sessions |
| `DISCORD_TOKEN` | 3 | Discord bot token |
| `ANTHROPIC_API_KEY` | 5 | Anthropic fallback |
| `GATEWAY_BIND` | 5 | Default: 127.0.0.1:8080 |
| `ENCRYPTION_KEY` | 5 | AES-256 key for secrets at rest |
