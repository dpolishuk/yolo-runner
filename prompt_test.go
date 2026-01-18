package main

import (
	"strings"
	"testing"
)

func TestBuildPromptIncludesSections(t *testing.T) {
	prompt := BuildPrompt("task-9", "Ship prompt parity", "Make it match", "It matches")

	required := []string{
		"Your task is: task-9 - Ship prompt parity",
		"**Description:**",
		"Make it match",
		"**Acceptance Criteria:**",
		"It matches",
		"**Strict TDD Protocol:**",
		"NEVER write implementation code before a failing test exists",
		"Watch test fail before writing code",
	}

	for _, item := range required {
		if !strings.Contains(prompt, item) {
			t.Fatalf("Expected prompt to include %q\nPrompt:\n%s", item, prompt)
		}
	}
}
