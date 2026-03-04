package tools

import (
	"context"
	"fmt"

	"github.com/spideynolove/gopenclaw/core"
)

func MemoryWriteTool(backend core.MemoryBackend, embed func(context.Context, string) ([]float32, error)) func(context.Context, map[string]any) (string, error) {
	return func(ctx context.Context, args map[string]any) (string, error) {
		content, ok := args["content"].(string)
		if !ok || content == "" {
			return "", fmt.Errorf("content arg required")
		}
		sessionID, _ := args["session_id"].(string)
		vec, err := embed(ctx, content)
		if err != nil {
			return "", err
		}
		err = backend.Store(ctx, core.Memory{
			SessionID: sessionID,
			Content:   content,
			Embedding: vec,
		})
		if err != nil {
			return "", err
		}
		return "memory stored", nil
	}
}
