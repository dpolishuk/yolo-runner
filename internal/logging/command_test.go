package logging

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLogCommandWritesStructuredJSON(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "runner-logs")
	logger := NewCommandLogger(logDir)

	err := logger.LogCommand([]string{"git", "status", "--porcelain"}, "stdout-line\n", "stderr-line\n", nil, tNow(2026, 1, 22, 10, 0, 0, 0))
	if err != nil {
		t.Fatalf("log command error: %v", err)
	}

	logFiles, err := os.ReadDir(logDir)
	if err != nil {
		t.Fatalf("read log dir: %v", err)
	}
	if len(logFiles) != 1 {
		t.Fatalf("expected one log file, got %d", len(logFiles))
	}

	content, err := os.ReadFile(filepath.Join(logDir, logFiles[0].Name()))
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	line := strings.TrimSpace(string(content))
	if line == "" {
		t.Fatal("expected non-empty log line")
	}

	if err := ValidateStructuredLogLine([]byte(line)); err != nil {
		t.Fatalf("expected valid structured log line: %v", err)
	}

	entry := map[string]interface{}{}
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		t.Fatalf("invalid json log line: %v", err)
	}

	if entry["command"] != "git status --porcelain" {
		t.Fatalf("expected command field, got %v", entry["command"])
	}
	if entry["status"] != "ok" {
		t.Fatalf("expected status ok, got %v", entry["status"])
	}
	if entry["stdout"] != "stdout-line\n" {
		t.Fatalf("expected stdout field, got %v", entry["stdout"])
	}
}

func TestLogCommandWritesErrorLevelForCommandErrors(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "runner-logs")
	logger := NewCommandLogger(logDir)

	err := logger.LogCommand([]string{"git", "status"}, "", "", assertError{}, tNow(2026, 1, 22, 10, 0, 1, 0))
	if err != nil {
		t.Fatalf("log command error: %v", err)
	}

	logFiles, err := os.ReadDir(logDir)
	if err != nil {
		t.Fatalf("read log dir: %v", err)
	}
	if len(logFiles) != 1 {
		t.Fatalf("expected one log file, got %d", len(logFiles))
	}

	content, err := os.ReadFile(filepath.Join(logDir, logFiles[0].Name()))
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	line := strings.TrimSpace(string(content))
	if line == "" {
		t.Fatal("expected non-empty log line")
	}

	entry := map[string]interface{}{}
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		t.Fatalf("invalid json log line: %v", err)
	}

	if entry["level"] != "error" {
		t.Fatalf("expected error level, got %v", entry["level"])
	}
}

type assertError struct{}

func (assertError) Error() string {
	return "command failed"
}

func tNow(year, month, day, hour, min, sec, nsec int) (ts time.Time) {
	return time.Date(year, time.Month(month), day, hour, min, sec, nsec, time.UTC)
}
