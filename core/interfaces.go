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

type MemoryBackend interface {
	Search(ctx context.Context, sessionID, query string, embedding []float32) ([]Memory, error)
	Store(ctx context.Context, m Memory) error
	Flush(ctx context.Context, sessionID string) error
	FlushSession(ctx context.Context, sessionID string) error
}
