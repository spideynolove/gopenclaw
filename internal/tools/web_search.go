package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func WebSearchTool(apiKey string) func(context.Context, map[string]any) (string, error) {
	return func(ctx context.Context, args map[string]any) (string, error) {
		query, ok := args["query"].(string)
		if !ok || query == "" {
			return "", fmt.Errorf("query arg required")
		}
		u := "https://api.duckduckgo.com/?q=" + url.QueryEscape(query) + "&format=json&no_html=1"
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return string(body), nil
	}
}
