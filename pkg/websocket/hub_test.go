package websocket

import (
	"digital.vasic.translator/pkg/events"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewHub tests hub creation
func TestNewHub(t *testing.T) {
	eventBus := events.NewEventBus()
	hub := NewHub(eventBus)

	require.NotNil(t, hub)
	assert.NotNil(t, hub.clients)
	assert.NotNil(t, hub.register)
	assert.NotNil(t, hub.unregister)
	assert.NotNil(t, hub.eventBus)
	assert.Equal(t, 0, hub.GetClientCount())
}

// TestHub_RegisterClient tests client registration
func TestHub_RegisterClient(t *testing.T) {
	eventBus := events.NewEventBus()
	hub := NewHub(eventBus)

	// Start hub in goroutine
	go hub.Run()

	client := &Client{
		ID:        "client-1",
		SessionID: "session-1",
		Send:      make(chan []byte, 256),
		Hub:       hub,
	}

	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, 1, hub.GetClientCount())
}

// TestHub_UnregisterClient tests client unregistration
func TestHub_UnregisterClient(t *testing.T) {
	eventBus := events.NewEventBus()
	hub := NewHub(eventBus)

	go hub.Run()

	client := &Client{
		ID:        "client-1",
		SessionID: "session-1",
		Send:      make(chan []byte, 256),
		Hub:       hub,
	}

	hub.Register(client)
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 1, hub.GetClientCount())

	hub.Unregister(client)
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 0, hub.GetClientCount())

	// Verify channel is closed
	_, ok := <-client.Send
	assert.False(t, ok, "Send channel should be closed")
}

// TestHub_Broadcast tests broadcasting to all clients
func TestHub_Broadcast(t *testing.T) {
	eventBus := events.NewEventBus()
	hub := NewHub(eventBus)

	go hub.Run()

	// Create multiple clients
	clients := make([]*Client, 3)
	for i := 0; i < 3; i++ {
		clients[i] = &Client{
			ID:        "client-" + string(rune('1'+i)),
			SessionID: "session-1",
			Send:      make(chan []byte, 256),
			Hub:       hub,
		}
		hub.Register(clients[i])
	}
	time.Sleep(10 * time.Millisecond)

	// Broadcast message
	message := []byte("test broadcast")
	hub.Broadcast(message)

	// Verify all clients received message
	for i, client := range clients {
		select {
		case msg := <-client.Send:
			assert.Equal(t, message, msg, "Client %d should receive broadcast", i)
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Client %d did not receive broadcast", i)
		}
	}
}

// TestHub_HandleEvent tests event handling from event bus
func TestHub_HandleEvent(t *testing.T) {
	eventBus := events.NewEventBus()
	hub := NewHub(eventBus)

	go hub.Run()

	client := &Client{
		ID:        "client-1",
		SessionID: "session-1",
		Send:      make(chan []byte, 256),
		Hub:       hub,
	}
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	// Publish event
	event := events.NewEvent(events.EventTranslationStarted, "Test event", map[string]interface{}{
		"test": "data",
	})
	event.SessionID = "session-1"
	eventBus.Publish(event)

	// Wait and verify client received event
	select {
	case msg := <-client.Send:
		var receivedEvent events.Event
		err := json.Unmarshal(msg, &receivedEvent)
		require.NoError(t, err)
		assert.Equal(t, events.EventTranslationStarted, receivedEvent.Type)
		assert.Equal(t, "Test event", receivedEvent.Message)
	case <-time.After(100 * time.Millisecond):
		t.Error("Client did not receive event")
	}
}

// TestHub_SessionFiltering tests filtering events by session ID
func TestHub_SessionFiltering(t *testing.T) {
	eventBus := events.NewEventBus()
	hub := NewHub(eventBus)

	go hub.Run()

	// Create clients with different sessions
	client1 := &Client{
		ID:        "client-1",
		SessionID: "session-1",
		Send:      make(chan []byte, 256),
		Hub:       hub,
	}
	client2 := &Client{
		ID:        "client-2",
		SessionID: "session-2",
		Send:      make(chan []byte, 256),
		Hub:       hub,
	}

	hub.Register(client1)
	hub.Register(client2)
	time.Sleep(10 * time.Millisecond)

	// Publish event for session-1
	event := events.NewEvent(events.EventTranslationProgress, "Session 1 event", nil)
	event.SessionID = "session-1"
	eventBus.Publish(event)

	time.Sleep(50 * time.Millisecond)

	// Verify only client1 received the event
	select {
	case msg := <-client1.Send:
		var receivedEvent events.Event
		err := json.Unmarshal(msg, &receivedEvent)
		require.NoError(t, err)
		assert.Equal(t, "session-1", receivedEvent.SessionID)
	case <-time.After(100 * time.Millisecond):
		t.Error("Client 1 did not receive event")
	}

	// Verify client2 did NOT receive the event
	select {
	case <-client2.Send:
		t.Error("Client 2 should not have received session-1 event")
	case <-time.After(50 * time.Millisecond):
		// Expected - no message
	}
}

// TestHub_BroadcastEvents tests broadcasting events to all sessions
func TestHub_BroadcastEvents(t *testing.T) {
	eventBus := events.NewEventBus()
	hub := NewHub(eventBus)

	go hub.Run()

	// Create clients with different sessions
	clients := []*Client{
		{ID: "client-1", SessionID: "session-1", Send: make(chan []byte, 256), Hub: hub},
		{ID: "client-2", SessionID: "session-2", Send: make(chan []byte, 256), Hub: hub},
	}

	for _, client := range clients {
		hub.Register(client)
	}
	time.Sleep(10 * time.Millisecond)

	// Publish event without session ID (broadcast to all)
	event := events.NewEvent(events.EventTranslationCompleted, "Broadcast event", nil)
	eventBus.Publish(event)

	time.Sleep(50 * time.Millisecond)

	// Verify all clients received the event
	for i, client := range clients {
		select {
		case msg := <-client.Send:
			var receivedEvent events.Event
			err := json.Unmarshal(msg, &receivedEvent)
			require.NoError(t, err)
			assert.Equal(t, events.EventTranslationCompleted, receivedEvent.Type)
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Client %d did not receive broadcast event", i)
		}
	}
}

// TestHub_ConcurrentRegistration tests concurrent client registration
func TestHub_ConcurrentRegistration(t *testing.T) {
	eventBus := events.NewEventBus()
	hub := NewHub(eventBus)

	go hub.Run()

	var wg sync.WaitGroup
	clientCount := 50

	// Register clients concurrently
	for i := 0; i < clientCount; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			client := &Client{
				ID:        "client-" + string(rune('0'+i%10)),
				SessionID: "session-1",
				Send:      make(chan []byte, 256),
				Hub:       hub,
			}
			hub.Register(client)
		}(i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, clientCount, hub.GetClientCount())
}

// TestHub_ConcurrentUnregistration tests concurrent unregistration
func TestHub_ConcurrentUnregistration(t *testing.T) {
	eventBus := events.NewEventBus()
	hub := NewHub(eventBus)

	go hub.Run()

	// Register clients
	clients := make([]*Client, 20)
	for i := 0; i < 20; i++ {
		clients[i] = &Client{
			ID:        "client-" + string(rune('0'+i%10)),
			SessionID: "session-1",
			Send:      make(chan []byte, 256),
			Hub:       hub,
		}
		hub.Register(clients[i])
	}
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 20, hub.GetClientCount())

	// Unregister concurrently
	var wg sync.WaitGroup
	for _, client := range clients {
		wg.Add(1)
		go func(c *Client) {
			defer wg.Done()
			hub.Unregister(c)
		}(client)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 0, hub.GetClientCount())
}

// TestHub_FullChannelHandling tests handling when client channel is full
func TestHub_FullChannelHandling(t *testing.T) {
	eventBus := events.NewEventBus()
	hub := NewHub(eventBus)

	go hub.Run()

	// Create client with small buffer
	client := &Client{
		ID:        "client-1",
		SessionID: "session-1",
		Send:      make(chan []byte, 1), // Small buffer
		Hub:       hub,
	}
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	// Fill the channel
	hub.Broadcast([]byte("message1"))
	hub.Broadcast([]byte("message2"))

	time.Sleep(10 * time.Millisecond)

	// Broadcast more messages (should not block)
	done := make(chan bool)
	go func() {
		hub.Broadcast([]byte("message3"))
		hub.Broadcast([]byte("message4"))
		done <- true
	}()

	select {
	case <-done:
		// Success - did not block
	case <-time.After(100 * time.Millisecond):
		t.Error("Broadcast blocked on full channel")
	}
}

// TestHub_JSONMarshaling tests event JSON marshaling
func TestHub_JSONMarshaling(t *testing.T) {
	eventBus := events.NewEventBus()
	hub := NewHub(eventBus)

	go hub.Run()

	client := &Client{
		ID:        "client-1",
		SessionID: "session-1",
		Send:      make(chan []byte, 256),
		Hub:       hub,
	}
	hub.Register(client)
	time.Sleep(10 * time.Millisecond)

	// Publish event with complex data
	event := events.NewEvent(events.EventTranslationProgress, "Complex data", map[string]interface{}{
		"string":  "value",
		"number":  42,
		"float":   3.14,
		"boolean": true,
		"array":   []string{"a", "b", "c"},
	})
	event.SessionID = "session-1"
	eventBus.Publish(event)

	select {
	case msg := <-client.Send:
		var receivedEvent events.Event
		err := json.Unmarshal(msg, &receivedEvent)
		require.NoError(t, err)
		assert.Equal(t, "Complex data", receivedEvent.Message)
		assert.NotNil(t, receivedEvent.Data)
	case <-time.After(100 * time.Millisecond):
		t.Error("Client did not receive event")
	}
}

// TestHub_GetClientCount tests client count tracking
func TestHub_GetClientCount(t *testing.T) {
	eventBus := events.NewEventBus()
	hub := NewHub(eventBus)

	go hub.Run()

	assert.Equal(t, 0, hub.GetClientCount())

	// Add clients
	clients := make([]*Client, 5)
	for i := 0; i < 5; i++ {
		clients[i] = &Client{
			ID:        "client-" + string(rune('1'+i)),
			SessionID: "session-1",
			Send:      make(chan []byte, 256),
			Hub:       hub,
		}
		hub.Register(clients[i])
		time.Sleep(5 * time.Millisecond)
		assert.Equal(t, i+1, hub.GetClientCount())
	}

	// Remove clients
	for i := 0; i < 5; i++ {
		hub.Unregister(clients[i])
		time.Sleep(5 * time.Millisecond)
		assert.Equal(t, 4-i, hub.GetClientCount())
	}
}

// TestHub_ThreadSafety tests concurrent operations
func TestHub_ThreadSafety(t *testing.T) {
	eventBus := events.NewEventBus()
	hub := NewHub(eventBus)

	go hub.Run()

	var wg sync.WaitGroup

	// Concurrent registrations, broadcasts, and unregistrations
	for i := 0; i < 20; i++ {
		wg.Add(3)

		// Register
		go func(i int) {
			defer wg.Done()
			client := &Client{
				ID:        "client-" + string(rune('0'+i%10)),
				SessionID: "session-1",
				Send:      make(chan []byte, 256),
				Hub:       hub,
			}
			hub.Register(client)
		}(i)

		// Broadcast
		go func() {
			defer wg.Done()
			hub.Broadcast([]byte("test message"))
		}()

		// Get count
		go func() {
			defer wg.Done()
			_ = hub.GetClientCount()
		}()
	}

	wg.Wait()
	// Test passes if no data races occur
}

// BenchmarkHub_Broadcast benchmarks broadcasting
func BenchmarkHub_Broadcast(b *testing.B) {
	eventBus := events.NewEventBus()
	hub := NewHub(eventBus)

	go hub.Run()

	// Add some clients
	for i := 0; i < 10; i++ {
		client := &Client{
			ID:        "client-" + string(rune('0'+i)),
			SessionID: "session-1",
			Send:      make(chan []byte, 256),
			Hub:       hub,
		}
		hub.Register(client)
	}
	time.Sleep(50 * time.Millisecond)

	message := []byte("benchmark message")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hub.Broadcast(message)
	}
}

// BenchmarkHub_Register benchmarks client registration
func BenchmarkHub_Register(b *testing.B) {
	eventBus := events.NewEventBus()
	hub := NewHub(eventBus)

	go hub.Run()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client := &Client{
			ID:        "client-bench",
			SessionID: "session-1",
			Send:      make(chan []byte, 256),
			Hub:       hub,
		}
		hub.Register(client)
	}
}

// BenchmarkHub_HandleEvent benchmarks event handling
func BenchmarkHub_HandleEvent(b *testing.B) {
	eventBus := events.NewEventBus()
	hub := NewHub(eventBus)

	go hub.Run()

	// Add client
	client := &Client{
		ID:        "client-1",
		SessionID: "session-1",
		Send:      make(chan []byte, 256),
		Hub:       hub,
	}
	hub.Register(client)
	time.Sleep(50 * time.Millisecond)

	// Drain client messages in background
	go func() {
		for range client.Send {
			// Drain
		}
	}()

	event := events.NewEvent(events.EventTranslationProgress, "Benchmark", nil)
	event.SessionID = "session-1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eventBus.Publish(event)
	}
}
