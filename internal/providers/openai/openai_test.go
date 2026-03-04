package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spideynolove/gopenclaw/core"
)

func TestComplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-api-key" {
			t.Errorf("expected Authorization header 'Bearer test-api-key', got '%s'", auth)
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected Content-Type 'application/json', got '%s'", contentType)
		}

		var req CompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if req.Model != "gpt-4" {
			t.Errorf("expected model 'gpt-4', got '%s'", req.Model)
		}

		if len(req.Messages) != 3 {
			t.Errorf("expected 3 messages, got %d", len(req.Messages))
		}

		if req.Messages[0].Role != "system" {
			t.Errorf("expected first message role 'system', got '%s'", req.Messages[0].Role)
		}

		if req.Messages[0].Content != "You are helpful" {
			t.Errorf("expected first message content 'You are helpful', got '%s'", req.Messages[0].Content)
		}

		resp := CompletionResponse{
			Choices: []Choice{
				{
					Message: Message{
						Role:    "assistant",
						Content: "Hello, this is a test response",
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := New("test-api-key", "gpt-4", server.URL)

	session := &core.Session{
		ID:           "test:123:456",
		SystemPrompt: "You are helpful",
		Messages: []core.Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there"},
		},
	}

	result, err := provider.Complete(context.Background(), session)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	expected := "Hello, this is a test response"
	if result != expected {
		t.Errorf("expected response '%s', got '%s'", expected, result)
	}
}

func TestCompleteWithoutSystemPrompt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req CompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if len(req.Messages) != 1 {
			t.Errorf("expected 1 message, got %d", len(req.Messages))
		}

		if req.Messages[0].Role != "user" {
			t.Errorf("expected first message role 'user', got '%s'", req.Messages[0].Role)
		}

		resp := CompletionResponse{
			Choices: []Choice{
				{
					Message: Message{
						Role:    "assistant",
						Content: "Response without system prompt",
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := New("test-api-key", "gpt-4", server.URL)

	session := &core.Session{
		ID:           "test:123:456",
		SystemPrompt: "",
		Messages: []core.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	result, err := provider.Complete(context.Background(), session)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	expected := "Response without system prompt"
	if result != expected {
		t.Errorf("expected response '%s', got '%s'", expected, result)
	}
}

func TestDefaultBaseURL(t *testing.T) {
	provider := New("test-api-key", "gpt-4", "")
	if provider.baseURL != "https://api.openai.com" {
		t.Errorf("expected default baseURL 'https://api.openai.com', got '%s'", provider.baseURL)
	}
}

func TestCustomBaseURL(t *testing.T) {
	customURL := "https://custom.example.com"
	provider := New("test-api-key", "gpt-4", customURL)
	if provider.baseURL != customURL {
		t.Errorf("expected baseURL '%s', got '%s'", customURL, provider.baseURL)
	}
}
