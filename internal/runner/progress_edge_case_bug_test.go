package runner

import (
	"testing"
)

// Test for the specific bug scenario where error+blocked causes double increment
func TestProgressCounter_ErrorBlockedDoubleIncrement(t *testing.T) {
	eventRecorder := &eventRecorder{}

	// Simulate a runOnce that returns error + "blocked" once, then succeeds
	attempts := 0
	runOnce := func(opts RunOnceOptions, deps RunOnceDeps) (string, error) {
		if attempts == 0 {
			attempts++
			// Return error + "blocked" to test double increment bug
			return "blocked", &fakeError{"simulated error"}
		}

		// Second attempt succeeds
		emitPhase(deps.Events, EventBeadsUpdate, "task", "Task", opts.Progress)
		return "completed", nil
	}

	progress := ProgressState{Total: 1}
	deps := RunOnceDeps{Events: eventRecorder}

	_, err := RunLoop(RunOnceOptions{Progress: progress}, deps, 0, runOnce)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	// If bug exists, completed might be > 1 due to double increment
	// Check events for exceeding total
	for i, event := range eventRecorder.events {
		if event.ProgressCompleted > event.ProgressTotal {
			t.Fatalf("BUG DETECTED: event %d shows progress [%d/%d] - completed incremented twice for same task!",
				i, event.ProgressCompleted, event.ProgressTotal)
		}
	}
}

type fakeError struct {
	msg string
}

func (e *fakeError) Error() string {
	return e.msg
}
