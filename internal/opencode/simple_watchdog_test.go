package opencode

import (
	"testing"
	"time"
)

func TestWatchdogWithEmptyLog(t *testing.T) {
	// Create a temporary log file
	logPath := t.TempDir() + "/test.log"

	// Create mock process that never completes
	mockProcess := &MockProcess{
		waitCh: make(chan error), // Never completes
		killCh: make(chan error, 1),
		stdin:  &nopWriteCloser{},
		stdout: &nopReadCloser{},
	}
	mockProcess.killCh <- nil // Kill succeeds

	// Create watchdog with very short timeout
	watchdog := NewWatchdog(WatchdogConfig{
		LogPath:         logPath,
		Timeout:         500 * time.Millisecond,
		Interval:        100 * time.Millisecond,
		CompletionGrace: 50 * time.Millisecond,
		TailLines:       5,
		Now:             time.Now,
		After:           time.After,
		NewTicker: func(d time.Duration) WatchdogTicker {
			return realWatchdogTicker{ticker: time.NewTicker(d)}
		},
	})

	start := time.Now()
	err := watchdog.Monitor(mockProcess)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	// Should timeout relatively quickly (within 2 seconds)
	if elapsed > 2*time.Second {
		t.Fatalf("watchdog took too long to timeout: %v", elapsed)
	}

	if !mockProcess.killCalled {
		t.Fatal("expected process.Kill() to be called")
	}
}
