package opencode

import (
	acp "github.com/ironpark/acp-go"
)

// Dummy implementation to make tests compile
type EventRouter struct {
	store *LogBubbleStore
}

func NewEventRouter(store *LogBubbleStore) *EventRouter {
	return &EventRouter{store: store}
}

func (er *EventRouter) RouteACPUpdate(update *acp.SessionUpdate) error {
	return nil
}
