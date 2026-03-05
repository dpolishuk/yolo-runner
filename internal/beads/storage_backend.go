package beads

import (
	"context"
	"fmt"
	"strings"

	"github.com/egv/yolo-runner/v2/internal/contracts"
)

// StorageBackend adapts beads to the storage-only contracts.StorageBackend API.
type StorageBackend struct {
	manager *TaskManager
}

var _ contracts.StorageBackend = (*StorageBackend)(nil)

func NewStorageBackend(runner Runner) *StorageBackend {
	return &StorageBackend{
		manager: NewTaskManager(runner),
	}
}

func (b *StorageBackend) GetTaskTree(ctx context.Context, rootID string) (*contracts.TaskTree, error) {
	if b == nil || b.manager == nil {
		return nil, fmt.Errorf("beads storage backend is not initialized")
	}
	rootID = strings.TrimSpace(rootID)
	if rootID == "" {
		return nil, fmt.Errorf("root task ID is required")
	}

	rootTask, err := b.manager.GetTask(ctx, rootID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(rootTask.ID) == "" {
		return nil, fmt.Errorf("root task %q not found", rootID)
	}

	// Build tree from root
	tree := &contracts.TaskTree{
		Root:  rootTask,
		Tasks: map[string]contracts.Task{rootTask.ID: rootTask},
	}
	return tree, nil
}

func (b *StorageBackend) GetTask(ctx context.Context, taskID string) (*contracts.Task, error) {
	if b == nil || b.manager == nil {
		return nil, fmt.Errorf("beads storage backend is not initialized")
	}

	task, err := b.manager.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(task.ID) == "" {
		return nil, fmt.Errorf("task %q not found", strings.TrimSpace(taskID))
	}
	return &task, nil
}

func (b *StorageBackend) SetTaskStatus(ctx context.Context, taskID string, status contracts.TaskStatus) error {
	if b == nil || b.manager == nil {
		return fmt.Errorf("beads storage backend is not initialized")
	}
	return b.manager.SetTaskStatus(ctx, taskID, status)
}

func (b *StorageBackend) SetTaskData(ctx context.Context, taskID string, data map[string]string) error {
	if b == nil || b.manager == nil {
		return fmt.Errorf("beads storage backend is not initialized")
	}
	return b.manager.SetTaskData(ctx, taskID, data)
}
