package core_test

import (
	"testing"

	"github.com/spideynolove/gopenclaw/core"
)

func TestTokenCount(t *testing.T) {
	msgs := []core.Message{
		{Role: "user", Content: "hello world"},
		{Role: "assistant", Content: "hi there, how can I help you today"},
	}
	count := core.EstimateTokens(msgs)
	if count <= 0 {
		t.Errorf("expected positive token count, got %d", count)
	}
}

func TestNeedsCompaction(t *testing.T) {
	var msgs []core.Message
	for i := 0; i < 300; i++ {
		msgs = append(msgs, core.Message{Role: "user", Content: "this is a message with some content in it"})
	}
	if !core.NeedsCompaction(msgs, 4096) {
		t.Error("expected compaction to be needed for 300 long messages with 4096 context window")
	}
}
