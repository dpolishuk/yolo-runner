package tk

import (
	"strings"
	"testing"
	"time"
)

func TestParseTicketFrontmatterConfigAcceptsValidValues(t *testing.T) {
	config, err := ParseTicketFrontmatterConfig(`
model: openai/gpt-5.3-codex
backend: codex
skillset: research
tools:
  - shell
  - git
timeout: 15m
mode: implement
`)
	if err != nil {
		t.Fatalf("expected valid frontmatter config, got error: %v", err)
	}
	if config.Model != "openai/gpt-5.3-codex" {
		t.Fatalf("expected model openai/gpt-5.3-codex, got %q", config.Model)
	}
	if config.Backend != "codex" {
		t.Fatalf("expected backend codex, got %q", config.Backend)
	}
	if config.Skillset != "research" {
		t.Fatalf("expected skillset research, got %q", config.Skillset)
	}
	if len(config.Tools) != 2 || config.Tools[0] != "shell" || config.Tools[1] != "git" {
		t.Fatalf("expected shell and git tools, got %#v", config.Tools)
	}
	if !config.HasTimeout || config.Timeout != 15*time.Minute {
		t.Fatalf("expected timeout 15m, got %#v (hasTimeout=%t)", config.Timeout, config.HasTimeout)
	}
	if config.Mode != "implement" {
		t.Fatalf("expected mode implement, got %q", config.Mode)
	}
}

func TestParseTicketFrontmatterConfigReturnsClearErrorsForInvalidValues(t *testing.T) {
	_, err := ParseTicketFrontmatterConfig(`
model: 12
backend: invalid-backend
skillset: 7
tools: shell
timeout: not-a-duration
mode: bad-mode
`)
	if err == nil {
		t.Fatalf("expected invalid config error, got nil")
	}

	message := err.Error()
	required := []string{
		"model must be a string",
		"backend must be one of: opencode, codex, claude, kimi",
		"skillset must be a string",
		"tools must be an array",
		"timeout must be a valid duration (for example 30s, 5m)",
		"mode must be one of: implement, review",
		"frontmatter validation failed",
	}
	for _, needle := range required {
		if !strings.Contains(message, needle) {
			t.Fatalf("expected validation message to include %q, got %q", needle, message)
		}
	}
}

