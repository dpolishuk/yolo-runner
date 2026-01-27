package opencode

import (
	"testing"

	acp "github.com/ironpark/acp-go"
)

// TestLogBubbleStore_CreateAndUpsertToolCall tests that tool call bubbles
// can be created and updated by id
func TestLogBubbleStore_CreateAndUpsertToolCall(t *testing.T) {
	store := NewLogBubbleStore()

	// Create a new tool call bubble
	pendingStatus := acp.ToolCallStatusPending
	executeKind := acp.ToolKindExecute
	toolCall1 := &acp.ToolCall{
		ToolCallId: "tool-1",
		Title:      "First Tool Call",
		Kind:       &executeKind,
		Status:     &pendingStatus,
	}

	store.UpsertToolCall(toolCall1)

	bubbles := store.GetBubbles()
	if len(bubbles) != 1 {
		t.Fatalf("expected 1 bubble after create, got %d", len(bubbles))
	}

	// Verify the bubble content contains the tool call details
	if !containsString(bubbles[0], "tool-1") {
		t.Errorf("expected bubble to contain tool call id 'tool-1', got: %s", bubbles[0])
	}
	if !containsString(bubbles[0], "First Tool Call") {
		t.Errorf("expected bubble to contain tool call title 'First Tool Call', got: %s", bubbles[0])
	}
	if !containsString(bubbles[0], "pending") {
		t.Errorf("expected bubble to contain status 'pending', got: %s", bubbles[0])
	}

	// Update the same tool call bubble with new status
	inProgressStatus := acp.ToolCallStatusInProgress
	toolCall1Update := &acp.ToolCall{
		ToolCallId: "tool-1",
		Title:      "First Tool Call",
		Kind:       &executeKind,
		Status:     &inProgressStatus,
	}

	store.UpsertToolCall(toolCall1Update)

	bubbles = store.GetBubbles()
	if len(bubbles) != 1 {
		t.Fatalf("expected 1 bubble after update, got %d", len(bubbles))
	}

	// Verify the bubble was updated
	if !containsString(bubbles[0], "in_progress") {
		t.Errorf("expected bubble to contain updated status 'in_progress', got: %s", bubbles[0])
	}
	if !containsString(bubbles[0], "tool-1") {
		t.Errorf("expected bubble to still contain tool call id 'tool-1', got: %s", bubbles[0])
	}
}

// TestLogBubbleStore_OrderingStability tests that updating a tool call
// bubble maintains its position in the ordering
func TestLogBubbleStore_OrderingStability(t *testing.T) {
	store := NewLogBubbleStore()

	// Create first tool call
	pendingStatus := acp.ToolCallStatusPending
	toolCall1 := &acp.ToolCall{
		ToolCallId: "tool-1",
		Title:      "First Tool Call",
		Status:     &pendingStatus,
	}

	store.UpsertToolCall(toolCall1)

	// Create second tool call
	toolCall2 := &acp.ToolCall{
		ToolCallId: "tool-2",
		Title:      "Second Tool Call",
		Status:     &pendingStatus,
	}

	store.UpsertToolCall(toolCall2)

	// Create third tool call
	toolCall3 := &acp.ToolCall{
		ToolCallId: "tool-3",
		Title:      "Third Tool Call",
		Status:     &pendingStatus,
	}

	store.UpsertToolCall(toolCall3)

	bubbles := store.GetBubbles()
	if len(bubbles) != 3 {
		t.Fatalf("expected 3 bubbles, got %d", len(bubbles))
	}

	// Verify initial ordering
	if !containsString(bubbles[0], "tool-1") {
		t.Errorf("expected first bubble to be tool-1, got: %s", bubbles[0])
	}
	if !containsString(bubbles[1], "tool-2") {
		t.Errorf("expected second bubble to be tool-2, got: %s", bubbles[1])
	}
	if !containsString(bubbles[2], "tool-3") {
		t.Errorf("expected third bubble to be tool-3, got: %s", bubbles[2])
	}

	// Update the second tool call - it should stay in position
	inProgressStatus := acp.ToolCallStatusInProgress
	toolCall2Update := &acp.ToolCall{
		ToolCallId: "tool-2",
		Title:      "Second Tool Call Updated",
		Status:     &inProgressStatus,
	}

	store.UpsertToolCall(toolCall2Update)

	bubbles = store.GetBubbles()
	if len(bubbles) != 3 {
		t.Fatalf("expected 3 bubbles after update, got %d", len(bubbles))
	}

	// Verify ordering is stable (tool-2 still in middle)
	if !containsString(bubbles[0], "tool-1") {
		t.Errorf("expected first bubble to still be tool-1, got: %s", bubbles[0])
	}
	if !containsString(bubbles[1], "tool-2") {
		t.Errorf("expected second bubble to still be tool-2, got: %s", bubbles[1])
	}
	if !containsString(bubbles[1], "Second Tool Call Updated") {
		t.Errorf("expected second bubble to be updated with new title, got: %s", bubbles[1])
	}
	if !containsString(bubbles[1], "in_progress") {
		t.Errorf("expected second bubble to have updated status, got: %s", bubbles[1])
	}
	if !containsString(bubbles[2], "tool-3") {
		t.Errorf("expected third bubble to still be tool-3, got: %s", bubbles[2])
	}
}

// TestLogBubbleStore_UpsertToolCallUpdate tests that tool_call_update
// events also upsert by id
func TestLogBubbleStore_UpsertToolCallUpdate(t *testing.T) {
	store := NewLogBubbleStore()

	// Create a tool call bubble
	pendingStatus := acp.ToolCallStatusPending
	toolCall1 := &acp.ToolCall{
		ToolCallId: "tool-1",
		Title:      "First Tool Call",
		Status:     &pendingStatus,
	}

	store.UpsertToolCall(toolCall1)

	bubbles := store.GetBubbles()
	if len(bubbles) != 1 {
		t.Fatalf("expected 1 bubble after create, got %d", len(bubbles))
	}

	// Update using ToolCallUpdate
	inProgressStatus := acp.ToolCallStatusInProgress
	toolCall1Update := &acp.ToolCallUpdate{
		ToolCallId: "tool-1",
		Title:      "First Tool Call",
		Status:     &inProgressStatus,
	}

	store.UpsertToolCallUpdate(toolCall1Update)

	bubbles = store.GetBubbles()
	if len(bubbles) != 1 {
		t.Fatalf("expected 1 bubble after update, got %d", len(bubbles))
	}

	// Verify the bubble was updated to tool_call_update format
	// Note: tool_call_update uses simplified format (emoji + label + title, no id/status)
	if !containsString(bubbles[0], "tool_call_update") {
		t.Errorf("expected bubble to contain 'tool_call_update', got: %s", bubbles[0])
	}
	// Should contain the rotating emoji for in_progress status (🔄)
	if !containsString(bubbles[0], "🔄") {
		t.Errorf("expected bubble to contain in_progress emoji, got: %s", bubbles[0])
	}
	if !containsString(bubbles[0], "First Tool Call") {
		t.Errorf("expected bubble to still contain tool call title, got: %s", bubbles[0])
	}
}

// TestLogBubbleStore_AddLogEntry tests that regular log entries
// can be added to the store

// TestLogBubbleStore_OrderingStabilityWithUpdate tests that updating a tool call
// bubble using UpsertToolCallUpdate maintains its position in the ordering
func TestLogBubbleStore_OrderingStabilityWithUpdate(t *testing.T) {
	store := NewLogBubbleStore()

	// Create first tool call using ToolCall
	pendingStatus := acp.ToolCallStatusPending
	toolCall1 := &acp.ToolCall{
		ToolCallId: "tool-1",
		Title:      "First Tool Call",
		Status:     &pendingStatus,
	}

	store.UpsertToolCall(toolCall1)

	// Create second tool call using ToolCall
	toolCall2 := &acp.ToolCall{
		ToolCallId: "tool-2",
		Title:      "Second Tool Call",
		Status:     &pendingStatus,
	}

	store.UpsertToolCall(toolCall2)

	// Create third tool call using ToolCall
	toolCall3 := &acp.ToolCall{
		ToolCallId: "tool-3",
		Title:      "Third Tool Call",
		Status:     &pendingStatus,
	}

	store.UpsertToolCall(toolCall3)

	bubbles := store.GetBubbles()
	if len(bubbles) != 3 {
		t.Fatalf("expected 3 bubbles, got %d", len(bubbles))
	}

	// Verify initial ordering
	if !containsString(bubbles[0], "tool-1") {
		t.Errorf("expected first bubble to be tool-1, got: %s", bubbles[0])
	}
	if !containsString(bubbles[1], "tool-2") {
		t.Errorf("expected second bubble to be tool-2, got: %s", bubbles[1])
	}
	if !containsString(bubbles[2], "tool-3") {
		t.Errorf("expected third bubble to be tool-3, got: %s", bubbles[2])
	}

	// Update the second tool call using UpsertToolCallUpdate - it should stay in position
	inProgressStatus := acp.ToolCallStatusInProgress
	toolCall2Update := &acp.ToolCallUpdate{
		ToolCallId: "tool-2",
		Title:      "Second Tool Call Updated",
		Status:     &inProgressStatus,
	}

	store.UpsertToolCallUpdate(toolCall2Update)

	bubbles = store.GetBubbles()
	if len(bubbles) != 3 {
		t.Fatalf("expected 3 bubbles after update, got %d", len(bubbles))
	}

	// Verify ordering is stable (tool-2 still in middle)
	// Note: tool_call_update format doesn't include the id, so we verify
	// by checking that the bubble is still in position 1 with updated content
	if !containsString(bubbles[0], "tool-1") {
		t.Errorf("expected first bubble to still be tool-1, got: %s", bubbles[0])
	}
	if !containsString(bubbles[1], "Second Tool Call Updated") {
		t.Errorf("expected second bubble to be updated with new title, got: %s", bubbles[1])
	}
	if !containsString(bubbles[1], "tool_call_update") {
		t.Errorf("expected second bubble to have tool_call_update format, got: %s", bubbles[1])
	}
	if !containsString(bubbles[2], "tool-3") {
		t.Errorf("expected third bubble to still be tool-3, got: %s", bubbles[2])
	}
}

func TestLogBubbleStore_AddLogEntry(t *testing.T) {
	store := NewLogBubbleStore()

	// Add a regular log entry
	store.AddLogEntry("Starting task...")

	bubbles := store.GetBubbles()
	if len(bubbles) != 1 {
		t.Fatalf("expected 1 bubble after adding log entry, got %d", len(bubbles))
	}

	if bubbles[0] != "Starting task..." {
		t.Errorf("expected bubble to be 'Starting task...', got: %s", bubbles[0])
	}
}

// TestLogBubbleStore_MixedEntriesWithUpdate tests that tool call bubbles
// can be upserted correctly even when interleaved with regular log entries
func TestLogBubbleStore_MixedEntriesWithUpdate(t *testing.T) {
	store := NewLogBubbleStore()

	// Add a regular log entry first
	store.AddLogEntry("Task started")

	// Create first tool call
	pendingStatus := acp.ToolCallStatusPending
	toolCall1 := &acp.ToolCall{
		ToolCallId: "tool-1",
		Title:      "First Tool",
		Status:     &pendingStatus,
	}
	store.UpsertToolCall(toolCall1)

	// Add another regular log entry
	store.AddLogEntry("Processing...")

	// Create second tool call
	toolCall2 := &acp.ToolCall{
		ToolCallId: "tool-2",
		Title:      "Second Tool",
		Status:     &pendingStatus,
	}
	store.UpsertToolCall(toolCall2)

	// Add one more regular log entry
	store.AddLogEntry("Almost done")

	bubbles := store.GetBubbles()
	if len(bubbles) != 5 {
		t.Fatalf("expected 5 bubbles, got %d", len(bubbles))
	}

	// Verify initial ordering: log, tool1, log, tool2, log
	if bubbles[0] != "Task started" {
		t.Errorf("expected bubble[0] to be 'Task started', got: %s", bubbles[0])
	}
	if !containsString(bubbles[1], "tool-1") {
		t.Errorf("expected bubble[1] to be tool-1, got: %s", bubbles[1])
	}
	if bubbles[2] != "Processing..." {
		t.Errorf("expected bubble[2] to be 'Processing...', got: %s", bubbles[2])
	}
	if !containsString(bubbles[3], "tool-2") {
		t.Errorf("expected bubble[3] to be tool-2, got: %s", bubbles[3])
	}
	if bubbles[4] != "Almost done" {
		t.Errorf("expected bubble[4] to be 'Almost done', got: %s", bubbles[4])
	}

	// Update first tool call - it should stay in position 1
	inProgressStatus := acp.ToolCallStatusInProgress
	toolCall1Update := &acp.ToolCall{
		ToolCallId: "tool-1",
		Title:      "First Tool Updated",
		Status:     &inProgressStatus,
	}
	store.UpsertToolCall(toolCall1Update)

	bubbles = store.GetBubbles()
	if len(bubbles) != 5 {
		t.Fatalf("expected 5 bubbles after update, got %d", len(bubbles))
	}

	// Verify ordering is maintained with regular log entries
	if bubbles[0] != "Task started" {
		t.Errorf("expected bubble[0] to still be 'Task started', got: %s", bubbles[0])
	}
	if !containsString(bubbles[1], "tool-1") {
		t.Errorf("expected bubble[1] to still be tool-1, got: %s", bubbles[1])
	}
	if !containsString(bubbles[1], "First Tool Updated") {
		t.Errorf("expected bubble[1] to be updated, got: %s", bubbles[1])
	}
	if bubbles[2] != "Processing..." {
		t.Errorf("expected bubble[2] to still be 'Processing...', got: %s", bubbles[2])
	}
	if !containsString(bubbles[3], "tool-2") {
		t.Errorf("expected bubble[3] to still be tool-2, got: %s", bubbles[3])
	}
	if bubbles[4] != "Almost done" {
		t.Errorf("expected bubble[4] to still be 'Almost done', got: %s", bubbles[4])
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestLogBubbleStore_NewLogBubbleStore tests that a new store starts empty
func TestLogBubbleStore_NewLogBubbleStore(t *testing.T) {
	store := NewLogBubbleStore()

	bubbles := store.GetBubbles()
	if len(bubbles) != 0 {
		t.Errorf("expected 0 bubbles in new store, got %d", len(bubbles))
	}
}

// TestLogBubbleStore_GetBubblesReturnsCopy tests that GetBubbles returns a copy
// and modifying the returned slice doesn't affect the store
func TestLogBubbleStore_GetBubblesReturnsCopy(t *testing.T) {
	store := NewLogBubbleStore()

	pendingStatus := acp.ToolCallStatusPending
	toolCall1 := &acp.ToolCall{
		ToolCallId: "tool-1",
		Title:      "First Tool",
		Status:     &pendingStatus,
	}
	store.UpsertToolCall(toolCall1)

	// Get bubbles and try to modify the returned slice
	bubbles := store.GetBubbles()
	if len(bubbles) != 1 {
		t.Fatalf("expected 1 bubble, got %d", len(bubbles))
	}

	// Modify the returned slice
	bubbles[0] = "MODIFIED"
	bubbles = append(bubbles, "EXTRA")

	// Get bubbles again - should not be affected
	bubbles2 := store.GetBubbles()
	if len(bubbles2) != 1 {
		t.Errorf("expected 1 bubble after modifying returned slice, got %d", len(bubbles2))
	}
	if bubbles2[0] == "MODIFIED" {
		t.Errorf("store was modified when returned slice was changed, got: %s", bubbles2[0])
	}
	if !containsString(bubbles2[0], "tool-1") {
		t.Errorf("expected original bubble content, got: %s", bubbles2[0])
	}
}

// TestLogBubbleStore_NilToolCallUpsert tests that nil tool calls are handled gracefully
func TestLogBubbleStore_NilToolCallUpsert(t *testing.T) {
	store := NewLogBubbleStore()

	// Upsert nil tool call - should not panic or add bubbles
	store.UpsertToolCall(nil)

	bubbles := store.GetBubbles()
	if len(bubbles) != 0 {
		t.Errorf("expected 0 bubbles after nil upsert, got %d", len(bubbles))
	}
}

// TestLogBubbleStore_NilToolCallUpdateUpsert tests that nil tool call updates are handled gracefully
func TestLogBubbleStore_NilToolCallUpdateUpsert(t *testing.T) {
	store := NewLogBubbleStore()

	// Upsert nil tool call update - should not panic or add bubbles
	store.UpsertToolCallUpdate(nil)

	bubbles := store.GetBubbles()
	if len(bubbles) != 0 {
		t.Errorf("expected 0 bubbles after nil upsert, got %d", len(bubbles))
	}
}

// TestLogBubbleStore_EmptyLogEntry tests that empty log entries are not added
func TestLogBubbleStore_EmptyLogEntry(t *testing.T) {
	store := NewLogBubbleStore()

	// Add empty log entry - should not be added
	store.AddLogEntry("")

	bubbles := store.GetBubbles()
	if len(bubbles) != 0 {
		t.Errorf("expected 0 bubbles after empty log entry, got %d", len(bubbles))
	}
}

// TestLogBubbleStore_MultipleRapidUpserts tests that multiple rapid upserts of the same tool call
// correctly update in place without creating duplicates
func TestLogBubbleStore_MultipleRapidUpserts(t *testing.T) {
	store := NewLogBubbleStore()

	toolCallId := "tool-rapid"

	// Create initial tool call
	pendingStatus := acp.ToolCallStatusPending
	toolCall1 := &acp.ToolCall{
		ToolCallId: acp.ToolCallId(toolCallId),
		Title:      "Rapid Tool",
		Status:     &pendingStatus,
	}
	store.UpsertToolCall(toolCall1)

	bubbles := store.GetBubbles()
	if len(bubbles) != 1 {
		t.Fatalf("expected 1 bubble after first upsert, got %d", len(bubbles))
	}

	// Rapidly update the same tool call 5 times
	for i := 0; i < 5; i++ {
		inProgressStatus := acp.ToolCallStatusInProgress
		toolCallUpdate := &acp.ToolCall{
			ToolCallId: acp.ToolCallId(toolCallId),
			Title:      "Rapid Tool",
			Status:     &inProgressStatus,
		}
		store.UpsertToolCall(toolCallUpdate)
	}

	bubbles = store.GetBubbles()
	// Should still only have 1 bubble, not 6
	if len(bubbles) != 1 {
		t.Errorf("expected 1 bubble after 5 rapid upserts, got %d", len(bubbles))
	}
	// Verify the bubble was updated
	if !containsString(bubbles[0], "tool-rapid") {
		t.Errorf("expected bubble to contain tool id, got: %s", bubbles[0])
	}
}

// TestLogBubbleStore_UpdateDifferentStatuses tests updating a tool call through all possible statuses
func TestLogBubbleStore_UpdateDifferentStatuses(t *testing.T) {
	store := NewLogBubbleStore()

	toolCallId := "tool-status"

	// Start with pending
	pendingStatus := acp.ToolCallStatusPending
	toolCall := &acp.ToolCall{
		ToolCallId: acp.ToolCallId(toolCallId),
		Title:      "Status Tool",
		Status:     &pendingStatus,
	}
	store.UpsertToolCall(toolCall)

	bubbles := store.GetBubbles()
	if len(bubbles) != 1 {
		t.Fatalf("expected 1 bubble, got %d", len(bubbles))
	}
	if !containsString(bubbles[0], "pending") {
		t.Errorf("expected pending status, got: %s", bubbles[0])
	}

	// Update to in_progress
	inProgressStatus := acp.ToolCallStatusInProgress
	toolCall.Status = &inProgressStatus
	store.UpsertToolCall(toolCall)

	bubbles = store.GetBubbles()
	if !containsString(bubbles[0], "in_progress") {
		t.Errorf("expected in_progress status, got: %s", bubbles[0])
	}

	// Update to completed
	completedStatus := acp.ToolCallStatusCompleted
	toolCall.Status = &completedStatus
	store.UpsertToolCall(toolCall)

	bubbles = store.GetBubbles()
	if !containsString(bubbles[0], "completed") {
		t.Errorf("expected completed status, got: %s", bubbles[0])
	}

	// Update to failed
	failedStatus := acp.ToolCallStatusFailed
	toolCall.Status = &failedStatus
	store.UpsertToolCall(toolCall)

	bubbles = store.GetBubbles()
	if !containsString(bubbles[0], "failed") {
		t.Errorf("expected failed status, got: %s", bubbles[0])
	}

	// Should still only have 1 bubble
	if len(bubbles) != 1 {
		t.Errorf("expected 1 bubble after multiple status updates, got %d", len(bubbles))
	}
}

// TestLogBubbleStore_OrderingStabilityMultipleUpdates tests that ordering is stable
// even when multiple bubbles are updated multiple times
func TestLogBubbleStore_OrderingStabilityMultipleUpdates(t *testing.T) {
	store := NewLogBubbleStore()

	pendingStatus := acp.ToolCallStatusPending
	inProgressStatus := acp.ToolCallStatusInProgress
	completedStatus := acp.ToolCallStatusCompleted

	// Create three tool calls
	toolCall1 := &acp.ToolCall{ToolCallId: "tool-1", Title: "Tool 1", Status: &pendingStatus}
	toolCall2 := &acp.ToolCall{ToolCallId: "tool-2", Title: "Tool 2", Status: &pendingStatus}
	toolCall3 := &acp.ToolCall{ToolCallId: "tool-3", Title: "Tool 3", Status: &pendingStatus}

	store.UpsertToolCall(toolCall1)
	store.UpsertToolCall(toolCall2)
	store.UpsertToolCall(toolCall3)

	bubbles := store.GetBubbles()
	if len(bubbles) != 3 {
		t.Fatalf("expected 3 bubbles, got %d", len(bubbles))
	}

	// Update tool-2 multiple times
	toolCall2.Status = &inProgressStatus
	store.UpsertToolCall(toolCall2)

	toolCall2.Status = &completedStatus
	store.UpsertToolCall(toolCall2)

	toolCall2.Title = "Tool 2 Updated"
	store.UpsertToolCall(toolCall2)

	// Update tool-1 once
	toolCall1.Status = &inProgressStatus
	store.UpsertToolCall(toolCall1)

	// Update tool-3 once
	toolCall3.Status = &completedStatus
	store.UpsertToolCall(toolCall3)

	bubbles = store.GetBubbles()
	if len(bubbles) != 3 {
		t.Fatalf("expected 3 bubbles after updates, got %d", len(bubbles))
	}

	// Verify ordering is still tool-1, tool-2, tool-3
	if !containsString(bubbles[0], "tool-1") {
		t.Errorf("expected first bubble to be tool-1, got: %s", bubbles[0])
	}
	if !containsString(bubbles[1], "tool-2") {
		t.Errorf("expected second bubble to be tool-2, got: %s", bubbles[1])
	}
	if !containsString(bubbles[2], "tool-3") {
		t.Errorf("expected third bubble to be tool-3, got: %s", bubbles[2])
	}
}

// TestLogBubbleStore_MixedEntriesWithToolCallUpdate tests ordering stability when
// using UpsertToolCallUpdate in a mixed scenario
func TestLogBubbleStore_MixedEntriesWithToolCallUpdate(t *testing.T) {
	store := NewLogBubbleStore()

	// Add regular log entry
	store.AddLogEntry("Task started")

	// Create tool call
	pendingStatus := acp.ToolCallStatusPending
	toolCall := &acp.ToolCall{
		ToolCallId: "tool-mixed",
		Title:      "Mixed Tool",
		Status:     &pendingStatus,
	}
	store.UpsertToolCall(toolCall)

	// Add another log entry
	store.AddLogEntry("Processing...")

	bubbles := store.GetBubbles()
	if len(bubbles) != 3 {
		t.Fatalf("expected 3 bubbles, got %d", len(bubbles))
	}

	// Verify ordering
	if bubbles[0] != "Task started" {
		t.Errorf("expected bubble[0] to be 'Task started', got: %s", bubbles[0])
	}
	if !containsString(bubbles[1], "tool-mixed") {
		t.Errorf("expected bubble[1] to be tool-mixed, got: %s", bubbles[1])
	}
	if bubbles[2] != "Processing..." {
		t.Errorf("expected bubble[2] to be 'Processing...', got: %s", bubbles[2])
	}

	// Update using UpsertToolCallUpdate
	inProgressStatus := acp.ToolCallStatusInProgress
	toolUpdate := &acp.ToolCallUpdate{
		ToolCallId: "tool-mixed",
		Title:      "Mixed Tool Updated",
		Status:     &inProgressStatus,
	}
	store.UpsertToolCallUpdate(toolUpdate)

	bubbles = store.GetBubbles()
	if len(bubbles) != 3 {
		t.Fatalf("expected 3 bubbles after update, got %d", len(bubbles))
	}

	// Verify ordering is maintained
	if bubbles[0] != "Task started" {
		t.Errorf("expected bubble[0] to still be 'Task started', got: %s", bubbles[0])
	}
	if !containsString(bubbles[1], "Mixed Tool Updated") {
		t.Errorf("expected bubble[1] to be updated, got: %s", bubbles[1])
	}
	if !containsString(bubbles[1], "tool_call_update") {
		t.Errorf("expected bubble[1] to have tool_call_update format, got: %s", bubbles[1])
	}
	if bubbles[2] != "Processing..." {
		t.Errorf("expected bubble[2] to still be 'Processing...', got: %s", bubbles[2])
	}
}
