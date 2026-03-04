package openai

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
	apiKey    string
	model     string
	baseURL   string
	httpClient *http.Client
}

func New(apiKey, model, baseURL string) *Provider {
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	return &Provider{
		apiKey:     apiKey,
		model:      model,
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CompletionRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Choice struct {
	Message Message `json:"message"`
}

type CompletionResponse struct {
	Choices []Choice `json:"choices"`
}

func (p *Provider) Complete(ctx context.Context, session *core.Session) (string, error) {
	messages := []Message{}

	if session.SystemPrompt != "" {
		messages = append(messages, Message{
			Role:    "system",
			Content: session.SystemPrompt,
		})
	}

	for _, msg := range session.Messages {
		messages = append(messages, Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	reqBody := CompletionRequest{
		Model:    p.model,
		Messages: messages,
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/chat/completions", bytes.NewReader(reqBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
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

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("api error: status %d, body: %s", resp.StatusCode, string(respBytes))
	}

	var respBody CompletionResponse
	if err := json.Unmarshal(respBytes, &respBody); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if len(respBody.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return respBody.Choices[0].Message.Content, nil
}
