package beads

import (
	"errors"
	"strings"
	"testing"
)

func TestLifecycleStrategyRoutingBD(t *testing.T) {
	runner := &scriptedRunner{responses: map[string]scriptResponse{
		"bd ready --parent root --json":         {output: `[{"id":"task-1","issue_type":"task","status":"open"}]`},
		"bd update task-1 --status in_progress": {},
		"bd sync":                               {},
	}}
	adapter := &Adapter{runner: runner, strategy: strategyFromCapabilities(TrackerCapabilities{Backend: backendBD, SyncMode: syncModeActive})}

	if _, err := adapter.Ready("root"); err != nil {
		t.Fatalf("ready failed: %v", err)
	}
	if err := adapter.UpdateStatus("task-1", "in_progress"); err != nil {
		t.Fatalf("update status failed: %v", err)
	}
	if err := adapter.Sync(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	if !containsCall(runner.calls, "bd ready --parent root --json") {
		t.Fatalf("expected bd ready call, got %v", runner.calls)
	}
	if !containsCall(runner.calls, "bd update task-1 --status in_progress") {
		t.Fatalf("expected bd update call, got %v", runner.calls)
	}
	if !containsCall(runner.calls, "bd sync") {
		t.Fatalf("expected bd sync call, got %v", runner.calls)
	}
}

func TestLifecycleStrategyRoutingBR(t *testing.T) {
	runner := &scriptedRunner{responses: map[string]scriptResponse{
		"br ready --parent root --json":         {output: `[{"id":"task-1","issue_type":"task","status":"open"}]`},
		"br update task-1 --status in_progress": {},
		"br sync --flush-only":                  {},
	}}
	adapter := &Adapter{runner: runner, strategy: strategyFromCapabilities(TrackerCapabilities{Backend: backendBR, SyncMode: syncModeFlushOnly})}

	if _, err := adapter.Ready("root"); err != nil {
		t.Fatalf("ready failed: %v", err)
	}
	if err := adapter.UpdateStatus("task-1", "in_progress"); err != nil {
		t.Fatalf("update status failed: %v", err)
	}
	if err := adapter.Sync(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	if !containsCall(runner.calls, "br ready --parent root --json") {
		t.Fatalf("expected br ready call, got %v", runner.calls)
	}
	if !containsCall(runner.calls, "br update task-1 --status in_progress") {
		t.Fatalf("expected br update call, got %v", runner.calls)
	}
	if !containsCall(runner.calls, "br sync --flush-only") {
		t.Fatalf("expected br flush-only sync call, got %v", runner.calls)
	}
}

func TestLifecycleStrategyRoutingNoopSyncSkipsCommand(t *testing.T) {
	runner := &scriptedRunner{responses: map[string]scriptResponse{}}
	adapter := &Adapter{runner: runner, strategy: strategyFromCapabilities(TrackerCapabilities{Backend: backendBD, SyncMode: syncModeNoop})}

	if err := adapter.Sync(); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("expected no sync command for noop mode, got %v", runner.calls)
	}
}

func TestNewWithCapabilityProbeSurfacesActionableError(t *testing.T) {
	runner := &scriptedRunner{responses: map[string]scriptResponse{
		"bd version": {err: errors.New("missing")},
		"br version": {err: errors.New("missing")},
	}}

	_, err := NewWithCapabilityProbe(runner)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "capability probe failed") {
		t.Fatalf("expected actionable error, got %v", err)
	}
}

func containsCall(calls [][]string, expected string) bool {
	for _, call := range calls {
		if strings.Join(call, " ") == expected {
			return true
		}
	}
	return false
}
