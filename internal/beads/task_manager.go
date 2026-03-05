package beads

import (
	"context"
	"strings"

	"github.com/egv/yolo-runner/v2/internal/contracts"
)

type TaskManager struct {
	adapter *Adapter
}

func NewTaskManager(runner Runner) *TaskManager {
	return &TaskManager{
		adapter: New(runner),
	}
}

func (m *TaskManager) NextTasks(_ context.Context, parentID string) ([]contracts.TaskSummary, error) {
	ready, err := m.adapter.Ready(parentID)
	if err != nil {
		return nil, err
	}

	if len(ready.Children) == 0 {
		if ready.ID == "" {
			return nil, nil
		}
		title := ready.ID
		return []contracts.TaskSummary{{ID: ready.ID, Title: title, Priority: ready.Priority}}, nil
	}

	tasks := make([]contracts.TaskSummary, 0, len(ready.Children))
	for _, child := range ready.Children {
		tasks = append(tasks, contracts.TaskSummary{
			ID:       child.ID,
			Title:    child.ID,
			Priority: child.Priority,
		})
	}
	return tasks, nil
}

func (m *TaskManager) GetTask(_ context.Context, taskID string) (contracts.Task, error) {
	bead, err := m.adapter.Show(taskID)
	if err != nil {
		return contracts.Task{}, err
	}
	return contracts.Task{
		ID:          bead.ID,
		Title:       bead.Title,
		Description: bead.Description,
		Status:      contracts.TaskStatus(bead.Status),
		Metadata:    nil,
	}, nil
}

func (m *TaskManager) SetTaskStatus(_ context.Context, taskID string, status contracts.TaskStatus) error {
	return m.adapter.UpdateStatus(taskID, strings.ToLower(string(status)))
}

func (m *TaskManager) SetTaskData(_ context.Context, taskID string, data map[string]string) error {
	// Beads doesn't support arbitrary key-value data; no-op for now
	_ = taskID
	_ = data
	return nil
}
