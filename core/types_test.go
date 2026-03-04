package core_test

import (
	"context"
	"testing"

	"github.com/spideynolove/gopenclaw/core"
)

func TestSessionID(t *testing.T) {
	id := core.SessionID("tenant1", "telegram", 123456, 789012)
	want := "tenant1:telegram:123456:789012"
	if id != want {
		t.Errorf("got %q want %q", id, want)
	}
}

func TestSessionMessages(t *testing.T) {
	s := &core.Session{
		ID:           "telegram:1:2",
		SystemPrompt: "You are helpful.",
		Messages:     []core.Message{{Role: "user", Content: "hi"}},
	}
	if len(s.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(s.Messages))
	}
	if s.Messages[0].Role != "user" {
		t.Errorf("expected role user, got %q", s.Messages[0].Role)
	}
}

func TestProviderInterface(t *testing.T) {
	var _ core.Provider = (*testProvider)(nil)
}

type testProvider struct{}

func (tp *testProvider) Complete(ctx context.Context, session *core.Session) (string, error) {
	return "response", nil
}

func TestChannelAdapterInterface(t *testing.T) {
	var _ core.ChannelAdapter = (*testChannelAdapter)(nil)
}

type testChannelAdapter struct{}

func (tca *testChannelAdapter) Start(ctx context.Context, out chan<- core.Event) error {
	return nil
}

func (tca *testChannelAdapter) Send(ctx context.Context, evt core.Event, text string) error {
	return nil
}

func TestSessionStoreInterface(t *testing.T) {
	var _ core.SessionStore = (*testSessionStore)(nil)
}

type testSessionStore struct{}

func (tss *testSessionStore) Load(ctx context.Context, sessionID string) (*core.Session, error) {
	return &core.Session{}, nil
}

func (tss *testSessionStore) Append(ctx context.Context, sessionID string, msg core.Message) error {
	return nil
}

func (tss *testSessionStore) SetSystemPrompt(ctx context.Context, sessionID, prompt string) error {
	return nil
}
