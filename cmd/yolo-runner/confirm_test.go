package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCleanupConfirmAcceptsYes(t *testing.T) {
	summary := " M file.go\nDiscard these changes? [y/N]"
	input := strings.NewReader("y\n")
	output := &bytes.Buffer{}

	ok, err := cleanupConfirmPrompt(summary, input, output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected confirmation to accept yes")
	}
	if !strings.Contains(output.String(), "Discard these changes? [y/N]") {
		t.Fatalf("expected prompt to be printed, got %q", output.String())
	}
}

func TestCleanupConfirmDefaultsToNo(t *testing.T) {
	summary := " M file.go\nDiscard these changes? [y/N]"
	input := strings.NewReader("\n")
	output := &bytes.Buffer{}

	ok, err := cleanupConfirmPrompt(summary, input, output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected confirmation to default to no")
	}
	if !strings.Contains(output.String(), "Discard these changes? [y/N]") {
		t.Fatalf("expected prompt to be printed, got %q", output.String())
	}
}
