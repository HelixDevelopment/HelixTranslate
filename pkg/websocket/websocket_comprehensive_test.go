package websocket

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClient_ReadPump tests client read pump functionality
func TestClient_ReadPump(t *testing.T) {
	eventBus := events.NewEventBus()
	hub := NewHub(eventBus)
	go hub.Run()

	// Create test server with WebSocket upgrade
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		
		client := &Client{
			ID:        "test-client",
			SessionID: "test-session",
			Conn:      conn,
			Send:      make(chan []byte, 256),
			Hub:       hub,
		}
		hub.Register(client)
		
		// Start read pump in goroutine
		go client.ReadPump()
		
		// Wait a bit then write message to trigger disconnect
		time.Sleep(100 * time.Millisecond)
		conn.WriteMessage(websocket.CloseMessage, []byte{})
	}))
	defer server.Close()

	// Connect WebSocket client
	wsURL := "ws" + server.URL[4:] // http -> ws
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Wait for connection to be established and read pump to start
	time.Sleep(200 * time.Millisecond)
}

// TestClient_WritePump tests client write pump functionality
func TestClient_WritePump(t *testing.T) {
	eventBus := events.NewEventBus()
	hub := NewHub(eventBus)
	go hub.Run()

	// Create test server with WebSocket upgrade
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		
		client := &Client{
			ID:        "test-client",
			SessionID: "test-session",
			Conn:      conn,
			Send:      make(chan []byte, 256),
			Hub:       hub,
		}
		hub.Register(client)
		
		// Start write pump in goroutine
		go client.WritePump()
		
		// Send test message
		client.Send <- []byte("test message")
		
		// Send more messages to test queuing
		client.Send <- []byte("second message")
		
		// Wait a bit then close channel to trigger shutdown
		time.Sleep(100 * time.Millisecond)
		close(client.Send)
	}))
	defer server.Close()

	// Connect WebSocket client
	wsURL := "ws" + server.URL[4:] // http -> ws
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Read messages
	_, message, err := conn.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, "test message\nsecond message", string(message))

	// Wait for connection to close
	time.Sleep(200 * time.Millisecond)
}

// TestClient_WritePump_ErrorHandling tests write pump error handling
func TestClient_WritePump_ErrorHandling(t *testing.T) {
	eventBus := events.NewEventBus()
	hub := NewHub(eventBus)
	go hub.Run()

	// We can't easily mock the WebSocket connection, so we'll test the error path
	// by creating a real connection that will be closed immediately
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		
		client := &Client{
			ID:        "test-client",
			SessionID: "test-session",
			Conn:      conn,
			Send:      make(chan []byte, 256),
			Hub:       hub,
		}
		hub.Register(client)
		
		// Start write pump
		go client.WritePump()
		
		// Immediately close connection to trigger error
		time.Sleep(10 * time.Millisecond)
		conn.Close()
		
		// Try to send message (will fail due to closed connection)
		select {
		case client.Send <- []byte("test message"):
			// Message sent
		case <-time.After(100 * time.Millisecond):
			t.Error("Failed to send message")
		}
	}))
	defer server.Close()

	// Connect and disconnect immediately
	wsURL := "ws" + server.URL[4:] // http -> ws
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	conn.Close()
	
	// Wait for error handling
	time.Sleep(200 * time.Millisecond)
}

// TestClient_WritePump_NextWriterError tests NextWriter error handling
func TestClient_WritePump_NextWriterError(t *testing.T) {
	eventBus := events.NewEventBus()
	hub := NewHub(eventBus)
	go hub.Run()

	// Test with a real connection that will be closed
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		
		client := &Client{
			ID:        "test-client",
			SessionID: "test-session",
			Conn:      conn,
			Send:      make(chan []byte, 256),
			Hub:       hub,
		}
		hub.Register(client)
		
		// Start write pump
		go client.WritePump()
		
		// Close connection immediately to trigger NextWriter error
		time.Sleep(10 * time.Millisecond)
		conn.Close()
		
		// Try to send message (will fail on NextWriter)
		select {
		case client.Send <- []byte("test message"):
			// Message sent
		case <-time.After(100 * time.Millisecond):
			t.Error("Failed to send message")
		}
	}))
	defer server.Close()

	// Connect and disconnect immediately
	wsURL := "ws" + server.URL[4:] // http -> ws
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	conn.Close()
	
	// Wait for error handling
	time.Sleep(200 * time.Millisecond)
}

// TestClient_Integration tests client integration with hub
func TestClient_Integration(t *testing.T) {
	eventBus := events.NewEventBus()
	hub := NewHub(eventBus)
	go hub.Run()

	// Create test server with WebSocket upgrade
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		
		client := &Client{
			ID:        "integration-client",
			SessionID: "integration-session",
			Conn:      conn,
			Send:      make(chan []byte, 256),
			Hub:       hub,
		}
		hub.Register(client)
		
		// Start both pumps
		go client.ReadPump()
		go client.WritePump()
		
		// Wait for operations
		time.Sleep(200 * time.Millisecond)
	}))
	defer server.Close()

	// Connect WebSocket client
	wsURL := "ws" + server.URL[4:] // http -> ws
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Verify client is registered
	assert.Equal(t, 1, hub.GetClientCount())

	// Test event broadcasting
	event := events.NewEvent(events.EventTranslationStarted, "Integration test", map[string]interface{}{
		"client_id": "integration-client",
	})
	event.SessionID = "integration-session"
	eventBus.Publish(event)

	// Read event message
	_, message, err := conn.ReadMessage()
	require.NoError(t, err)
	assert.Contains(t, string(message), "Integration test")

	// Wait and verify
	time.Sleep(100 * time.Millisecond)
}

// TestHub_EventSubscription tests hub event subscription
func TestHub_EventSubscription(t *testing.T) {
	eventBus := events.NewEventBus()
	
	// Create hub
	hub := NewHub(eventBus)
	
	// Verify event bus subscription (internal implementation)
	// We can't directly test the subscription, but we can test
	// that events are handled correctly
	
	go hub.Run()
	
	// Create mock client to capture events
	receivedEvents := make(chan []byte, 10)
	client := &Client{
		ID:        "subscription-client",
		SessionID: "test-session",
		Conn:      nil, // Not needed for this test
		Send:      receivedEvents,
		Hub:       hub,
	}
	hub.Register(client)
	time.Sleep(50 * time.Millisecond)
	
	// Publish different event types
	eventTypes := []struct {
		eventType events.EventType
		message   string
	}{
		{events.EventTranslationStarted, "Started"},
		{events.EventTranslationProgress, "Progress"},
		{events.EventTranslationCompleted, "Completed"},
		{events.EventTranslationError, "Error"},
	}
	
	for _, eventInfo := range eventTypes {
		event := events.NewEvent(eventInfo.eventType, eventInfo.message, nil)
		event.SessionID = "test-session"
		eventBus.Publish(event)
	}
	
	// Verify all events were received
	for i := 0; i < len(eventTypes); i++ {
		select {
		case msg := <-receivedEvents:
			assert.NotEmpty(t, msg)
		case <-time.After(100 * time.Millisecond):
			t.Errorf("Event %d not received", i)
		}
	}
}

// TestWebSocket_ErrorPaths tests various error conditions
func TestWebSocket_ErrorPaths(t *testing.T) {
	// Test hub with nil clients
	eventBus := events.NewEventBus()
	hub := NewHub(eventBus)
	go hub.Run()

	// Operations with no clients should not panic
	hub.Broadcast([]byte("test"))
	assert.Equal(t, 0, hub.GetClientCount())

	// Test unregistering non-existent client
	client := &Client{
		ID:        "non-existent",
		SessionID: "test",
		Send:      make(chan []byte),
		Hub:       hub,
	}
	hub.Unregister(client) // Should not panic
	assert.Equal(t, 0, hub.GetClientCount())
}

// Benchmark tests for WebSocket operations
func BenchmarkHub_EventHandling(b *testing.B) {
	eventBus := events.NewEventBus()
	hub := NewHub(eventBus)
	go hub.Run()

	// Add some clients
	for i := 0; i < 5; i++ {
		client := &Client{
			ID:        "client-" + string(rune('0'+i)),
			SessionID: "test-session",
			Send:      make(chan []byte, 1000), // Large buffer to avoid blocking
			Hub:       hub,
		}
		hub.Register(client)
	}
	time.Sleep(50 * time.Millisecond)

	event := events.NewEvent(events.EventTranslationProgress, "Benchmark", nil)
	event.SessionID = "test-session"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eventBus.Publish(event)
	}
}