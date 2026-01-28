package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// MarkdownBubble is a Bubble Tea component for rendering markdown content
// It uses glamour for markdown rendering and follows the teacup pattern
type MarkdownBubble struct {
	content string
	width   int
}

// NewMarkdownBubble creates a new markdown bubble component
func NewMarkdownBubble() MarkdownBubble {
	return MarkdownBubble{
		width: 80,
	}
}

// Init initializes markdown bubble component
func (m MarkdownBubble) Init() tea.Cmd {
	return nil
}

// SetMarkdownContentMsg is a message to set markdown content in bubble
type SetMarkdownContentMsg struct {
	Content string
}

// Update handles messages and updates the markdown bubble state
func (m MarkdownBubble) Update(msg tea.Msg) (MarkdownBubble, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typed.Width
	case SetMarkdownContentMsg:
		m.content = typed.Content
	}

	return m, nil
}

// View returns the rendered markdown
func (m MarkdownBubble) View() string {
	if m.content == "" {
		// Return styled empty line to maintain component visibility (follows teacup pattern)
		style := lipgloss.NewStyle().Width(m.width)
		return style.Render("")
	}

	// Strip control sequences before rendering
	strippedContent := stripControlSequences(m.content)

	// Normalize newlines before rendering
	normalizedContent := normalizeMarkdownNewlines(strippedContent)

	// Create a glamour renderer for terminal markdown rendering
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(m.width),
	)
	if err != nil {
		// Fallback to plain text if glamour fails
		return normalizedContent
	}

	rendered, err := renderer.Render(normalizedContent)
	if err != nil {
		return normalizedContent
	}

	return rendered
}

// stripControlSequences removes ANSI escape codes and other control sequences
// from text before rendering as markdown
func stripControlSequences(text string) string {
	// Remove ANSI escape sequences (colors, etc.)
	// ANSI escape sequences start with \x1b[ and end with m
	result := text
	for {
		start := findSubstring(result, "\x1b[")
		if start < 0 {
			break
		}
		end := findSubstring(result[start:], "m")
		if end < 0 {
			break
		}
		result = result[:start] + result[start+end+1:]
	}

	// Remove null characters
	result = strings.ReplaceAll(result, "\x00", "")

	// Remove other ASCII control characters (except newline, tab, carriage return)
	// Keep: \n (0x0A), \r (0x0D), \t (0x09)
	clean := make([]rune, 0, len(result))
	for _, r := range result {
		if r == '\n' || r == '\r' || r == '\t' || r >= 32 {
			clean = append(clean, r)
		}
	}
	result = string(clean)

	return result
}

// findSubstring is a helper to find substring index
func findSubstring(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// SetWidth sets the width for markdown rendering
func (m *MarkdownBubble) SetWidth(width int) {
	m.width = width
}

// normalizeMarkdownNewlines normalizes newlines in markdown content
// It converts all newline variants (\r\n, \n, \r) to standard Unix newlines (\n)
func normalizeMarkdownNewlines(text string) string {
	// Replace Windows line endings (\r\n) with Unix newlines (\n)
	text = strings.ReplaceAll(text, "\r\n", "\n")
	// Replace Mac line endings (\r) with Unix newlines (\n)
	text = strings.ReplaceAll(text, "\r", "\n")
	return text
}
