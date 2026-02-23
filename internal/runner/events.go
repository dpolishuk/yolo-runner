package runner

import "time"

type EventType string

const (
	EventSelectTask    EventType = "select_task"
	EventBeadsUpdate   EventType = "beads_update"
	EventOpenCodeStart EventType = "opencode_start"
	EventOpenCodeEnd   EventType = "opencode_end"
	EventGitAdd        EventType = "git_add"
	EventGitStatus     EventType = "git_status"
	EventGitCommit     EventType = "git_commit"
	EventBeadsClose    EventType = "beads_close"
	EventBeadsVerify   EventType = "beads_verify"
	EventBeadsSync     EventType = "beads_sync"
)

type Event struct {
	Type              EventType `json:"type"`
	IssueID           string    `json:"issue_id"`
	Title             string    `json:"title"`
	Phase             string    `json:"phase"`
	ProgressCompleted int       `json:"progress_completed"`
	ProgressTotal     int       `json:"progress_total"`
	Model             string    `json:"model"`
	Message           string    `json:"message"`
	Thought           string    `json:"thought"` // Agent thoughts in markdown format
	EmittedAt         time.Time `json:"emitted_at"`
}

// RunnerEventType exposes the event type as a string for UI adapters.
func (e Event) RunnerEventType() string { return string(e.Type) }

// RunnerEventTitle exposes the event title for UI adapters.
func (e Event) RunnerEventTitle() string { return e.Title }

// RunnerEventThought exposes the event thought for UI adapters.
func (e Event) RunnerEventThought() string { return e.Thought }

// RunnerEventMessage exposes the event message for UI adapters.
func (e Event) RunnerEventMessage() string { return e.Message }

type EventEmitter interface {
	Emit(event Event)
}
