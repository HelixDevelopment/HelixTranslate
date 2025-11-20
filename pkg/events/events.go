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

// Publish sends an event to all subscribed handlers
func (eb *EventBus) Publish(event Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	// Send to specific handlers
	if handlers, ok := eb.handlers[event.Type]; ok {
		for _, handler := range handlers {
			go handler(event)
		}
	}

	// Send to all-event handlers
	for _, handler := range eb.allEvents {
		go handler(event)
	}
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
