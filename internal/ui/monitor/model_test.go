package monitor

import (
	"testing"
	"time"

	"github.com/anomalyco/yolo-runner/internal/contracts"
)

func TestModelTracksCurrentTaskPhaseAgeAndHistory(t *testing.T) {
	now := time.Date(2026, 2, 10, 12, 0, 10, 0, time.UTC)
	model := NewModel(func() time.Time { return now })

	model.Apply(contracts.Event{Type: contracts.EventTypeTaskStarted, TaskID: "task-1", Message: "started", Timestamp: now.Add(-5 * time.Second)})
	model.Apply(contracts.Event{Type: contracts.EventTypeRunnerStarted, TaskID: "task-1", Message: "runner started", Timestamp: now.Add(-3 * time.Second)})
	model.Apply(contracts.Event{Type: contracts.EventTypeRunnerFinished, TaskID: "task-1", Message: "runner finished", Timestamp: now.Add(-1 * time.Second)})

	view := model.View()
	assertContains(t, view, "Current Task: task-1")
	assertContains(t, view, "Phase: runner_finished")
	assertContains(t, view, "Last Output Age: 1s")
	assertContains(t, view, "runner started")
	assertContains(t, view, "runner finished")
}

func assertContains(t *testing.T, text string, expected string) {
	t.Helper()
	if !contains(text, expected) {
		t.Fatalf("expected %q in %q", expected, text)
	}
}

func contains(text string, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i+len(sub) <= len(text); i++ {
		if text[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
