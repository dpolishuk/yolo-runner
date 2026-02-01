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

func TestProgressLastOutputAgeDoubleSBug(t *testing.T) {
	// Test specifically for the double 's' bug in progress output
	// The bug has been reported where "2ss" appears instead of "2s"

	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "issue-1.jsonl")
	if err := os.WriteFile(logPath, []byte("start"), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	baseTime := time.Date(2026, 1, 20, 10, 0, 0, 0, time.UTC)
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

	// Rapid updates that might trigger race conditions
	for i := 1; i <= 10; i++ {
		t.Run(fmt.Sprintf("rapid_update_%d", i), func(t *testing.T) {
			buffer.Reset()
			current = baseTime.Add(time.Duration(i) * time.Second)
			ticker.Tick(current)
			output := waitForOutput(t, buffer)

			// Extract the last rendered line
			renderedLine := lastRender(output)

			// Look specifically for double 's' pattern
			if strings.Contains(renderedLine, "ss") && !strings.Contains(renderedLine, "ms") {
				// Check if it's specifically in the age part
				agePart := extractAgePart(renderedLine)
				if strings.Contains(agePart, "ss") {
					t.Errorf("found double 's' bug in rapid update %d: age='%s' full='%q'", i, agePart, renderedLine)
				}
			}

			// Also verify the basic format is correct
			expected := fmt.Sprintf("%ds", i)
			if !strings.Contains(renderedLine, "last output "+expected) {
				t.Errorf("incorrect age format in update %d: expected '%s', got '%q'", i, expected, renderedLine)
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

// Test calling SetState rapidly which might cause variable reuse issues
func TestProgressSetStateWithDoubleSBug(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "issue-1.jsonl")

	baseTime := time.Date(2026, 1, 20, 10, 0, 0, 0, time.UTC)
	current := baseTime
	now := func() time.Time { return current }
	buffer := &bytes.Buffer{}
	ticker := newFakeProgressTicker()
	progress := NewProgress(ProgressConfig{
		Writer:  buffer,
		State:   "initial state",
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

	// Set different states rapidly, checking each time
	states := []string{"state 1", "state 2", "opencode running", "final state"}
	for i, state := range states {
		t.Run(fmt.Sprintf("state_change_%d", i), func(t *testing.T) {
			buffer.Reset()
			current = baseTime.Add(time.Duration(i+1) * time.Second)

			progress.SetState(state)
			ticker.Tick(current)
			output := waitForOutput(t, buffer)

			// Extract the last rendered line
			renderedLine := lastRender(output)

			// Check for double 's' bug
			if strings.Contains(renderedLine, "ss") && !strings.Contains(renderedLine, "ms") {
				agePart := extractAgePart(renderedLine)
				if strings.Contains(agePart, "ss") {
					t.Errorf("found double 's' bug after state change to '%s': age='%s' full='%q'", state, agePart, renderedLine)
				}
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
