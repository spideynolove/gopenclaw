package core

import "context"

type Provider interface {
	Complete(ctx context.Context, session *Session) (string, error)
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
