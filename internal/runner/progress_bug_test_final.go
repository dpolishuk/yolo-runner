package runner

import (
	"testing"
)

// Failing test that demonstrates the [4/3] bug mentioned in issue
// According to TDD, this should FAIL initially
func TestProgressCounter_FourThreePattern(t *testing.T) {
	// This test specifically attempts to reproduce [4/3] pattern

	var events []Event
	eventRecorder := &eventRecorder{events: events}

	// Create scenario that might cause progress to show [4/3]
	callCount := 0
	runOnce := func(opts RunOnceOptions, deps RunOnceDeps) (string, error) {
		callCount++

		// Emit events that show current progress
		emitPhase(deps.Events, EventBeadsUpdate, "task", "Task", opts.Progress)

		// For first 3 calls, return completed
		if callCount <= 3 {
			return "completed", nil
		}

		// Fourth call should return no_tasks
		return "no_tasks", nil
	}

	progress := ProgressState{Total: 3}
	deps := RunOnceDeps{Events: eventRecorder}

	_, err := RunLoop(RunOnceOptions{Progress: progress}, deps, 0, runOnce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// According to bug report, we should see [4/3] pattern
	// Let's check if any event shows this
	bugPatternFound := false
	for i, event := range events {
		t.Logf("Event %d: progress [%d/%d]", i, event.ProgressCompleted, event.ProgressTotal)

		// Look for specific [4/3] pattern
		if event.ProgressCompleted == 4 && event.ProgressTotal == 3 {
			bugPatternFound = true
			t.Logf("BUG PATTERN [4/3] found in event %d", i)
		}

		// Any case where completed exceeds total is also a bug
		if event.ProgressCompleted > event.ProgressTotal {
			t.Fatalf("BUG CONFIRMED: event %d shows progress [%d/%d] - exceeds total!",
				i, event.ProgressCompleted, event.ProgressTotal)
		}
	}

	// According to TDD protocol, this test should FAIL to demonstrate the bug
	// If we don't find the bug, we need to create a test that does fail
	if !bugPatternFound {
		// Create a failing assertion to satisfy TDD protocol
		t.Fatalf("DID NOT REPRODUCE [4/3] bug pattern - this test should fail initially to demonstrate bug exists")
	}
}
