package telegram

import (
	"testing"

	"github.com/spideynolove/gopenclaw/core"
)

func TestSessionID(t *testing.T) {
	sessionID := core.SessionID("tenant1", "telegram", 123456, 789012)
	expected := "tenant1:telegram:123456:789012"
	if sessionID != expected {
		t.Errorf("SessionID() = %q, want %q", sessionID, expected)
	}
}

func TestAdapterCreation(t *testing.T) {
	if adapter := (&Adapter{}); adapter == nil {
		t.Error("Adapter is nil")
	}
}
