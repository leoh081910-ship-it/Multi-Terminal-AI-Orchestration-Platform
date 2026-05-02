package server

import (
	"sync"
)

// EventBus provides a simple in-process pub/sub for internal events.
// Workers publish events, and the WebSocket hub subscribes to broadcast them.
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[string][]func(Event)
}

// Event represents an internal platform event.
type Event struct {
	Type      string      `json:"type"`
	ProjectID string      `json:"project_id,omitempty"`
	TaskID    string      `json:"task_id,omitempty"`
	AgentID   string      `json:"agent_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

// NewEventBus creates a new EventBus.
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]func(Event)),
	}
}

// Subscribe registers a callback for a specific event type.
// Use "*" to subscribe to all events.
func (b *EventBus) Subscribe(eventType string, handler func(Event)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscribers[eventType] = append(b.subscribers[eventType], handler)
}

// Publish sends an event to all matching subscribers.
func (b *EventBus) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Type-specific subscribers
	for _, handler := range b.subscribers[event.Type] {
		go handler(event)
	}
	// Wildcard subscribers
	for _, handler := range b.subscribers["*"] {
		go handler(event)
	}
}

// Common event types
const (
	EventTaskStateChanged = "task.state_changed"
	EventTaskCreated      = "task.created"
	EventTaskCompleted    = "task.completed"
	EventTaskFailed       = "task.failed"
	EventAgentHeartbeat   = "agent.heartbeat"
	EventAgentOffline     = "agent.offline"
	EventDecompositionComplete = "decomposition.complete"
	EventMessageSent      = "message.sent"
)
