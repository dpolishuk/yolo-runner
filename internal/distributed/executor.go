package distributed

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/egv/yolo-runner/v2/internal/contracts"
)

type ExecutorWorkerOptions struct {
	ID                string
	Bus               Bus
	Runner            contracts.AgentRunner
	Subjects          EventSubjects
	Capabilities      []Capability
	HeartbeatInterval time.Duration
	RequestTimeout    time.Duration
	Clock             func() time.Time
}

type ExecutorWorker struct {
	id                string
	bus               Bus
	runner            contracts.AgentRunner
	subjects          EventSubjects
	capabilities      CapabilitySet
	heartbeatInterval time.Duration
	requestTimeout    time.Duration
	clock             func() time.Time
}

func NewExecutorWorker(cfg ExecutorWorkerOptions) *ExecutorWorker {
	subjects := cfg.Subjects
	if subjects.Register == "" {
		subjects = DefaultEventSubjects("yolo")
	}
	return &ExecutorWorker{
		id:                strings.TrimSpace(cfg.ID),
		bus:               cfg.Bus,
		runner:            cfg.Runner,
		subjects:          subjects,
		capabilities:      NewCapabilitySet(cfg.Capabilities...),
		heartbeatInterval: cfg.HeartbeatInterval,
		requestTimeout:    cfg.RequestTimeout,
		clock: func() time.Time {
			if cfg.Clock != nil {
				return cfg.Clock().UTC()
			}
			return time.Now().UTC()
		},
	}
}

func (w *ExecutorWorker) ID() string {
	if strings.TrimSpace(w.id) != "" {
		return strings.TrimSpace(w.id)
	}
	return "executor-" + w.clock().Format("20060102150405.000")
}

func (w *ExecutorWorker) Start(ctx context.Context) error {
	if w == nil || w.bus == nil {
		return fmt.Errorf("executor worker bus is required")
	}
	if w.runner == nil {
		return fmt.Errorf("executor worker runner is required")
	}
	interval := w.heartbeatInterval
	if interval <= 0 {
		interval = 5 * time.Second
	}
	if w.requestTimeout <= 0 {
		w.requestTimeout = 30 * time.Second
	}

	if err := w.publishRegistration(ctx); err != nil {
		return err
	}
	dispatchCh, unsubscribeDispatch, err := w.bus.Subscribe(ctx, w.subjects.TaskDispatch)
	if err != nil {
		return err
	}
	defer unsubscribeDispatch()

	heartbeatTicker := time.NewTicker(interval)
	defer heartbeatTicker.Stop()
	if err := w.publishHeartbeat(ctx); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-heartbeatTicker.C:
			if err := w.publishHeartbeat(ctx); err != nil {
				return err
			}
		case raw, ok := <-dispatchCh:
			if !ok {
				return nil
			}
			go w.handleDispatch(ctx, raw)
		}
	}
}

func (w *ExecutorWorker) publishRegistration(ctx context.Context) error {
	registration := ExecutorRegistrationPayload{
		ExecutorID:   w.ID(),
		Capabilities: keys(w.capabilities),
		StartedAt:    w.clock(),
	}
	event, err := NewEventEnvelope(EventTypeExecutorRegistered, w.ID(), "", registration)
	if err != nil {
		return err
	}
	return w.bus.Publish(ctx, w.subjects.Register, event)
}

func (w *ExecutorWorker) publishHeartbeat(ctx context.Context) error {
	heartbeat := ExecutorHeartbeatPayload{
		ExecutorID: w.ID(),
		SeenAt:     w.clock(),
	}
	event, err := NewEventEnvelope(EventTypeExecutorHeartbeat, w.ID(), "", heartbeat)
	if err != nil {
		return err
	}
	return w.bus.Publish(ctx, w.subjects.Heartbeat, event)
}

func (w *ExecutorWorker) handleDispatch(ctx context.Context, env EventEnvelope) {
	payload := TaskDispatchPayload{}
	if len(env.Payload) == 0 {
		return
	}
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return
	}
	if strings.TrimSpace(payload.TargetExecutorID) != "" && payload.TargetExecutorID != w.ID() {
		return
	}
	if !w.capabilities.HasAll(payload.RequiredCapabilities...) {
		return
	}

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
	transport := runnerTransportRequest{}
	if len(payload.Request) == 0 {
		return
	}
	if err := json.Unmarshal(payload.Request, &transport); err != nil {
		return
	}
	request := contracts.RunnerRequest{
		TaskID:   transport.TaskID,
		ParentID: transport.ParentID,
		Prompt:   transport.Prompt,
		Mode:     transport.Mode,
		Model:    transport.Model,
		RepoRoot: transport.RepoRoot,
		Timeout:  transport.Timeout,
		Metadata: transport.Metadata,
	}
	requestCtx := ctx
	if request.Timeout > 0 {
		deadline, cancel := context.WithTimeout(ctx, request.Timeout)
		defer cancel()
		requestCtx = deadline
	} else if w.requestTimeout > 0 {
		deadline, cancel := context.WithTimeout(ctx, w.requestTimeout)
		defer cancel()
		requestCtx = deadline
	}
	result, err := w.runner.Run(requestCtx, request)
	response := TaskResultPayload{
		CorrelationID: payload.CorrelationID,
		ExecutorID:    w.ID(),
		Result:        result,
	}
	if err != nil {
		response.Result = contracts.RunnerResult{
			Status: contracts.RunnerResultFailed,
			Reason: err.Error(),
		}
		response.Error = err.Error()
	}
	responseEnv, err := NewEventEnvelope(EventTypeTaskResult, w.ID(), payload.CorrelationID, response)
	if err != nil {
		return
	}
	_ = w.bus.Publish(requestCtx, w.subjects.TaskResult, responseEnv)
}

func (w *ExecutorWorker) RequestService(ctx context.Context, request ServiceRequestPayload) (ServiceResponsePayload, error) {
	if w == nil || w.bus == nil {
		return ServiceResponsePayload{}, fmt.Errorf("executor worker not ready")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(request.RequestID) == "" {
		request.RequestID = strings.TrimSpace(request.TaskID) + "-" + strings.ReplaceAll(w.clock().Format(time.RFC3339Nano), ":", "")
	}
	request.ExecutorID = w.ID()
	if strings.TrimSpace(request.CorrelationID) == "" {
		request.CorrelationID = request.RequestID
	}

	responseCh, unsubscribeResponse, err := w.bus.Subscribe(ctx, w.subjects.ServiceResult)
	if err != nil {
		return ServiceResponsePayload{}, err
	}
	defer unsubscribeResponse()

	event, err := NewEventEnvelope(EventTypeServiceRequest, w.ID(), request.CorrelationID, request)
	if err != nil {
		return ServiceResponsePayload{}, err
	}
	if err := w.bus.Publish(ctx, w.subjects.ServiceRequest, event); err != nil {
		return ServiceResponsePayload{}, err
	}

	for {
		select {
		case raw, ok := <-responseCh:
			if !ok {
				return ServiceResponsePayload{}, fmt.Errorf("service response channel closed")
			}
			if raw.CorrelationID != request.CorrelationID {
				continue
			}
			response := ServiceResponsePayload{}
			if len(raw.Payload) == 0 {
				continue
			}
			if err := json.Unmarshal(raw.Payload, &response); err != nil {
				continue
			}
			if response.RequestID != request.RequestID {
				continue
			}
			if response.Error != "" {
				return response, fmt.Errorf("%s", response.Error)
			}
			return response, nil
		case <-ctx.Done():
			return ServiceResponsePayload{}, ctx.Err()
		}
	}
}

func keys(values CapabilitySet) []Capability {
	out := make([]Capability, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	return out
}
