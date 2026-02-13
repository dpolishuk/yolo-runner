package webhook

import (
	"context"
	"sync"
)

type AsyncDispatcher struct {
	sink  QueueSink
	queue chan Job
	done  chan struct{}

	mu        sync.RWMutex
	closed    bool
	closeOnce sync.Once
}

func NewAsyncDispatcher(sink QueueSink, bufferSize int) *AsyncDispatcher {
	if bufferSize <= 0 {
		bufferSize = 1
	}
	d := &AsyncDispatcher{
		sink:  sink,
		queue: make(chan Job, bufferSize),
		done:  make(chan struct{}),
	}
	go d.loop()
	return d
}

func (d *AsyncDispatcher) Dispatch(ctx context.Context, job Job) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.closed {
		return ErrDispatcherClosed
	}

	select {
	case d.queue <- job:
		return nil
	default:
		return ErrQueueFull
	}
}

func (d *AsyncDispatcher) Close(ctx context.Context) error {
	d.closeOnce.Do(func() {
		d.mu.Lock()
		d.closed = true
		close(d.queue)
		d.mu.Unlock()
	})

	select {
	case <-d.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (d *AsyncDispatcher) loop() {
	defer close(d.done)
	for job := range d.queue {
		if d.sink == nil {
			continue
		}
		_ = d.sink.Enqueue(context.Background(), job)
	}
}
