package runner

import (
	"testing"
)

// Test acceptance criteria directly
func TestProgressCounter_AcceptanceCriteria(t *testing.T) {
	// Test 1: Progress [x/y] never exceeds y
	t.Run("NeverExceedsTotal", func(t *testing.T) {
		eventRecorder := &eventRecorder{}

		runOnce := func(opts RunOnceOptions, deps RunOnceDeps) (string, error) {
			emitPhase(deps.Events, EventBeadsUpdate, "task", "Task", opts.Progress)
			return "completed", nil
		}

		progress := ProgressState{Total: 3}
		deps := RunOnceDeps{Events: eventRecorder}

		_, err := RunLoop(RunOnceOptions{Progress: progress}, deps, 3, runOnce)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for i, event := range eventRecorder.events {
			if event.ProgressCompleted > event.ProgressTotal {
				t.Fatalf("event %d: progress [%d/%d] exceeds total",
					i, event.ProgressCompleted, event.ProgressTotal)
			}
		}
	})

	// Test 2: x increments once per completed/blocked leaf
	t.Run("IncrementsOncePerCompleted", func(t *testing.T) {
		eventRecorder := &eventRecorder{}

		runOnce := func(opts RunOnceOptions, deps RunOnceDeps) (string, error) {
			emitPhase(deps.Events, EventBeadsUpdate, "task", "Task", opts.Progress)
			return "completed", nil
		}

		progress := ProgressState{Total: 4}
		deps := RunOnceDeps{Events: eventRecorder}

		count, err := RunLoop(RunOnceOptions{Progress: progress}, deps, 4, runOnce)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 4 {
			t.Fatalf("expected 4 completed, got %d", count)
		}

		// Should see exactly one increment per run: [0/4], [1/4], [2/4], [3/4], [4/4]
		expectedProgress := []int{0, 1, 2, 3, 4}
		if len(eventRecorder.events) != len(expectedProgress) {
			t.Fatalf("expected %d events, got %d", len(expectedProgress), len(eventRecorder.events))
		}

		for i, event := range eventRecorder.events {
			if event.ProgressCompleted != expectedProgress[i] {
				t.Fatalf("event %d: expected progress [%d/4], got [%d/4]",
					i, expectedProgress[i], event.ProgressCompleted)
			}
		}
	})

	// Test 3: y is total runnable leaves under root
	t.Run("TotalIsRunnableLeaves", func(t *testing.T) {
		tree := Issue{
			ID:        "root",
			IssueType: "epic",
			Status:    "open",
			Children: []Issue{
				{ID: "task-1", IssueType: "task", Status: "open"},          // runnable
				{ID: "task-2", IssueType: "task", Status: "open"},          // runnable
				{ID: "task-3", IssueType: "task", Status: "open"},          // runnable
				{ID: "container-1", IssueType: "molecule", Status: "open"}, // not runnable (container)
			},
		}

		expectedTotal := CountRunnableLeaves(tree)
		if expectedTotal != 3 {
			t.Fatalf("expected 3 runnable leaves, got %d", expectedTotal)
		}

		eventRecorder := &eventRecorder{}

		runOnce := func(opts RunOnceOptions, deps RunOnceDeps) (string, error) {
			emitPhase(deps.Events, EventBeadsUpdate, "task", "Task", opts.Progress)
			return "completed", nil
		}

		progress := ProgressState{Total: expectedTotal}
		deps := RunOnceDeps{Events: eventRecorder}

		_, err := RunLoop(RunOnceOptions{Progress: progress}, deps, 3, runOnce)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// All events should have total = 3
		for i, event := range eventRecorder.events {
			if event.ProgressTotal != 3 {
				t.Fatalf("event %d: expected total 3 (runnable leaves), got %d",
					i, event.ProgressTotal)
			}
		}
	})
}
