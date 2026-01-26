package opencode

import (
	"testing"

	acp "github.com/ironpark/acp-go"
)

func TestAgentMessageAggregator_BasicAggregation(t *testing.T) {
	aggregator := NewAgentMessageAggregator()

	// Send first chunk without newline
	update1 := acp.NewSessionUpdateAgentMessageChunk(acp.NewContentBlockText("Hello"))
	got1 := aggregator.ProcessUpdate(&update1)
	if got1 != "" {
		t.Fatalf("expected empty output for incomplete chunk, got: %q", got1)
	}

	// Send second chunk with newline - should trigger output
	update2 := acp.NewSessionUpdateAgentMessageChunk(acp.NewContentBlockText(" world\n"))
	got2 := aggregator.ProcessUpdate(&update2)
	expected := "agent_message \"Hello world\\n\""
	if got2 != expected {
		t.Fatalf("expected aggregated message: %q, got: %q", expected, got2)
	}
}

func TestAgentMessageAggregator_MultipleChunksBeforeNewline(t *testing.T) {
	aggregator := NewAgentMessageAggregator()

	// Send multiple chunks before newline
	update1 := acp.NewSessionUpdateAgentMessageChunk(acp.NewContentBlockText("Chunk1"))
	got1 := aggregator.ProcessUpdate(&update1)
	if got1 != "" {
		t.Fatalf("expected empty output for first chunk, got: %q", got1)
	}

	update2 := acp.NewSessionUpdateAgentMessageChunk(acp.NewContentBlockText(" Chunk2"))
	got2 := aggregator.ProcessUpdate(&update2)
	if got2 != "" {
		t.Fatalf("expected empty output for second chunk, got: %q", got2)
	}

	update3 := acp.NewSessionUpdateAgentMessageChunk(acp.NewContentBlockText(" Chunk3\n"))
	got3 := aggregator.ProcessUpdate(&update3)
	expected := "agent_message \"Chunk1 Chunk2 Chunk3\\n\""
	if got3 != expected {
		t.Fatalf("expected aggregated message: %q, got: %q", expected, got3)
	}
}

func TestAgentMessageAggregator_MultipleMessages(t *testing.T) {
	aggregator := NewAgentMessageAggregator()

	// First complete message
	update1 := acp.NewSessionUpdateAgentMessageChunk(acp.NewContentBlockText("First message\n"))
	got1 := aggregator.ProcessUpdate(&update1)
	expected1 := "agent_message \"First message\\n\""
	if got1 != expected1 {
		t.Fatalf("expected first message: %q, got: %q", expected1, got1)
	}

	// Second complete message
	update2 := acp.NewSessionUpdateAgentMessageChunk(acp.NewContentBlockText("Second message\n"))
	got2 := aggregator.ProcessUpdate(&update2)
	expected2 := "agent_message \"Second message\\n\""
	if got2 != expected2 {
		t.Fatalf("expected second message: %q, got: %q", expected2, got2)
	}
}

func TestAgentMessageAggregator_PreservesExactText(t *testing.T) {
	aggregator := NewAgentMessageAggregator()

	// Test with special characters and formatting
	update := acp.NewSessionUpdateAgentMessageChunk(acp.NewContentBlockText("Line 1\nLine 2\nLine 3\n"))
	got := aggregator.ProcessUpdate(&update)
	expected := "agent_message \"Line 1\\nLine 2\\nLine 3\\n\""
	if got != expected {
		t.Fatalf("expected exact text preservation: %q, got: %q", expected, got)
	}
}

func TestAgentMessageAggregator_NonAgentMessageChunks(t *testing.T) {
	aggregator := NewAgentMessageAggregator()

	// Test that non-agent-message chunks are not affected
	toolUpdate := acp.NewSessionUpdateToolCall(
		acp.ToolCallId("tool-1"),
		"Read file",
		acp.ToolKindPtr(acp.ToolKindRead),
		acp.ToolCallStatusPtr(acp.ToolCallStatusPending),
		nil,
		nil,
	)

	got := aggregator.ProcessUpdate(&toolUpdate)
	expected := "⏳ \x1b[33mtool_call\x1b[0m id=tool-1 title=\"Read file\" kind=read status=pending"
	if got != expected {
		t.Fatalf("expected tool call to have status badge: %q, got: %q", expected, got)
	}
}
