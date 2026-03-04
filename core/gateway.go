package core

import (
	"context"
	"fmt"
	"log/slog"
)

type Gateway struct {
	provider Provider
	channel  ChannelAdapter
	store    SessionStore
	memory   MemoryBackend
}

func NewGateway(provider Provider, channel ChannelAdapter, store SessionStore) *Gateway {
	return &Gateway{
		provider: provider,
		channel:  channel,
		store:    store,
		memory:   nil,
	}
}

func NewGatewayWithMemory(provider Provider, channel ChannelAdapter, store SessionStore, memory MemoryBackend) *Gateway {
	return &Gateway{
		provider: provider,
		channel:  channel,
		store:    store,
		memory:   memory,
	}
}

func (g *Gateway) Run(ctx context.Context) error {
	events := make(chan Event, 32)

	if err := g.channel.Start(ctx, events); err != nil {
		return fmt.Errorf("channel start: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case evt, ok := <-events:
			if !ok {
				return nil
			}
			g.handle(ctx, evt)
		}
	}
}

func (g *Gateway) handle(ctx context.Context, evt Event) {
	session, err := g.store.Load(ctx, evt.SessionID)
	if err != nil {
		slog.Error("failed to load session", "sessionID", evt.SessionID, "err", err)
		return
	}

	userMsg := Message{Role: "user", Content: evt.Text}
	if err := g.store.Append(ctx, evt.SessionID, userMsg); err != nil {
		slog.Error("failed to append user message", "sessionID", evt.SessionID, "err", err)
		return
	}
	session.Messages = append(session.Messages, userMsg)

	reply, err := g.provider.Complete(ctx, session)
	if err != nil {
		slog.Error("failed to complete", "sessionID", evt.SessionID, "err", err)
		return
	}

	assistantMsg := Message{Role: "assistant", Content: reply}
	if err := g.store.Append(ctx, evt.SessionID, assistantMsg); err != nil {
		slog.Error("failed to append assistant message", "sessionID", evt.SessionID, "err", err)
		return
	}

	if err := g.channel.Send(ctx, evt, reply); err != nil {
		slog.Error("failed to send reply", "sessionID", evt.SessionID, "err", err)
		return
	}

	if NeedsCompaction(session.Messages, 128000) {
		g.compact(ctx, evt.SessionID, session)
	}
}

func (g *Gateway) compact(ctx context.Context, sessionID string, session *Session) {
	if g.memory == nil {
		return
	}

	systemPrompt := session.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = "You are a helpful assistant. Summarize the key facts from the conversation."
	}

	compactSession := &Session{
		ID:           sessionID,
		SystemPrompt: systemPrompt + "\n\nSummarize the key facts from this conversation in 1-2 sentences.",
		Messages:     session.Messages,
	}

	summary, err := g.provider.Complete(ctx, compactSession)
	if err != nil {
		slog.Error("compaction: provider complete failed", "sessionID", sessionID, "err", err)
		return
	}

	mem := Memory{
		SessionID: sessionID,
		Content:   summary,
		Embedding: nil,
	}
	if err := g.memory.Store(ctx, mem); err != nil {
		slog.Error("compaction: store memory failed", "sessionID", sessionID, "err", err)
		return
	}

	if err := g.memory.Flush(ctx, sessionID); err != nil {
		slog.Error("compaction: flush memory failed", "sessionID", sessionID, "err", err)
	}
}
