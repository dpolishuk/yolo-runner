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

// TestProgressLastOutputRegressionForDoubleSBug tests the specific double 's' bug reported in yolo-runner-127.6.3
// This is a regression test to ensure the bug doesn't reoccur
func TestProgressLastOutputRegressionForDoubleSBug(t *testing.T) {
	// Regression test for specific issue: last output age showing "2ss" instead of "2s"
	// Issue yolo-runner-127.6.3 reported this exact problem

	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "issue-regression.jsonl")
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

	// Test the exact scenario from bug report
	testCases := []struct {
		name        string
		timeDiff    time.Duration
		expectedAge string
	}{
		{"exactly_2_seconds", 2 * time.Second, "2s"},
		{"exactly_12_seconds", 12 * time.Second, "12s"},
		{"exactly_176_seconds", 176 * time.Second, "176s"},
		{"zero_seconds", 0 * time.Second, "0s"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buffer.Reset()
			current = baseTime.Add(tc.timeDiff)
			ticker.Tick(current)
			output := waitForOutput(t, buffer)

			// Extract the rendered line
			renderedLine := lastRender(output)

			// CRITICAL: Check for double 's' bug - this is the main regression check
			// The bug was reported as showing "2ss", "12ss", "176ss" instead of "2s", "12s", "176s"
			agePart := extractAgePartForRegression(renderedLine)

			// Verify the expected age is present
			expectedFragment := fmt.Sprintf("last output %s", tc.expectedAge)
			if !strings.Contains(renderedLine, expectedFragment) {
				t.Errorf("expected fragment '%s' not found in output: %q", expectedFragment, renderedLine)
			}

			// Look for the specific patterns mentioned in the bug report
			buggyPattern := fmt.Sprintf("%ss", tc.expectedAge) // e.g., "2ss", "12ss", "176ss"
			if strings.Contains(agePart, buggyPattern) {
				t.Errorf("REGRESSION: Double 's' bug detected! Found '%s' instead of '%s' in line: %q",
					buggyPattern, tc.expectedAge, renderedLine)
			}

			// Also check for any unexpected double 's' patterns
			if strings.Contains(agePart, "ss") && !strings.Contains(agePart, "ms") {
				t.Errorf("REGRESSION: Unexpected double 's' pattern found in age '%s' from line: %q", agePart, renderedLine)
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

// extractAgePartForRegression extracts just the age portion from a rendered line
// e.g., from "/ opencode running - last output 60s" extracts "60s"
func extractAgePartForRegression(line string) string {
	parts := strings.Split(line, "last output ")
	if len(parts) < 2 {
		return ""
	}
	agePart := parts[1]
	// Remove any trailing characters that aren't part of the age
	agePart = strings.Trim(agePart, " \t\n\r")
	return agePart
}
