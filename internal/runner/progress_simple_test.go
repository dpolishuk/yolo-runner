package runner

import (
	"testing"
)

// Test that reproduces the exact bug scenario where progress counter shows [4/3]
// This test is designed to fail with current implementation, demonstrating the bug
func TestProgressCounterSimple_Reproduction(t *testing.T) {
	// This test attempts to reproduce the [4/3] bug scenario

	runs := 0
	eventRecorder := &eventRecorder{}

	// Simulate running 3 tasks, where progress should go [0/3], [1/3], [2/3], [3/3]
	runOnce := func(opts RunOnceOptions, deps RunOnceDeps) (string, error) {
		var result string

		switch runs {
		case 0:
			result = "completed"
		case 1:
			result = "completed"
		case 2:
			result = "completed"
		case 3:
			result = "no_tasks"
		default:
			t.Fatalf("unexpected run %d", runs)
		}

		// Emit progress for this run
		emitPhase(deps.Events, EventBeadsUpdate, "task", "Task", opts.Progress)

		runs++
		return result, nil
	}

	// Start with 3 total tasks
	progress := ProgressState{Total: 3}
	deps := RunOnceDeps{Events: eventRecorder}

	count, err := RunLoop(RunOnceOptions{Progress: progress}, deps, 0, runOnce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected count 3, got %d", count)
	}

	// Check all events for bug pattern
	expectedProgress := []int{0, 1, 2, 3} // Should see [0/3], [1/3], [2/3], [3/3]

	if len(eventRecorder.events) != len(expectedProgress) {
		t.Fatalf("expected %d events, got %d", len(expectedProgress), len(eventRecorder.events))
	}

	for i, event := range eventRecorder.events {
		expectedCompleted := expectedProgress[i]
		actualCompleted := event.ProgressCompleted
		total := event.ProgressTotal

		if actualCompleted != expectedCompleted {
			t.Fatalf("event %d: expected progress [%d/%d], got [%d/%d]",
				i, expectedCompleted, total, actualCompleted, total)
		}

		if actualCompleted > total {
			t.Fatalf("BUG REPRODUCED: event %d shows progress [%d/%d] which exceeds total!",
				i, actualCompleted, total)
		}

		// Look specifically for [4/3] pattern mentioned in bug report
		if actualCompleted == 4 && total == 3 {
			t.Fatalf("BUG REPRODUCED: event %d shows exact bug pattern [4/3]!", i)
		}
	}
}
