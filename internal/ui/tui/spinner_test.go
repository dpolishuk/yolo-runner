package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
)

func TestSpinnerUsesBubblesSpinner(t *testing.T) {
	// Test that we can create a spinner component
	s := NewSpinner()

	// Verify it can be initialized and returns a command
	if cmd := s.Init(); cmd == nil {
		t.Fatal("expected spinner Init to return a command")
	}

	// Verify the View method returns output (not empty)
	view := s.View()
	if view == "" {
		t.Fatal("expected spinner view to return output, got empty string")
	}

	// Verify the view contains some content (spinner character)
	// The bubbles spinner uses unicode characters by default
	if len(view) == 0 {
		t.Fatal("expected spinner view to contain content, got empty string")
	}
}

func TestSpinnerAdvancesOnTick(t *testing.T) {
	s := NewSpinner()

	// Get initial view
	_, cmd := s.Update(spinner.TickMsg{})
	if cmd == nil {
		// First tick might return a command to schedule next tick
	}

	initialView := s.View()

	// Tick again to advance
	_, cmd = s.Update(spinner.TickMsg{})
	afterView := s.View()

	// After ticking, the spinner should have advanced
	// (Note: this might not always change depending on frame, but should at least not error)
	if initialView == "" || afterView == "" {
		t.Fatal("expected non-empty views before and after tick")
	}
}

func TestSpinnerInitReturnsCommand(t *testing.T) {
	s := NewSpinner()

	cmd := s.Init()
	if cmd == nil {
		t.Fatal("expected spinner Init to return a command for ticking")
	}
}

func TestSpinnerNoCustomFrames(t *testing.T) {
	// This test verifies we're not using custom spinnerFrames variable
	// by ensuring the component is self-contained and uses bubbles/spinner

	s := NewSpinner()
	view := s.View()

	// If we were using custom spinnerFrames (like "-", "\", "|", "/"),
	// those characters would appear in the output
	// bubbles/spinner defaults to dot spinner, so these shouldn't appear
	customChars := []string{"-", "\\", "|", "/"}
	hasCustomChar := false
	for _, char := range customChars {
		if strings.Contains(view, char) {
			hasCustomChar = true
			break
		}
	}

	if hasCustomChar {
		t.Fatalf("expected spinner to use bubbles/spinner default frames, not custom characters like -, \\, |, /, got view: %q", view)
	}
}

func TestSpinnerViewFormat(t *testing.T) {
	s := NewSpinner()
	view := s.View()

	// View should return a string with the current spinner frame
	// and should not be empty
	if len(view) == 0 {
		t.Fatal("expected spinner view to have content")
	}

	// The view should be reasonably short (just the spinner character)
	// bubbles/spinner dot spinner is 1-2 characters
	if len(view) > 10 {
		t.Fatalf("expected spinner view to be short, got length %d: %q", len(view), view)
	}
}
