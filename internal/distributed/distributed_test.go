package distributed

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/egv/yolo-runner/v2/internal/contracts"
)

type fakeRunner struct {
	result contracts.RunnerResult
	err    error
}

func (r fakeRunner) Run(_ context.Context, request contracts.RunnerRequest) (contracts.RunnerResult, error) {
	return r.result, r.err
}

func TestParseEventEnvelopeSupportsLegacyAndV1Schemas(t *testing.T) {
	t.Run("legacy event defaults to v0", func(t *testing.T) {
		legacyPayload := []byte(`{"type":"executor_registered","source":"old-exec","payload":{"executor_id":"exec-1","capabilities":["implement"]}}`)
		evt, err := ParseEventEnvelope(legacyPayload)
		if err != nil {
			t.Fatalf("parse legacy envelope: %v", err)
		}
		if evt.SchemaVersion != EventSchemaVersionV0 {
			t.Fatalf("expected legacy schema version %q, got %q", EventSchemaVersionV0, evt.SchemaVersion)
		}
	})

	t.Run("versioned event preserves schema and type", func(t *testing.T) {
		msg, err := NewEventEnvelope(EventTypeExecutorHeartbeat, "exec", "corr", ExecutorHeartbeatPayload{ExecutorID: "exec"})
		if err != nil {
			t.Fatalf("new envelope: %v", err)
		}
		raw, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("marshal envelope: %v", err)
		}
		parsed, err := ParseEventEnvelope(raw)
		if err != nil {
			t.Fatalf("parse envelope: %v", err)
		}
		if parsed.SchemaVersion != EventSchemaVersionV1 {
			t.Fatalf("expected schema %q, got %q", EventSchemaVersionV1, parsed.SchemaVersion)
		}
		if parsed.Type != EventTypeExecutorHeartbeat {
			t.Fatalf("expected type %q, got %q", EventTypeExecutorHeartbeat, parsed.Type)
		}
	})
}

func TestExecutorRegistryRoutesByCapabilitiesAndEvictsStale(t *testing.T) {
	registry := NewExecutorRegistry(20*time.Millisecond, func() time.Time { return time.Now().UTC() })
	registry.Register(ExecutorRegistrationPayload{ExecutorID: "implement-only", Capabilities: []Capability{CapabilityImplement}})
	registry.Register(ExecutorRegistrationPayload{ExecutorID: "reviewer", Capabilities: []Capability{CapabilityReview, CapabilityImplement}})

	reviewer, err := registry.Pick(CapabilityReview)
	if err != nil {
		t.Fatalf("expected review executor, got error %v", err)
	}
	if reviewer.ID != "reviewer" {
		t.Fatalf("expected reviewer, got %q", reviewer.ID)
	}

	// advance clock forward to expire entries
	time.Sleep(30 * time.Millisecond)
	_, err = registry.Pick(CapabilityReview)
	if err == nil {
		t.Fatalf("expected stale registry to return no capable executors")
	}
}

func TestMastermindRoutesTaskBasedOnCapabilities(t *testing.T) {
	bus := NewMemoryBus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mastermind := NewMastermind(MastermindOptions{
		ID:             "mastermind",
		Bus:            bus,
		RequestTimeout: 2 * time.Second,
		RegistryTTL:    2 * time.Second,
	})
	if err := mastermind.Start(ctx); err != nil {
		t.Fatalf("start mastermind: %v", err)
	}
	resultCh := make(chan string, 8)
	reviewExecutor := NewExecutorWorker(ExecutorWorkerOptions{
		ID:           "review-exec",
		Bus:          bus,
		Runner:       fakeRunner{result: contracts.RunnerResult{Status: contracts.RunnerResultCompleted, Artifacts: map[string]string{"worker": "review"}}},
		Capabilities: []Capability{CapabilityReview},
	})
	implementExecutor := NewExecutorWorker(ExecutorWorkerOptions{
		ID:           "impl-exec",
		Bus:          bus,
		Runner:       fakeRunner{result: contracts.RunnerResult{Status: contracts.RunnerResultCompleted, Artifacts: map[string]string{"worker": "implement"}}, err: fmt.Errorf("should not execute")},
		Capabilities: []Capability{CapabilityImplement},
	})
	go func() { _ = reviewExecutor.Start(ctx) }()
	go func() { _ = implementExecutor.Start(ctx) }()
	_ = resultCh

	time.Sleep(20 * time.Millisecond)
	result, err := mastermind.DispatchTask(ctx, TaskDispatchRequest{
		RunnerRequest: contracts.RunnerRequest{TaskID: "task-review", Mode: contracts.RunnerModeReview},
	})
	if err != nil {
		t.Fatalf("dispatch review task: %v", err)
	}
	if result.Status != contracts.RunnerResultCompleted {
		t.Fatalf("expected completed result, got %q", result.Status)
	}
	if result.Artifacts["worker"] != "review" {
		t.Fatalf("expected review worker result, got %v", result.Artifacts)
	}
}

func TestExecutorCanRequestServiceFromMastermind(t *testing.T) {
	bus := NewMemoryBus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serviceHandled := make(chan string, 1)
	mastermind := NewMastermind(MastermindOptions{
		ID:          "mastermind",
		Bus:         bus,
		RegistryTTL: 2 * time.Second,
		ServiceHandler: func(ctx context.Context, request ServiceRequestPayload) (ServiceResponsePayload, error) {
			serviceHandled <- request.Service
			return ServiceResponsePayload{Artifacts: map[string]string{"service": request.Service}}, nil
		},
	})
	if err := mastermind.Start(ctx); err != nil {
		t.Fatalf("start mastermind: %v", err)
	}

	executor := NewExecutorWorker(ExecutorWorkerOptions{
		ID:           "executor",
		Bus:          bus,
		Runner:       fakeRunner{result: contracts.RunnerResult{Status: contracts.RunnerResultCompleted}},
		Capabilities: []Capability{CapabilityImplement},
	})
	go func() { _ = executor.Start(ctx) }()

	time.Sleep(20 * time.Millisecond)
	response, err := executor.RequestService(ctx, ServiceRequestPayload{TaskID: "t1", Service: "review-with-larger-model"})
	if err != nil {
		t.Fatalf("request service: %v", err)
	}
	if response.Artifacts["service"] != "review-with-larger-model" {
		t.Fatalf("expected service response artifact, got %v", response.Artifacts)
	}
	select {
	case name := <-serviceHandled:
		if name != "review-with-larger-model" {
			t.Fatalf("expected service review-with-larger-model, got %q", name)
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("expected service handler to run")
	}
}

func TestMastermindReturnsErrorWhenExecutorDisconnects(t *testing.T) {
	bus := NewMemoryBus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	clock := time.Now
	mastermind := NewMastermind(MastermindOptions{
		ID:             "mastermind",
		Bus:            bus,
		RegistryTTL:    10 * time.Millisecond,
		RequestTimeout: 80 * time.Millisecond,
		Clock:          clock,
	})
	if err := mastermind.Start(ctx); err != nil {
		t.Fatalf("start mastermind: %v", err)
	}
	executor := NewExecutorWorker(ExecutorWorkerOptions{
		ID:           "executor",
		Bus:          bus,
		Runner:       fakeRunner{result: contracts.RunnerResult{Status: contracts.RunnerResultCompleted}},
		Capabilities: []Capability{CapabilityImplement},
		Clock:        clock,
	})
	go func() { _ = executor.Start(ctx) }()
	time.Sleep(20 * time.Millisecond)
	time.Sleep(30 * time.Millisecond)
	_, err := mastermind.DispatchTask(ctx, TaskDispatchRequest{
		RunnerRequest: contracts.RunnerRequest{TaskID: "disconnect", Mode: contracts.RunnerModeImplement},
	})
	if err == nil {
		t.Fatalf("expected dispatch to fail after executor heartbeat expires")
	}
}
