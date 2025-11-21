package events

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewEventBus tests event bus creation
func TestNewEventBus(t *testing.T) {
	bus := NewEventBus()
	require.NotNil(t, bus)
	assert.NotNil(t, bus.handlers)
	assert.NotNil(t, bus.allEvents)
}

// TestEventBus_Subscribe tests subscribing to specific event type
func TestEventBus_Subscribe(t *testing.T) {
	bus := NewEventBus()
	received := false

	handler := func(event Event) {
		received = true
	}

	bus.Subscribe(EventTranslationStarted, handler)

	event := NewEvent(EventTranslationStarted, "Test started", nil)
	bus.Publish(event)

	// Give handler time to execute
	time.Sleep(10 * time.Millisecond)

	assert.True(t, received, "Handler should have received event")
}

// TestEventBus_SubscribeAll tests subscribing to all events
func TestEventBus_SubscribeAll(t *testing.T) {
	bus := NewEventBus()
	var receivedEvents []EventType
	var mu sync.Mutex

	handler := func(event Event) {
		mu.Lock()
		defer mu.Unlock()
		receivedEvents = append(receivedEvents, event.Type)
	}

	bus.SubscribeAll(handler)

	// Publish different event types
	events := []EventType{
		EventTranslationStarted,
		EventTranslationProgress,
		EventTranslationCompleted,
	}

	for _, eventType := range events {
		bus.Publish(NewEvent(eventType, "Test", nil))
	}

	// Give handlers time to execute
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Len(t, receivedEvents, 3, "Should receive all 3 events")
}

// TestEventBus_MultipleHandlers tests multiple handlers for same event
func TestEventBus_MultipleHandlers(t *testing.T) {
	bus := NewEventBus()
	var count1, count2 int
	var mu sync.Mutex

	handler1 := func(event Event) {
		mu.Lock()
		defer mu.Unlock()
		count1++
	}

	handler2 := func(event Event) {
		mu.Lock()
		defer mu.Unlock()
		count2++
	}

	bus.Subscribe(EventTranslationStarted, handler1)
	bus.Subscribe(EventTranslationStarted, handler2)

	bus.Publish(NewEvent(EventTranslationStarted, "Test", nil))

	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, count1, "Handler 1 should be called once")
	assert.Equal(t, 1, count2, "Handler 2 should be called once")
}

// TestEventBus_DifferentEventTypes tests routing to correct handlers
func TestEventBus_DifferentEventTypes(t *testing.T) {
	bus := NewEventBus()
	var startedCount, completedCount int
	var mu sync.Mutex

	startedHandler := func(event Event) {
		mu.Lock()
		defer mu.Unlock()
		startedCount++
	}

	completedHandler := func(event Event) {
		mu.Lock()
		defer mu.Unlock()
		completedCount++
	}

	bus.Subscribe(EventTranslationStarted, startedHandler)
	bus.Subscribe(EventTranslationCompleted, completedHandler)

	// Publish started event
	bus.Publish(NewEvent(EventTranslationStarted, "Started", nil))
	time.Sleep(10 * time.Millisecond)

	// Publish completed event
	bus.Publish(NewEvent(EventTranslationCompleted, "Completed", nil))
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, startedCount, "Started handler called once")
	assert.Equal(t, 1, completedCount, "Completed handler called once")
}

// TestNewEvent tests event creation
func TestNewEvent(t *testing.T) {
	data := map[string]interface{}{
		"test_key": "test_value",
		"count":    42,
	}

	event := NewEvent(EventTranslationProgress, "Test message", data)

	assert.NotEmpty(t, event.ID, "Event should have ID")
	assert.Equal(t, EventTranslationProgress, event.Type)
	assert.Equal(t, "Test message", event.Message)
	assert.Equal(t, data, event.Data)
	assert.False(t, event.Timestamp.IsZero(), "Event should have timestamp")
}

// TestEvent_WithSessionID tests event with session ID
func TestEvent_WithSessionID(t *testing.T) {
	event := NewEvent(EventTranslationStarted, "Test", nil)
	event.SessionID = "session-123"

	assert.Equal(t, "session-123", event.SessionID)
}

// TestEventTypes_Constants tests all event type constants
func TestEventTypes_Constants(t *testing.T) {
	eventTypes := []EventType{
		EventTranslationStarted,
		EventTranslationProgress,
		EventTranslationCompleted,
		EventTranslationError,
		EventConversionStarted,
		EventConversionProgress,
		EventConversionCompleted,
		EventConversionError,
	}

	// Verify all are unique
	seen := make(map[EventType]bool)
	for _, et := range eventTypes {
		assert.False(t, seen[et], "Event type should be unique: %s", et)
		assert.NotEmpty(t, string(et), "Event type should not be empty")
		seen[et] = true
	}

	assert.Len(t, seen, 8, "Should have 8 unique event types")
}

// TestEventBus_ConcurrentPublish tests concurrent event publishing
func TestEventBus_ConcurrentPublish(t *testing.T) {
	bus := NewEventBus()
	var count int
	var mu sync.Mutex

	handler := func(event Event) {
		mu.Lock()
		defer mu.Unlock()
		count++
	}

	bus.SubscribeAll(handler)

	// Publish 100 events concurrently
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			bus.Publish(NewEvent(EventTranslationProgress, "Test", map[string]interface{}{
				"index": i,
			}))
		}(i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 100, count, "Should receive all 100 events")
}

// TestEventBus_ThreadSafety tests thread-safe subscribe and publish
func TestEventBus_ThreadSafety(t *testing.T) {
	bus := NewEventBus()
	var wg sync.WaitGroup

	// Concurrent subscribes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Subscribe(EventTranslationProgress, func(event Event) {})
		}()
	}

	// Concurrent publishes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Publish(NewEvent(EventTranslationProgress, "Test", nil))
		}()
	}

	wg.Wait()
	// Test passes if no data races occur
}

// TestEventBus_HandlerPanic tests that panicking handler doesn't crash
func TestEventBus_HandlerPanic(t *testing.T) {
	bus := NewEventBus()
	var normalHandlerCalled bool
	var mu sync.Mutex

	panicHandler := func(event Event) {
		panic("test panic")
	}

	normalHandler := func(event Event) {
		mu.Lock()
		defer mu.Unlock()
		normalHandlerCalled = true
	}

	bus.Subscribe(EventTranslationStarted, panicHandler)
	bus.Subscribe(EventTranslationStarted, normalHandler)

	// Should not crash even if handler panics
	bus.Publish(NewEvent(EventTranslationStarted, "Test", nil))
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.True(t, normalHandlerCalled, "Normal handler should still be called")
}

// TestEvent_DataTypes tests different data types in event data
func TestEvent_DataTypes(t *testing.T) {
	data := map[string]interface{}{
		"string":  "test",
		"int":     42,
		"float":   3.14,
		"bool":    true,
		"nil":     nil,
		"slice":   []string{"a", "b", "c"},
		"map":     map[string]int{"x": 1, "y": 2},
	}

	event := NewEvent(EventTranslationProgress, "Test", data)

	assert.Equal(t, "test", event.Data["string"])
	assert.Equal(t, 42, event.Data["int"])
	assert.Equal(t, 3.14, event.Data["float"])
	assert.Equal(t, true, event.Data["bool"])
	assert.Nil(t, event.Data["nil"])
	assert.Len(t, event.Data["slice"], 3)
	assert.Len(t, event.Data["map"], 2)
}

// TestEventID_Uniqueness tests that event IDs are unique
func TestEventID_Uniqueness(t *testing.T) {
	ids := make(map[string]bool)

	for i := 0; i < 1000; i++ {
		event := NewEvent(EventTranslationProgress, "Test", nil)
		assert.False(t, ids[event.ID], "Event ID should be unique")
		ids[event.ID] = true
		time.Sleep(time.Microsecond)
	}

	assert.Len(t, ids, 1000, "Should have 1000 unique IDs")
}

// BenchmarkEventBus_Publish benchmarks event publishing
func BenchmarkEventBus_Publish(b *testing.B) {
	bus := NewEventBus()
	event := NewEvent(EventTranslationProgress, "Benchmark", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Publish(event)
	}
}

// BenchmarkEventBus_PublishWithHandlers benchmarks with handlers
func BenchmarkEventBus_PublishWithHandlers(b *testing.B) {
	bus := NewEventBus()

	// Add 10 handlers
	for i := 0; i < 10; i++ {
		bus.Subscribe(EventTranslationProgress, func(event Event) {})
	}

	event := NewEvent(EventTranslationProgress, "Benchmark", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Publish(event)
	}
}

// BenchmarkNewEvent benchmarks event creation
func BenchmarkNewEvent(b *testing.B) {
	data := map[string]interface{}{
		"key": "value",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewEvent(EventTranslationProgress, "Benchmark", data)
	}
}
