package webhook

import (
	"context"
	"sync"
	"testing"
	"time"
)

type recordSink struct {
	mu   sync.Mutex
	jobs []Job
}

func (s *recordSink) Enqueue(_ context.Context, job Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs = append(s.jobs, job)
	return nil
}

func (s *recordSink) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.jobs)
}

func TestAsyncDispatcherDeliversJobsToSink(t *testing.T) {
	sink := &recordSink{}
	dispatcher := NewAsyncDispatcher(sink, 8)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = dispatcher.Close(ctx)
	}()

	job := Job{ID: "job-1"}
	if err := dispatcher.Dispatch(context.Background(), job); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	deadline := time.After(2 * time.Second)
	for sink.count() == 0 {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for job delivery")
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func TestAsyncDispatcherRejectsWhenClosed(t *testing.T) {
	sink := &recordSink{}
	dispatcher := NewAsyncDispatcher(sink, 2)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := dispatcher.Close(ctx); err != nil {
		t.Fatalf("close dispatcher: %v", err)
	}

	if err := dispatcher.Dispatch(context.Background(), Job{ID: "job-1"}); err == nil {
		t.Fatal("expected dispatch after close to fail")
	}
}

func TestAsyncDispatcherReturnsQueueFullWhenBufferIsSaturated(t *testing.T) {
	sink := &slowQueueSink{release: make(chan struct{}), started: make(chan struct{}, 1)}
	dispatcher := NewAsyncDispatcher(sink, 1)
	defer func() {
		close(sink.release)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = dispatcher.Close(ctx)
	}()

	if err := dispatcher.Dispatch(context.Background(), Job{ID: "job-1"}); err != nil {
		t.Fatalf("first dispatch failed: %v", err)
	}
	select {
	case <-sink.started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for sink to start")
	}
	if err := dispatcher.Dispatch(context.Background(), Job{ID: "job-2"}); err != nil {
		t.Fatalf("second dispatch failed: %v", err)
	}
	if err := dispatcher.Dispatch(context.Background(), Job{ID: "job-3"}); err != ErrQueueFull {
		t.Fatalf("expected ErrQueueFull, got %v", err)
	}
}
