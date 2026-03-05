package beads

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/egv/yolo-runner/v2/internal/contracts"
)

// TaskManager implements contracts.TaskManager for beads tracker
type TaskManager struct {
	adapter       *Adapter
	runner        Runner
	initErr       error
	terminalMu    sync.RWMutex
	terminalState map[string]contracts.TaskStatus
}

// NewTaskManager creates a new beads task manager
func NewTaskManager(runner Runner) *TaskManager {
	manager, err := NewTaskManagerWithCapabilityProbe(runner)
	if err == nil {
		return manager
	}
	adapter := New(runner)
	return &TaskManager{
		adapter:       adapter,
		runner:        runner,
		initErr:       err,
		terminalState: map[string]contracts.TaskStatus{},
	}
}

// NewTaskManagerWithCapabilityProbe creates a beads task manager and fails when
// backend capabilities cannot be determined.
func NewTaskManagerWithCapabilityProbe(runner Runner) (*TaskManager, error) {
	adapter, err := NewWithCapabilityProbe(runner)
	if err != nil {
		return nil, err
	}
	return &TaskManager{
		adapter:       adapter,
		runner:        runner,
		terminalState: map[string]contracts.TaskStatus{},
	}, nil
}

func (m *TaskManager) startupError() error {
	if m == nil || m.initErr == nil {
		return nil
	}
	return fmt.Errorf("beads capability probe failed: %w", m.initErr)
}

// NextTasks returns the next ready tasks for the given parent
func (m *TaskManager) NextTasks(_ context.Context, parentID string) ([]contracts.TaskSummary, error) {
	if err := m.startupError(); err != nil {
		return nil, err
	}
	ready, err := m.adapter.Ready(parentID)
	if err != nil {
		return nil, err
	}

	if len(ready.Children) == 0 {
		if ready.ID == "" {
			return nil, nil
		}
		if m.isTerminal(ready.ID) {
			return nil, nil
		}
		return []contracts.TaskSummary{{ID: ready.ID, Title: ready.ID, Priority: ready.Priority}}, nil
	}

	tasks := make([]contracts.TaskSummary, 0, len(ready.Children))
	for _, child := range ready.Children {
		if m.isTerminal(child.ID) {
			continue
		}
		tasks = append(tasks, contracts.TaskSummary{ID: child.ID, Title: child.ID, Priority: child.Priority})
	}
	sort.SliceStable(tasks, func(i, j int) bool {
		if tasks[i].Priority == nil || tasks[j].Priority == nil {
			return false
		}
		return *tasks[i].Priority < *tasks[j].Priority
	})
	return tasks, nil
}

// GetTask retrieves a task by ID
func (m *TaskManager) GetTask(_ context.Context, taskID string) (contracts.Task, error) {
	if err := m.startupError(); err != nil {
		return contracts.Task{}, err
	}
	bead, err := m.adapter.Show(taskID)
	if err != nil {
		return contracts.Task{}, err
	}
	return contracts.Task{
		ID:          bead.ID,
		Title:       bead.Title,
		Description: bead.Description,
		Status:      contracts.TaskStatus(bead.Status),
	}, nil
}

// SetTaskStatus updates the status of a task
func (m *TaskManager) SetTaskStatus(_ context.Context, taskID string, status contracts.TaskStatus) error {
	if err := m.startupError(); err != nil {
		return err
	}
	if err := m.adapter.UpdateStatus(taskID, string(status)); err != nil {
		return err
	}
	switch status {
	case contracts.TaskStatusFailed, contracts.TaskStatusBlocked:
		m.markTerminal(taskID, status)
	default:
		m.clearTerminal(taskID)
	}
	return nil
}

// SetTaskData sets additional data on a task
func (m *TaskManager) SetTaskData(_ context.Context, taskID string, data map[string]string) error {
	if err := m.startupError(); err != nil {
		return err
	}
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if err := m.adapter.UpdateNotes(taskID, key+"="+data[key]); err != nil {
			return err
		}
	}
	return nil
}

func (m *TaskManager) isTerminal(taskID string) bool {
	if taskID == "" {
		return false
	}
	m.terminalMu.RLock()
	defer m.terminalMu.RUnlock()
	state, ok := m.terminalState[taskID]
	if !ok {
		return false
	}
	return state == contracts.TaskStatusFailed || state == contracts.TaskStatusBlocked
}

func (m *TaskManager) markTerminal(taskID string, state contracts.TaskStatus) {
	if taskID == "" {
		return
	}
	m.terminalMu.Lock()
	defer m.terminalMu.Unlock()
	m.terminalState[taskID] = state
}

func (m *TaskManager) clearTerminal(taskID string) {
	if taskID == "" {
		return
	}
	m.terminalMu.Lock()
	defer m.terminalMu.Unlock()
	delete(m.terminalState, taskID)
}
