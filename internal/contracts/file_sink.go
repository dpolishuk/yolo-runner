package contracts

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type FileEventSink struct {
	path string
	mu   sync.Mutex
}

func NewFileEventSink(path string) *FileEventSink {
	return &FileEventSink{path: path}
}

func (s *FileEventSink) Emit(_ context.Context, event Event) error {
	if s == nil || s.path == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	line, err := MarshalEventJSONL(event)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(line)
	return err
}
