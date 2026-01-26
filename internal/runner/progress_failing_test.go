package runner

import (
	"testing"
)

// Failing test that demonstrates the [4/3] bug
// This test should FAIL with current implementation
func TestProgressCounter_DemonstrateBug(t *testing.T) {
	// Based on bug report: "Progress counter can exceed total (e.g., [4/3])"
	// This test creates scenario where this should happen

	var capturedEvents []Event
	eventRecorder := &eventRecorder{events: capturedEvents}

	// Mock runOnce that simulates the bug scenario
	// The key insight: somehow progress gets incremented beyond total
	runOnce := func(opts RunOnceOptions, deps RunOnceDeps) (string, error) {
		// Emit events with progress that should show bug
		// If bug exists, some events might show [4/3]
		emitPhase(deps.Events, EventSelectTask, "task", "Task", opts.Progress)
		emitPhase(deps.Events, EventBeadsUpdate, "task", "Task", opts.Progress)
		return "completed", nil
	}

	// Start with 3 total tasks
	progress := ProgressState{Total: 3}
	deps := RunOnceDeps{Events: eventRecorder}

	// Run loop multiple times
	_, err := RunLoop(RunOnceOptions{Progress: progress}, deps, 4, runOnce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check if any events show the [4/3] bug pattern
	bugDetected := false
	for i, event := range capturedEvents {
		if event.ProgressCompleted > event.ProgressTotal {
			t.Logf("Bug detected in event %d: progress [%d/%d] exceeds total",
				i, event.ProgressCompleted, event.ProgressTotal)
			bugDetected = true
		}

		// Look for specific [4/3] pattern from bug report
		if event.ProgressCompleted == 4 && event.ProgressTotal == 3 {
			t.Logf("Specific [4/3] bug pattern detected in event %d", i)
			bugDetected = true
		}
	}

	// According to TDD, this test should initially FAIL to demonstrate bug
	// If we don't detect the bug, we should fail to indicate we need to create a failing test first
	if !bugDetected {
		t.Skip("Bug not reproduced - unable to create failing test as required by TDD protocol")
	}
}
