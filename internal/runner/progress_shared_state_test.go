package runner

import (
	"testing"
)

func TestProgressCounterSharedStructBug(t *testing.T) {
	// This test demonstrates the actual bug:
	// When the same ProgressState struct instance is passed to RunLoop,
	// and RunLoop modifies it by setting current.Progress.Completed = completed,
	// the same struct instance gets modified across loop iterations.
	// If events are emitted later with stale references, they can show wrong values.

	runs := 0
	var capturedProgressRefs []ProgressState

	results := []string{"completed", "blocked", "completed", "no_tasks"}

	runOnce := func(opts RunOnceOptions, deps RunOnceDeps) (string, error) {
		if runs >= len(results) {
			t.Fatalf("unexpected run %d, only have %d results", runs+1, len(results))
		}

		// Capture the actual ProgressState being passed to this run
		// This should be a copy, but let's see what we actually get
		capturedProgressRefs = append(capturedProgressRefs, opts.Progress)

		result := results[runs]
		runs++
		return result, nil
	}

	// Use a single ProgressState instance - this is the key to reproduce the bug
	progress := ProgressState{Total: 3}

	count, err := RunLoop(RunOnceOptions{Progress: progress}, RunOnceDeps{}, 0, runOnce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected count 3, got %d", count)
	}

	// Let's examine what progress values were actually passed to each run
	expectedCompleted := []int{0, 1, 2, 3} // What we expect to see

	for i, capturedProgress := range capturedProgressRefs {
		expected := expectedCompleted[i]
		actual := capturedProgress.Completed
		if actual != expected {
			t.Fatalf("run %d: expected progress completed %d, got %d", i, expected, actual)
		}
		if capturedProgress.Completed > capturedProgress.Total {
			t.Fatalf("run %d: BUG - progress [%d/%d] exceeds total",
				i, capturedProgress.Completed, capturedProgress.Total)
		}
	}
}

func TestProgressCounterWithEventsRaceCondition(t *testing.T) {
	// Test scenario where events are emitted with progress state
	// that might be getting modified concurrently

	runs := 0
	var events []Event
	eventEmitter := &eventRecorder{events: events}

	results := []string{"completed", "blocked", "completed", "no_tasks"}

	runOnce := func(opts RunOnceOptions, deps RunOnceDeps) (string, error) {
		if runs >= len(results) {
			t.Fatalf("unexpected run %d, only have %d results", runs+1, len(results))
		}

		// Emit events with the progress state that was passed in
		// This simulates what happens in real RunOnce
		result := results[runs]

		// Emit various phases - all use the same opts.Progress
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

	// Single shared ProgressState instance
	progress := ProgressState{Total: 3}
	deps := RunOnceDeps{Events: eventEmitter}

	count, err := RunLoop(RunOnceOptions{Progress: progress}, deps, 0, runOnce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected count 3, got %d", count)
	}

	// The potential bug: if emitPhase somehow captures a reference to the progress
	// state instead of copying it, we might see wrong values
	for i, event := range events {
		if event.ProgressCompleted > event.ProgressTotal {
			t.Fatalf("event %d: BUG - progress [%d/%d] exceeds total",
				i, event.ProgressCompleted, event.ProgressTotal)
		}
	}

	// Verify that all events within a single run have the same progress value
	eventsPerRun := 10                 // Number of phases emitted per run
	if len(events) != eventsPerRun*4 { // 4 runs total
		t.Fatalf("expected %d events, got %d", eventsPerRun*4, len(events))
	}

	for run := 0; run < 4; run++ {
		startIdx := run * eventsPerRun
		endIdx := startIdx + eventsPerRun
		expectedProgress := run

		for i := startIdx; i < endIdx; i++ {
			if events[i].ProgressCompleted != expectedProgress {
				t.Fatalf("run %d event %d: expected progress %d, got %d",
					run, i-startIdx, expectedProgress, events[i].ProgressCompleted)
			}
		}
	}
}
