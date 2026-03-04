package core

import (
	"context"
	"testing"
)

type mockStore struct {
	sessions map[string]*Session
	messages map[string][]Message
}

func newMockStore() *mockStore {
	return &mockStore{
		sessions: make(map[string]*Session),
		messages: make(map[string][]Message),
	}
}

func (m *mockStore) Load(ctx context.Context, sessionID string) (*Session, error) {
	if s, ok := m.sessions[sessionID]; ok {
		return &Session{
			ID:           s.ID,
			SystemPrompt: s.SystemPrompt,
			Messages:     append([]Message{}, s.Messages...),
		}, nil
	}
	return &Session{
		ID:       sessionID,
		Messages: []Message{},
	}, nil
}

func (m *mockStore) Append(ctx context.Context, sessionID string, msg Message) error {
	m.messages[sessionID] = append(m.messages[sessionID], msg)
	if s, ok := m.sessions[sessionID]; ok {
		s.Messages = append(s.Messages, msg)
	} else {
		m.sessions[sessionID] = &Session{
			ID:       sessionID,
			Messages: []Message{msg},
		}
	}
	return nil
}

func (m *mockStore) SetSystemPrompt(ctx context.Context, sessionID, prompt string) error {
	if s, ok := m.sessions[sessionID]; ok {
		s.SystemPrompt = prompt
	} else {
		m.sessions[sessionID] = &Session{
			ID:           sessionID,
			SystemPrompt: prompt,
			Messages:     []Message{},
		}
	}
	return nil
}

type mockProvider struct {
	responses map[string]string
	failWith  error
}

func newMockProvider() *mockProvider {
	return &mockProvider{
		responses: map[string]string{
			"ping": "pong",
		},
	}
}

func (m *mockProvider) Complete(ctx context.Context, session *Session) (string, error) {
	if m.failWith != nil {
		return "", m.failWith
	}
	if len(session.Messages) > 0 {
		lastMsg := session.Messages[len(session.Messages)-1]
		if resp, ok := m.responses[lastMsg.Content]; ok {
			return resp, nil
		}
	}
	return "default response", nil
}

type mockAdapter struct {
	sendCalls []struct {
		evt  Event
		text string
	}
	startErr error
	sendErr  error
	eventsChan chan Event
}

func newMockAdapter() *mockAdapter {
	return &mockAdapter{
		sendCalls: []struct {
			evt  Event
			text string
		}{},
	}
}

func (m *mockAdapter) Start(ctx context.Context, out chan<- Event) error {
	if m.startErr != nil {
		return m.startErr
	}
	eventsChan := make(chan Event)
	m.eventsChan = eventsChan
	go func() {
		for evt := range eventsChan {
			out <- evt
		}
	}()
	return nil
}

func (m *mockAdapter) Send(ctx context.Context, evt Event, text string) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	m.sendCalls = append(m.sendCalls, struct {
		evt  Event
		text string
	}{evt, text})
	return nil
}

func TestGatewayRoutesSingleMessage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := newMockStore()
	provider := newMockProvider()
	adapter := newMockAdapter()

	gateway := NewGateway(provider, adapter, store)

	go func() {
		gateway.Run(ctx)
	}()

	adapter.eventsChan <- Event{
		SessionID: "test:123:456",
		ChatID:    123,
		UserID:    456,
		Text:      "ping",
	}

	cancel()

	if len(adapter.sendCalls) != 1 {
		t.Fatalf("expected 1 send call, got %d", len(adapter.sendCalls))
	}

	if adapter.sendCalls[0].text != "pong" {
		t.Errorf("expected reply 'pong', got '%s'", adapter.sendCalls[0].text)
	}
}

func TestGatewayPersistsMessages(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := newMockStore()
	provider := newMockProvider()
	adapter := newMockAdapter()

	gateway := NewGateway(provider, adapter, store)

	go func() {
		gateway.Run(ctx)
	}()

	sessionID := "test:123:456"
	adapter.events <- Event{
		SessionID: sessionID,
		ChatID:    123,
		UserID:    456,
		Text:      "ping",
	}

	cancel()

	if len(store.messages[sessionID]) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(store.messages[sessionID]))
	}

	if store.messages[sessionID][0].Role != "user" || store.messages[sessionID][0].Content != "ping" {
		t.Errorf("first message should be user 'ping', got %v", store.messages[sessionID][0])
	}

	if store.messages[sessionID][1].Role != "assistant" || store.messages[sessionID][1].Content != "pong" {
		t.Errorf("second message should be assistant 'pong', got %v", store.messages[sessionID][1])
	}
}
