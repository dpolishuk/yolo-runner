package distributed

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/egv/yolo-runner/v2/internal/contracts"
)

const (
	EventSchemaVersionV1 SchemaVersion = "1"
	EventSchemaVersionV0 SchemaVersion = "0"
)

type SchemaVersion string

type EventType string

const (
	EventTypeExecutorRegistered EventType = "executor_registered"
	EventTypeExecutorHeartbeat  EventType = "executor_heartbeat"
	EventTypeExecutorOffline    EventType = "executor_offline"
	EventTypeTaskDispatch       EventType = "task_dispatch"
	EventTypeTaskResult         EventType = "task_result"
	EventTypeServiceRequest     EventType = "service_request"
	EventTypeServiceResponse    EventType = "service_response"
)

type Capability string

const (
	CapabilityImplement    Capability = "implement"
	CapabilityReview       Capability = "review"
	CapabilityRewriteTask  Capability = "rewrite_task"
	CapabilityLargerModel  Capability = "larger_model"
	CapabilityServiceProxy Capability = "service_proxy"
)

type EventEnvelope struct {
	SchemaVersion SchemaVersion   `json:"schema_version"`
	Type          EventType       `json:"type"`
	CorrelationID string          `json:"correlation_id,omitempty"`
	Source        string          `json:"source"`
	Timestamp     time.Time       `json:"timestamp"`
	Payload       json.RawMessage `json:"payload,omitempty"`
}

type EventPayload struct {
	SchemaVersion string                 `json:"schema_version,omitempty"`
	Type          string                 `json:"type"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	Source        string                 `json:"source"`
	Timestamp     string                 `json:"timestamp,omitempty"`
	Payload       map[string]interface{} `json:"payload,omitempty"`
}

type ExecutorRegistrationPayload struct {
	ExecutorID   string            `json:"executor_id"`
	Capabilities []Capability      `json:"capabilities"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	StartedAt    time.Time         `json:"started_at"`
}

type ExecutorHeartbeatPayload struct {
	ExecutorID string            `json:"executor_id"`
	SeenAt     time.Time         `json:"seen_at"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type TaskDispatchPayload struct {
	CorrelationID        string          `json:"correlation_id"`
	TaskID               string          `json:"task_id"`
	TargetExecutorID     string          `json:"target_executor_id,omitempty"`
	RequiredCapabilities []Capability    `json:"required_capabilities"`
	Request              json.RawMessage `json:"request"`
}

type TaskResultPayload struct {
	CorrelationID string                 `json:"correlation_id"`
	ExecutorID    string                 `json:"executor_id"`
	Result        contracts.RunnerResult `json:"result"`
	Error         string                 `json:"error,omitempty"`
}

type ServiceRequestPayload struct {
	RequestID     string            `json:"request_id"`
	CorrelationID string            `json:"correlation_id"`
	ExecutorID    string            `json:"executor_id"`
	TaskID        string            `json:"task_id"`
	Service       string            `json:"service"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type ServiceResponsePayload struct {
	RequestID     string            `json:"request_id"`
	CorrelationID string            `json:"correlation_id"`
	ExecutorID    string            `json:"executor_id"`
	Service       string            `json:"service"`
	Artifacts     map[string]string `json:"artifacts,omitempty"`
	Error         string            `json:"error,omitempty"`
}

func NewEventEnvelope(typ EventType, source string, correlationID string, payload any) (EventEnvelope, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return EventEnvelope{}, fmt.Errorf("marshal payload: %w", err)
	}
	return EventEnvelope{
		SchemaVersion: EventSchemaVersionV1,
		Type:          typ,
		CorrelationID: correlationID,
		Source:        strings.TrimSpace(source),
		Timestamp:     time.Now().UTC(),
		Payload:       raw,
	}, nil
}

func ParseEventEnvelope(raw []byte) (EventEnvelope, error) {
	var evt EventEnvelope
	if err := json.Unmarshal(raw, &evt); err == nil && evt.Type != "" {
		if strings.TrimSpace(string(evt.SchemaVersion)) == "" {
			evt.SchemaVersion = EventSchemaVersionV0
		}
		return evt, nil
	}

	var legacy struct {
		Type         EventType       `json:"type"`
		Source       string          `json:"source"`
		Correlation  string          `json:"correlation_id"`
		Schema       SchemaVersion   `json:"schema_version"`
		Timestamp    time.Time       `json:"timestamp"`
		TS           string          `json:"ts"`
		Payload      json.RawMessage `json:"payload"`
		EventPayload map[string]any  `json:"event"`
		Data         json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &legacy); err != nil {
		return EventEnvelope{}, err
	}
	if legacy.Type == "" {
		return EventEnvelope{}, fmt.Errorf("missing event type")
	}
	payload := legacy.Payload
	if len(payload) == 0 {
		payload = legacy.Data
	}
	parsed := EventEnvelope{
		SchemaVersion: legacy.Schema,
		Type:          legacy.Type,
		CorrelationID: legacy.Correlation,
		Source:        legacy.Source,
		Timestamp:     legacy.Timestamp,
		Payload:       payload,
	}
	if parsed.SchemaVersion == "" {
		parsed.SchemaVersion = EventSchemaVersionV0
	}
	if parsed.Timestamp.IsZero() && strings.TrimSpace(legacy.TS) != "" {
		if parsedTS, err := time.Parse(time.RFC3339, legacy.TS); err == nil {
			parsed.Timestamp = parsedTS
		}
	}
	return parsed, nil
}
