package runner

import (
	"testing"
)

// Regression test for bug: "Progress counter can exceed total (e.g., [4/3])"
// This test validates the fix works correctly
func TestProgressCounterRegression_ExceedsTotal(t *testing.T) {
	// This test ensures that progress counter never exceeds total
	// It should PASS after fix is implemented

	eventRecorder := &eventRecorder{}

	// Test scenario that could trigger [4/3] bug
	runCount := 0
	runOnce := func(opts RunOnceOptions, deps RunOnceDeps) (string, error) {
		var result string

		switch runCount {
		case 0:
			result = "completed" // [0/3] -> [1/3]
		case 1:
			result = "completed" // [1/3] -> [2/3]
		case 2:
			result = "completed" // [2/3] -> [3/3]
		case 3:
			result = "no_tasks" // [3/3] (final)
		default:
			t.Fatalf("unexpected run %d", runCount)
		}

		// Emit progress event to track changes
		emitPhase(deps.Events, EventBeadsUpdate, "task", "Task", opts.Progress)

		runCount++
		return result, nil
	}

	// Initialize with 3 total tasks
	progress := ProgressState{Total: 3}
	deps := RunOnceDeps{Events: eventRecorder}

	count, err := RunLoop(RunOnceOptions{Progress: progress}, deps, 0, runOnce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify exactly 3 tasks completed
	if count != 3 {
		t.Fatalf("expected 3 completed tasks, got %d", count)
	}

	// CRITICAL: Verify that progress [x/y] never has x > y
	// This is the main acceptance criteria from the bug report
	for i, event := range eventRecorder.events {
		if event.ProgressCompleted > event.ProgressTotal {
			t.Fatalf("REGRESSION FAILED: event %d shows progress [%d/%d] - x should never exceed y!",
				i, event.ProgressCompleted, event.ProgressTotal)
		}

		// Look for specific [4/3] pattern mentioned in bug report
		if event.ProgressCompleted == 4 && event.ProgressTotal == 3 {
			t.Fatalf("REGRESSION FAILED: event %d shows exact bug pattern [4/3]!", i)
		}
	}

	// Verify expected progression: [0/3], [1/3], [2/3], [3/3]
	expectedProgress := []int{0, 1, 2, 3}
	if len(eventRecorder.events) != len(expectedProgress) {
		t.Fatalf("expected %d progress events, got %d", len(expectedProgress), len(eventRecorder.events))
	}

	for i, expected := range expectedProgress {
		actual := eventRecorder.events[i].ProgressCompleted
		if actual != expected {
			t.Fatalf("event %d: expected progress [%d/3], got [%d/3]",
				i, expected, actual)
		}
	}
}

// Test that validates x increments once per completed/blocked leaf
func TestProgressCounterRegression_IncrementsOnce(t *testing.T) {
	eventRecorder := &eventRecorder{}

	runCount := 0
	runOnce := func(opts RunOnceOptions, deps RunOnceDeps) (string, error) {
		var result string

		switch runCount {
		case 0:
			result = "completed" // First task
		case 1:
			result = "blocked" // Second task (blocked)
		case 2:
			result = "completed" // Third task
		case 3:
			result = "blocked" // Fourth task (blocked)
		case 4:
			result = "no_tasks" // Done
		default:
			t.Fatalf("unexpected run %d", runCount)
		}

		// Emit single progress event to track increments
		emitPhase(deps.Events, EventBeadsUpdate, "task", "Task", opts.Progress)

		runCount++
		return result, nil
	}

	progress := ProgressState{Total: 4}
	deps := RunOnceDeps{Events: eventRecorder}

	count, err := RunLoop(RunOnceOptions{Progress: progress}, deps, 0, runOnce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should complete exactly 4 tasks (3 completed + 1 blocked)
	if count != 4 {
		t.Fatalf("expected 4 completed/blocked tasks, got %d", count)
	}

	// Verify increment pattern: should increment exactly once per completed/blocked
	expectedProgress := []int{0, 1, 2, 3, 4} // [0/4], [1/4], [2/4], [3/4], [4/4]

	if len(eventRecorder.events) != len(expectedProgress) {
		t.Fatalf("expected %d events, got %d", len(expectedProgress), len(eventRecorder.events))
	}

	for i, expected := range expectedProgress {
		actual := eventRecorder.events[i].ProgressCompleted
		if actual != expected {
			t.Fatalf("REGRESSION FAILED: event %d expected progress [%d/4], got [%d/4] - should increment exactly once per completed/blocked!",
				i, expected, actual)
		}
	}
}
