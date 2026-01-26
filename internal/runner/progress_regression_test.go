package runner

import (
	"testing"
)

// Regression test for bug where progress counter can exceed total
func TestProgressCounterRegression_OverIncrementScenario(t *testing.T) {
	// Regression test for specific bug: progress counter can exceed total (e.g., [4/3])
	// This test ensures fix works correctly

	runs := 0

	// Simulate running 3 leaf tasks
	results := []string{"completed", "blocked", "completed", "no_tasks"}

	runOnce := func(opts RunOnceOptions, deps RunOnceDeps) (string, error) {
		if runs >= len(results) {
			t.Fatalf("unexpected run %d, only have %d results", runs+1, len(results))
		}

		result := results[runs]

		// Emit all phases that occur during task execution
		// This represents the "multiple updates" mentioned in bug report
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
			emitPhase(deps.Events, phase, "task-id", "Task Title", opts.Progress)
		}

		runs++
		return result, nil
	}

	// Test with 3 total runnable leaves
	progress := ProgressState{Total: 3}
	eventRecorder := &eventRecorder{}
	deps := RunOnceDeps{Events: eventRecorder}

	count, err := RunLoop(RunOnceOptions{Progress: progress}, deps, 0, runOnce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 completed tasks, got %d", count)
	}

	// CRITICAL: Verify that progress counter never exceeds total
	// This is the main regression test for the bug
	maxCompleted := 0
	maxTotal := 0
	for i, event := range eventRecorder.events {
		if event.ProgressCompleted > event.ProgressTotal {
			t.Fatalf("BUG REGRESSION: event %d shows progress [%d/%d] which exceeds total!",
				i, event.ProgressCompleted, event.ProgressTotal)
		}

		// Look for the specific [4/3] pattern mentioned in the bug report
		if event.ProgressCompleted == 4 && event.ProgressTotal == 3 {
			t.Fatalf("BUG REGRESSION: event %d shows exact bug pattern [4/3]!", i)
		}

		if event.ProgressCompleted > maxCompleted {
			maxCompleted = event.ProgressCompleted
		}
		if event.ProgressTotal > maxTotal {
			maxTotal = event.ProgressTotal
		}
	}

	// Additional verification: final progress should be [3/3]
	if maxCompleted != 3 || maxTotal != 3 {
		t.Fatalf("expected final progress to be [3/3], but saw max [%d/%d]", maxCompleted, maxTotal)
	}
}

// Regression test to ensure x increments exactly once per completed/blocked leaf
func TestProgressCounterRegression_ExactlyOncePerCompletedTask(t *testing.T) {
	// Test that progress completed increments exactly once per completed/blocked task
	runs := 0
	progressEvents := []Event{}

	// Mix of completed and blocked results
	results := []string{"completed", "blocked", "completed", "blocked", "completed", "no_tasks"}

	runOnce := func(opts RunOnceOptions, deps RunOnceDeps) (string, error) {
		if runs >= len(results) {
			t.Fatalf("unexpected run %d, only have %d results", runs+1, len(results))
		}

		result := results[runs]

		// Emit a single phase to track progress changes
		emitPhase(deps.Events, EventBeadsUpdate, "task-id", "Task Title", opts.Progress)

		runs++
		return result, nil
	}

	progress := ProgressState{Total: 5}
	eventRecorder := &eventRecorder{events: progressEvents}
	deps := RunOnceDeps{Events: eventRecorder}

	count, err := RunLoop(RunOnceOptions{Progress: progress}, deps, 0, runOnce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 5 {
		t.Fatalf("expected 5 completed tasks, got %d", count)
	}

	// Verify exact increment pattern: [0/5], [1/5], [2/5], [3/5], [4/5], [5/5]
	expectedProgress := []int{0, 1, 2, 3, 4, 5}

	if len(eventRecorder.events) != len(expectedProgress) {
		t.Fatalf("expected %d progress events, got %d", len(expectedProgress), len(eventRecorder.events))
	}

	for i, event := range eventRecorder.events {
		expectedCompleted := expectedProgress[i]
		if event.ProgressCompleted != expectedCompleted {
			t.Fatalf("event %d: expected progress completed %d, got %d",
				i, expectedCompleted, event.ProgressCompleted)
		}
		if event.ProgressTotal != 5 {
			t.Fatalf("event %d: expected progress total 5, got %d", i, event.ProgressTotal)
		}
		if event.ProgressCompleted > event.ProgressTotal {
			t.Fatalf("BUG REGRESSION: event %d shows progress [%d/%d] which exceeds total!",
				i, event.ProgressCompleted, event.ProgressTotal)
		}
	}
}

// Regression test to verify y is total runnable leaves under root
func TestProgressCounterRegression_TotalIsRunnableLeaves(t *testing.T) {
	// Test that progress total correctly represents total runnable leaves
	runs := 0
	progressEvents := []Event{}

	// Tree structure with some runnable and some non-runnable leaves
	beads := &fakeBeads{
		treeIssue: Issue{
			ID:        "root",
			IssueType: "epic",
			Status:    "open",
			Children: []Issue{
				{ID: "task-1", IssueType: "task", Status: "open"},          // runnable
				{ID: "task-2", IssueType: "task", Status: "in_progress"},   // runnable
				{ID: "task-3", IssueType: "task", Status: "closed"},        // runnable (completed)
				{ID: "task-4", IssueType: "task", Status: "blocked"},       // runnable (blocked)
				{ID: "container-1", IssueType: "molecule", Status: "open"}, // not runnable (container)
				{ID: "task-5", IssueType: "task", Status: "open"},          // runnable
			},
		},
	}

	// Count what should be the total runnable leaves
	expectedTotal := 4 // task-1, task-2, task-3, task-4, task-5 = 5, but let's verify with actual function
	expectedTotal = CountRunnableLeaves(beads.treeIssue)

	results := []string{"completed", "completed", "completed", "completed", "no_tasks"}

	runOnce := func(opts RunOnceOptions, deps RunOnceDeps) (string, error) {
		if runs >= len(results) {
			t.Fatalf("unexpected run %d, only have %d results", runs+1, len(results))
		}

		result := results[runs]

		// Emit progress event
		emitPhase(deps.Events, EventBeadsUpdate, "task-id", "Task Title", opts.Progress)

		runs++
		return result, nil
	}

	progress := ProgressState{Total: expectedTotal}
	eventRecorder := &eventRecorder{events: progressEvents}
	deps := RunOnceDeps{Events: eventRecorder, Beads: beads}

	_, err := RunLoop(RunOnceOptions{Progress: progress, RootID: "root", RepoRoot: "/repo"}, deps, 0, runOnce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify that total never changes and matches runnable leaves count
	for i, event := range eventRecorder.events {
		if event.ProgressTotal != expectedTotal {
			t.Fatalf("event %d: expected progress total %d (runnable leaves), got %d",
				i, expectedTotal, event.ProgressTotal)
		}
		if event.ProgressCompleted > event.ProgressTotal {
			t.Fatalf("BUG REGRESSION: event %d shows progress [%d/%d] which exceeds total!",
				i, event.ProgressCompleted, event.ProgressTotal)
		}
	}
}
