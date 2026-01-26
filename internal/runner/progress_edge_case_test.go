package runner

import (
	"testing"
)

func TestProgressCounterOverIncrementSpecificCase(t *testing.T) {
	// Test the specific case where progress counter exceeds total
	// This test attempts to reproduce the exact bug: [4/3] pattern

	runs := 0

	runOnce := func(opts RunOnceOptions, deps RunOnceDeps) (string, error) {
		if runs >= 4 {
			t.Fatalf("unexpected run %d", runs+1)
		}

		// Emit events with current progress state
		result := "completed"
		if runs < 3 {
			result = "completed"
		} else {
			result = "no_tasks"
		}

		// Emit multiple phases to simulate "multiple updates"
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

	// Start with 3 total tasks
	progress := ProgressState{Total: 3}
	eventRecorder := &eventRecorder{}
	deps := RunOnceDeps{Events: eventRecorder}

	count, err := RunLoop(RunOnceOptions{Progress: progress}, deps, 0, runOnce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected count 3, got %d", count)
	}

	// The critical test: check for [4/3] pattern or any exceeded progress
	for i, event := range eventRecorder.events {
		if event.ProgressCompleted > event.ProgressTotal {
			t.Fatalf("BUG DETECTED: event %d shows progress [%d/%d] which exceeds total!",
				i, event.ProgressCompleted, event.ProgressTotal)
		}

		// Specifically look for the [4/3] pattern mentioned in bug report
		if event.ProgressCompleted == 4 && event.ProgressTotal == 3 {
			t.Fatalf("BUG DETECTED: event %d shows exact bug pattern [4/3]!", i)
		}
	}

	// Final sanity check
	if len(eventRecorder.events) == 0 {
		t.Fatalf("no events were emitted")
	}

	lastEvent := eventRecorder.events[len(eventRecorder.events)-1]
	if lastEvent.ProgressCompleted > lastEvent.ProgressTotal {
		t.Fatalf("BUG DETECTED: final event shows progress [%d/%d] which exceeds total!",
			lastEvent.ProgressCompleted, lastEvent.ProgressTotal)
	}
}

func TestProgressCounterEdgeCaseWithMaxParameter(t *testing.T) {
	// Test scenario where max parameter might cause issues
	runs := 0

	runOnce := func(opts RunOnceOptions, deps RunOnceDeps) (string, error) {
		if runs >= 4 {
			t.Fatalf("unexpected run %d", runs+1)
		}

		result := "completed"
		if runs >= 3 {
			result = "no_tasks"
		}

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

	progress := ProgressState{Total: 3}
	eventRecorder := &eventRecorder{}
	deps := RunOnceDeps{Events: eventRecorder}

	// Set max to 4 (more than total) to see if this causes issues
	count, err := RunLoop(RunOnceOptions{Progress: progress}, deps, 4, runOnce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected count 3, got %d", count)
	}

	// Check for any exceeded progress
	for i, event := range eventRecorder.events {
		if event.ProgressCompleted > event.ProgressTotal {
			t.Fatalf("event %d: progress [%d/%d] exceeds total!",
				i, event.ProgressCompleted, event.ProgressTotal)
		}
	}
}
