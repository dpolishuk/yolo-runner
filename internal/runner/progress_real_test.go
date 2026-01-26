package runner

import (
	"bytes"
	"testing"
)

func TestProgressCounterRealScenario(t *testing.T) {
	// Test the real scenario described in bug report
	// Use actual RunOnce function to see if bug manifests

	recorder := &callRecorder{}

	// Create a tree with 3 leaf tasks
	tree := Issue{
		ID:        "root",
		IssueType: "epic",
		Status:    "open",
		Children: []Issue{
			{ID: "task-1", IssueType: "task", Status: "open"},
			{ID: "task-2", IssueType: "task", Status: "open"},
			{ID: "task-3", IssueType: "task", Status: "open"},
		},
	}

	beads := &fakeBeads{
		recorder:   recorder,
		readyIssue: tree,
		treeIssue:  tree,
		showQueue: []Bead{
			{ID: "task-1", Title: "Task 1", Status: "open"},
			{ID: "task-1", Status: "closed"},
			{ID: "task-2", Title: "Task 2", Status: "open"},
			{ID: "task-2", Status: "closed"},
			{ID: "task-3", Title: "Task 3", Status: "open"},
			{ID: "task-3", Status: "closed"},
		},
	}

	eventRecorder := &eventRecorder{}
	output := &bytes.Buffer{}

	deps := RunOnceDeps{
		Beads:    beads,
		Prompt:   &fakePrompt{recorder: recorder, prompt: "PROMPT"},
		OpenCode: &fakeOpenCode{recorder: recorder},
		Git:      &fakeGit{recorder: recorder, dirty: true, rev: "deadbeef"},
		Logger:   &fakeLogger{recorder: recorder},
		Events:   eventRecorder,
	}

	// Run the loop multiple times to see if progress exceeds total
	progress := ProgressState{Total: 3}
	count, err := RunLoop(RunOnceOptions{
		RepoRoot: "/repo",
		RootID:   "root",
		Out:      output,
		Progress: progress,
	}, deps, 0, RunOnce)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected count 3, got %d", count)
	}

	// Check all emitted events for the bug
	for i, event := range eventRecorder.events {
		if event.ProgressCompleted > event.ProgressTotal {
			t.Fatalf("event %d: BUG FOUND - progress [%d/%d] exceeds total!",
				i, event.ProgressCompleted, event.ProgressTotal)
		}
	}

	// Additionally, check that final progress makes sense
	finalEvents := eventRecorder.events
	if len(finalEvents) == 0 {
		t.Fatalf("no events emitted")
	}

	// Look at last few events to see final progress
	lastEvent := finalEvents[len(finalEvents)-1]
	if lastEvent.ProgressCompleted != 3 || lastEvent.ProgressTotal != 3 {
		t.Fatalf("expected final progress [3/3], got [%d/%d]",
			lastEvent.ProgressCompleted, lastEvent.ProgressTotal)
	}
}

func TestProgressCounterPotentialMutationBug(t *testing.T) {
	// Test for potential mutation of shared ProgressState
	// The hypothesis: somehow the same ProgressState struct is being modified
	// in a way that causes events to be emitted with wrong values

	recorder := &callRecorder{}
	eventRecorder := &eventRecorder{}

	// Create a scenario where we deliberately try to trigger the bug
	beads := &fakeBeads{
		recorder:   recorder,
		readyIssue: Issue{ID: "task-1", IssueType: "task", Status: "open"},
		showQueue: []Bead{
			{ID: "task-1", Title: "Task", Status: "open"},
			{ID: "task-1", Status: "closed"},
		},
	}

	deps := RunOnceDeps{
		Beads:    beads,
		Prompt:   &fakePrompt{recorder: recorder, prompt: "PROMPT"},
		OpenCode: &fakeOpenCode{recorder: recorder},
		Git:      &fakeGit{recorder: recorder, dirty: true, rev: "deadbeef"},
		Logger:   &fakeLogger{recorder: recorder},
		Events:   eventRecorder,
	}

	// Test with multiple runs that reuse same progress struct
	progress := ProgressState{Total: 1}

	// Run multiple times to see if any mutation occurs
	for i := 0; i < 3; i++ {
		_, err := RunOnce(RunOnceOptions{
			RepoRoot: "/repo",
			RootID:   "root",
			Out:      &bytes.Buffer{},
			Progress: progress,
		}, deps)

		if err != nil && err.Error() != "no tasks available" {
			t.Fatalf("run %d: unexpected error: %v", i, err)
		}
	}

	// Check for the bug pattern in events
	for i, event := range eventRecorder.events {
		if event.ProgressCompleted > event.ProgressTotal {
			t.Fatalf("event %d: BUG - progress [%d/%d] exceeds total",
				i, event.ProgressCompleted, event.ProgressTotal)
		}

		// Look for the specific [4/3] pattern mentioned in bug report
		if event.ProgressCompleted == 4 && event.ProgressTotal == 3 {
			t.Fatalf("event %d: Found exact bug pattern [4/3]!", i)
		}
	}
}
