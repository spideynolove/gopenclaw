package tools_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/spideynolove/gopenclaw/core"
	"github.com/spideynolove/gopenclaw/internal/tools"
)

type mockMemory struct {
	stored []core.Memory
}

func (m *mockMemory) Search(ctx context.Context, sessionID, query string, embedding []float32) ([]core.Memory, error) {
	return nil, nil
}

func (m *mockMemory) Store(ctx context.Context, mem core.Memory) error {
	m.stored = append(m.stored, mem)
	return nil
}

func (m *mockMemory) Flush(ctx context.Context, sessionID string) error {
	return nil
}

func (m *mockMemory) FlushSession(ctx context.Context, sessionID string) error {
	return nil
}

func TestExecutorRegisterAndExecute(t *testing.T) {
	chain := tools.PolicyChain{
		Global: core.Policy{Allow: []string{"test_tool"}},
	}
	executor := tools.NewExecutor(chain)

	called := false
	executor.Register("test_tool", func(ctx context.Context, args map[string]any) (string, error) {
		called = true
		return "success", nil
	})

	result, err := executor.Execute(context.Background(), core.ToolCall{
		ID:   "call1",
		Name: "test_tool",
		Args: map[string]any{},
	}, core.Policy{})

	if err != nil {
		t.Errorf("execute failed: %v", err)
	}
	if !called {
		t.Error("tool function was not called")
	}
	if result.Content != "success" {
		t.Errorf("expected success, got %s", result.Content)
	}
	if result.IsError {
		t.Error("result should not be marked as error")
	}
	if result.CallID != "call1" {
		t.Errorf("expected call ID call1, got %s", result.CallID)
	}
}

func TestExecutorRejectUnpermittedTool(t *testing.T) {
	chain := tools.PolicyChain{
		Global: core.Policy{Allow: []string{"allowed_tool"}},
	}
	executor := tools.NewExecutor(chain)

	executor.Register("denied_tool", func(ctx context.Context, args map[string]any) (string, error) {
		return "should not be called", nil
	})

	result, err := executor.Execute(context.Background(), core.ToolCall{
		ID:   "call1",
		Name: "denied_tool",
		Args: map[string]any{},
	}, core.Policy{})

	if err != nil {
		t.Errorf("execute should not error on permission denial: %v", err)
	}
	if !result.IsError {
		t.Error("result should be marked as error")
	}
	if result.Content != "tool not permitted" {
		t.Errorf("expected 'tool not permitted', got %s", result.Content)
	}
}

func TestExecutorUnknownTool(t *testing.T) {
	chain := tools.PolicyChain{
		Global: core.Policy{Allow: []string{"unknown_tool"}},
	}
	executor := tools.NewExecutor(chain)

	_, err := executor.Execute(context.Background(), core.ToolCall{
		ID:   "call1",
		Name: "unknown_tool",
		Args: map[string]any{},
	}, core.Policy{})

	if err == nil {
		t.Error("execute should error on unknown tool")
	}
}

func TestExecutorToolError(t *testing.T) {
	chain := tools.PolicyChain{
		Global: core.Policy{Allow: []string{"error_tool"}},
	}
	executor := tools.NewExecutor(chain)

	executor.Register("error_tool", func(ctx context.Context, args map[string]any) (string, error) {
		return "", fmt.Errorf("tool failed")
	})

	result, err := executor.Execute(context.Background(), core.ToolCall{
		ID:   "call1",
		Name: "error_tool",
		Args: map[string]any{},
	}, core.Policy{})

	if err != nil {
		t.Errorf("execute should not error, error should be in result: %v", err)
	}
	if !result.IsError {
		t.Error("result should be marked as error")
	}
	if result.Content != "tool failed" {
		t.Errorf("expected 'tool failed', got %s", result.Content)
	}
}

func TestMemoryWriteTool(t *testing.T) {
	mem := &mockMemory{}
	embed := func(ctx context.Context, text string) ([]float32, error) {
		return []float32{0.1, 0.2, 0.3}, nil
	}

	toolFn := tools.MemoryWriteTool(mem, embed)
	result, err := toolFn(context.Background(), map[string]any{
		"content":    "test fact",
		"session_id": "session123",
	})

	if err != nil {
		t.Errorf("memory write failed: %v", err)
	}
	if result != "memory stored" {
		t.Errorf("expected 'memory stored', got %s", result)
	}
	if len(mem.stored) != 1 {
		t.Errorf("expected 1 stored memory, got %d", len(mem.stored))
	}
	if mem.stored[0].Content != "test fact" {
		t.Errorf("expected 'test fact', got %s", mem.stored[0].Content)
	}
	if mem.stored[0].SessionID != "session123" {
		t.Errorf("expected 'session123', got %s", mem.stored[0].SessionID)
	}
}

func TestMemoryWriteToolMissingContent(t *testing.T) {
	mem := &mockMemory{}
	embed := func(ctx context.Context, text string) ([]float32, error) {
		return []float32{}, nil
	}

	toolFn := tools.MemoryWriteTool(mem, embed)
	_, err := toolFn(context.Background(), map[string]any{
		"session_id": "session123",
	})

	if err == nil {
		t.Error("memory write should error without content")
	}
}

func TestWebSearchTool(t *testing.T) {
	toolFn := tools.WebSearchTool("")
	result, err := toolFn(context.Background(), map[string]any{
		"query": "golang",
	})

	if err != nil {
		t.Errorf("web search failed: %v", err)
	}
	if result == "" {
		t.Error("web search should return non-empty result")
	}
}

func TestWebSearchToolMissingQuery(t *testing.T) {
	toolFn := tools.WebSearchTool("")
	_, err := toolFn(context.Background(), map[string]any{})

	if err == nil {
		t.Error("web search should error without query")
	}
}

func TestWebSearchToolEmptyQuery(t *testing.T) {
	toolFn := tools.WebSearchTool("")
	_, err := toolFn(context.Background(), map[string]any{
		"query": "",
	})

	if err == nil {
		t.Error("web search should error with empty query")
	}
}
