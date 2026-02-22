package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogBrowserRendersTaskAndLogFileList(t *testing.T) {
	tmpDir := t.TempDir()

	taskOne := filepath.Join(tmpDir, "runner-logs", "opencode", "task-1.jsonl")
	taskTwo := filepath.Join(tmpDir, "runner-logs", "opencode", "task-2.jsonl")
	if err := os.MkdirAll(filepath.Dir(taskOne), 0o755); err != nil {
		t.Fatalf("mkdirall task log dir: %v", err)
	}
	if err := os.WriteFile(taskOne, []byte("task-1 log line\n"), 0o600); err != nil {
		t.Fatalf("write task-1 log: %v", err)
	}
	if err := os.WriteFile(taskTwo, []byte("task-2 log line\n"), 0o600); err != nil {
		t.Fatalf("write task-2 log: %v", err)
	}

	browser, err := NewLogBrowser(filepath.Join(tmpDir, "runner-logs"))
	if err != nil {
		t.Fatalf("new log browser: %v", err)
	}

	view := browser.View()
	if !strings.Contains(view, "Tasks and Log Files:") {
		t.Fatalf("expected task list heading, got %q", view)
	}
	if !strings.Contains(view, "task-1") {
		t.Fatalf("expected task-1 in list, got %q", view)
	}
	if !strings.Contains(view, "task-2") {
		t.Fatalf("expected task-2 in list, got %q", view)
	}
	if !strings.Contains(view, "task-1.jsonl") {
		t.Fatalf("expected task-1 log file in list, got %q", view)
	}
	if !strings.Contains(view, "task-2.jsonl") {
		t.Fatalf("expected task-2 log file in list, got %q", view)
	}
	if !strings.Contains(view, "task-1 log line") {
		t.Fatalf("expected selected task log content in view, got %q", view)
	}
}

func TestLogBrowserSelectionUpdatesDisplayedLogContent(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, "task-1.jsonl"), []byte("primary output\n"), 0o600); err != nil {
		t.Fatalf("write task-1 log: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "task-1.stderr.log"), []byte("secondary output\n"), 0o600); err != nil {
		t.Fatalf("write task-1 stderr log: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "task-2.jsonl"), []byte("task-2 output\n"), 0o600); err != nil {
		t.Fatalf("write task-2 log: %v", err)
	}

	browser, err := NewLogBrowser(tmpDir)
	if err != nil {
		t.Fatalf("new log browser: %v", err)
	}

	if got := browser.CurrentTask(); got != "task-1" {
		t.Fatalf("expected initial selected task task-1, got %q", got)
	}
	if got := browser.CurrentLogFile(); !strings.HasSuffix(filepath.ToSlash(got), "task-1.jsonl") {
		t.Fatalf("expected initial selected log task-1.jsonl, got %q", got)
	}
	if view := browser.View(); !strings.Contains(view, "primary output") {
		t.Fatalf("expected initial log content, got %q", view)
	}

	browser.SelectTask(1)
	if got := browser.CurrentTask(); got != "task-2" {
		t.Fatalf("expected selected task task-2 after SelectTask, got %q", got)
	}
	if !strings.Contains(browser.View(), "task-2 output") {
		t.Fatalf("expected selected task content in view, got %q", browser.View())
	}

	browser.SelectTask(0)
	if view := browser.View(); !strings.Contains(view, "primary output") {
		t.Fatalf("expected first file content after task selection, got %q", view)
	}

	browser.SelectLogFile(1)
	if got := filepath.Base(browser.CurrentLogFile()); got != "task-1.stderr.log" {
		t.Fatalf("expected second file to be selected, got %q", got)
	}
	if !strings.Contains(browser.View(), "secondary output") {
		t.Fatalf("expected secondary file output after file selection, got %q", browser.View())
	}
}
