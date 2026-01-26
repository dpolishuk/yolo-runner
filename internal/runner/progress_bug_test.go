package runner

import (
	"testing"
)

func TestProgressCounterNeverExceedsTotal(t *testing.T) {
	// Test that demonstrates the bug where progress counter can exceed total
	// This test should fail initially, showing the bug exists
	runs := 0
	capture := []ProgressState{}
	results := []string{"completed", "blocked", "completed", "no_tasks"} // 3 tasks total + no_tasks

	// Initial progress state with 3 total tasks
	initialProgress := ProgressState{Total: 3}

	runOnce := func(opts RunOnceOptions, deps RunOnceDeps) (string, error) {
		if runs >= len(results) {
			t.Fatalf("unexpected run %d, only have %d results", runs+1, len(results))
		}
		capture = append(capture, opts.Progress)
		result := results[runs]
		runs++
		return result, nil
	}

	count, err := RunLoop(RunOnceOptions{Progress: initialProgress}, RunOnceDeps{}, 0, runOnce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected count 3, got %d", count)
	}

	// Verify that progress never exceeds total
	for i, progress := range capture {
		if progress.Completed > progress.Total {
			t.Fatalf("iteration %d: BUG: progress completed (%d) exceeds total (%d)", i, progress.Completed, progress.Total)
		}
	}

	// Verify final progress is correct
	finalProgress := capture[len(capture)-1]
	if finalProgress.Completed != 3 || finalProgress.Total != 3 {
		t.Fatalf("expected final progress [3/3], got [%d/%d]", finalProgress.Completed, finalProgress.Total)
	}
}

func TestProgressCounterWithOverIncrementScenario(t *testing.T) {
	// Test the specific scenario mentioned in the bug report
	// where multiple updates cause the counter to exceed total
	runs := 0
	capture := []ProgressState{}

	// Simulate the problematic scenario with 3 leaf tasks
	results := []string{"completed", "blocked", "completed", "no_tasks"} // Should end at [3/3]

	runOnce := func(opts RunOnceOptions, deps RunOnceDeps) (string, error) {
		if runs >= len(results) {
			t.Fatalf("unexpected run %d, only have %d results", runs+1, len(results))
		}
		capture = append(capture, opts.Progress)
		result := results[runs]
		runs++
		return result, nil
	}

	// Start with a ProgressState that might get modified during execution
	initialProgress := ProgressState{Total: 3}

	count, err := RunLoop(RunOnceOptions{Progress: initialProgress}, RunOnceDeps{}, 0, runOnce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected count 3, got %d", count)
	}

	// Check each captured progress state
	expectedProgress := [][]int{{0, 3}, {1, 3}, {2, 3}, {3, 3}} // [completed, total]

	for i, progress := range capture {
		expectedCompleted, expectedTotal := expectedProgress[i][0], expectedProgress[i][1]
		if progress.Completed != expectedCompleted || progress.Total != expectedTotal {
			t.Fatalf("iteration %d: expected progress [%d/%d], got [%d/%d]",
				i, expectedCompleted, expectedTotal, progress.Completed, progress.Total)
		}
		if progress.Completed > progress.Total {
			t.Fatalf("iteration %d: BUG: progress completed (%d) exceeds total (%d)",
				i, progress.Completed, progress.Total)
		}
	}
}

func TestProgressCounterIncrementsOncePerCompletedBlockedLeaf(t *testing.T) {
	// Test that x increments exactly once per completed/blocked leaf task
	runs := 0
	capture := []ProgressState{}

	// Mix of completed and blocked results
	results := []string{"completed", "blocked", "completed", "blocked", "completed", "no_tasks"}

	runOnce := func(opts RunOnceOptions, deps RunOnceDeps) (string, error) {
		if runs >= len(results) {
			t.Fatalf("unexpected run %d, only have %d results", runs+1, len(results))
		}
		capture = append(capture, opts.Progress)
		result := results[runs]
		runs++
		return result, nil
	}

	count, err := RunLoop(RunOnceOptions{Progress: ProgressState{Total: 5}}, RunOnceDeps{}, 0, runOnce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 5 {
		t.Fatalf("expected count 5, got %d", count)
	}

	// Verify that completed increments by exactly 1 each time
	for i := 1; i < len(capture); i++ {
		prevCompleted := capture[i-1].Completed
		currCompleted := capture[i].Completed
		expectedIncrement := 1

		// Check if this run should have incremented (not the final no_tasks run)
		if i < len(results) && results[i-1] != "no_tasks" {
			if currCompleted-prevCompleted != expectedIncrement {
				t.Fatalf("iteration %d: expected increment of %d, got increment of %d (%d -> %d)",
					i, expectedIncrement, currCompleted-prevCompleted, prevCompleted, currCompleted)
			}
		}
	}

	// Final check - should never exceed total
	for i, progress := range capture {
		if progress.Completed > progress.Total {
			t.Fatalf("iteration %d: progress exceeded total: [%d/%d]", i, progress.Completed, progress.Total)
		}
	}
}
