package opencode

import (
	"testing"

	acp "github.com/ironpark/acp-go"
)

func TestFormatSessionUpdateToolCall(t *testing.T) {
	update := acp.NewSessionUpdateToolCall(
		acp.ToolCallId("tool-1"),
		"Read file",
		acp.ToolKindPtr(acp.ToolKindRead),
		acp.ToolCallStatusPtr(acp.ToolCallStatusPending),
		nil,
		nil,
	)

	got := formatSessionUpdate(&update)
	expected := "⏳ \x1b[33mtool_call\x1b[0m id=tool-1 title=\"Read file\" kind=read status=pending"
	if got != expected {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestFormatSessionUpdateAgentMessage(t *testing.T) {
	update := acp.NewSessionUpdateAgentMessageChunk(acp.NewContentBlockText("Hello there"))

	got := formatSessionUpdate(&update)
	expected := "agent_message \"Hello there\""
	if got != expected {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestFormatACPRequest(t *testing.T) {
	got := formatACPRequest("permission", "allow")
	expected := "request permission allow"
	if got != expected {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestFormatToolCallWithStatusBadges(t *testing.T) {
	tests := []struct {
		name     string
		status   *acp.ToolCallStatus
		expected string
	}{
		{
			name:     "pending status",
			status:   acp.ToolCallStatusPtr(acp.ToolCallStatusPending),
			expected: "⏳ \x1b[33mtool_call\x1b[0m id=tool-1 title=\"Read file\" kind=read status=pending",
		},
		{
			name:     "in_progress status", 
			status:   acp.ToolCallStatusPtr(acp.ToolCallStatusInProgress),
			expected: "🔄 \x1b[34mtool_call\x1b[0m id=tool-1 title=\"Read file\" kind=read status=in_progress",
		},
		{
			name:     "completed status",
			status:   acp.ToolCallStatusPtr(acp.ToolCallStatusCompleted),
			expected: "✅ \x1b[32mtool_call\x1b[0m id=tool-1 title=\"Read file\" kind=read status=completed",
		},
		{
			name:     "failed status",
			status:   acp.ToolCallStatusPtr(acp.ToolCallStatusFailed),
			expected: "❌ \x1b[31mtool_call\x1b[0m id=tool-1 title=\"Read file\" kind=read status=failed",
		},
		{
			name:     "unknown status",
			status:   acp.ToolCallStatusPtr("unknown"),
			expected: "⚪ \x1b[37mtool_call\x1b[0m id=tool-1 title=\"Read file\" kind=read status=unknown",
		},
		{
			name:     "nil status",
			status:   nil,
			expected: "⚪ \x1b[37mtool_call\x1b[0m id=tool-1 title=\"Read file\" kind=read",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update := acp.NewSessionUpdateToolCall(
				acp.ToolCallId("tool-1"),
				"Read file",
				acp.ToolKindPtr(acp.ToolKindRead),
				tt.status,
				nil,
				nil,
			)

			got := formatSessionUpdate(&update)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestFormatToolCallUpdateWithStatusBadges(t *testing.T) {
	update := acp.NewSessionUpdateToolCallUpdate(
		acp.ToolCallId("tool-1"),
		acp.ToolCallStatusPtr(acp.ToolCallStatusCompleted),
		nil,
		nil,
	)

	got := formatSessionUpdate(&update)
	expected := "✅ \x1b[32mtool_call_update\x1b[0m id=tool-1 status=completed"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}
