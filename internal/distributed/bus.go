package distributed

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

type Bus interface {
	Publish(ctx context.Context, subject string, event EventEnvelope) error
	Subscribe(ctx context.Context, subject string) (<-chan EventEnvelope, func(), error)
	Close() error
}

type MemoryBus struct {
	mu        sync.RWMutex
	channels  map[string][]chan EventEnvelope
	closed    bool
	closeOnce sync.Once
}

func NewMemoryBus() *MemoryBus {
	return &MemoryBus{
		channels: make(map[string][]chan EventEnvelope),
	}
}

func (b *MemoryBus) Publish(_ context.Context, subject string, event EventEnvelope) error {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return fmt.Errorf("bus closed")
	}
	consumers := append([]chan EventEnvelope{}, b.channels[subject]...)
	b.mu.RUnlock()

	for _, ch := range consumers {
		select {
		case ch <- event:
		default:
		}
	}
	return nil
}

func (b *MemoryBus) Subscribe(_ context.Context, subject string) (<-chan EventEnvelope, func(), error) {
	if b == nil {
		return nil, nil, fmt.Errorf("bus is nil")
	}
	ch := make(chan EventEnvelope, 32)
	b.mu.Lock()
	if b.closed {
		close(ch)
		b.mu.Unlock()
		return nil, nil, fmt.Errorf("bus closed")
	}
	b.channels[subject] = append(b.channels[subject], ch)
	b.mu.Unlock()

	unsub := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		subscribers := b.channels[subject]
		for i, candidate := range subscribers {
			if candidate == ch {
				b.channels[subject] = append(subscribers[:i], subscribers[i+1:]...)
				close(ch)
				return
			}
		}
	}
	return ch, unsub, nil
}

func (b *MemoryBus) Close() error {
	if b == nil {
		return nil
	}
	b.closeOnce.Do(func() {
		b.mu.Lock()
		b.closed = true
		for subject, subscribers := range b.channels {
			for _, ch := range subscribers {
				close(ch)
			}
			delete(b.channels, subject)
		}
		b.mu.Unlock()
	})
	return nil
}

func MustMarshal(raw json.RawMessage, fallback map[string]any) ([]byte, error) {
	if len(raw) > 0 {
		return raw, nil
	}
	return json.Marshal(fallback)
}
