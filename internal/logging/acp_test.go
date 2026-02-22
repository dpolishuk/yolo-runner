package logging

import (
	"os"
	"path/filepath"
	"testing"
	"strings"
)

func TestAppendACPRequestWritesJSONL(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "runner-logs", "opencode", "issue-1.jsonl")
	if err := AppendACPRequest(logPath, ACPRequestEntry{
		IssueID:     "issue-1",
		RequestType: "permission",
		Decision:    "allow",
	}); err != nil {
		t.Fatalf("append error: %v", err)
	}
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(content)), "\n") {
		if err := ValidateStructuredLogLine([]byte(line)); err != nil {
			t.Fatalf("logged entry should conform to schema: %v", err)
		}
	}
	if len(content) == 0 || content[len(content)-1] != '\n' {
		t.Fatalf("expected newline-terminated jsonl")
	}
}
