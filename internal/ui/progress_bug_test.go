package ui

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProgressLastOutputAgeFormatting(t *testing.T) {
	// Test that "last output age" is formatted correctly without duplicate 's' suffix
	// This test specifically checks for bug where age shows as "2ss" instead of "2s"

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

	// Test various time differences that might trigger bug
	testCases := []struct {
		timeDiff    time.Duration
		expectedAge string
		description string
	}{
		{2 * time.Second, "2s", "simple seconds"},
		{1 * time.Minute, "60s", "minute format"},
		{5*time.Minute + 30*time.Second, "330s", "complex format"},
		{0 * time.Second, "0s", "zero seconds"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			buffer.Reset()
			current = baseTime.Add(tc.timeDiff)
			ticker.Tick(current)
			output := waitForOutput(t, buffer)

			// Extract the last rendered line
			renderedLine := lastRender(output)

			// Check that expected age appears in output
			if !strings.Contains(renderedLine, "last output "+tc.expectedAge) {
				t.Errorf("expected 'last output %s' in output %q, got %q", tc.expectedAge, renderedLine, output)
			}

			// Specifically check for duplicate 's' bug - look for "ss" anywhere in age part
			// Pattern to catch: "2ss", "60ss", "330ss", etc.
			agePart := extractAgePart(renderedLine)
			if strings.Contains(agePart, "ss") && !strings.Contains(agePart, "ms") {
				t.Errorf("found duplicate 's' in age formatting: age='%s' full='%q'", agePart, renderedLine)
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

// extractAgePart extracts just the age portion from a rendered line
// e.g., from "/ opencode running - last output 60s" extracts "60s"
func extractAgePart(line string) string {
	parts := strings.Split(line, "last output ")
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}
