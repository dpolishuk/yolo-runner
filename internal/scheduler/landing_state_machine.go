package scheduler

import "fmt"

type LandingState string

const (
	LandingStateQueued   LandingState = "queued"
	LandingStateLanding  LandingState = "landing"
	LandingStateRetrying LandingState = "retrying"
	LandingStateBlocked  LandingState = "blocked"
	LandingStateLanded   LandingState = "landed"
)

func (s LandingState) IsTerminal() bool {
	return s == LandingStateBlocked || s == LandingStateLanded
}

type LandingEvent string

const (
	LandingEventBegin           LandingEvent = "begin"
	LandingEventSucceeded       LandingEvent = "succeeded"
	LandingEventFailedRetryable LandingEvent = "failed_retryable"
	LandingEventFailedPermanent LandingEvent = "failed_permanent"
	LandingEventRequeued        LandingEvent = "requeued"
)

type LandingQueueStateMachine struct {
	state       LandingState
	attempts    int
	maxAttempts int
}

func NewLandingQueueStateMachine(maxAttempts int) *LandingQueueStateMachine {
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	return &LandingQueueStateMachine{state: LandingStateQueued, maxAttempts: maxAttempts}
}

func (m *LandingQueueStateMachine) State() LandingState {
	if m == nil {
		return LandingStateQueued
	}
	return m.state
}

func (m *LandingQueueStateMachine) Attempts() int {
	if m == nil {
		return 0
	}
	return m.attempts
}

func (m *LandingQueueStateMachine) Apply(event LandingEvent) error {
	if m == nil {
		return fmt.Errorf("state machine is nil")
	}
	if m.state.IsTerminal() {
		return fmt.Errorf("cannot apply %q in terminal state %q", event, m.state)
	}

	switch m.state {
	case LandingStateQueued:
		if event != LandingEventBegin {
			return m.invalidTransition(event)
		}
		m.state = LandingStateLanding
		return nil

	case LandingStateLanding:
		switch event {
		case LandingEventSucceeded:
			m.state = LandingStateLanded
			return nil
		case LandingEventFailedPermanent:
			m.attempts++
			m.state = LandingStateBlocked
			return nil
		case LandingEventFailedRetryable:
			m.attempts++
			if m.attempts >= m.maxAttempts {
				m.state = LandingStateBlocked
				return nil
			}
			m.state = LandingStateRetrying
			return nil
		default:
			return m.invalidTransition(event)
		}

	case LandingStateRetrying:
		switch event {
		case LandingEventRequeued:
			m.state = LandingStateQueued
			return nil
		case LandingEventBegin:
			m.state = LandingStateLanding
			return nil
		case LandingEventFailedPermanent:
			m.state = LandingStateBlocked
			return nil
		default:
			return m.invalidTransition(event)
		}
	}

	return m.invalidTransition(event)
}

func (m *LandingQueueStateMachine) invalidTransition(event LandingEvent) error {
	return fmt.Errorf("invalid landing transition: state=%q event=%q", m.state, event)
}
