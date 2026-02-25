package distributed

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

type fakeRedisPubSub struct {
	messages   chan *redis.Message
	closeCalls int32
}

func (p *fakeRedisPubSub) Channel(...redis.ChannelOption) <-chan *redis.Message {
	return p.messages
}

func (p *fakeRedisPubSub) Close() error {
	if atomic.CompareAndSwapInt32(&p.closeCalls, 0, 1) {
		close(p.messages)
	}
	return nil
}

type fakeRedisClient struct {
	pubSub redisPubSub
}

func (c *fakeRedisClient) Publish(_ context.Context, _ string, _ interface{}) *redis.IntCmd {
	return redis.NewIntResult(1, nil)
}

func (c *fakeRedisClient) Subscribe(_ context.Context, _ ...string) redisPubSub {
	if c.pubSub == nil {
		c.pubSub = &fakeRedisPubSub{messages: make(chan *redis.Message)}
	}
	return c.pubSub
}

func (c *fakeRedisClient) Close() error {
	if c == nil || c.pubSub == nil {
		return nil
	}
	return c.pubSub.Close()
}

func TestRedisBusSubscribeReturnsUnsubscribeThatClosesUnderlyingSubscription(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			t.Fatalf("unexpected panic: %v", err)
		}
	}()

	fakePubSub := &fakeRedisPubSub{messages: make(chan *redis.Message)}
	bus := &RedisBus{client: &fakeRedisClient{pubSub: fakePubSub}}

	out, unsubscribe, err := bus.Subscribe(context.Background(), "events")
	if err != nil {
		t.Fatalf("subscribe should return channel: %v", err)
	}
	if out == nil {
		t.Fatalf("expected output channel")
	}

	unsubscribe()
	select {
	case _, ok := <-out:
		if ok {
			t.Fatalf("expected unsubscribe to close output channel")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected output channel to close after unsubscribe")
	}

	if atomic.LoadInt32(&fakePubSub.closeCalls) != 1 {
		t.Fatalf("expected close called exactly once, got %d", atomic.LoadInt32(&fakePubSub.closeCalls))
	}

	unsubscribe()
	if atomic.LoadInt32(&fakePubSub.closeCalls) != 1 {
		t.Fatalf("expected close to be idempotent")
	}
}

func TestRedisBusSubscribeClosesUnderlyingSubscriptionOnContextCancel(t *testing.T) {
	fakePubSub := &fakeRedisPubSub{messages: make(chan *redis.Message)}
	bus := &RedisBus{client: &fakeRedisClient{pubSub: fakePubSub}}
	ctx, cancel := context.WithCancel(context.Background())

	out, _, err := bus.Subscribe(ctx, "events")
	if err != nil {
		t.Fatalf("subscribe should return channel: %v", err)
	}

	cancel()
	select {
	case _, ok := <-out:
		if ok {
			t.Fatalf("expected output channel to close after context cancel")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected output channel to close after cancel")
	}

	if atomic.LoadInt32(&fakePubSub.closeCalls) != 1 {
		t.Fatalf("expected pubsub close on context cancel, got %d", atomic.LoadInt32(&fakePubSub.closeCalls))
	}
}
