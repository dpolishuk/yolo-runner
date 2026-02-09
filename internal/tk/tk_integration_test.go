package tk

import (
	"os/exec"
	"strings"
	"testing"
)

func TestAdapterReadyUsesRealTKOrderingAndDeps(t *testing.T) {
	if _, err := exec.LookPath("tk"); err != nil {
		t.Skip("tk CLI is required for integration tests")
	}

	workdir := t.TempDir()
	r := commandRunner{dir: workdir}
	a := New(r)

	rootID := mustCreateTicket(t, r, "Roadmap", "epic", "0", "")
	firstID := mustCreateTicket(t, r, "First", "task", "0", rootID)
	secondID := mustCreateTicket(t, r, "Second", "task", "1", rootID)
	mustRun(t, r, "tk", "dep", secondID, firstID)

	ready, err := a.Ready(rootID)
	if err != nil {
		t.Fatalf("ready failed: %v", err)
	}

	if ready.ID != firstID {
		t.Fatalf("expected first ready task %q, got %q", firstID, ready.ID)
	}
}

func TestAdapterUpdateStatusWithReasonWritesNote(t *testing.T) {
	if _, err := exec.LookPath("tk"); err != nil {
		t.Skip("tk CLI is required for integration tests")
	}

	workdir := t.TempDir()
	r := commandRunner{dir: workdir}
	a := New(r)

	taskID := mustCreateTicket(t, r, "Needs unblock", "task", "2", "")
	if err := a.UpdateStatusWithReason(taskID, "blocked", "runner timeout"); err != nil {
		t.Fatalf("update status with reason failed: %v", err)
	}

	show, err := r.Run("tk", "show", taskID)
	if err != nil {
		t.Fatalf("tk show failed: %v", err)
	}
	if !strings.Contains(show, "runner timeout") {
		t.Fatalf("expected note text in tk show output")
	}
}

type commandRunner struct {
	dir string
}

func (c commandRunner) Run(args ...string) (string, error) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = c.dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func mustCreateTicket(t *testing.T, runner commandRunner, title string, issueType string, priority string, parent string) string {
	t.Helper()
	args := []string{"tk", "create", title, "-t", issueType, "-p", priority}
	if parent != "" {
		args = append(args, "--parent", parent)
	}
	out, err := runner.Run(args...)
	if err != nil {
		t.Fatalf("create ticket failed: %v (%s)", err, out)
	}
	return strings.TrimSpace(out)
}

func mustRun(t *testing.T, runner commandRunner, args ...string) {
	t.Helper()
	if out, err := runner.Run(args...); err != nil {
		t.Fatalf("command failed: %v (%s)", err, out)
	}
}
