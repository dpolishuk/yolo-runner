package logging

import (
	"strings"
	"testing"
)

func TestValidateStructuredLogLineAcceptsRequiredFields(t *testing.T) {
	samples := []string{
		`{"timestamp":"2026-02-22T10:00:00Z","level":"info","component":"runner","task_id":"task-99","run_id":"run-99","message":"runner started","issue_id":"task-99"}`,
		`{"timestamp":"2026-02-22T10:01:00Z","level":"debug","component":"opencode","task_id":"task-101","run_id":"run-101","issue_id":"task-101","request_type":"update","decision":"allow","message":"stream update"}`,
	}

	for _, line := range samples {
		if err := ValidateStructuredLogLine([]byte(line)); err != nil {
			t.Fatalf("expected valid schema line, got: %v", err)
		}
	}
}

func TestValidateStructuredLogLineRejectsMissingRequiredField(t *testing.T) {
	line := `{"timestamp":"2026-02-22T10:00:00Z","level":"info","component":"runner","task_id":"task-99","message":"missing run_id"}`
	if err := ValidateStructuredLogLine([]byte(line)); err == nil {
		t.Fatal("expected validation failure for missing run_id")
	}
}

func TestValidateStructuredLogLineRejectsInvalidTimestamp(t *testing.T) {
	line := `{"timestamp":"not-a-timestamp","level":"info","component":"runner","task_id":"task-99","run_id":"run-99"}`
	if err := ValidateStructuredLogLine([]byte(line)); err == nil {
		t.Fatal("expected validation failure for invalid timestamp")
	}
}

func TestValidateStructuredLogLineRejectsBlankLine(t *testing.T) {
	if err := ValidateStructuredLogLine([]byte("")); err == nil {
		t.Fatal("expected validation failure for blank line")
	}
	if err := ValidateStructuredLogLine([]byte("   \n")); err == nil {
		t.Fatal("expected validation failure for whitespace-only line")
	}
}

func TestValidateStructuredLogLineForLoggedEntries(t *testing.T) {
	lines := strings.TrimSpace(`{"timestamp":"2026-02-22T10:00:00Z","level":"info","component":"runner","task_id":"task-1","run_id":"run-1","issue_id":"task-1"}
{"timestamp":"2026-02-22T10:00:01Z","level":"info","component":"opencode","task_id":"task-2","run_id":"run-2","issue_id":"task-2","request_type":"permission"}`)

	for _, line := range strings.Split(lines, "\n") {
		if err := ValidateStructuredLogLine([]byte(line)); err != nil {
			t.Fatalf("expected logged entry to conform: %v", err)
		}
	}
}
