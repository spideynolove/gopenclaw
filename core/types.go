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
}

type Session struct {
	ID           string
	SystemPrompt string
	Messages     []Message
}

func SessionID(channel string, chatID, userID int64) string {
	return fmt.Sprintf("%s:%d:%d", channel, chatID, userID)
}
