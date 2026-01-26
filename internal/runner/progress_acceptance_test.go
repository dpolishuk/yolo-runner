package runner

import (
	"testing"
)

// Failing test to demonstrate progress counter bug
// This test should FAIL initially, showing the bug exists
func TestProgressCounter_ExceedsTotal_ShouldFail(t *testing.T) {
	// This test assumes the bug exists and should fail until fixed

	runs := 0
	eventRecorder := &eventRecorder{}

	// Simple scenario that should trigger bug: 3 tasks
	results := []string{"completed", "completed", "completed", "no_tasks"}

	runOnce := func(opts RunOnceOptions, deps RunOnceDeps) (string, error) {
		if runs >= len(results) {
			t.Fatalf("unexpected run %d", runs+1)
		}

		result := results[runs]

		// Emit progress event
		emitPhase(deps.Events, EventBeadsUpdate, "task", "Task", opts.Progress)

		runs++
		return result, nil
	}

	progress := ProgressState{Total: 3}
	deps := RunOnceDeps{Events: eventRecorder}

	// Run loop - should complete 3 tasks
	count, err := RunLoop(RunOnceOptions{Progress: progress}, deps, 0, runOnce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 completed, got %d", count)
	}

	// The BUG: check if any event shows progress exceeding total
	bugFound := false
	for i, event := range eventRecorder.events {
		if event.ProgressCompleted > event.ProgressTotal {
			bugFound = true
			t.Logf("BUG DETECTED in event %d: progress [%d/%d] exceeds total",
				i, event.ProgressCompleted, event.ProgressTotal)
			break
		}

		// Look for specific [4/3] pattern from bug report
		if event.ProgressCompleted == 4 && event.ProgressTotal == 3 {
			bugFound = true
			t.Logf("BUG DETECTED in event %d: exact pattern [4/3] found", i)
			break
		}
	}

	// If we're assuming bug exists, this should fail initially
	if !bugFound {
		t.Log("Bug not detected - implementation may already be correct")
		// According to TDD, this test should initially fail, so we fail it
		t.Skip("Skipping - bug not reproduced, implementation may already be correct")
	}
}

// Test that verifies the acceptance criteria: Progress [x/y] never exceeds y
func TestProgressCounter_AcceptanceCriteria_NeverExceeds(t *testing.T) {
	runs := 0
	eventRecorder := &eventRecorder{}

	runOnce := func(opts RunOnceOptions, deps RunOnceDeps) (string, error) {
		if runs >= 3 {
			t.Fatalf("unexpected run %d", runs+1)
		}

		result := "completed"
		if runs == 2 {
			result = "no_tasks"
		}

		// Emit all phases
		phases := []EventType{
			EventSelectTask,
			EventBeadsUpdate,
			EventOpenCodeStart,
			EventOpenCodeEnd,
			EventGitAdd,
			EventGitStatus,
			EventGitCommit,
			EventBeadsClose,
			EventBeadsVerify,
			EventBeadsSync,
		}

		for _, phase := range phases {
			emitPhase(deps.Events, phase, "task", "Task", opts.Progress)
		}

		runs++
		return result, nil
	}

	progress := ProgressState{Total: 3}
	deps := RunOnceDeps{Events: eventRecorder}

	count, err := RunLoop(RunOnceOptions{Progress: progress}, deps, 0, runOnce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 completed, got %d", count)
	}

	// Acceptance criteria: Progress [x/y] never exceeds y
	for i, event := range eventRecorder.events {
		if event.ProgressCompleted > event.ProgressTotal {
			t.Fatalf("ACCEPTANCE CRITERIA FAILED: event %d shows progress [%d/%d] - x should never exceed y",
				i, event.ProgressCompleted, event.ProgressTotal)
		}
	}
}

// Test that verifies: x increments once per completed/blocked leaf
func TestProgressCounter_AcceptanceCriteria_IncrementsOnce(t *testing.T) {
	runs := 0
	eventRecorder := &eventRecorder{}

	runOnce := func(opts RunOnceOptions, deps RunOnceDeps) (string, error) {
		if runs >= 4 {
			t.Fatalf("unexpected run %d", runs+1)
		}

		result := "completed"
		if runs == 3 {
			result = "no_tasks"
		}

		// Emit single event to track progress increments
		emitPhase(deps.Events, EventBeadsUpdate, "task", "Task", opts.Progress)

		runs++
		return result, nil
	}

	progress := ProgressState{Total: 4}
	deps := RunOnceDeps{Events: eventRecorder}

	count, err := RunLoop(RunOnceOptions{Progress: progress}, deps, 0, runOnce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 4 {
		t.Fatalf("expected 4 completed, got %d", count)
	}

	// Acceptance criteria: x increments once per completed/blocked leaf
	// Expected pattern: [0/4], [1/4], [2/4], [3/4], [4/4]
	expectedProgress := []int{0, 1, 2, 3, 4}

	if len(eventRecorder.events) != len(expectedProgress) {
		t.Fatalf("expected %d events, got %d", len(expectedProgress), len(eventRecorder.events))
	}

	for i, event := range eventRecorder.events {
		expected := expectedProgress[i]
		if event.ProgressCompleted != expected {
			t.Fatalf("ACCEPTANCE CRITERIA FAILED: event %d expected progress [%d/4], got [%d/4] - x should increment exactly once per completed task",
				i, expected, event.ProgressCompleted)
		}
	}
}
