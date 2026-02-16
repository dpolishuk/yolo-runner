package main

import (
	"strings"
	"testing"
)

func TestTrackerConfigServiceLoadModelDefaultsWhenConfigMissing(t *testing.T) {
	svc := newTrackerConfigService()

	model, err := svc.LoadModel(t.TempDir())
	if err != nil {
		t.Fatalf("expected missing config to fall back to defaults, got %v", err)
	}
	if model.DefaultProfile != defaultProfileName {
		t.Fatalf("expected default profile %q, got %q", defaultProfileName, model.DefaultProfile)
	}
	profile, ok := model.Profiles[defaultProfileName]
	if !ok {
		t.Fatalf("expected default profile %q to exist", defaultProfileName)
	}
	if profile.Tracker.Type != trackerTypeTK {
		t.Fatalf("expected default tracker type %q, got %q", trackerTypeTK, profile.Tracker.Type)
	}
}

func TestTrackerConfigServiceLoadModelRejectsInvalidYAML(t *testing.T) {
	repoRoot := t.TempDir()
	writeTrackerConfigYAML(t, repoRoot, `
profiles:
  default:
    tracker:
      type: tk
      tk: [
`)

	svc := newTrackerConfigService()
	_, err := svc.LoadModel(repoRoot)
	if err == nil {
		t.Fatalf("expected invalid YAML to fail")
	}
	if !strings.Contains(err.Error(), "cannot parse config file") {
		t.Fatalf("expected parse failure, got %q", err.Error())
	}
}

func TestTrackerConfigServiceLoadModelRejectsUnknownFields(t *testing.T) {
	repoRoot := t.TempDir()
	writeTrackerConfigYAML(t, repoRoot, `
profiles:
  default:
    tracker:
      type: tk
unexpected_key: true
`)

	svc := newTrackerConfigService()
	_, err := svc.LoadModel(repoRoot)
	if err == nil {
		t.Fatalf("expected unknown field to fail")
	}
	if !strings.Contains(err.Error(), "cannot parse config file") {
		t.Fatalf("expected parse failure, got %q", err.Error())
	}
}

func TestTrackerConfigServiceResolveAgentDefaultsRejectsBadNumber(t *testing.T) {
	repoRoot := t.TempDir()
	writeTrackerConfigYAML(t, repoRoot, `
profiles:
  default:
    tracker:
      type: tk
agent:
  concurrency: 0
`)

	svc := newTrackerConfigService()
	_, err := svc.ResolveAgentDefaults(repoRoot)
	if err == nil {
		t.Fatalf("expected invalid agent defaults to fail")
	}
	if !strings.Contains(err.Error(), "agent.concurrency") {
		t.Fatalf("expected numeric validation error, got %q", err.Error())
	}
}

func TestTrackerConfigServiceResolveAgentDefaultsRejectsBadDuration(t *testing.T) {
	repoRoot := t.TempDir()
	writeTrackerConfigYAML(t, repoRoot, `
profiles:
  default:
    tracker:
      type: tk
agent:
  runner_timeout: soon
`)

	svc := newTrackerConfigService()
	_, err := svc.ResolveAgentDefaults(repoRoot)
	if err == nil {
		t.Fatalf("expected invalid duration to fail")
	}
	if !strings.Contains(err.Error(), "agent.runner_timeout") {
		t.Fatalf("expected duration validation error, got %q", err.Error())
	}
}

func TestTrackerConfigServiceResolveAgentDefaultsRejectsUnsupportedBackend(t *testing.T) {
	repoRoot := t.TempDir()
	writeTrackerConfigYAML(t, repoRoot, `
profiles:
  default:
    tracker:
      type: tk
agent:
  backend: unsupported
`)

	svc := newTrackerConfigService()
	_, err := svc.ResolveAgentDefaults(repoRoot)
	if err == nil {
		t.Fatalf("expected unsupported backend to fail")
	}
	if !strings.Contains(err.Error(), "agent.backend") {
		t.Fatalf("expected backend field guidance, got %q", err.Error())
	}
}

func TestTrackerConfigServiceResolveTrackerProfileRejectsMissingAuthToken(t *testing.T) {
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

	svc := newTrackerConfigService()
	_, err := svc.ResolveTrackerProfile(repoRoot, "", "root-1", func(string) string { return "" })
	if err == nil {
		t.Fatalf("expected missing token validation to fail")
	}
	if !strings.Contains(err.Error(), "missing auth token") {
		t.Fatalf("expected auth token guidance, got %q", err.Error())
	}
}
