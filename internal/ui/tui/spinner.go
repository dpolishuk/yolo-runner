package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// Spinner is a wrapper around bubbles/spinner for the TUI status bar
type Spinner struct {
	spinner spinner.Model
}

// NewSpinner creates a new spinner component using bubbles/spinner
func NewSpinner() Spinner {
	return Spinner{
		spinner: spinner.New(spinner.WithSpinner(spinner.Dot)),
	}
}

// Init returns the initial command for the spinner
func (s Spinner) Init() tea.Cmd {
	return spinner.Tick
}

// Update handles spinner tick messages
func (s Spinner) Update(msg tea.Msg) (Spinner, tea.Cmd) {
	var cmd tea.Cmd
	s.spinner, cmd = s.spinner.Update(msg)
	return s, cmd
}

// View returns the current spinner frame
func (s Spinner) View() string {
	return s.spinner.View()
}
