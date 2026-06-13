package events

import (
	"sync"
	"time"
)

// EventType represents different types of events
type EventType string

const (
	EventTranslationStarted   EventType = "translation_started"
	EventTranslationProgress  EventType = "translation_progress"
	EventTranslationCompleted EventType = "translation_completed"
	EventTranslationError     EventType = "translation_error"
	EventConversionStarted    EventType = "conversion_started"
	EventConversionProgress   EventType = "conversion_progress"
	EventConversionCompleted  EventType = "conversion_completed"
	EventConversionError      EventType = "conversion_error"

	// LLMsVerifier model lifecycle events
	EventModelDiscovered      EventType = "model_discovered"
	EventModelUpdated         EventType = "model_updated"
	EventModelRemoved         EventType = "model_removed"
	EventVerificationCompleted EventType = "verification_completed"
	EventVerificationFailed   EventType = "verification_failed"
)

// Event represents a system event
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
}

// EventHandler is a function that processes events
type EventHandler func(event Event)

// EventBus manages event distribution
type EventBus struct {
	mu        sync.RWMutex
	handlers  map[EventType][]EventHandler
	allEvents []EventHandler
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		handlers:  make(map[EventType][]EventHandler),
		allEvents: make([]EventHandler, 0),
	}
}

// Subscribe adds a handler for a specific event type
func (eb *EventBus) Subscribe(eventType EventType, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

// SubscribeAll adds a handler for all events
func (eb *EventBus) SubscribeAll(handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.allEvents = append(eb.allEvents, handler)
}

// Publish sends an event to all subscribed handlers.
//
// Handlers are invoked synchronously, in subscription order (type-specific
// handlers first, then all-event handlers). Each handler is isolated with a
// recover() so a panicking handler cannot crash the publisher or prevent later
// handlers from running.
//
// Concurrency: the handler slices are snapshotted under the read lock and the
// lock is released BEFORE any handler runs. This is deliberate and load-bearing:
//
//   - It keeps handler execution out of the lock-held region, so a handler may
//     not deadlock or stall a concurrent Subscribe/SubscribeAll (which need the
//     write lock). Holding RLock across handler invocation previously starved
//     writers under sustained Publish load.
//   - It removes the previous unbounded "one goroutine per handler per event"
//     fan-out, which leaked goroutines without backpressure under load.
//
// The snapshot also makes the set of handlers for a given Publish stable: a
// handler added concurrently is either fully included or fully excluded, never
// observed mid-append.
func (eb *EventBus) Publish(event Event) {
	eb.mu.RLock()
	// Copy the handler slices so we can release the lock before invoking them.
	// append(nil, src...) returns nil for an empty/absent slice, which the
	// range loops below handle correctly (zero iterations).
	typeHandlers := append([]EventHandler(nil), eb.handlers[event.Type]...)
	allHandlers := append([]EventHandler(nil), eb.allEvents...)
	eb.mu.RUnlock()

	// Send to specific handlers, then all-event handlers — synchronously and in
	// order, each isolated from panics.
	for _, handler := range typeHandlers {
		invokeHandler(handler, event)
	}
	for _, handler := range allHandlers {
		invokeHandler(handler, event)
	}
}

// invokeHandler runs a single handler with panic isolation so one misbehaving
// handler cannot crash the publisher or skip the handlers that follow it.
func invokeHandler(h EventHandler, event Event) {
	defer func() { _ = recover() }()
	h(event)
}

// NewEvent creates a new event with timestamp and unique ID
func NewEvent(eventType EventType, message string, data map[string]interface{}) Event {
	return Event{
		ID:        generateEventID(),
		Type:      eventType,
		Timestamp: time.Now(),
		Message:   message,
		Data:      data,
	}
}

// generateEventID creates a unique event ID
func generateEventID() string {
	return time.Now().Format("20060102150405.000000")
}
