package opencode

import (
	"bufio"
	"context"
	"os"
	"strings"
	"time"
)

// monitorInitFailures monitors stderr logs for Serena initialization failures
// and cancels the context if detected
func monitorInitFailures(ctx context.Context, stderrPath string, cancel context.CancelFunc) {
	if stderrPath == "" {
		return
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if hasSerenaInitError(stderrPath) {
				writeConsoleLine(os.Stderr, "Serena language server initialization failure detected")
				cancel()
				return
			}
		}
	}
}

// hasSerenaInitError checks if stderr log contains Serena initialization error
func hasSerenaInitError(stderrPath string) bool {
	file, err := os.Open(stderrPath)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "language server manager is not initialized") {
			return true
		}
	}
	return false
}
