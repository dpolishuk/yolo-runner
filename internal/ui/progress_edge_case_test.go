package ui

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Test for potential edge cases that might cause double 's' in age formatting
func TestProgressEdgeCasesForDoubleSBug(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "issue-edge.jsonl")

	baseTime := time.Date(2026, 1, 20, 10, 0, 0, 0, time.UTC)
	current := baseTime
	now := func() time.Time { return current }
	buffer := &bytes.Buffer{}
	ticker := newFakeProgressTicker()
	progress := NewProgress(ProgressConfig{
		Writer:  buffer,
		State:   "opencode running",
		LogPath: logPath,
		Ticker:  ticker,
		Now:     now,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		progress.Run(ctx)
		close(done)
	}()

	// Test edge case: zero time
	t.Run("zero_time_edge", func(t *testing.T) {
		buffer.Reset()
		current = time.Time{} // Zero time
		ticker.Tick(current)
		output := waitForOutput(t, buffer)
		renderedLine := lastRender(output)

		// Should show "n/a" not cause any formatting issues
		if strings.Contains(renderedLine, "ss") {
			t.Errorf("found double 's' with zero time: %q", renderedLine)
		}
	})

	// Test edge case: negative duration (should be clamped to 0)
	t.Run("negative_duration_edge", func(t *testing.T) {
		buffer.Reset()
		// Set up scenario where time calculation might go negative
		current = baseTime.Add(-1 * time.Second)
		ticker.Tick(current)
		output := waitForOutput(t, buffer)
		renderedLine := lastRender(output)

		if strings.Contains(renderedLine, "ss") {
			t.Errorf("found double 's' with negative time: %q", renderedLine)
		}

		// Should show "0s" not negative
		if !strings.Contains(renderedLine, "last output 0s") {
			t.Errorf("expected 0s for negative time, got: %q", renderedLine)
		}
	})

	// Test edge case: very large time value
	t.Run("large_time_edge", func(t *testing.T) {
		buffer.Reset()
		current = baseTime.Add(999999 * time.Second) // Very large
		ticker.Tick(current)
		output := waitForOutput(t, buffer)
		renderedLine := lastRender(output)

		if strings.Contains(renderedLine, "ss") {
			t.Errorf("found double 's' with large time: %q", renderedLine)
		}

		// Should format correctly without overflow issues
		expected := "999999s"
		if !strings.Contains(renderedLine, "last output "+expected) {
			t.Errorf("expected %s for large time, got: %q", expected, renderedLine)
		}
	})

	cancel()
	ticker.Stop()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("progress did not stop")
	}
}

// Test potential race conditions in age formatting
func TestProgressRaceConditionDoubleSBug(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "issue-race.jsonl")

	baseTime := time.Date(2026, 1, 20, 10, 0, 0, 0, time.UTC)
	if err := os.WriteFile(logPath, []byte("start"), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}
	if err := os.Chtimes(logPath, baseTime, baseTime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	current := baseTime
	now := func() time.Time { return current }
	buffer := &bytes.Buffer{}
	ticker := newFakeProgressTicker()
	progress := NewProgress(ProgressConfig{
		Writer:  buffer,
		State:   "opencode running",
		LogPath: logPath,
		Ticker:  ticker,
		Now:     now,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		progress.Run(ctx)
		close(done)
	}()

	// Rapid state changes combined with time updates to trigger potential races
	for i := 0; i < 20; i++ {
		t.Run(fmt.Sprintf("race_test_%d", i), func(t *testing.T) {
			buffer.Reset()

			// Change state rapidly
			stateName := fmt.Sprintf("state_%d", i)
			progress.SetState(stateName)

			// Immediately update time
			current = baseTime.Add(time.Duration(i+1) * time.Second)
			ticker.Tick(current)
			output := waitForOutput(t, buffer)
			renderedLine := lastRender(output)

			// Check for double 's' bug
			if strings.Contains(renderedLine, "ss") && !strings.Contains(renderedLine, "ms") {
				t.Errorf("found double 's' in race condition test %d: %q", i, renderedLine)
			}

			// Verify state is present
			if !strings.Contains(renderedLine, stateName) {
				t.Errorf("state not found in rendered line %d: %q", i, renderedLine)
			}
		})
	}

	cancel()
	ticker.Stop()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("progress did not stop")
	}
}
