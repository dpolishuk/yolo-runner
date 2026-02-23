package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunMainConfigInitCreatesStarterConfig(t *testing.T) {
	repoRoot := t.TempDir()

	stdout := captureStdout(t, func() {
		code := RunMain([]string{"config", "init", "--repo", repoRoot}, nil)
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d", code)
		}
	})

	configPath := filepath.Join(repoRoot, trackerConfigRelPath)
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read generated config: %v", err)
	}
	expected := "default_profile: default\nprofiles:\n  default:\n    tracker:\n      type: tk\n"
	if string(content) != expected {
		t.Fatalf("unexpected template content:\n%s", string(content))
	}

	svc := newTrackerConfigService()
	resolved, err := svc.ResolveTrackerProfile(repoRoot, "", "", "root-1", nil)
	if err != nil {
		t.Fatalf("expected generated config to resolve tracker profile, got %v", err)
	}
	if resolved.Name != defaultProfileName {
		t.Fatalf("expected default profile name %q, got %q", defaultProfileName, resolved.Name)
	}
	if resolved.Tracker.Type != trackerTypeTK {
		t.Fatalf("expected default tracker type %q, got %q", trackerTypeTK, resolved.Tracker.Type)
	}
	if _, err := svc.ResolveAgentDefaults(repoRoot); err != nil {
		t.Fatalf("expected generated config to resolve agent defaults, got %v", err)
	}
	if !strings.Contains(stdout, trackerConfigRelPath) {
		t.Fatalf("expected success output to mention %q, got %q", trackerConfigRelPath, stdout)
	}
}

func TestRunMainConfigInitRefusesToOverwriteWithoutForce(t *testing.T) {
	repoRoot := t.TempDir()
	writeTrackerConfigYAML(t, repoRoot, `
default_profile: custom
profiles:
  custom:
    tracker:
      type: tk
`)

	configPath := filepath.Join(repoRoot, trackerConfigRelPath)
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read existing config: %v", err)
	}

	errText := captureStderr(t, func() {
		code := RunMain([]string{"config", "init", "--repo", repoRoot}, nil)
		if code != 1 {
			t.Fatalf("expected exit code 1, got %d", code)
		}
	})

	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config after init: %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("expected existing config to remain unchanged")
	}
	if !strings.Contains(errText, "already exists") {
		t.Fatalf("expected overwrite safety message, got %q", errText)
	}
	if !strings.Contains(errText, "--force") {
		t.Fatalf("expected overwrite guidance to mention --force, got %q", errText)
	}
}

func TestRunMainConfigInitOverwritesWithForce(t *testing.T) {
	repoRoot := t.TempDir()
	writeTrackerConfigYAML(t, repoRoot, `
profiles:
  default:
    tracker:
      type: linear
      linear:
        scope:
          workspace: anomaly
        auth:
          token_env: LINEAR_TOKEN
`)

	code := RunMain([]string{"config", "init", "--repo", repoRoot, "--force"}, nil)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	configPath := filepath.Join(repoRoot, trackerConfigRelPath)
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read generated config: %v", err)
	}
	expected := "default_profile: default\nprofiles:\n  default:\n    tracker:\n      type: tk\n"
	if string(content) != expected {
		t.Fatalf("unexpected template content after force overwrite:\n%s", string(content))
	}

	svc := newTrackerConfigService()
	if _, err := svc.ResolveTrackerProfile(repoRoot, "", "", "root-1", nil); err != nil {
		t.Fatalf("expected generated config to pass tracker profile validation, got %v", err)
	}
	if _, err := svc.ResolveAgentDefaults(repoRoot); err != nil {
		t.Fatalf("expected generated config to pass agent defaults validation, got %v", err)
	}
}

func TestRunMainConfigInitHelpReturnsZero(t *testing.T) {
	errText := captureStderr(t, func() {
		code := RunMain([]string{"config", "init", "--help"}, nil)
		if code != 0 {
			t.Fatalf("expected exit code 0 for help, got %d", code)
		}
	})

	if !strings.Contains(errText, "Usage of yolo-agent config init:") {
		t.Fatalf("expected help usage text, got %q", errText)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = original
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("close reader: %v", err)
	}
	return string(data)
}
