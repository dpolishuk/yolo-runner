package contracts

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestTaskStatusConstants(t *testing.T) {
	expected := []TaskStatus{TaskStatusOpen, TaskStatusInProgress, TaskStatusBlocked, TaskStatusClosed, TaskStatusFailed}
	for _, status := range expected {
		if status == "" {
			t.Fatalf("status constant must not be empty")
		}
	}
}

func TestRunnerResultValidate(t *testing.T) {
	valid := RunnerResult{Status: RunnerResultCompleted}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected completed to be valid: %v", err)
	}

	invalid := RunnerResult{Status: RunnerResultStatus("unknown")}
	if err := invalid.Validate(); err == nil {
		t.Fatalf("expected unknown status to fail validation")
	}
}

func TestLoopSummaryCounts(t *testing.T) {
	summary := LoopSummary{Completed: 1, Blocked: 2, Failed: 3, Skipped: 4}
	if summary.TotalProcessed() != 10 {
		t.Fatalf("expected total processed to be 10, got %d", summary.TotalProcessed())
	}
}

func TestContractInterfacesCanBeImplementedByFakes(t *testing.T) {
	ctx := context.Background()
	manager := fakeTaskManager{}
	runner := fakeAgentRunner{}
	vcs := fakeVCS{}

	tasks, err := manager.NextTasks(ctx, "root")
	if err != nil {
		t.Fatalf("next tasks failed: %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != "t-1" {
		t.Fatalf("unexpected tasks: %#v", tasks)
	}

	result, err := runner.Run(ctx, RunnerRequest{TaskID: "t-1", Mode: RunnerModeImplement})
	if err != nil {
		t.Fatalf("runner failed: %v", err)
	}
	if result.Status != RunnerResultCompleted {
		t.Fatalf("unexpected runner status: %s", result.Status)
	}

	branch, err := vcs.CreateTaskBranch(ctx, "t-1")
	if err != nil {
		t.Fatalf("create task branch failed: %v", err)
	}
	if branch == "" {
		t.Fatalf("expected non-empty branch")
	}
}

type fakeTaskManager struct{}

func (fakeTaskManager) NextTasks(context.Context, string) ([]TaskSummary, error) {
	return []TaskSummary{{ID: "t-1", Title: "test"}}, nil
}

func (fakeTaskManager) GetTask(context.Context, string) (Task, error) {
	return Task{ID: "t-1", Title: "test"}, nil
}

func (fakeTaskManager) SetTaskStatus(context.Context, string, TaskStatus) error {
	return nil
}

func (fakeTaskManager) SetTaskData(context.Context, string, map[string]string) error {
	return nil
}

type fakeAgentRunner struct{}

func (fakeAgentRunner) Run(context.Context, RunnerRequest) (RunnerResult, error) {
	return RunnerResult{Status: RunnerResultCompleted}, nil
}

type fakeVCS struct{}

func (fakeVCS) EnsureMain(context.Context) error { return nil }

func (fakeVCS) CreateTaskBranch(context.Context, string) (string, error) { return "task/t-1", nil }

func (fakeVCS) Checkout(context.Context, string) error { return nil }

func (fakeVCS) CommitAll(context.Context, string) (string, error) { return "abc123", nil }

func (fakeVCS) MergeToMain(context.Context, string) error { return nil }

func (fakeVCS) PushBranch(context.Context, string) error { return nil }

func (fakeVCS) PushMain(context.Context) error { return nil }

func TestEventDefaults(t *testing.T) {
	event := Event{Type: EventTypeTaskStarted, TaskID: "t-1", Timestamp: time.Now().UTC()}
	if event.Type == "" || event.TaskID == "" || event.Timestamp.IsZero() {
		t.Fatalf("event fields should be populated")
	}
}

func TestRunnerResultRequiresStatus(t *testing.T) {
	err := (RunnerResult{}).Validate()
	if !errors.Is(err, ErrInvalidRunnerResultStatus) {
		t.Fatalf("expected ErrInvalidRunnerResultStatus, got %v", err)
	}
}
