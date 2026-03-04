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
}

func NewGateway(provider Provider, channel ChannelAdapter, store SessionStore) *Gateway {
	return &Gateway{
		provider: provider,
		channel:  channel,
		store:    store,
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
}
