package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRunMainRendersMonitorViewFromEventsFile(t *testing.T) {
	tempDir := t.TempDir()
	eventsPath := filepath.Join(tempDir, "events.jsonl")
	content := "{\"type\":\"task_started\",\"task_id\":\"task-1\",\"task_title\":\"Readable task\",\"message\":\"started\",\"ts\":\"2026-02-10T12:00:00Z\"}\n" +
		"{\"type\":\"runner_finished\",\"task_id\":\"task-1\",\"message\":\"done\",\"ts\":\"2026-02-10T12:00:05Z\"}\n"
	if err := os.WriteFile(eventsPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write events: %v", err)
	}

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	code := RunMain([]string{"--events", eventsPath}, out, errOut)
	if code != 0 {
		t.Fatalf("expected code 0, got %d stderr=%q", code, errOut.String())
	}
	if out.String() == "" {
		t.Fatalf("expected rendered view")
	}
	if !contains(out.String(), "Current Task: task-1 - Readable task") {
		t.Fatalf("expected current task in output, got %q", out.String())
	}
}

func contains(text string, sub string) bool {
	for i := 0; i+len(sub) <= len(text); i++ {
		if text[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
