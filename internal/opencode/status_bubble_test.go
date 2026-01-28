package opencode

import (
	"testing"

	acp "github.com/ironpark/acp-go"
)

// TestFormatToolCallDirectly tests the formatToolCall function directly
// These tests verify status bubble formatting for tool calls with different statuses.
// Following TDD: tests define expected behavior, implementation should match.
func TestFormatToolCallDirectly(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		id       acp.ToolCallId
		title    string
		kind     *acp.ToolKind
		status   *acp.ToolCallStatus
		expected string
	}{
		{
			name:     "pending status with all fields",
			prefix:   "tool_call",
			id:       "tool-1",
			title:    "Read file",
			kind:     acp.ToolKindPtr(acp.ToolKindRead),
			status:   acp.ToolCallStatusPtr(acp.ToolCallStatusPending),
			expected: "⏳ \x1b[33mtool_call\x1b[0m id=tool-1 title=\"Read file\" kind=read status=pending",
		},
		{
			name:     "in_progress status",
			prefix:   "tool_call",
			id:       "tool-2",
			title:    "Write file",
			kind:     acp.ToolKindPtr(acp.ToolKindRead),
			status:   acp.ToolCallStatusPtr(acp.ToolCallStatusInProgress),
			expected: "🔄 \x1b[34mtool_call\x1b[0m id=tool-2 title=\"Write file\" kind=read status=in_progress",
		},
		{
			name:     "completed status",
			prefix:   "tool_call",
			id:       "tool-3",
			title:    "Execute command",
			kind:     acp.ToolKindPtr(acp.ToolKindRead),
			status:   acp.ToolCallStatusPtr(acp.ToolCallStatusCompleted),
			expected: "✅ \x1b[32mtool_call\x1b[0m id=tool-3 title=\"Execute command\" kind=read status=completed",
		},
		{
			name:     "failed status",
			prefix:   "tool_call",
			id:       "tool-4",
			title:    "Delete file",
			kind:     acp.ToolKindPtr(acp.ToolKindRead),
			status:   acp.ToolCallStatusPtr(acp.ToolCallStatusFailed),
			expected: "❌ \x1b[31mtool_call\x1b[0m id=tool-4 title=\"Delete file\" kind=read status=failed",
		},
		{
			name:     "nil status",
			prefix:   "tool_call",
			id:       "tool-5",
			title:    "Unknown status",
			kind:     acp.ToolKindPtr(acp.ToolKindRead),
			status:   nil,
			expected: "⚪ \x1b[37mtool_call\x1b[0m id=tool-5 title=\"Unknown status\" kind=read",
		},
		{
			name:     "empty title",
			prefix:   "tool_call",
			id:       "tool-6",
			title:    "",
			kind:     acp.ToolKindPtr(acp.ToolKindRead),
			status:   acp.ToolCallStatusPtr(acp.ToolCallStatusPending),
			expected: "⏳ \x1b[33mtool_call\x1b[0m id=tool-6 kind=read status=pending",
		},
		{
			name:     "nil kind",
			prefix:   "tool_call",
			id:       "tool-7",
			title:    "No kind",
			kind:     nil,
			status:   acp.ToolCallStatusPtr(acp.ToolCallStatusPending),
			expected: "⏳ \x1b[33mtool_call\x1b[0m id=tool-7 title=\"No kind\" status=pending",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatToolCall(tt.prefix, tt.id, tt.title, tt.kind, tt.status)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

// TestFormatToolCallUpdateSimplifiedFormatDirectly tests formatToolCall with tool_call_update prefix
func TestFormatToolCallUpdateSimplifiedFormatDirectly(t *testing.T) {
	tests := []struct {
		name     string
		id       acp.ToolCallId
		title    string
		kind     *acp.ToolKind
		status   *acp.ToolCallStatus
		expected string
	}{
		{
			name:     "pending status simplified",
			id:       "tool-1",
			title:    "Pending task",
			kind:     acp.ToolKindPtr(acp.ToolKindRead),
			status:   acp.ToolCallStatusPtr(acp.ToolCallStatusPending),
			expected: "⏳ \x1b[33mtool_call_update\x1b[0m Pending task",
		},
		{
			name:     "in_progress status simplified",
			id:       "tool-2",
			title:    "Running task",
			kind:     acp.ToolKindPtr(acp.ToolKindRead),
			status:   acp.ToolCallStatusPtr(acp.ToolCallStatusInProgress),
			expected: "🔄 \x1b[34mtool_call_update\x1b[0m Running task",
		},
		{
			name:     "completed status simplified",
			id:       "tool-3",
			title:    "Done",
			kind:     acp.ToolKindPtr(acp.ToolKindRead),
			status:   acp.ToolCallStatusPtr(acp.ToolCallStatusCompleted),
			expected: "✅ \x1b[32mtool_call_update\x1b[0m Done",
		},
		{
			name:     "failed status simplified",
			id:       "tool-4",
			title:    "Error",
			kind:     acp.ToolKindPtr(acp.ToolKindRead),
			status:   acp.ToolCallStatusPtr(acp.ToolCallStatusFailed),
			expected: "❌ \x1b[31mtool_call_update\x1b[0m Error",
		},
		{
			name:     "nil status simplified",
			id:       "tool-5",
			title:    "Unknown",
			kind:     nil,
			status:   nil,
			expected: "⚪ \x1b[37mtool_call_update\x1b[0m Unknown",
		},
		{
			name:     "empty title simplified",
			id:       "tool-6",
			title:    "",
			kind:     acp.ToolKindPtr(acp.ToolKindRead),
			status:   acp.ToolCallStatusPtr(acp.ToolCallStatusPending),
			expected: "⏳ \x1b[33mtool_call_update\x1b[0m ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatToolCall("tool_call_update", tt.id, tt.title, tt.kind, tt.status)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

// TestFormatToolCallStatusEmoji verifies correct emoji mapping for each status
func TestFormatToolCallStatusEmoji(t *testing.T) {
	tests := []struct {
		name     string
		status   *acp.ToolCallStatus
		expected string
	}{
		{
			name:     "pending status emoji",
			status:   acp.ToolCallStatusPtr(acp.ToolCallStatusPending),
			expected: "⏳",
		},
		{
			name:     "in_progress status emoji",
			status:   acp.ToolCallStatusPtr(acp.ToolCallStatusInProgress),
			expected: "🔄",
		},
		{
			name:     "completed status emoji",
			status:   acp.ToolCallStatusPtr(acp.ToolCallStatusCompleted),
			expected: "✅",
		},
		{
			name:     "failed status emoji",
			status:   acp.ToolCallStatusPtr(acp.ToolCallStatusFailed),
			expected: "❌",
		},
		{
			name:     "nil status emoji",
			status:   nil,
			expected: "⚪",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use tool_call prefix and minimal data to just check emoji
			got := formatToolCall("tool_call", "test-id", "", nil, tt.status)
			if !containsSubstring(got, tt.expected) {
				t.Errorf("expected emoji %q in output, got %q", tt.expected, got)
			}
		})
	}
}

// TestFormatToolCallColorCodes verifies correct ANSI color codes for each status
func TestFormatToolCallColorCodes(t *testing.T) {
	tests := []struct {
		name     string
		status   *acp.ToolCallStatus
		expected string
	}{
		{
			name:     "pending status color (yellow)",
			status:   acp.ToolCallStatusPtr(acp.ToolCallStatusPending),
			expected: "\x1b[33m",
		},
		{
			name:     "in_progress status color (blue)",
			status:   acp.ToolCallStatusPtr(acp.ToolCallStatusInProgress),
			expected: "\x1b[34m",
		},
		{
			name:     "completed status color (green)",
			status:   acp.ToolCallStatusPtr(acp.ToolCallStatusCompleted),
			expected: "\x1b[32m",
		},
		{
			name:     "failed status color (red)",
			status:   acp.ToolCallStatusPtr(acp.ToolCallStatusFailed),
			expected: "\x1b[31m",
		},
		{
			name:     "nil status color (white)",
			status:   nil,
			expected: "\x1b[37m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatToolCall("tool_call", "test-id", "", nil, tt.status)
			if !containsSubstring(got, tt.expected) {
				t.Errorf("expected color code %q in output, got %q", tt.expected, got)
			}
		})
	}
}

// TestFormatToolCallResetColor verifies that color codes are reset
func TestFormatToolCallResetColor(t *testing.T) {
	resetColor := "\x1b[0m"

	tests := []struct {
		name   string
		status *acp.ToolCallStatus
	}{
		{
			name:   "pending status reset",
			status: acp.ToolCallStatusPtr(acp.ToolCallStatusPending),
		},
		{
			name:   "in_progress status reset",
			status: acp.ToolCallStatusPtr(acp.ToolCallStatusInProgress),
		},
		{
			name:   "completed status reset",
			status: acp.ToolCallStatusPtr(acp.ToolCallStatusCompleted),
		},
		{
			name:   "failed status reset",
			status: acp.ToolCallStatusPtr(acp.ToolCallStatusFailed),
		},
		{
			name:   "nil status reset",
			status: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatToolCall("tool_call", "test-id", "", nil, tt.status)
			if !containsSubstring(got, resetColor) {
				t.Errorf("expected reset color code %q in output, got %q", resetColor, got)
			}
		})
	}
}

// TestFormatToolCallDifferentKinds verifies formatting with different tool kinds
func TestFormatToolCallDifferentKinds(t *testing.T) {
	// Test with the only defined ToolKind constant
	kind := acp.ToolKindRead
	status := acp.ToolCallStatusPtr(acp.ToolCallStatusPending)
	got := formatToolCall("tool_call", "test-id", "Test", acp.ToolKindPtr(kind), status)
	expectedKind := "kind=" + string(kind)
	if !containsSubstring(got, expectedKind) {
		t.Errorf("expected %q in output, got %q", expectedKind, got)
	}
}
