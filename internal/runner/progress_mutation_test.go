package runner

import (
	"testing"
)

func TestProgressCounterMutationBug(t *testing.T) {
	// This test demonstrates the actual bug: when the same ProgressState
	// struct instance is modified across loop iterations, it can cause
	// events to be emitted with incorrect progress values

	runs := 0
	emittedEvents := []Event{}

	// Capture events to see the actual progress values being emitted
	eventEmitter := &eventRecorder{events: emittedEvents}

	results := []string{"completed", "blocked", "completed", "no_tasks"}

	runOnce := func(opts RunOnceOptions, deps RunOnceDeps) (string, error) {
		if runs >= len(results) {
			t.Fatalf("unexpected run %d, only have %d results", runs+1, len(results))
		}

		// Simulate what happens in RunOnce - emit events with the progress state
		// Use the same progress state that will be passed to next iteration
		result := results[runs]

		// Emit events that would normally be emitted during RunOnce
		emitPhase(deps.Events, EventSelectTask, "task-id", "Task Title", opts.Progress)
		emitPhase(deps.Events, EventBeadsUpdate, "task-id", "Task Title", opts.Progress)
		emitPhase(deps.Events, EventOpenCodeStart, "task-id", "Task Title", opts.Progress)
		emitPhase(deps.Events, EventOpenCodeEnd, "task-id", "Task Title", opts.Progress)
		emitPhase(deps.Events, EventGitAdd, "task-id", "Task Title", opts.Progress)
		emitPhase(deps.Events, EventGitStatus, "task-id", "Task Title", opts.Progress)
		emitPhase(deps.Events, EventGitCommit, "task-id", "Task Title", opts.Progress)
		emitPhase(deps.Events, EventBeadsClose, "task-id", "Task Title", opts.Progress)
		emitPhase(deps.Events, EventBeadsVerify, "task-id", "Task Title", opts.Progress)
		emitPhase(deps.Events, EventBeadsSync, "task-id", "Task Title", opts.Progress)

		runs++
		return result, nil
	}

	// Start with a ProgressState - this is the key: the same struct gets modified
	progress := ProgressState{Total: 3}
	deps := RunOnceDeps{Events: eventEmitter}

	count, err := RunLoop(RunOnceOptions{Progress: progress}, deps, 0, runOnce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected count 3, got %d", count)
	}

	// Now check the emitted events for the bug
	// We expect to see events with progress values like [4/3] if the bug exists
	for i, event := range eventEmitter.events {
		if event.ProgressCompleted > event.ProgressTotal {
			t.Fatalf("event %d: BUG found - progress [%d/%d] exceeds total",
				i, event.ProgressCompleted, event.ProgressTotal)
		}
	}
}

func TestProgressCounterSharedStateBug(t *testing.T) {
	// Test the specific scenario where a shared ProgressState gets mutated
	// causing the "over-increment" issue mentioned in the bug report

	runs := 0
	emittedEvents := []Event{}
	eventEmitter := &eventRecorder{events: emittedEvents}

	// This simulates the real scenario where a single ProgressState instance
	// gets passed through multiple loop iterations
	progress := ProgressState{Total: 3}

	results := []string{"completed", "blocked", "completed", "no_tasks"}

	runOnce := func(opts RunOnceOptions, deps RunOnceDeps) (string, error) {
		if runs >= len(results) {
			t.Fatalf("unexpected run %d, only have %d results", runs+1, len(results))
		}

		result := results[runs]

		// Emit multiple phases like the real RunOnce does
		// The key insight: all these events use the same opts.Progress reference
		// which gets mutated in RunLoop after this runOnce call returns
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

	deps := RunOnceDeps{Events: eventEmitter}

	count, err := RunLoop(RunOnceOptions{Progress: progress}, deps, 0, runOnce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected count 3, got %d", count)
	}

	// The bug would manifest if events are emitted with progress values that exceed total
	// or if the progress values don't match the expected pattern
	expectedProgressPerRun := []int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // Run 1: [0/3] for all events
		1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // Run 2: [1/3] for all events
		2, 2, 2, 2, 2, 2, 2, 2, 2, 2, // Run 3: [2/3] for all events
		3, 3, 3, 3, 3, 3, 3, 3, 3, 3} // Run 4 (no_tasks): [3/3] for all events

	if len(eventEmitter.events) != len(expectedProgressPerRun) {
		t.Fatalf("expected %d events, got %d", len(expectedProgressPerRun), len(eventEmitter.events))
	}

	for i, event := range eventEmitter.events {
		expectedCompleted := expectedProgressPerRun[i]
		if event.ProgressCompleted != expectedCompleted {
			t.Fatalf("event %d: expected progress completed %d, got %d",
				i, expectedCompleted, event.ProgressCompleted)
		}
		if event.ProgressTotal != 3 {
			t.Fatalf("event %d: expected progress total 3, got %d",
				i, event.ProgressTotal)
		}
		if event.ProgressCompleted > event.ProgressTotal {
			t.Fatalf("event %d: BUG - progress [%d/%d] exceeds total",
				i, event.ProgressCompleted, event.ProgressTotal)
		}
	}
}
