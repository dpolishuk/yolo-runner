package distributed

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/egv/yolo-runner/v2/internal/contracts"
)

type ServiceHandler func(ctx context.Context, request ServiceRequestPayload) (ServiceResponsePayload, error)

type MastermindOptions struct {
	ID             string
	Bus            Bus
	Subjects       EventSubjects
	RegistryTTL    time.Duration
	RequestTimeout time.Duration
	Clock          func() time.Time
	ServiceHandler ServiceHandler
}

type TaskDispatchRequest struct {
	RunnerRequest        contracts.RunnerRequest
	RequiredCapabilities []Capability
}

type Mastermind struct {
	id             string
	bus            Bus
	subjects       EventSubjects
	registry       *ExecutorRegistry
	requestTimeout time.Duration
	clock          func() time.Time
	serviceHandler ServiceHandler
}

func NewMastermind(cfg MastermindOptions) *Mastermind {
	subjects := cfg.Subjects
	if subjects.Register == "" {
		subjects = DefaultEventSubjects("yolo")
	}
	return &Mastermind{
		id:             strings.TrimSpace(cfg.ID),
		bus:            cfg.Bus,
		subjects:       subjects,
		registry:       NewExecutorRegistry(cfg.RegistryTTL, cfg.Clock),
		requestTimeout: cfg.RequestTimeout,
		clock: func() time.Time {
			if cfg.Clock != nil {
				return cfg.Clock().UTC()
			}
			return time.Now().UTC()
		},
		serviceHandler: cfg.ServiceHandler,
	}
}

func (m *Mastermind) Registry() *ExecutorRegistry {
	return m.registry
}

func (m *Mastermind) Start(ctx context.Context) error {
	if m == nil || m.bus == nil {
		return fmt.Errorf("mastermind bus is required")
	}
	registerCh, unregister, err := m.bus.Subscribe(ctx, m.subjects.Register)
	if err != nil {
		return err
	}
	heartbeatCh, unsubscribeHeartbeat, err := m.bus.Subscribe(ctx, m.subjects.Heartbeat)
	if err != nil {
		unregister()
		return err
	}
	serviceCh, unsubscribeService, err := m.bus.Subscribe(ctx, m.subjects.ServiceRequest)
	if err != nil {
		unregister()
		unsubscribeHeartbeat()
		return err
	}

	go func() {
		defer unregister()
		for {
			select {
			case raw, ok := <-registerCh:
				if !ok {
					return
				}
				registration := ExecutorRegistrationPayload{}
				if len(raw.Payload) == 0 {
					continue
				}
				if err := json.Unmarshal(raw.Payload, &registration); err != nil {
					continue
				}
				m.registry.Register(registration)
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		defer unsubscribeHeartbeat()
		for {
			select {
			case raw, ok := <-heartbeatCh:
				if !ok {
					return
				}
				heartbeat := ExecutorHeartbeatPayload{}
				if len(raw.Payload) == 0 {
					continue
				}
				if err := json.Unmarshal(raw.Payload, &heartbeat); err != nil {
					continue
				}
				m.registry.Heartbeat(heartbeat)
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		defer unsubscribeService()
		for {
			select {
			case raw, ok := <-serviceCh:
				if !ok {
					return
				}
				if err := m.handleServiceRequest(ctx, raw); err != nil {
					_ = err
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (m *Mastermind) DispatchTask(ctx context.Context, req TaskDispatchRequest) (contracts.RunnerResult, error) {
	if m == nil {
		return contracts.RunnerResult{}, fmt.Errorf("mastermind is nil")
	}
	if m.bus == nil {
		return contracts.RunnerResult{}, fmt.Errorf("mastermind bus is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if req.RunnerRequest.TaskID == "" {
		return contracts.RunnerResult{}, fmt.Errorf("task id is required")
	}
	capabilities := req.RequiredCapabilities
	if len(capabilities) == 0 {
		mode := req.RunnerRequest.Mode
		if mode == "" {
			mode = req.RunnerRequest.Mode
		}
		switch mode {
		case contracts.RunnerModeReview:
			capabilities = []Capability{CapabilityReview}
		default:
			capabilities = []Capability{CapabilityImplement}
		}
	}
	executor, err := m.registry.Pick(capabilities...)
	if err != nil {
		return contracts.RunnerResult{}, err
	}
	correlationID := req.RunnerRequest.TaskID + "-" + strings.ReplaceAll(time.Now().UTC().Format(time.RFC3339Nano), ":", "")
	dispatchTimeout := m.requestTimeout
	if dispatchTimeout <= 0 {
		dispatchTimeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, dispatchTimeout)
	defer cancel()

	resultCh, unsubResult, err := m.bus.Subscribe(ctx, m.subjects.TaskResult)
	if err != nil {
		return contracts.RunnerResult{}, err
	}
	defer unsubResult()

	dispatch := TaskDispatchPayload{
		CorrelationID:        correlationID,
		TaskID:               req.RunnerRequest.TaskID,
		TargetExecutorID:     executor.ID,
		RequiredCapabilities: capabilities,
		Request:              nil,
	}
	payloadRequest, err := requestForTransport(req.RunnerRequest)
	if err != nil {
		return contracts.RunnerResult{}, err
	}
	dispatch.Request = payloadRequest
	env, err := NewEventEnvelope(EventTypeTaskDispatch, m.id, correlationID, dispatch)
	if err != nil {
		return contracts.RunnerResult{}, err
	}
	env.CorrelationID = correlationID
	if err := m.bus.Publish(ctx, m.subjects.TaskDispatch, env); err != nil {
		return contracts.RunnerResult{}, err
	}

	timeoutTicker := time.NewTicker(50 * time.Millisecond)
	defer timeoutTicker.Stop()
	for {
		select {
		case raw := <-resultCh:
			if raw.CorrelationID != correlationID {
				continue
			}
			payload := TaskResultPayload{}
			if err := json.Unmarshal(raw.Payload, &payload); err != nil {
				continue
			}
			if strings.TrimSpace(payload.CorrelationID) != correlationID {
				continue
			}
			if strings.TrimSpace(payload.ExecutorID) == "" {
				payload.ExecutorID = strings.TrimSpace(payload.ExecutorID)
			}
			if payload.Error != "" {
				return contracts.RunnerResult{}, fmt.Errorf("executor failed: %s", payload.Error)
			}
			return payload.Result, nil
		case <-timeoutTicker.C:
			if !m.registry.IsAvailable(executor.ID, m.clock()) {
				return contracts.RunnerResult{}, fmt.Errorf("executor %s disconnected", executor.ID)
			}
		case <-ctx.Done():
			return contracts.RunnerResult{}, fmt.Errorf("task dispatch timed out: %w", ctx.Err())
		}
	}
}

func (m *Mastermind) handleServiceRequest(ctx context.Context, env EventEnvelope) error {
	if m == nil || m.serviceHandler == nil {
		return fmt.Errorf("service handler unavailable")
	}
	request := ServiceRequestPayload{}
	if len(env.Payload) == 0 {
		return fmt.Errorf("empty service request payload")
	}
	if err := json.Unmarshal(env.Payload, &request); err != nil {
		return err
	}
	response, err := m.serviceHandler(ctx, request)
	if err != nil {
		response.Error = err.Error()
	}
	response.RequestID = strings.TrimSpace(request.RequestID)
	response.CorrelationID = strings.TrimSpace(request.CorrelationID)
	response.ExecutorID = strings.TrimSpace(request.ExecutorID)
	response.Service = strings.TrimSpace(request.Service)
	responseEnv, err := NewEventEnvelope(EventTypeServiceResponse, m.id, response.CorrelationID, response)
	if err != nil {
		return err
	}
	responseEnv.CorrelationID = response.CorrelationID
	return m.bus.Publish(ctx, m.subjects.ServiceResult, responseEnv)
}

func requestForTransport(request contracts.RunnerRequest) (json.RawMessage, error) {
	type runnerTransportRequest struct {
		TaskID   string               `json:"task_id"`
		ParentID string               `json:"parent_id"`
		Prompt   string               `json:"prompt"`
		Mode     contracts.RunnerMode `json:"mode"`
		Model    string               `json:"model"`
		RepoRoot string               `json:"repo_root"`
		Timeout  time.Duration        `json:"timeout"`
		Metadata map[string]string    `json:"metadata,omitempty"`
	}
	transport := runnerTransportRequest{
		TaskID:   request.TaskID,
		ParentID: request.ParentID,
		Prompt:   request.Prompt,
		Mode:     request.Mode,
		Model:    request.Model,
		RepoRoot: request.RepoRoot,
		Timeout:  request.Timeout,
		Metadata: request.Metadata,
	}
	raw, err := json.Marshal(transport)
	if err != nil {
		return nil, err
	}
	return raw, nil
}
