package opencode

import (
	"testing"

	acp "github.com/ironpark/acp-go"
)

func TestFormatToolCallUpdateSimplifiedFormat(t *testing.T) {
	tests := []struct {
		name     string
		status   *acp.ToolCallStatus
		title    string
		expected string
	}{
		{
			name:     "in_progress status",
			status:   acp.ToolCallStatusPtr(acp.ToolCallStatusInProgress),
			title:    "Processing file",
			expected: "🔄 \x1b[34mtool_call_update\x1b[0m Processing file",
		},
		{
			name:     "completed status with title",
			status:   acp.ToolCallStatusPtr(acp.ToolCallStatusCompleted),
			title:    "Read file",
			expected: "✅ \x1b[32mtool_call_update\x1b[0m Read file",
		},
		{
			name:     "failed status",
			status:   acp.ToolCallStatusPtr(acp.ToolCallStatusFailed),
			title:    "Write file",
			expected: "❌ \x1b[31mtool_call_update\x1b[0m Write file",
		},
		{
			name:     "pending status",
			status:   acp.ToolCallStatusPtr(acp.ToolCallStatusPending),
			title:    "Delete file",
			expected: "⏳ \x1b[33mtool_call_update\x1b[0m Delete file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update := acp.NewSessionUpdateToolCallUpdate(
				acp.ToolCallId("tool-1"),
				tt.status,
				nil,
				nil,
			)
			update.GetToolcallupdate().Title = tt.title

			got := formatSessionUpdate(&update)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestFormatToolCallUpdateWithNilStatus(t *testing.T) {
	update := acp.NewSessionUpdateToolCallUpdate(
		acp.ToolCallId("tool-1"),
		nil,
		nil,
		nil,
	)

	got := formatSessionUpdate(&update)
	expected := "⚪ \x1b[37mtool_call_update\x1b[0m "
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}
