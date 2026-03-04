package providers

import (
	"context"
	"fmt"
	"testing"

	"github.com/spideynolove/gopenclaw/core"
)

type mockProvider struct {
	shouldFail bool
	failErr    string
	result     string
}

func (m *mockProvider) Complete(ctx context.Context, session *core.Session) (string, error) {
	if m.shouldFail {
		return "", fmt.Errorf("%s", m.failErr)
	}
	return m.result, nil
}

func TestChainUsesFirstProvider(t *testing.T) {
	p1 := &mockProvider{shouldFail: false, result: "success"}
	chain := NewChain(p1)

	session := &core.Session{}
	result, err := chain.Complete(context.Background(), session)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result != "success" {
		t.Fatalf("expected 'success', got %q", result)
	}
}

func TestChainFailsOverToSecond(t *testing.T) {
	p1 := &mockProvider{shouldFail: true, failErr: "status 429 too many requests"}
	p2 := &mockProvider{shouldFail: false, result: "fallback"}

	chain := NewChain(p1, p2)

	session := &core.Session{}
	result, err := chain.Complete(context.Background(), session)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result != "fallback" {
		t.Fatalf("expected 'fallback', got %q", result)
	}
}

func TestChainReturnsErrorWhenAllFail(t *testing.T) {
	p1 := &mockProvider{shouldFail: true, failErr: "status 429 too many requests"}
	p2 := &mockProvider{shouldFail: true, failErr: "status 500 internal server error"}

	chain := NewChain(p1, p2)

	session := &core.Session{}
	_, err := chain.Complete(context.Background(), session)

	if err == nil {
		t.Fatal("expected error when all providers fail")
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"429 status", fmt.Errorf("status 429"), true},
		{"500 status", fmt.Errorf("status 500"), true},
		{"503 status", fmt.Errorf("status 503"), true},
		{"timeout", fmt.Errorf("context deadline exceeded"), true},
		{"non-retryable", fmt.Errorf("invalid api key"), false},
		{"nil error", nil, false},
	}

	for _, test := range tests {
		result := isRetryableError(test.err)
		if result != test.expected {
			t.Fatalf("%s: expected %v, got %v", test.name, test.expected, result)
		}
	}
}

func TestIsHTTPRetryable(t *testing.T) {
	tests := []struct {
		code     int
		expected bool
	}{
		{429, true},
		{500, true},
		{502, true},
		{503, true},
		{504, true},
		{400, false},
		{401, false},
		{200, false},
	}

	for _, test := range tests {
		result := IsHTTPRetryable(test.code)
		if result != test.expected {
			t.Fatalf("code %d: expected %v, got %v", test.code, test.expected, result)
		}
	}
}
