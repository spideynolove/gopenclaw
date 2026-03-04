package core

import (
	"fmt"
	"time"
)

type Message struct {
	Role    string
	Content string
}

type Memory struct {
	ID        int64
	SessionID string
	Content   string
	Embedding []float32
	CreatedAt time.Time
}

type Event struct {
	SessionID string
	ChatID    int64
	UserID    int64
	Text      string
	TenantID  string
}

type Session struct {
	ID           string
	TenantID     string
	AgentID      string
	SystemPrompt string
	Messages     []Message
}

type ToolCall struct {
	ID   string
	Name string
	Args map[string]any
}

type ToolResult struct {
	CallID  string
	Content string
	IsError bool
}

type Policy struct {
	Allow []string
	Deny  []string
}

func SessionID(tenantID, channel string, chatID, userID int64) string {
	return fmt.Sprintf("%s:%s:%d:%d", tenantID, channel, chatID, userID)
}
