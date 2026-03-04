package tools

import (
	"context"
	"fmt"

	"github.com/spideynolove/gopenclaw/core"
)

type Executor struct {
	chain PolicyChain
	tools map[string]func(context.Context, map[string]any) (string, error)
}

func NewExecutor(chain PolicyChain) *Executor {
	e := &Executor{
		chain: chain,
		tools: make(map[string]func(context.Context, map[string]any) (string, error)),
	}
	return e
}

func (e *Executor) Register(name string, fn func(context.Context, map[string]any) (string, error)) {
	e.tools[name] = fn
}

func (e *Executor) Execute(ctx context.Context, call core.ToolCall, policy core.Policy) (core.ToolResult, error) {
	if !e.chain.Permitted(call.Name) {
		return core.ToolResult{CallID: call.ID, Content: "tool not permitted", IsError: true}, nil
	}
	fn, ok := e.tools[call.Name]
	if !ok {
		return core.ToolResult{}, fmt.Errorf("unknown tool: %s", call.Name)
	}
	content, err := fn(ctx, call.Args)
	if err != nil {
		return core.ToolResult{CallID: call.ID, Content: err.Error(), IsError: true}, nil
	}
	return core.ToolResult{CallID: call.ID, Content: content}, nil
}
