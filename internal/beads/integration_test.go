package beads

import (
	"context"
	"errors"
	"testing"

	"github.com/egv/yolo-runner/v2/internal/contracts"
)

func TestTaskManagerGetTaskReturnsTaskDetails(t *testing.T) {
	runner := &scriptedRunner{responses: map[string]scriptResponse{
		"bd version":            {output: "bd version 0.55.1"},
		"bd show task-1 --json": {output: `[{"id":"task-1","title":"Fix bug","description":"Fix the login bug","status":"open"}]`},
	}}

	manager, err := NewTaskManagerWithCapabilityProbe(runner)
	if err != nil {
		t.Fatalf("build manager: %v", err)
	}

	task, err := manager.GetTask(context.Background(), "task-1")
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if task.ID != "task-1" {
		t.Fatalf("expected id task-1, got %q", task.ID)
	}
	if task.Title != "Fix bug" {
		t.Fatalf("expected title 'Fix bug', got %q", task.Title)
	}
	if task.Description != "Fix the login bug" {
		t.Fatalf("expected description, got %q", task.Description)
	}
	if task.Status != contracts.TaskStatusOpen {
		t.Fatalf("expected status open, got %v", task.Status)
	}
}

func TestTaskManagerTerminalStateBlocksTaskFromFutureNextTasks(t *testing.T) {
	runner := &scriptedRunner{responses: map[string]scriptResponse{
		"bd version":                       {output: "bd version 0.55.1"},
		"bd ready --parent root --json":    {output: `[{"id":"task-1","issue_type":"task","status":"open","priority":1}]`},
		"bd update task-1 --status failed": {},
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

	if err := manager.SetTaskStatus(context.Background(), "task-1", contracts.TaskStatusFailed); err != nil {
		t.Fatalf("set status: %v", err)
	}

	tasks, err = manager.NextTasks(context.Background(), "root")
	if err != nil {
		t.Fatalf("next tasks after failure: %v", err)
	}
	if len(tasks) != 0 {
		t.Fatalf("expected no tasks after failure, got: %#v", tasks)
	}
}

func TestTaskManagerTerminalStateBlocksTaskWithBlockedStatus(t *testing.T) {
	runner := &scriptedRunner{responses: map[string]scriptResponse{
		"bd version":                        {output: "bd version 0.55.1"},
		"bd ready --parent root --json":     {output: `[{"id":"task-1","issue_type":"task","status":"open","priority":1},{"id":"task-2","issue_type":"task","status":"open","priority":2}]`},
		"bd update task-1 --status blocked": {},
	}}

	manager, err := NewTaskManagerWithCapabilityProbe(runner)
	if err != nil {
		t.Fatalf("build manager: %v", err)
	}

	tasks, err := manager.NextTasks(context.Background(), "root")
	if err != nil {
		t.Fatalf("next tasks: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}

	if err := manager.SetTaskStatus(context.Background(), "task-1", contracts.TaskStatusBlocked); err != nil {
		t.Fatalf("set status: %v", err)
	}

	tasks, err = manager.NextTasks(context.Background(), "root")
	if err != nil {
		t.Fatalf("next tasks after blocked: %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != "task-2" {
		t.Fatalf("expected only task-2, got: %#v", tasks)
	}
}

func TestTaskManagerNextTasksSortsByPriority(t *testing.T) {
	priority3 := 3
	priority1 := 1
	priority2 := 2
	runner := &scriptedRunner{responses: map[string]scriptResponse{
		"bd version":                    {output: "bd version 0.55.1"},
		"bd ready --parent root --json": {output: `[{"id":"task-3","issue_type":"task","status":"open","priority":3},{"id":"task-1","issue_type":"task","status":"open","priority":1},{"id":"task-2","issue_type":"task","status":"open","priority":2}]`},
	}}

	manager, err := NewTaskManagerWithCapabilityProbe(runner)
	if err != nil {
		t.Fatalf("build manager: %v", err)
	}

	tasks, err := manager.NextTasks(context.Background(), "root")
	if err != nil {
		t.Fatalf("next tasks: %v", err)
	}
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}
	if tasks[0].ID != "task-1" || tasks[1].ID != "task-2" || tasks[2].ID != "task-3" {
		t.Fatalf("expected tasks sorted by priority, got: %#v", tasks)
	}
	if *tasks[0].Priority != priority1 || *tasks[1].Priority != priority2 || *tasks[2].Priority != priority3 {
		t.Fatalf("priorities not preserved correctly")
	}
}

func TestTaskManagerNextTasksHandlesNilPriority(t *testing.T) {
	runner := &scriptedRunner{responses: map[string]scriptResponse{
		"bd version":                    {output: "bd version 0.55.1"},
		"bd ready --parent root --json": {output: `[{"id":"task-1","issue_type":"task","status":"open"},{"id":"task-2","issue_type":"task","status":"open","priority":1}]`},
	}}

	manager, err := NewTaskManagerWithCapabilityProbe(runner)
	if err != nil {
		t.Fatalf("build manager: %v", err)
	}

	tasks, err := manager.NextTasks(context.Background(), "root")
	if err != nil {
		t.Fatalf("next tasks: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
}

func TestTaskManagerSetTaskDataSortsKeys(t *testing.T) {
	runner := &scriptedRunner{responses: map[string]scriptResponse{
		"bd version":                       {output: "bd version 0.55.1"},
		"bd update task-1 --notes alpha=2": {},
		"bd update task-1 --notes beta=3":  {},
		"bd update task-1 --notes zebra=1": {},
	}}

	manager, err := NewTaskManagerWithCapabilityProbe(runner)
	if err != nil {
		t.Fatalf("build manager: %v", err)
	}

	err = manager.SetTaskData(context.Background(), "task-1", map[string]string{
		"zebra": "1",
		"alpha": "2",
		"beta":  "3",
	})
	if err != nil {
		t.Fatalf("set data: %v", err)
	}

	if len(runner.calls) != 4 {
		t.Fatalf("expected 4 calls (version + 3 notes), got %d: %v", len(runner.calls), runner.calls)
	}
	if runner.calls[1][4] != "alpha=2" {
		t.Fatalf("expected alpha second (after version), got %q", runner.calls[1][4])
	}
	if runner.calls[2][4] != "beta=3" {
		t.Fatalf("expected beta third, got %q", runner.calls[2][4])
	}
	if runner.calls[3][4] != "zebra=1" {
		t.Fatalf("expected zebra last, got %q", runner.calls[3][4])
	}
}

func TestTaskManagerStartupErrorPropagates(t *testing.T) {
	runner := &scriptedRunner{responses: map[string]scriptResponse{
		"bd version": {err: errors.New("bd not found")},
		"br version": {err: errors.New("br not found")},
	}}

	manager := NewTaskManager(runner)

	_, err := manager.NextTasks(context.Background(), "root")
	if err == nil {
		t.Fatalf("expected error when manager has startup error")
	}
	if !containsString(err.Error(), "beads capability probe failed") {
		t.Fatalf("expected capability probe error, got: %v", err)
	}
}

func TestAdapterTreeReturnsIssueTree(t *testing.T) {
	runner := &scriptedRunner{responses: map[string]scriptResponse{
		"bd version":                   {output: "bd version 0.55.1"},
		"bd list --parent root --json": {output: `[{"id":"root","issue_type":"epic","status":"open","children":[{"id":"task-1","issue_type":"task","status":"open"}]}]`},
	}}

	adapter, err := NewWithCapabilityProbe(runner)
	if err != nil {
		t.Fatalf("build adapter: %v", err)
	}

	issue, err := adapter.Tree("root")
	if err != nil {
		t.Fatalf("tree: %v", err)
	}
	if issue.ID != "root" {
		t.Fatalf("expected root id, got %q", issue.ID)
	}
	if len(issue.Children) != 1 || issue.Children[0].ID != "task-1" {
		t.Fatalf("expected child task-1, got: %#v", issue.Children)
	}
}

func TestAdapterCloseEligibleRoutesToBD(t *testing.T) {
	runner := &scriptedRunner{responses: map[string]scriptResponse{
		"bd version":             {output: "bd version 0.55.1"},
		"bd epic close-eligible": {},
	}}

	adapter, err := NewWithCapabilityProbe(runner)
	if err != nil {
		t.Fatalf("build adapter: %v", err)
	}

	err = adapter.CloseEligible()
	if err != nil {
		t.Fatalf("close eligible: %v", err)
	}
	if !containsCall(runner.calls, "bd epic close-eligible") {
		t.Fatalf("expected bd epic close-eligible call, got %v", runner.calls)
	}
}

func TestAdapterCloseEligibleRoutesToBR(t *testing.T) {
	runner := &scriptedRunner{responses: map[string]scriptResponse{
		"bd version":             {err: errors.New("bd not found")},
		"br version":             {output: "br version 0.1.14"},
		"br epic close-eligible": {},
	}}

	adapter, err := NewWithCapabilityProbe(runner)
	if err != nil {
		t.Fatalf("build adapter: %v", err)
	}

	err = adapter.CloseEligible()
	if err != nil {
		t.Fatalf("close eligible: %v", err)
	}
	if !containsCall(runner.calls, "br epic close-eligible") {
		t.Fatalf("expected br epic close-eligible call, got %v", runner.calls)
	}
}

func TestAdapterCloseRoutesToBD(t *testing.T) {
	runner := &scriptedRunner{responses: map[string]scriptResponse{
		"bd version":      {output: "bd version 0.55.1"},
		"bd close task-1": {},
	}}

	adapter, err := NewWithCapabilityProbe(runner)
	if err != nil {
		t.Fatalf("build adapter: %v", err)
	}

	err = adapter.Close("task-1")
	if err != nil {
		t.Fatalf("close: %v", err)
	}
	if !containsCall(runner.calls, "bd close task-1") {
		t.Fatalf("expected bd close call, got %v", runner.calls)
	}
}

func TestAdapterUpdateStatusWithReasonSanitizesAndUpdates(t *testing.T) {
	runner := &scriptedRunner{responses: map[string]scriptResponse{
		"bd version":                             {output: "bd version 0.55.1"},
		"bd update task-1 --status failed":       {},
		"bd update task-1 --notes Test; failure": {},
	}}

	adapter, err := NewWithCapabilityProbe(runner)
	if err != nil {
		t.Fatalf("build adapter: %v", err)
	}

	err = adapter.UpdateStatusWithReason("task-1", "failed", "Test\nfailure")
	if err != nil {
		t.Fatalf("update status with reason: %v", err)
	}
	if !containsCall(runner.calls, "bd update task-1 --status failed") {
		t.Fatalf("expected status update call, got %v", runner.calls)
	}
	if !containsCall(runner.calls, "bd update task-1 --notes Test; failure") {
		t.Fatalf("expected notes update call, got %v", runner.calls)
	}
}

func TestIsAvailableReturnsFalseWhenBeadsDirMissing(t *testing.T) {
	available := IsAvailable("/nonexistent/path")
	if available {
		t.Fatalf("expected beads to not be available for nonexistent path")
	}
}

func TestSanitizeReasonHandlesEmptyString(t *testing.T) {
	result := sanitizeReason("")
	if result != "" {
		t.Fatalf("expected empty string, got %q", result)
	}
}

func TestSanitizeReasonHandlesWhitespaceOnly(t *testing.T) {
	result := sanitizeReason("   \n\r   ")
	if result != "" {
		t.Fatalf("expected empty string, got %q", result)
	}
}

func TestSanitizeReasonReplacesNewlines(t *testing.T) {
	result := sanitizeReason("line1\nline2\r\nline3")
	if result != "line1; line2; line3" {
		t.Fatalf("expected newlines replaced with semicolons, got %q", result)
	}
}

func TestAdapterReadyWithEmptyChildrenReturnsLeafTask(t *testing.T) {
	runner := &scriptedRunner{responses: map[string]scriptResponse{
		"bd version":                    {output: "bd version 0.55.1"},
		"bd ready --parent root --json": {output: `[{"id":"task-1","issue_type":"task","status":"open","priority":1}]`},
	}}

	adapter, err := NewWithCapabilityProbe(runner)
	if err != nil {
		t.Fatalf("build adapter: %v", err)
	}

	issue, err := adapter.Ready("root")
	if err != nil {
		t.Fatalf("ready: %v", err)
	}
	if issue.ID != "task-1" {
		t.Fatalf("expected task-1, got %q", issue.ID)
	}
}

func TestAdapterReadyWithMultipleChildrenReturnsEpicWithChildren(t *testing.T) {
	runner := &scriptedRunner{responses: map[string]scriptResponse{
		"bd version":                    {output: "bd version 0.55.1"},
		"bd ready --parent root --json": {output: `[{"id":"task-1","issue_type":"task","status":"open","priority":1},{"id":"task-2","issue_type":"task","status":"open","priority":2}]`},
	}}

	adapter, err := NewWithCapabilityProbe(runner)
	if err != nil {
		t.Fatalf("build adapter: %v", err)
	}

	issue, err := adapter.Ready("root")
	if err != nil {
		t.Fatalf("ready: %v", err)
	}
	if issue.ID != "root" {
		t.Fatalf("expected root id, got %q", issue.ID)
	}
	if len(issue.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(issue.Children))
	}
}

func TestAdapterShowReturnsBeadDetails(t *testing.T) {
	runner := &scriptedRunner{responses: map[string]scriptResponse{
		"bd version":            {output: "bd version 0.55.1"},
		"bd show task-1 --json": {output: `[{"id":"task-1","title":"Fix bug","description":"The bug fix","status":"open"}]`},
	}}

	adapter, err := NewWithCapabilityProbe(runner)
	if err != nil {
		t.Fatalf("build adapter: %v", err)
	}

	bead, err := adapter.Show("task-1")
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	if bead.ID != "task-1" {
		t.Fatalf("expected task-1, got %q", bead.ID)
	}
	if bead.Title != "Fix bug" {
		t.Fatalf("expected title, got %q", bead.Title)
	}
}

func TestAdapterShowReturnsEmptyWhenNotFound(t *testing.T) {
	runner := &scriptedRunner{responses: map[string]scriptResponse{
		"bd version":                 {output: "bd version 0.55.1"},
		"bd show nonexistent --json": {output: `[]`},
	}}

	adapter, err := NewWithCapabilityProbe(runner)
	if err != nil {
		t.Fatalf("build adapter: %v", err)
	}

	bead, err := adapter.Show("nonexistent")
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	if bead.ID != "" {
		t.Fatalf("expected empty bead, got %q", bead.ID)
	}
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
