package opencode

import (
	"testing"

	acp "github.com/ironpark/acp-go"
)

// Additional edge case tests to ensure robustness
func TestAgentMessageAggregator_EdgeCases(t *testing.T) {
	t.Run("empty chunks", func(t *testing.T) {
		aggregator := NewAgentMessageAggregator()
		update := acp.NewSessionUpdateAgentMessageChunk(acp.NewContentBlockText(""))
		got := aggregator.ProcessUpdate(&update)
		if got != "" {
			t.Fatalf("expected empty output for empty chunk, got: %q", got)
		}
	})

	t.Run("multiple newlines in one chunk", func(t *testing.T) {
		aggregator := NewAgentMessageAggregator()
		update := acp.NewSessionUpdateAgentMessageChunk(acp.NewContentBlockText("Line1\nLine2\n"))
		got := aggregator.ProcessUpdate(&update)
		expected := "agent_message \"Line1\\nLine2\\n\""
		if got != expected {
			t.Fatalf("expected: %q, got: %q", expected, got)
		}
	})

	t.Run("trailing content without newline", func(t *testing.T) {
		aggregator := NewAgentMessageAggregator()

		// Send complete message
		update1 := acp.NewSessionUpdateAgentMessageChunk(acp.NewContentBlockText("Complete\n"))
		got1 := aggregator.ProcessUpdate(&update1)
		expected1 := "agent_message \"Complete\\n\""
		if got1 != expected1 {
			t.Fatalf("expected first message: %q, got: %q", expected1, got1)
		}

		// Send incomplete message (no newline)
		update2 := acp.NewSessionUpdateAgentMessageChunk(acp.NewContentBlockText("Incomplete"))
		got2 := aggregator.ProcessUpdate(&update2)
		if got2 != "" {
			t.Fatalf("expected no output for incomplete trailing message, got: %q", got2)
		}
	})

	t.Run("nil content block", func(t *testing.T) {
		aggregator := NewAgentMessageAggregator()
		// This test ensures we handle nil content blocks gracefully
		// (though in practice this shouldn't happen with the ACP library)
		// We can't easily create a nil content block with the ACP library,
		// so we'll test the processMessageChunk method directly
		got := aggregator.processMessageChunk(nil)
		if got != "" {
			t.Fatalf("expected empty output for nil content, got: %q", got)
		}
	})
}
