package runner

import (
	"testing"
)

func TestProgressCounterTotalChangingBug(t *testing.T) {
	// This test simulates a scenario where the total number of runnable leaves
	// changes during execution, which could cause the progress counter logic
	// to become inconsistent

	runs := 0
	var events []Event
	eventEmitter := &eventRecorder{events: events}

	// Mock beads that returns different tree structures on each call
	treeCalls := 0
	beads := &fakeBeads{
		recorder: nil,
		treeIssue: Issue{
			ID:        "root",
			IssueType: "epic",
			Status:    "open",
			Children: []Issue{
				{ID: "task-1", IssueType: "task", Status: "open"},
				{ID: "task-2", IssueType: "task", Status: "open"},
				{ID: "task-3", IssueType: "task", Status: "open"},
			},
		},
	}

	results := []string{"completed", "blocked", "completed", "no_tasks"}

	runOnce := func(opts RunOnceOptions, deps RunOnceDeps) (string, error) {
		if runs >= len(results) {
			t.Fatalf("unexpected run %d, only have %d results", runs+1, len(results))
		}

		// Simulate tree structure changing between runs
		// For example, tasks might get re-evaluated as runnable/non-runnable
		if treeCalls == 0 {
			// First call: 3 runnable leaves
			deps.Beads.(*fakeBeads).treeIssue = Issue{
				ID:        "root",
				IssueType: "epic",
				Status:    "open",
				Children: []Issue{
					{ID: "task-1", IssueType: "task", Status: "open"},
					{ID: "task-2", IssueType: "task", Status: "open"},
					{ID: "task-3", IssueType: "task", Status: "open"},
				},
			}
		} else if treeCalls == 1 {
			// Second call: maybe only 2 runnable leaves now
			deps.Beads.(*fakeBeads).treeIssue = Issue{
				ID:        "root",
				IssueType: "epic",
				Status:    "open",
				Children: []Issue{
					{ID: "task-2", IssueType: "task", Status: "open"},
					{ID: "task-3", IssueType: "task", Status: "open"},
				},
			}
		}

		treeCalls++

		// Emit events with current progress
		result := results[runs]

		emitPhase(deps.Events, EventSelectTask, "task", "Task", opts.Progress)
		emitPhase(deps.Events, EventBeadsUpdate, "task", "Task", opts.Progress)
		emitPhase(deps.Events, EventOpenCodeStart, "task", "Task", opts.Progress)
		emitPhase(deps.Events, EventOpenCodeEnd, "task", "Task", opts.Progress)
		emitPhase(deps.Events, EventGitAdd, "task", "Task", opts.Progress)
		emitPhase(deps.Events, EventGitStatus, "task", "Task", opts.Progress)
		emitPhase(deps.Events, EventGitCommit, "task", "Task", opts.Progress)
		emitPhase(deps.Events, EventBeadsClose, "task", "Task", opts.Progress)
		emitPhase(deps.Events, EventBeadsVerify, "task", "Task", opts.Progress)
		emitPhase(deps.Events, EventBeadsSync, "task", "Task", opts.Progress)

		runs++
		return result, nil
	}

	// Start with initial progress state
	progress := ProgressState{Total: 3}
	deps := RunOnceDeps{Events: eventEmitter, Beads: beads}

	count, err := RunLoop(RunOnceOptions{Progress: progress, RepoRoot: "/repo", RootID: "root"}, deps, 0, runOnce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected count 3, got %d", count)
	}

	// Check for the bug: progress completed exceeding total
	for i, event := range events {
		if event.ProgressCompleted > event.ProgressTotal {
			t.Fatalf("event %d: BUG - progress [%d/%d] exceeds total",
				i, event.ProgressCompleted, event.ProgressTotal)
		}
	}
}

func TestProgressCounterRaceConditionWithSharedProgress(t *testing.T) {
	// Test exact scenario described in bug report:
	// Run runner on root with 3 leaf tasks and observe progress display after multiple updates

	runs := 0
	eventRecorder := &eventRecorder{}
	eventRecorder.events = []Event{}

	// Set up a scenario with 3 leaf tasks
	results := []string{"completed", "blocked", "completed", "no_tasks"} // 3 leaf tasks

	runOnce := func(opts RunOnceOptions, deps RunOnceDeps) (string, error) {
		if runs >= len(results) {
			t.Fatalf("unexpected run %d, only have %d results", runs+1, len(results))
		}

		result := results[runs]

		// Emit the same sequence of events that happens in real RunOnce
		// This represents the "multiple updates" mentioned in bug report
		emitPhase(deps.Events, EventSelectTask, "task-1", "Task 1", opts.Progress)
		emitPhase(deps.Events, EventBeadsUpdate, "task-1", "Task 1", opts.Progress)
		emitPhase(deps.Events, EventOpenCodeStart, "task-1", "Task 1", opts.Progress)
		emitPhase(deps.Events, EventOpenCodeEnd, "task-1", "Task 1", opts.Progress)
		emitPhase(deps.Events, EventGitAdd, "task-1", "Task 1", opts.Progress)
		emitPhase(deps.Events, EventGitStatus, "task-1", "Task 1", opts.Progress)
		emitPhase(deps.Events, EventGitCommit, "task-1", "Task 1", opts.Progress)
		emitPhase(deps.Events, EventBeadsClose, "task-1", "Task 1", opts.Progress)
		emitPhase(deps.Events, EventBeadsVerify, "task-1", "Task 1", opts.Progress)
		emitPhase(deps.Events, EventBeadsSync, "task-1", "Task 1", opts.Progress)

		runs++
		return result, nil
	}

	// Start with progress showing 3 total tasks
	progress := ProgressState{Total: 3}
	deps := RunOnceDeps{Events: eventRecorder}

	count, err := RunLoop(RunOnceOptions{Progress: progress}, deps, 0, runOnce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected count 3, got %d", count)
	}

	// The bug would manifest as events showing progress like [4/3]
	// Let's check the exact pattern that should emerge:
	eventsPerRun := 10
	expectedPattern := []struct {
		completed int
		total     int
	}{
		{0, 3}, // Run 1 - all 10 events should show [0/3]
		{1, 3}, // Run 2 - all 10 events should show [1/3]
		{2, 3}, // Run 3 - all 10 events should show [2/3]
		{3, 3}, // Run 4 - all 10 events should show [3/3]
	}

	if len(eventRecorder.events) != eventsPerRun*len(expectedPattern) {
		t.Fatalf("expected %d events, got %d", eventsPerRun*len(expectedPattern), len(eventRecorder.events))
	}

	eventIdx := 0
	for run, expected := range expectedPattern {
		for phase := 0; phase < eventsPerRun; phase++ {
			event := eventRecorder.events[eventIdx]
			if event.ProgressCompleted != expected.completed {
				t.Fatalf("run %d phase %d: expected completed %d, got %d",
					run, phase, expected.completed, event.ProgressCompleted)
			}
			if event.ProgressTotal != expected.total {
				t.Fatalf("run %d phase %d: expected total %d, got %d",
					run, phase, expected.total, event.ProgressTotal)
			}
			if event.ProgressCompleted > event.ProgressTotal {
				t.Fatalf("run %d phase %d: BUG DETECTED - progress [%d/%d] exceeds total",
					run, phase, event.ProgressCompleted, event.ProgressTotal)
			}
			eventIdx++
		}
	}
}
