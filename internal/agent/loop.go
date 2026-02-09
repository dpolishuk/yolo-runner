package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/anomalyco/yolo-runner/internal/contracts"
)

type LoopOptions struct {
	ParentID       string
	MaxRetries     int
	MaxTasks       int
	DryRun         bool
	Stop           <-chan struct{}
	RepoRoot       string
	Model          string
	RunnerTimeout  time.Duration
	VCS            contracts.VCS
	RequireReview  bool
	MergeOnSuccess bool
}

type Loop struct {
	tasks   contracts.TaskManager
	runner  contracts.AgentRunner
	events  contracts.EventSink
	options LoopOptions
}

func NewLoop(tasks contracts.TaskManager, runner contracts.AgentRunner, events contracts.EventSink, options LoopOptions) *Loop {
	return &Loop{tasks: tasks, runner: runner, events: events, options: options}
}

func (l *Loop) Run(ctx context.Context) (contracts.LoopSummary, error) {
	summary := contracts.LoopSummary{}
	for {
		if l.stopRequested() {
			return summary, nil
		}
		if l.options.MaxTasks > 0 && summary.TotalProcessed() >= l.options.MaxTasks {
			return summary, nil
		}

		next, err := l.tasks.NextTasks(ctx, l.options.ParentID)
		if err != nil {
			return summary, err
		}
		if len(next) == 0 {
			return summary, nil
		}

		taskID := next[0].ID
		task, err := l.tasks.GetTask(ctx, taskID)
		if err != nil {
			return summary, err
		}

		if l.options.DryRun {
			summary.Skipped++
			return summary, nil
		}

		taskBranch := ""
		if l.options.VCS != nil {
			if err := l.options.VCS.EnsureMain(ctx); err != nil {
				return summary, err
			}
			branch, err := l.options.VCS.CreateTaskBranch(ctx, task.ID)
			if err != nil {
				return summary, err
			}
			taskBranch = branch
			if err := l.options.VCS.Checkout(ctx, branch); err != nil {
				return summary, err
			}
		}

		retries := 0
		for {
			if err := l.tasks.SetTaskStatus(ctx, task.ID, contracts.TaskStatusInProgress); err != nil {
				return summary, err
			}

			result, err := l.runner.Run(ctx, contracts.RunnerRequest{
				TaskID:   task.ID,
				ParentID: l.options.ParentID,
				Mode:     contracts.RunnerModeImplement,
				RepoRoot: l.options.RepoRoot,
				Model:    l.options.Model,
				Timeout:  l.options.RunnerTimeout,
				Prompt:   buildPrompt(task, contracts.RunnerModeImplement),
			})
			if err != nil {
				return summary, err
			}

			if result.Status == contracts.RunnerResultCompleted && l.options.RequireReview {
				reviewResult, reviewErr := l.runner.Run(ctx, contracts.RunnerRequest{
					TaskID:   task.ID,
					ParentID: l.options.ParentID,
					Mode:     contracts.RunnerModeReview,
					RepoRoot: l.options.RepoRoot,
					Model:    l.options.Model,
					Timeout:  l.options.RunnerTimeout,
					Prompt:   buildPrompt(task, contracts.RunnerModeReview),
				})
				if reviewErr != nil {
					return summary, reviewErr
				}
				if reviewResult.Status != contracts.RunnerResultCompleted {
					result = reviewResult
				}
			}

			switch result.Status {
			case contracts.RunnerResultCompleted:
				if l.options.MergeOnSuccess && l.options.VCS != nil && taskBranch != "" {
					if err := l.options.VCS.MergeToMain(ctx, taskBranch); err != nil {
						return summary, err
					}
					if err := l.options.VCS.PushMain(ctx); err != nil {
						return summary, err
					}
				}
				if err := l.tasks.SetTaskStatus(ctx, task.ID, contracts.TaskStatusClosed); err != nil {
					return summary, err
				}
				summary.Completed++
			case contracts.RunnerResultBlocked:
				if err := l.tasks.SetTaskStatus(ctx, task.ID, contracts.TaskStatusBlocked); err != nil {
					return summary, err
				}
				if result.Reason != "" {
					if err := l.tasks.SetTaskData(ctx, task.ID, map[string]string{"blocked_reason": result.Reason}); err != nil {
						return summary, err
					}
				}
				summary.Blocked++
			case contracts.RunnerResultFailed:
				retries++
				if retries <= l.options.MaxRetries {
					if err := l.tasks.SetTaskData(ctx, task.ID, map[string]string{"retry_count": fmt.Sprintf("%d", retries)}); err != nil {
						return summary, err
					}
					if err := l.tasks.SetTaskStatus(ctx, task.ID, contracts.TaskStatusOpen); err != nil {
						return summary, err
					}
					continue
				}
				if err := l.tasks.SetTaskStatus(ctx, task.ID, contracts.TaskStatusFailed); err != nil {
					return summary, err
				}
				summary.Failed++
			default:
				if err := l.tasks.SetTaskStatus(ctx, task.ID, contracts.TaskStatusFailed); err != nil {
					return summary, err
				}
				summary.Failed++
			}
			break
		}
	}
}

func buildPrompt(task contracts.Task, mode contracts.RunnerMode) string {
	modeLine := "Implementation"
	if mode == contracts.RunnerModeReview {
		modeLine = "Review"
	}
	sections := []string{
		"Mode: " + modeLine,
		"Task ID: " + task.ID,
		"Title: " + task.Title,
	}
	if strings.TrimSpace(task.Description) != "" {
		sections = append(sections, "Description:\n"+task.Description)
	}
	return strings.Join(sections, "\n\n")
}

func (l *Loop) stopRequested() bool {
	if l.options.Stop == nil {
		return false
	}
	select {
	case <-l.options.Stop:
		return true
	default:
		return false
	}
}
