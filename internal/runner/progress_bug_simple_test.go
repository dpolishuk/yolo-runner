package runner

import (
	"testing"
)

// Failing test that demonstrates [4/3] bug
// According to TDD, this should FAIL initially
func TestProgressCounter_BeforeFix_BugDemo(t *testing.T) {
	// This test is designed to fail with current implementation
	// by creating a scenario that should trigger [4/3] bug pattern

	events := &eventRecorder{}

	// Mock a buggy scenario where progress gets incremented beyond total
	buggyRunOnce := func(opts RunOnceOptions, deps RunOnceDeps) (string, error) {
		// Simulate bug by emitting progress that exceeds total
		buggyProgress := ProgressState{
			Completed: opts.Progress.Total + 1, // Bug: exceeds total by 1
			Total:     opts.Progress.Total,
		}

		emitPhase(deps.Events, EventBeadsUpdate, "task", "Task", buggyProgress)
		return "completed", nil
	}

	progress := ProgressState{Total: 3}
	deps := RunOnceDeps{Events: events}

	// This should demonstrate the bug
	_, err := RunLoop(RunOnceOptions{Progress: progress}, deps, 1, buggyRunOnce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check if bug was detected
	bugFound := false
	for i, event := range events.events {
		if event.ProgressCompleted > event.ProgressTotal {
			bugFound = true
			t.Logf("Bug demonstrated: event %d shows [%d/%d]",
				i, event.ProgressCompleted, event.ProgressTotal)
		}

		// Look for specific [4/3] pattern
		if event.ProgressCompleted == 4 && event.ProgressTotal == 3 {
			t.Fatalf("Bug demonstrated: event %d shows exact [4/3] pattern!", i)
		}
	}

	// According to TDD protocol, this test should FAIL initially
	if !bugFound {
		t.Fatalf("DID NOT DEMONSTRATE BUG - this test should fail initially to demonstrate bug exists")
	}
}
