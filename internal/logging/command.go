package logging

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CommandLogger handles logging of command stdout/stderr to files
type CommandLogger struct {
	logDir string
}

// CommandLogEntry is the structured command log format.
type CommandLogEntry struct {
	LoggingSchemaFields
	IssueID   string `json:"issue_id"`
	Command   string `json:"command"`
	StartTime string `json:"start_time"`
	Elapsed   string `json:"elapsed"`
	ExitCode  int    `json:"exit_code"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
	Error     string `json:"error,omitempty"`
}

// NewCommandLogger creates a new command logger
func NewCommandLogger(logDir string) *CommandLogger {
	return &CommandLogger{
		logDir: logDir,
	}
}

// LogCommand logs a command's execution details, stdout, and stderr to files
func (cl *CommandLogger) LogCommand(command []string, stdout string, stderr string, err error, startTime time.Time) error {
	if cl == nil {
		return nil
	}
	if len(command) == 0 {
		command = []string{"unknown"}
	}
	if err := os.MkdirAll(cl.logDir, 0o755); err != nil {
		return err
	}

	elapsed := time.Since(startTime).Round(time.Millisecond)
	level := "info"
	status := "ok"
	if err != nil {
		level = "error"
		status = "failed"
	}

	entry := CommandLogEntry{
		LoggingSchemaFields: populateRequiredLogFields(LoggingSchemaFields{
			Level:     level,
			Component: "runner",
			TaskID:    "runtime",
			RunID:     "runtime",
		}, "runtime"),
		IssueID:   "runtime",
		Command:   strings.Join(command, " "),
		StartTime: startTime.UTC().Format(time.RFC3339),
		Elapsed:   elapsed.String(),
		ExitCode:  exitCodeFromError(err),
		Status:    status,
		Message:   "command completed",
		Stdout:    stdout,
		Stderr:    stderr,
	}
	if err != nil {
		entry.Error = err.Error()
	}

	payload, marshalErr := json.Marshal(entry)
	if marshalErr != nil {
		return marshalErr
	}

	logPath := filepath.Join(cl.logDir, logFileName(command))
	logFile, openErr := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if openErr != nil {
		return openErr
	}
	defer logFile.Close()

	if _, writeErr := logFile.Write(append(payload, '\n')); writeErr != nil {
		return writeErr
	}
	return nil
}

func logFileName(command []string) string {
	timestamp := time.Now().UTC().Format("20060102_150405_000000")
	commandName := strings.Join(command[:min(3, len(command))], "_")
	commandName = strings.ReplaceAll(commandName, "/", "_")
	commandName = strings.ReplaceAll(commandName, " ", "_")
	return fmt.Sprintf("%s_%s.log", timestamp, commandName)
}

func exitCodeFromError(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return 1
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
