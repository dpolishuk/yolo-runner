package scheduler

import "testing"

func TestLandingQueueStateMachineHappyPath(t *testing.T) {
	sm := NewLandingQueueStateMachine(3)

	if err := sm.Apply(LandingEventBegin); err != nil {
		t.Fatalf("begin landing failed: %v", err)
	}
	if sm.State() != LandingStateLanding {
		t.Fatalf("expected landing state, got %s", sm.State())
	}

	if err := sm.Apply(LandingEventSucceeded); err != nil {
		t.Fatalf("landing success failed: %v", err)
	}
	if sm.State() != LandingStateLanded {
		t.Fatalf("expected landed state, got %s", sm.State())
	}
}

func TestLandingQueueStateMachineRetryPath(t *testing.T) {
	sm := NewLandingQueueStateMachine(3)

	if err := sm.Apply(LandingEventBegin); err != nil {
		t.Fatalf("begin landing failed: %v", err)
	}
	if err := sm.Apply(LandingEventFailedRetryable); err != nil {
		t.Fatalf("mark retryable failure failed: %v", err)
	}
	if sm.State() != LandingStateRetrying {
		t.Fatalf("expected retrying state, got %s", sm.State())
	}
	if sm.Attempts() != 1 {
		t.Fatalf("expected attempts=1, got %d", sm.Attempts())
	}

	if err := sm.Apply(LandingEventRequeued); err != nil {
		t.Fatalf("requeue failed: %v", err)
	}
	if sm.State() != LandingStateQueued {
		t.Fatalf("expected queued state, got %s", sm.State())
	}

	if err := sm.Apply(LandingEventBegin); err != nil {
		t.Fatalf("begin landing second attempt failed: %v", err)
	}
	if err := sm.Apply(LandingEventSucceeded); err != nil {
		t.Fatalf("landing success failed: %v", err)
	}
	if sm.State() != LandingStateLanded {
		t.Fatalf("expected landed state, got %s", sm.State())
	}
}

func TestLandingQueueStateMachineBlocksAfterMaxAttempts(t *testing.T) {
	sm := NewLandingQueueStateMachine(2)

	if err := sm.Apply(LandingEventBegin); err != nil {
		t.Fatalf("begin landing failed: %v", err)
	}
	if err := sm.Apply(LandingEventFailedRetryable); err != nil {
		t.Fatalf("mark retryable failure failed: %v", err)
	}
	if err := sm.Apply(LandingEventRequeued); err != nil {
		t.Fatalf("requeue failed: %v", err)
	}

	if err := sm.Apply(LandingEventBegin); err != nil {
		t.Fatalf("begin landing second attempt failed: %v", err)
	}
	if err := sm.Apply(LandingEventFailedRetryable); err != nil {
		t.Fatalf("mark retryable failure failed: %v", err)
	}

	if sm.State() != LandingStateBlocked {
		t.Fatalf("expected blocked state, got %s", sm.State())
	}
	if sm.Attempts() != 2 {
		t.Fatalf("expected attempts=2, got %d", sm.Attempts())
	}
}

func TestLandingQueueStateMachineRejectsInvalidTransitions(t *testing.T) {
	sm := NewLandingQueueStateMachine(2)

	if err := sm.Apply(LandingEventSucceeded); err == nil {
		t.Fatalf("expected invalid transition error")
	}
	if sm.State() != LandingStateQueued {
		t.Fatalf("expected state to remain queued, got %s", sm.State())
	}
}

func TestLandingQueueStateMachineTerminalStatesRejectFurtherEvents(t *testing.T) {
	tests := []LandingState{LandingStateBlocked, LandingStateLanded}
	for _, terminal := range tests {
		t.Run(string(terminal), func(t *testing.T) {
			sm := NewLandingQueueStateMachine(2)
			sm.state = terminal
			if err := sm.Apply(LandingEventBegin); err == nil {
				t.Fatalf("expected terminal transition to fail")
			}
			if sm.State() != terminal {
				t.Fatalf("expected state to remain %s, got %s", terminal, sm.State())
			}
		})
	}
}
