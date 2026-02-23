package beads

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/anomalyco/yolo-runner/internal/contracts"
)

func TestTaskManagerRoutesBDCommandsForLifecycle(t *testing.T) {
	runner := &scriptedRunner{responses: map[string]scriptResponse{
		"bd version":                            {output: "bd version 0.55.1"},
		"bd ready --parent root --json":         {output: `[{"id":"task-1","issue_type":"task","status":"open","priority":1}]`},
		"bd update task-1 --status in_progress": {},
		"bd update task-1 --notes alpha=1":      {},
		"bd update task-1 --notes beta=2":       {},
	}}

	manager, err := NewTaskManagerWithCapabilityProbe(runner)
	if err != nil {
		t.Fatalf("build manager: %v", err)
	}

	tasks, err := manager.NextTasks(context.Background(), "root")
	if err != nil {
		t.Fatalf("next tasks: %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != "task-1" {
		t.Fatalf("unexpected tasks: %#v", tasks)
	}

	if err := manager.SetTaskStatus(context.Background(), "task-1", contracts.TaskStatusInProgress); err != nil {
		t.Fatalf("set status: %v", err)
	}
	if err := manager.SetTaskData(context.Background(), "task-1", map[string]string{"beta": "2", "alpha": "1"}); err != nil {
		t.Fatalf("set data: %v", err)
	}

	if !containsCall(runner.calls, "bd ready --parent root --json") {
		t.Fatalf("expected bd ready call, got %v", runner.calls)
	}
	if !containsCall(runner.calls, "bd update task-1 --status in_progress") {
		t.Fatalf("expected bd status call, got %v", runner.calls)
	}
	if !containsCall(runner.calls, "bd update task-1 --notes alpha=1") {
		t.Fatalf("expected sorted alpha note call, got %v", runner.calls)
	}
	if !containsCall(runner.calls, "bd update task-1 --notes beta=2") {
		t.Fatalf("expected sorted beta note call, got %v", runner.calls)
	}
}

func TestTaskManagerRoutesBRCommandsForLifecycle(t *testing.T) {
	runner := &scriptedRunner{responses: map[string]scriptResponse{
		"bd version":                           {err: errors.New("bd unavailable")},
		"br version":                           {output: "br version 0.1.14"},
		"br ready --parent root --json":        {output: `[{"id":"task-2","issue_type":"task","status":"open","priority":2}]`},
		"br update task-2 --status blocked":    {},
		"br update task-2 --notes context=abc": {},
	}}

	manager, err := NewTaskManagerWithCapabilityProbe(runner)
	if err != nil {
		t.Fatalf("build manager: %v", err)
	}

	if _, err := manager.NextTasks(context.Background(), "root"); err != nil {
		t.Fatalf("next tasks: %v", err)
	}
	if err := manager.SetTaskStatus(context.Background(), "task-2", contracts.TaskStatusBlocked); err != nil {
		t.Fatalf("set status: %v", err)
	}
	if err := manager.SetTaskData(context.Background(), "task-2", map[string]string{"context": "abc"}); err != nil {
		t.Fatalf("set data: %v", err)
	}

	if !containsCall(runner.calls, "br ready --parent root --json") {
		t.Fatalf("expected br ready call, got %v", runner.calls)
	}
	if !containsCall(runner.calls, "br update task-2 --status blocked") {
		t.Fatalf("expected br status call, got %v", runner.calls)
	}
	if !containsCall(runner.calls, "br update task-2 --notes context=abc") {
		t.Fatalf("expected br notes call, got %v", runner.calls)
	}
}

func TestTaskManagerProbeFailureReturnsActionableStartupError(t *testing.T) {
	runner := &scriptedRunner{responses: map[string]scriptResponse{
		"bd version": {err: errors.New("missing")},
		"br version": {err: errors.New("missing")},
	}}

	_, err := NewTaskManagerWithCapabilityProbe(runner)
	if err == nil {
		t.Fatalf("expected probe failure")
	}
	if !strings.Contains(err.Error(), "capability probe failed") {
		t.Fatalf("expected actionable probe error, got %v", err)
	}
}
