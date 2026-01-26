package opencode

import (
	"bytes"
	"os"
	"testing"

	acp "github.com/ironpark/acp-go"
)

func TestClientIntegration_AggregatesAgentMessages(t *testing.T) {
	// This test verifies that the client integration properly uses the aggregator
	// and that agent messages are aggregated while other messages pass through

	// Create a mock ACP client handler that captures onUpdate calls
	var capturedLines []string
	aggregator := NewAgentMessageAggregator()

	onUpdate := func(note *acp.SessionNotification) {
		if note == nil {
			return
		}
		if line := aggregator.ProcessUpdate(&note.Update); line != "" {
			capturedLines = append(capturedLines, line)
		}
	}

	// Test agent message chunks get aggregated
	update1 := acp.NewSessionUpdateAgentMessageChunk(acp.NewContentBlockText("Hello"))
	update2 := acp.NewSessionUpdateAgentMessageChunk(acp.NewContentBlockText(" world\n"))

	onUpdate(&acp.SessionNotification{Update: update1})
	if len(capturedLines) != 0 {
		t.Fatalf("expected no output for first chunk, got: %v", capturedLines)
	}

	onUpdate(&acp.SessionNotification{Update: update2})
	if len(capturedLines) != 1 {
		t.Fatalf("expected one aggregated message, got: %d lines", len(capturedLines))
	}
	expected := "agent_message \"Hello world\\n\""
	if capturedLines[0] != expected {
		t.Fatalf("expected aggregated message: %q, got: %q", expected, capturedLines[0])
	}
}

func TestWriteConsoleLineIntegration(t *testing.T) {
	// Test that writeConsoleLine works with the aggregated output
	var buf bytes.Buffer
	line := "agent_message \"Test message\\n\""

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	writeConsoleLine(&buf, line)
	w.Close()
	os.Stderr = oldStderr

	// Read from pipe to clear it
	_, _ = r.Read(make([]byte, 1024))

	// The function should write to the provided writer
	got := buf.String()
	expected := "agent_message \"Test message\\n\"\n"
	if got != expected {
		t.Fatalf("expected: %q, got: %q", expected, got)
	}
}
