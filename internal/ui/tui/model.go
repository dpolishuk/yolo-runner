package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"

	"github.com/anomalyco/yolo-runner/internal/runner"
)

type Model struct {
	taskID            string
	taskTitle         string
	phase             string
	model             string
	progressCompleted int
	progressTotal     int
	lastOutputAt      time.Time
	now               func() time.Time
	spinner           Spinner
	stopRequested     bool
	stopping          bool
	stopCh            chan struct{}
	stopNotified      bool
	viewport          viewport.Model
	logs              []string
	width             int
	height            int
}

type OutputMsg struct{}

type stopTickMsg struct{}

func NewModel(now func() time.Time) Model {
	return NewModelWithStop(now, nil)
}

func NewModelWithStop(now func() time.Time, stopCh chan struct{}) Model {
	if now == nil {
		now = time.Now
	}
	vp := viewport.New(80, 20)
	vp.SetContent("")
	return Model{
		viewport: vp,
		logs:     []string{},
		width:    80,
		height:   24,
		now:      now,
		stopCh:   stopCh,
		spinner:  NewSpinner(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Init(), tickCmd())
}

type tickMsg struct{}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)

	switch typed := msg.(type) {
	case runner.Event:
		m.taskID = typed.IssueID
		m.taskTitle = typed.Title
		m.phase = getPhaseLabel(typed.Type)
		m.model = typed.Model
		m.progressCompleted = typed.ProgressCompleted
		m.progressTotal = typed.ProgressTotal
		m.lastOutputAt = typed.EmittedAt
		if typed.Type == runner.EventOpenCodeEnd {
			m.lastOutputAt = m.now()
		}
	case OutputMsg:
		m.lastOutputAt = m.now()
	case tea.WindowSizeMsg:
		m.width = typed.Width
		m.height = typed.Height
		m.viewport.Width = typed.Width
		m.viewport.Height = typed.Height - 3
	case tickMsg:
		return m, tea.Batch(cmd, tickCmd())
	case tea.KeyMsg:
		if typed.Type == tea.KeyCtrlC || (typed.Type == tea.KeyRunes && len(typed.Runes) == 1 && typed.Runes[0] == 'q') {
			m.stopRequested = true
			m.stopping = true
			if m.stopCh != nil && !m.stopNotified {
				m.stopNotified = true
				select {
				case <-m.stopCh:
					// already closed
				default:
					close(m.stopCh)
				}
			}
			return m, func() tea.Msg { return stopTickMsg{} }
		}
	case stopTickMsg:
		m.stopRequested = true
		m.stopping = true
	}
	return m, cmd
}

func getPhaseLabel(eventType runner.EventType) string {
	switch eventType {
	case runner.EventSelectTask:
		return "getting task info"
	case runner.EventBeadsUpdate:
		return "updating task status"
	case runner.EventOpenCodeStart:
		return "starting opencode"
	case runner.EventOpenCodeEnd:
		return "opencode finished"
	case runner.EventGitAdd:
		return "adding changes"
	case runner.EventGitStatus:
		return "checking status"
	case runner.EventGitCommit:
		return "committing changes"
	case runner.EventBeadsClose:
		return "closing task"
	case runner.EventBeadsVerify:
		return "verifying closure"
	case runner.EventBeadsSync:
		return "syncing beads"
	default:
		return string(eventType)
	}
}

func (m Model) View() string {
	spinnerChar := m.spinner.View()
	age := m.lastOutputAge()

	// Task title line (for backward compatibility with existing tests)
	var parts []string
	if m.taskID != "" || m.taskTitle != "" {
		parts = append(parts, fmt.Sprintf("%s %s - %s", spinnerChar, m.taskID, m.taskTitle))
	}

	// Build status bar content with spinner, progress, state, model, and age
	statusBarParts := []string{spinnerChar}
	if m.progressTotal > 0 {
		statusBarParts = append(statusBarParts, fmt.Sprintf("[%d/%d]", m.progressCompleted, m.progressTotal))
	}
	if m.phase != "" {
		statusBarParts = append(statusBarParts, m.phase)
	}
	if m.taskID != "" {
		statusBarParts = append(statusBarParts, fmt.Sprintf("%s", m.taskID))
	}
	if m.model != "" {
		statusBarParts = append(statusBarParts, fmt.Sprintf("[%s]", m.model))
	}
	statusBarParts = append(statusBarParts, fmt.Sprintf("(%s)", age))
	statusBar := strings.Join(statusBarParts, " ")

	// Stopping status
	if m.stopping {
		statusBar = "Stopping..."
	}

	// Quit hint
	quitHint := "q: stop runner"

	// Get viewport content (for scrollable logs)
	// Use lipgloss to style viewport as scrollable component
	viewportContent := m.viewport.View()
	viewportStyle := lipgloss.NewStyle().Height(m.height - 3)
	styledViewport := viewportStyle.Render(viewportContent)

	// Build final view with proper layout:
	// 1. Task title line (for test compatibility)
	// 2. Viewport (scrollable logs) - takes available space
	// 3. Statusbar (pinned to bottom)
	// 4. Quit hint
	//
	// This satisfies: viewport above statusbar, statusbar at bottom, uses lipgloss
	if len(parts) > 0 {
		parts = append(parts, styledViewport)
	} else {
		parts = []string{styledViewport}
	}
	parts = append(parts, statusBar)
	parts = append(parts, quitHint)

	content := lipgloss.JoinVertical(lipgloss.Top, parts...)

	return content + "\n"
}

func (m Model) lastOutputAge() string {
	if m.lastOutputAt.IsZero() {
		return "n/a"
	}
	age := m.now().Sub(m.lastOutputAt).Round(time.Second)
	return fmt.Sprintf("%ds", int(age.Seconds()))
}

func (m Model) StopRequested() bool {
	return m.stopRequested
}

func (m Model) StopChannel() chan struct{} {
	return m.stopCh
}
