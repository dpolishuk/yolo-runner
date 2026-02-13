package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type JSONLQueue struct {
	path string
	mu   sync.Mutex
}

type queuedJob struct {
	EnqueuedAt time.Time `json:"enqueuedAt"`
	Job
}

func NewJSONLQueue(path string) *JSONLQueue {
	return &JSONLQueue{path: strings.TrimSpace(path)}
}

func (q *JSONLQueue) Enqueue(ctx context.Context, job Job) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	if q.path == "" {
		return fmt.Errorf("queue path is required")
	}
	if err := os.MkdirAll(filepath.Dir(q.path), 0o755); err != nil {
		return fmt.Errorf("create queue directory: %w", err)
	}

	file, err := os.OpenFile(q.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open queue file: %w", err)
	}
	defer file.Close()

	record := queuedJob{
		EnqueuedAt: time.Now().UTC(),
		Job:        job,
	}
	line, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal queued job: %w", err)
	}
	line = append(line, '\n')

	if _, err := file.Write(line); err != nil {
		return fmt.Errorf("append queued job: %w", err)
	}
	return nil
}
