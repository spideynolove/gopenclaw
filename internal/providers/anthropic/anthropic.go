package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spideynolove/gopenclaw/core"
)

type Provider struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

func New(apiKey, model string) *Provider {
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}
	return &Provider{
		apiKey:     apiKey,
		model:      model,
		baseURL:    "https://api.anthropic.com",
		httpClient: &http.Client{},
	}
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CompletionRequest struct {
	Model       string    `json:"model"`
	MaxTokens   int       `json:"max_tokens"`
	System      string    `json:"system,omitempty"`
	Messages    []Message `json:"messages"`
}

type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type CompletionResponse struct {
	Content []ContentBlock `json:"content"`
}

func (p *Provider) Complete(ctx context.Context, session *core.Session) (string, error) {
	messages := []Message{}

	for _, msg := range session.Messages {
		messages = append(messages, Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	reqBody := CompletionRequest{
		Model:     p.model,
		MaxTokens: 2048,
		System:    session.SystemPrompt,
		Messages:  messages,
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/messages", bytes.NewReader(reqBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError {
		return "", fmt.Errorf("api retryable error: status %d", resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("api error: status %d, body: %s", resp.StatusCode, string(respBytes))
	}

	var respBody CompletionResponse
	if err := json.Unmarshal(respBytes, &respBody); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if len(respBody.Content) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	return respBody.Content[0].Text, nil
}
