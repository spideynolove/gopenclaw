package providers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/spideynolove/gopenclaw/core"
)

type Chain struct {
	providers []core.Provider
}

func NewChain(providers ...core.Provider) *Chain {
	return &Chain{
		providers: providers,
	}
}

func (c *Chain) Complete(ctx context.Context, session *core.Session) (string, error) {
	var lastErr error

	for i, provider := range c.providers {
		result, err := provider.Complete(ctx, session)
		if err == nil {
			return result, nil
		}

		lastErr = err
		isRetryable := isRetryableError(err)

		slog.Warn("provider failed, trying next",
			"provider_index", i,
			"error", err,
			"retryable", isRetryable)

		if !isRetryable {
			return "", fmt.Errorf("non-retryable error from provider %d: %w", i, err)
		}
	}

	return "", fmt.Errorf("all providers failed, last error: %w", lastErr)
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	patterns := []string{
		"429",
		"status 429",
		"500",
		"502",
		"503",
		"504",
		"timeout",
		"deadline",
		"connection refused",
		"i/o timeout",
		"retryable error",
	}

	for _, pattern := range patterns {
		if len(errStr) >= len(pattern) {
			for i := 0; i <= len(errStr)-len(pattern); i++ {
				if errStr[i:i+len(pattern)] == pattern {
					return true
				}
			}
		}
	}

	return false
}

func IsHTTPRetryable(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests ||
		statusCode == http.StatusInternalServerError ||
		statusCode == http.StatusBadGateway ||
		statusCode == http.StatusServiceUnavailable ||
		statusCode == http.StatusGatewayTimeout
}
