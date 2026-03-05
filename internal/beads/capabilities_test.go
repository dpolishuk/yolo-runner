package beads

import (
	"errors"
	"strings"
	"testing"
)

type scriptResponse struct {
	output string
	err    error
}

type scriptedRunner struct {
	responses map[string]scriptResponse
	calls     [][]string
}

func (s *scriptedRunner) Run(args ...string) (string, error) {
	s.calls = append(s.calls, append([]string{}, args...))
	key := strings.Join(args, " ")
	if response, ok := s.responses[key]; ok {
		return response.output, response.err
	}
	return "", errors.New("unexpected command: " + key)
}

func TestProbeTrackerCapabilitiesDetectsBDAndActiveSync(t *testing.T) {
	runner := &scriptedRunner{responses: map[string]scriptResponse{
		"bd version": {output: "bd version 0.55.1"},
	}}

	capabilities, err := ProbeTrackerCapabilities(runner)
	if err != nil {
		t.Fatalf("probe failed: %v", err)
	}
	if capabilities.Backend != backendBD {
		t.Fatalf("expected backend %q, got %q", backendBD, capabilities.Backend)
	}
	if capabilities.SyncMode != syncModeActive {
		t.Fatalf("expected sync mode %q, got %q", syncModeActive, capabilities.SyncMode)
	}
}

func TestProbeTrackerCapabilitiesDetectsBDSyncNoopFromVersion(t *testing.T) {
	runner := &scriptedRunner{responses: map[string]scriptResponse{
		"bd version": {output: "bd version 0.56.1"},
	}}

	capabilities, err := ProbeTrackerCapabilities(runner)
	if err != nil {
		t.Fatalf("probe failed: %v", err)
	}
	if capabilities.Backend != backendBD {
		t.Fatalf("expected backend %q, got %q", backendBD, capabilities.Backend)
	}
	if capabilities.SyncMode != syncModeNoop {
		t.Fatalf("expected sync mode %q, got %q", syncModeNoop, capabilities.SyncMode)
	}
}

func TestProbeTrackerCapabilitiesDetectsBRAndFlushOnly(t *testing.T) {
	runner := &scriptedRunner{responses: map[string]scriptResponse{
		"bd version": {err: errors.New("bd not found")},
		"br version": {output: "br version 0.1.14"},
	}}

	capabilities, err := ProbeTrackerCapabilities(runner)
	if err != nil {
		t.Fatalf("probe failed: %v", err)
	}
	if capabilities.Backend != backendBR {
		t.Fatalf("expected backend %q, got %q", backendBR, capabilities.Backend)
	}
	if capabilities.SyncMode != syncModeFlushOnly {
		t.Fatalf("expected sync mode %q, got %q", syncModeFlushOnly, capabilities.SyncMode)
	}
}

func TestProbeTrackerCapabilitiesFailsWhenNoBackendDetected(t *testing.T) {
	runner := &scriptedRunner{responses: map[string]scriptResponse{
		"bd version": {err: errors.New("bd not found")},
		"br version": {err: errors.New("br not found")},
	}}

	_, err := ProbeTrackerCapabilities(runner)
	if err == nil {
		t.Fatalf("expected probe failure")
	}
	if !strings.Contains(err.Error(), "capability probe failed") {
		t.Fatalf("expected actionable error, got %v", err)
	}
}
