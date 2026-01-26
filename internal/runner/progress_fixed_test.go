package runner

import (
	"testing"
)

// Regression test for bug where progress counter can exceed total
// This test ensures the fix works correctly
func TestProgressCounter_ExceedsTotal_Fixed(t *testing.T) {
	// Test that validates the main acceptance criteria:
	// "Progress [x/y] never exceeds y"

	eventRecorder := &eventRecorder{}

	// Simple scenario: 3 total tasks
	results := []string{"completed", "completed", "completed", "no_tasks"}

	runOnce := func(opts RunOnceOptions, deps RunOnceDeps) (string, error) {
		if len(results) == 0 {
			return "no_tasks", nil
		}

		result := results[0]
		results = results[1:]

		// Emit progress event
		emitPhase(deps.Events, EventBeadsUpdate, "task", "Task", opts.Progress)

		return result, nil
	}

	progress := ProgressState{Total: 3}
	deps := RunOnceDeps{Events: eventRecorder}

	count, err := RunLoop(RunOnceOptions{Progress: progress}, deps, 0, runOnce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should complete exactly 3 tasks
	if count != 3 {
		t.Fatalf("expected 3 completed tasks, got %d", count)
	}

	// Critical check: no event should show progress exceeding total
	for i, event := range eventRecorder.events {
		if event.ProgressCompleted > event.ProgressTotal {
			t.Fatalf("REGRESSION FAILED: event %d shows progress [%d/%d] which exceeds total!",
				i, event.ProgressCompleted, event.ProgressTotal)
		}

		// Look for specific [4/3] pattern mentioned in bug report
		if event.ProgressCompleted == 4 && event.ProgressTotal == 3 {
			t.Fatalf("REGRESSION FAILED: event %d shows exact bug pattern [4/3]!", i)
		}
	}
}

// Test that validates: x increments once per completed/blocked leaf
func TestProgressCounter_IncrementsOnce_Fixed(t *testing.T) {
	eventRecorder := &eventRecorder{}

	// Mix of completed and blocked results
	results := []string{"completed", "blocked", "completed", "blocked", "completed", "no_tasks"}

	runOnce := func(opts RunOnceOptions, deps RunOnceDeps) (string, error) {
		if len(results) == 0 {
			return "no_tasks", nil
		}

		result := results[0]
		results = results[1:]

		// Emit single progress event
		emitPhase(deps.Events, EventBeadsUpdate, "task", "Task", opts.Progress)

		return result, nil
	}

	progress := ProgressState{Total: 5}
	deps := RunOnceDeps{Events: eventRecorder}

	count, err := RunLoop(RunOnceOptions{Progress: progress}, deps, 0, runOnce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should complete exactly 5 tasks
	if count != 5 {
		t.Fatalf("expected 5 completed/blocked tasks, got %d", count)
	}

	// Should see exact increment pattern: [0/5], [1/5], [2/5], [3/5], [4/5], [5/5]
	expectedProgress := []int{0, 1, 2, 3, 4, 5}

	if len(eventRecorder.events) != len(expectedProgress) {
		t.Fatalf("expected %d progress events, got %d", len(expectedProgress), len(eventRecorder.events))
	}

	for i, event := range eventRecorder.events {
		expected := expectedProgress[i]
		if event.ProgressCompleted != expected {
			t.Fatalf("REGRESSION FAILED: event %d expected progress [%d/5], got [%d/5] - should increment exactly once per completed/blocked!",
				i, expected, event.ProgressCompleted)
		}
	}
}
