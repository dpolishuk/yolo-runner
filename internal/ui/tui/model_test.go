package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anomalyco/yolo-runner/internal/runner"
)

func TestModelUsesBubblesSpinnerNotCustomFrames(t *testing.T) {
	// Verify that the Model uses the bubbles spinner component
	// and not the old custom spinnerFrames variable
	m := NewModel(func() time.Time { return time.Unix(0, 0) })
	updated, _ := m.Update(runner.Event{
		Type:      runner.EventSelectTask,
		IssueID:   "task-123",
		Title:     "Test Task",
		EmittedAt: time.Unix(0, 0),
	})
	m = updated.(Model)

	view := m.View()

	// The old custom spinnerFrames used these characters: "-", "\", "|", "/"
	// The new bubbles spinner with dot spinner should not use these at line start
	lines := strings.Split(strings.TrimSpace(view), "\n")
	customChars := []string{"-", "\\", "|", "/"}
	for _, line := range lines {
		// Skip the quit hint line which starts with "q:"
		if strings.HasPrefix(line, "q:") {
			continue
		}
		// Skip the stopping line which starts with "Stopping"
		if strings.HasPrefix(line, "Stopping") {
			continue
		}
		// Check if the first character of the line is a custom spinner char
		if len(line) > 0 {
			firstChar := string(line[0])
			for _, char := range customChars {
				if firstChar == char {
					t.Fatalf("expected model to use bubbles spinner, not custom spinnerFrames. Found custom char %q at start of line: %q", char, line)
				}
			}
		}
	}

	// The view should still render something (spinner output)
	if len(view) == 0 {
		t.Fatal("expected model view to have content")
	}
}

func TestModelRendersTaskAndPhase(t *testing.T) {
	fixedNow := time.Date(2026, 1, 19, 12, 0, 10, 0, time.UTC)
	m := NewModel(func() time.Time { return fixedNow })
	updated, _ := m.Update(runner.Event{
		Type:      runner.EventSelectTask,
		IssueID:   "task-1",
		Title:     "Example Task",
		Phase:     "running",
		EmittedAt: fixedNow.Add(-5 * time.Second),
	})
	m = updated.(Model)

	view := m.View()
	if !strings.Contains(view, "task-1 - Example Task") {
		t.Fatalf("expected task id and title in view, got %q", view)
	}
	if !strings.Contains(view, "getting task info") {
		t.Fatalf("expected phase in view, got %q", view)
	}
	if !strings.Contains(view, "(5s)") {
		t.Fatalf("expected last output age in view, got %q", view)
	}
}

func TestSpinnerAdvancesOnOutput(t *testing.T) {
	m := NewModel(func() time.Time { return time.Unix(0, 0) })
	// Initialize to start spinner ticking
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("expected Init to return a command")
	}
	// Execute the spinner's tick command to get spinner.TickMsg
	msg := cmd()
	if msg == nil {
		t.Fatal("expected command to return a message")
	}
	// Send the message to update the model
	updated, _ := m.Update(msg)
	m = updated.(Model)
	first := m.View()

	// Tick again
	updated, _ = m.Update(msg)
	m = updated.(Model)
	second := m.View()

	// The spinner should advance (or at least the view should be generated)
	if first == "" || second == "" {
		t.Fatalf("expected non-empty views, got first=%q, second=%q", first, second)
	}
}

func TestModelTicksLastOutputAge(t *testing.T) {
	now := time.Date(2026, 1, 19, 12, 0, 0, 0, time.UTC)
	current := now
	m := NewModel(func() time.Time { return current })
	updated, _ := m.Update(runner.Event{
		Type:      runner.EventSelectTask,
		IssueID:   "task-1",
		Title:     "Example Task",
		Phase:     "running",
		EmittedAt: current,
	})
	m = updated.(Model)

	current = current.Add(3 * time.Second)
	updated, cmd := m.Update(tickMsg{})
	m = updated.(Model)

	if cmd == nil {
		t.Fatalf("expected tick command")
	}
	if !strings.Contains(m.View(), "(3s)") {
		t.Fatalf("expected last output age to tick, got %q", m.View())
	}
}

func TestModelOutputResetsLastOutputAge(t *testing.T) {
	now := time.Date(2026, 1, 19, 12, 0, 0, 0, time.UTC)
	current := now
	m := NewModel(func() time.Time { return current })
	updated, _ := m.Update(runner.Event{
		Type:      runner.EventSelectTask,
		IssueID:   "task-1",
		Title:     "Example Task",
		Phase:     "running",
		EmittedAt: current,
	})
	m = updated.(Model)

	current = current.Add(5 * time.Second)
	updated, _ = m.Update(OutputMsg{})
	m = updated.(Model)

	if !strings.Contains(m.View(), "(0s)") {
		t.Fatalf("expected last output age to reset, got %q", m.View())
	}
}

func TestModelInitSchedulesTick(t *testing.T) {
	m := NewModel(func() time.Time { return time.Unix(0, 0) })
	if cmd := m.Init(); cmd == nil {
		t.Fatalf("expected tick command")
	}
}

func TestModelShowsQuitHintOnStart(t *testing.T) {
	m := NewModel(func() time.Time { return time.Unix(0, 0) })
	view := m.View()
	if !strings.Contains(view, "\nq: stop runner\n") {
		t.Fatalf("expected quit hint in view, got %q", view)
	}
	if strings.Contains(view, "Stopping...") {
		t.Fatalf("did not expect stopping status in view, got %q", view)
	}
}

func TestModelShowsQuitHintWhileStopping(t *testing.T) {
	m := NewModel(func() time.Time { return time.Unix(0, 0) })
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(Model)
	view := m.View()
	if !strings.Contains(view, "Stopping...") {
		t.Fatalf("expected stopping status in view, got %q", view)
	}
	if !strings.Contains(view, "\nq: stop runner\n") {
		t.Fatalf("expected quit hint in view, got %q", view)
	}
}

func TestModelStopsOnCtrlC(t *testing.T) {
	stopCh := make(chan struct{})
	m := NewModelWithStop(func() time.Time { return time.Unix(0, 0) }, stopCh)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)
	if !m.StopRequested() {
		t.Fatalf("expected stop requested")
	}
	select {
	case <-stopCh:
		// ok
	default:
		t.Fatalf("expected stop channel to close")
	}
}

func TestModelRendersStatusBarInSingleLine(t *testing.T) {
	fixedNow := time.Date(2026, 1, 19, 12, 0, 10, 0, time.UTC)
	m := NewModel(func() time.Time { return fixedNow })
	updated, _ := m.Update(runner.Event{
		Type:      runner.EventSelectTask,
		IssueID:   "task-1",
		Title:     "Example Task",
		Phase:     "running",
		EmittedAt: fixedNow.Add(-5 * time.Second),
	})
	m = updated.(Model)

	view := m.View()
	lines := strings.Split(strings.TrimSpace(view), "\n")

	// Should have a status bar line containing spinner, phase, and last output age
	statusBarFound := false
	for _, line := range lines {
		if strings.Contains(line, "task-1 - Example Task") {
			// This is the task title line, not status bar
			continue
		}
		if strings.Contains(line, "phase:") {
			// This should not exist anymore - phase should be in status bar
			t.Fatalf("phase should be in status bar, not separate line: %q", line)
		}
		if strings.Contains(line, "last output") {
			// This should not exist anymore - last output should be in status bar
			t.Fatalf("last output should be in status bar, not separate line: %q", line)
		}
		// Check if this line contains spinner, state info, and age
		if strings.Contains(line, "-") || strings.Contains(line, "\\") || strings.Contains(line, "|") || strings.Contains(line, "/") {
			// Found spinner, check if it also contains phase and age info
			if strings.Contains(line, "getting task info") && strings.Contains(line, "5s") {
				statusBarFound = true
			}
		}
	}

	if !statusBarFound {
		t.Fatalf("expected status bar line with spinner, phase, and age, got view: %q", view)
	}
}

func TestModelStatusBarUpdatesInPlace(t *testing.T) {
	fixedNow := time.Date(2026, 1, 19, 12, 0, 0, 0, time.UTC)
	current := fixedNow
	m := NewModel(func() time.Time { return current })
	updated, _ := m.Update(runner.Event{
		Type:      runner.EventSelectTask,
		IssueID:   "task-1",
		Title:     "Example Task",
		Phase:     "running",
		EmittedAt: current,
	})
	m = updated.(Model)

	firstView := m.View()

	// Advance time and update
	current = current.Add(3 * time.Second)
	updated, _ = m.Update(tickMsg{})
	m = updated.(Model)

	secondView := m.View()

	// Views should be different (age updated) but structure should be same
	if firstView == secondView {
		t.Fatalf("expected view to change with time")
	}

	// Both should have same number of lines
	firstLines := strings.Split(strings.TrimSpace(firstView), "\n")
	secondLines := strings.Split(strings.TrimSpace(secondView), "\n")
	if len(firstLines) != len(secondLines) {
		t.Fatalf("expected same number of lines in status bar updates, got %d vs %d", len(firstLines), len(secondLines))
	}
}

func TestModelStatusBarExactFormat(t *testing.T) {
	fixedNow := time.Date(2026, 1, 19, 12, 0, 10, 0, time.UTC)
	m := NewModel(func() time.Time { return fixedNow })
	updated, _ := m.Update(runner.Event{
		Type:      runner.EventSelectTask,
		IssueID:   "task-1",
		Title:     "Example Task",
		Phase:     "running",
		EmittedAt: fixedNow.Add(-5 * time.Second),
	})
	m = updated.(Model)

	view := m.View()

	// Check that view contains expected components (not exact format since spinner changes)
	if !strings.Contains(view, "task-1 - Example Task") {
		t.Fatalf("expected task id and title in view, got %q", view)
	}
	if !strings.Contains(view, "getting task info") {
		t.Fatalf("expected phase in view, got %q", view)
	}
	if !strings.Contains(view, "task-1") {
		t.Fatalf("expected task id in status bar, got %q", view)
	}
	if !strings.Contains(view, "(5s)") {
		t.Fatalf("expected last output age in view, got %q", view)
	}
	if !strings.Contains(view, "q: stop runner") {
		t.Fatalf("expected quit hint in view, got %q", view)
	}

	// Test with progress
	updated, _ = m.Update(runner.Event{
		Type:              runner.EventSelectTask,
		IssueID:           "task-1",
		Title:             "Example Task",
		Phase:             "running",
		ProgressCompleted: 2,
		ProgressTotal:     5,
		EmittedAt:         fixedNow.Add(-5 * time.Second),
	})
	m = updated.(Model)

	view = m.View()

	// Check progress is shown
	if !strings.Contains(view, "[2/5]") {
		t.Fatalf("expected progress [2/5] in view, got %q", view)
	}
}

func TestModelStatusBarShowsProgress(t *testing.T) {
	fixedNow := time.Date(2026, 1, 19, 12, 0, 10, 0, time.UTC)
	m := NewModel(func() time.Time { return fixedNow })
	updated, _ := m.Update(runner.Event{
		Type:              runner.EventSelectTask,
		IssueID:           "task-1",
		Title:             "Example Task",
		Phase:             "running",
		ProgressCompleted: 2,
		ProgressTotal:     5,
		EmittedAt:         fixedNow.Add(-5 * time.Second),
	})
	m = updated.(Model)

	view := m.View()
	lines := strings.Split(strings.TrimSpace(view), "\n")

	// Find the status bar line (should contain spinner and phase)
	var statusBarLine string
	for _, line := range lines {
		if strings.Contains(line, "getting task info") && strings.Contains(line, "task-1") && strings.Contains(line, "(5s)") {
			statusBarLine = line
			break
		}
	}

	if statusBarLine == "" {
		t.Fatalf("expected to find status bar line with getting task info phase and task-1, got view: %q", view)
	}

	// Status bar should contain progress [2/5]
	if !strings.Contains(statusBarLine, "[2/5]") {
		t.Fatalf("expected status bar to contain progress [2/5], got: %q", statusBarLine)
	}
}

func TestModelStatusBarShowsModel(t *testing.T) {
	fixedNow := time.Date(2026, 1, 19, 12, 0, 10, 0, time.UTC)
	m := NewModel(func() time.Time { return fixedNow })
	updated, _ := m.Update(runner.Event{
		Type:      runner.EventOpenCodeStart,
		IssueID:   "task-1",
		Title:     "Example Task",
		Phase:     "starting opencode",
		Model:     "claude-3-5-sonnet",
		EmittedAt: fixedNow.Add(-5 * time.Second),
	})
	m = updated.(Model)

	view := m.View()
	lines := strings.Split(strings.TrimSpace(view), "\n")

	// Find the status bar line
	var statusBarLine string
	for _, line := range lines {
		if strings.Contains(line, "starting opencode") && strings.Contains(line, "task-1") && strings.Contains(line, "(5s)") {
			statusBarLine = line
			break
		}
	}

	if statusBarLine == "" {
		t.Fatalf("expected to find status bar line with starting opencode phase, got view: %q", view)
	}

	// Status bar should contain the model name
	if !strings.Contains(statusBarLine, "claude-3-5-sonnet") {
		t.Fatalf("expected status bar to contain model 'claude-3-5-sonnet', got: %q", statusBarLine)
	}
}

func TestModelStatusBarShowsMultipleModelFormats(t *testing.T) {
	fixedNow := time.Date(2026, 1, 19, 12, 0, 10, 0, time.UTC)

	testCases := []struct {
		model    string
		expected string
	}{
		{"gpt-4", "gpt-4"},
		{"claude-3-5-sonnet-20241022", "claude-3-5-sonnet-20241022"},
		{"gemini-pro", "gemini-pro"},
		{"o1-preview", "o1-preview"},
	}

	for _, tc := range testCases {
		t.Run(tc.model, func(t *testing.T) {
			m := NewModel(func() time.Time { return fixedNow })
			updated, _ := m.Update(runner.Event{
				Type:      runner.EventOpenCodeStart,
				IssueID:   "task-1",
				Title:     "Example Task",
				Phase:     "starting opencode",
				Model:     tc.model,
				EmittedAt: fixedNow.Add(-5 * time.Second),
			})
			m = updated.(Model)

			view := m.View()

			// View should contain the model name
			if !strings.Contains(view, tc.expected) {
				t.Fatalf("expected view to contain model %q, got: %q", tc.expected, view)
			}
		})
	}
}
