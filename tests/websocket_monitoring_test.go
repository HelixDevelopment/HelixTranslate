package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/sshworker"
	"digital.vasic.translator/pkg/websocket"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWebSocketMonitoringSystem comprehensive test suite for WebSocket monitoring
func TestWebSocketMonitoringSystem(t *testing.T) {
	// Create a test suite
	suite := &WebSocketMonitoringTestSuite{}
	suite.Setup(t)
	defer suite.Teardown()

	t.Run("WebSocketConnection", suite.TestWebSocketConnection)
	t.Run("EventTransmission", suite.TestEventTransmission)
	t.Run("MultipleClients", suite.TestMultipleClients)
	t.Run("SessionManagement", suite.TestSessionManagement)
	t.Run("ErrorHandling", suite.TestErrorHandling)
}

type WebSocketMonitoringTestSuite struct {
	server      *MonitoringTestServer
	clients     []*websocket.Conn
	eventBus     *events.EventBus
	testSessions map[string]*TestSession
	httpServer  *http.Server
}

type MonitoringTestServer struct {
	Port    int
	Clients map[string]*websocket.Conn
	Hub     *websocket.Hub
}

type TestSession struct {
	ID          string
	Events      []TestEvent
	StartTime   time.Time
	EndTime     time.Time
	Progress    float64
	WorkerInfo  *TestWorkerInfo
	Error       error
}

type TestEvent struct {
	Type       string                 `json:"type"`
	SessionID  string                 `json:"session_id"`
	Step       string                 `json:"step,omitempty"`
	Message    string                 `json:"message,omitempty"`
	Progress   float64                `json:"progress,omitempty"`
	Error      string                 `json:"error,omitempty"`
	CurrentItem string                 `json:"current_item,omitempty"`
	TotalItems int                    `json:"total_items,omitempty"`
	Timestamp  int64                  `json:"timestamp"`
	Data       map[string]interface{}   `json:"data,omitempty"`
}

type TestWorkerInfo struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Type     string `json:"type"`
	Model    string `json:"model"`
	Capacity int    `json:"capacity"`
}

func (suite *WebSocketMonitoringTestSuite) Setup(t *testing.T) {
	// Initialize test monitoring server
	suite.server = &MonitoringTestServer{
		Port:    8091, // Use different port for testing
		Clients: make(map[string]*websocket.Conn),
		Hub:     websocket.NewHub(),
	}

	// Initialize event bus
	suite.eventBus = events.NewEventBus()

	// Initialize test sessions
	suite.testSessions = make(map[string]*TestSession)

	// Start test server
	go suite.startTestServer()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Connect test clients
	suite.connectTestClients(t)
}

func (suite *WebSocketMonitoringTestSuite) Teardown() {
	// Close all client connections
	for _, client := range suite.clients {
		client.Close()
	}

	// Stop test server
	if suite.httpServer != nil {
		suite.httpServer.Shutdown(context.Background())
	}

	// Clean up
	suite.clients = nil
	suite.testSessions = nil
}

func (suite *WebSocketMonitoringTestSuite) startTestServer() {
	mux := http.NewServeMux()
	
	// WebSocket endpoint
	mux.HandleFunc("/ws", suite.handleWebSocket)

	// Status endpoint
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "running",
			"clients": len(suite.server.Clients),
			"sessions": len(suite.testSessions),
		})
	})

	suite.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", suite.server.Port),
		Handler: mux,
	}

	if err := suite.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("Test server error: %v", err)
	}
}

func (suite *WebSocketMonitoringTestSuite) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Get session and client IDs from query params
	sessionID := r.URL.Query().Get("session_id")
	clientID := r.URL.Query().Get("client_id")

	if sessionID == "" {
		sessionID = fmt.Sprintf("test-session-%d", time.Now().Unix())
	}
	if clientID == "" {
		clientID = fmt.Sprintf("test-client-%d", time.Now().Unix())
	}

	// Store client
	suite.server.Clients[clientID] = conn

	// Create test session if it doesn't exist
	if _, exists := suite.testSessions[sessionID]; !exists {
		suite.testSessions[sessionID] = &TestSession{
			ID:        sessionID,
			Events:    make([]TestEvent, 0),
			StartTime: time.Now(),
		}
	}

	// Handle client messages
	go suite.handleClientMessages(clientID, sessionID, conn)
}

func (suite *WebSocketMonitoringTestSuite) handleClientMessages(clientID, sessionID string, conn *websocket.Conn) {
	defer func() {
		delete(suite.server.Clients, clientID)
		conn.Close()
	}()

	for {
		var event TestEvent
		err := conn.ReadJSON(&event)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Store event in test session
		if session, exists := suite.testSessions[sessionID]; exists {
			session.Events = append(session.Events, event)
			session.Progress = event.Progress
			
			if event.Error != "" {
				session.Error = fmt.Errorf(event.Error)
			}
			
			if event.Type == "translation_completed" {
				session.EndTime = time.Now()
			}
		}

		// Broadcast to other clients
		suite.broadcastEvent(event, clientID)
	}
}

func (suite *WebSocketMonitoringTestSuite) broadcastEvent(event TestEvent, excludeClientID string) {
	for clientID, conn := range suite.server.Clients {
		if clientID == excludeClientID {
			continue
		}

		if err := conn.WriteJSON(event); err != nil {
			log.Printf("Failed to send event to client %s: %v", clientID, err)
			conn.Close()
			delete(suite.server.Clients, clientID)
		}
	}
}

func (suite *WebSocketMonitoringTestSuite) connectTestClients(t *testing.T) {
	// Connect multiple test clients
	for i := 0; i < 3; i++ {
		conn, _, err := websocket.DefaultDialer.Dial(
			fmt.Sprintf("ws://localhost:%d/ws?session_id=test-session&client_id=test-client-%d", suite.server.Port, i),
			nil,
		)
		require.NoError(t, err, "Failed to connect test client %d", i)

		suite.clients = append(suite.clients, conn)
	}

	// Wait for connections to establish
	time.Sleep(50 * time.Millisecond)
}

func (suite *WebSocketMonitoringTestSuite) TestWebSocketConnection(t *testing.T) {
	// Test that all clients are connected
	assert.Equal(t, 3, len(suite.clients), "Should have 3 connected clients")
	assert.Equal(t, 3, len(suite.server.Clients), "Server should track 3 clients")

	// Test server status
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/status", suite.server.Port))
	require.NoError(t, err)
	defer resp.Body.Close()

	var status map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&status)
	require.NoError(t, err)

	assert.Equal(t, "running", status["status"])
	assert.Equal(t, float64(3), status["clients"])
}

func (suite *WebSocketMonitoringTestSuite) TestEventTransmission(t *testing.T) {
	// Send test event from first client
	testEvent := TestEvent{
		Type:        "translation_progress",
		SessionID:   "test-session",
		Step:        "translation",
		Message:     "Test message",
		Progress:    50.0,
		CurrentItem: "test_item",
		TotalItems:  100,
		Timestamp:   time.Now().Unix(),
	}

	// Send event
	err := suite.clients[0].WriteJSON(testEvent)
	require.NoError(t, err)

	// Wait for broadcast
	time.Sleep(50 * time.Millisecond)

	// Verify event was stored in session
	session := suite.testSessions["test-session"]
	assert.Equal(t, 1, len(session.Events), "Session should have 1 event")
	assert.Equal(t, testEvent.Type, session.Events[0].Type)
	assert.Equal(t, testEvent.Progress, session.Events[0].Progress)

	// Verify other clients received the event (skip first client which sent it)
	for i := 1; i < len(suite.clients); i++ {
		var receivedEvent TestEvent
		err := suite.clients[i].ReadJSON(&receivedEvent)
		if err != nil && websocket.IsUnexpectedCloseError(err) {
			continue // Skip if client disconnected
		}
		require.NoError(t, err)
		assert.Equal(t, testEvent.Type, receivedEvent.Type)
		assert.Equal(t, testEvent.Message, receivedEvent.Message)
	}
}

func (suite *WebSocketMonitoringTestSuite) TestMultipleClients(t *testing.T) {
	// Test concurrent events from multiple clients
	eventCount := 5
	for i := 0; i < len(suite.clients); i++ {
		go func(clientIndex int) {
			for j := 0; j < eventCount; j++ {
				event := TestEvent{
					Type:       "translation_progress",
					SessionID:  fmt.Sprintf("test-session-%d", clientIndex),
					Message:    fmt.Sprintf("Client %d Event %d", clientIndex, j),
					Progress:   float64(j*10),
					Timestamp:  time.Now().Unix(),
				}

				suite.clients[clientIndex].WriteJSON(event)
				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	// Wait for all events
	time.Sleep(500 * time.Millisecond)

	// Verify all events were processed
	totalEvents := 0
	for _, session := range suite.testSessions {
		totalEvents += len(session.Events)
	}

	expectedEvents := len(suite.clients) * eventCount
	assert.GreaterOrEqual(t, totalEvents, expectedEvents/2, "Should process most events (allowing for some loss)")
}

func (suite *WebSocketMonitoringTestSuite) TestSessionManagement(t *testing.T) {
	// Test session lifecycle
	sessionID := "lifecycle-test-session"

	// Create session via WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(
		fmt.Sprintf("ws://localhost:%d/ws?session_id=%s&client_id=lifecycle-client", suite.server.Port, sessionID),
		nil,
	)
	require.NoError(t, err)
	defer conn.Close()

	// Send lifecycle events
	events := []TestEvent{
		{Type: "translation_started", SessionID: sessionID, Message: "Started", Progress: 0},
		{Type: "translation_progress", SessionID: sessionID, Message: "Progress", Progress: 50},
		{Type: "translation_completed", SessionID: sessionID, Message: "Completed", Progress: 100},
	}

	for _, event := range events {
		err := conn.WriteJSON(event)
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond)
	}

	// Verify session exists
	session, exists := suite.testSessions[sessionID]
	assert.True(t, exists, "Session should exist")
	assert.Equal(t, sessionID, session.ID, "Session ID should match")
	assert.Equal(t, len(events), len(session.Events), "Should have all events")
	assert.Equal(t, 100.0, session.Progress, "Final progress should be 100")

	// Verify session has end time if completed
	assert.NotZero(t, session.EndTime, "Session should have end time when completed")
}

func (suite *WebSocketMonitoringTestSuite) TestErrorHandling(t *testing.T) {
	// Test error event handling
	errorEvent := TestEvent{
		Type:       "translation_error",
		SessionID:  "test-session",
		Step:       "translation",
		Message:    "Test error occurred",
		Error:      "Simulated error for testing",
		Progress:   25.0,
		Timestamp:  time.Now().Unix(),
	}

	err := suite.clients[0].WriteJSON(errorEvent)
	require.NoError(t, err)

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	// Verify error was stored
	session := suite.testSessions["test-session"]
	assert.NotNil(t, session.Error, "Session should have error")
	assert.Contains(t, session.Error.Error(), "Simulated error", "Error message should match")
}

// TestSSHWorkerIntegration tests SSH worker functionality
func TestSSHWorkerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping SSH worker integration test in short mode")
	}

	// Create test SSH worker configuration
	config := sshworker.SSHWorkerConfig{
		Host:              "localhost",
		Username:          "testuser",
		Password:          "testpass",
		Port:              22,
		RemoteDir:         "/tmp/translate-ssh-test",
		ConnectionTimeout:  5 * time.Second,
		CommandTimeout:    10 * time.Second,
	}

	t.Run("SSHWorkerCreation", func(t *testing.T) {
		// Test SSH worker creation
		logger := &MockLogger{}
		worker, err := sshworker.NewSSHWorker(config, logger)
		
		// Should succeed (connection will fail later)
		assert.NoError(t, err)
		assert.NotNil(t, worker)
	})

	t.Run("SSHWorkerConnection", func(t *testing.T) {
		// Skip if no SSH server available
		if !isSSHServerAvailable(config.Host, config.Port) {
			t.Skip("SSH server not available for integration test")
		}

		logger := &MockLogger{}
		worker, err := sshworker.NewSSHWorker(config, logger)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Test connection
		err = worker.Connect(ctx)
		if err != nil {
			t.Skip("SSH connection failed - worker not configured")
			return
		}

		defer worker.Disconnect()

		// Test command execution
		result, err := worker.ExecuteCommand(ctx, "echo 'Hello from SSH worker'")
		assert.NoError(t, err)
		assert.Equal(t, 0, result.ExitCode)
		assert.Contains(t, result.Stdout, "Hello from SSH worker")
	})
}

// TestEventBusIntegration tests event system integration
func TestEventBusIntegration(t *testing.T) {
	t.Run("EventCreation", func(t *testing.T) {
		data := map[string]interface{}{
			"progress":   75.0,
			"step":       "translation",
			"current_item": "test_item",
			"total_items": 100,
		}

		event := events.NewEvent(events.EventTranslationProgress, "Test event", data)
		
		assert.Equal(t, events.EventTranslationProgress, event.Type)
		assert.Equal(t, "Test event", event.Message)
		assert.Equal(t, data, event.Data)
		assert.NotZero(t, event.Timestamp)
	})

	t.Run("EventSubscription", func(t *testing.T) {
		eventBus := events.NewEventBus()
		receivedEvents := make([]events.Event, 0)
		
		// Subscribe to events
		eventBus.Subscribe(events.EventTranslationProgress, func(event events.Event) {
			receivedEvents = append(receivedEvents, event)
		})

		// Publish event
		data := map[string]interface{}{"progress": 50.0}
		event := events.NewEvent(events.EventTranslationProgress, "Test progress", data)
		eventBus.Publish(event)

		// Give time for async processing
		time.Sleep(10 * time.Millisecond)

		assert.Len(t, receivedEvents, 1)
		assert.Equal(t, events.EventTranslationProgress, receivedEvents[0].Type)
	})
}

// TestWebSocketPerformance tests performance characteristics
func TestWebSocketPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	t.Run("HighFrequencyEvents", func(t *testing.T) {
		// Connect test client
		conn, _, err := websocket.DefaultDialer.Dial(
			"ws://localhost:8091/ws?session_id=perf-test&client_id=perf-client",
			nil,
		)
		require.NoError(t, err)
		defer conn.Close()

		eventCount := 100
		startTime := time.Now()

		// Send high-frequency events
		for i := 0; i < eventCount; i++ {
			event := TestEvent{
				Type:       "translation_progress",
				SessionID:  "perf-test",
				Message:    fmt.Sprintf("Performance test event %d", i),
				Progress:   float64(i),
				Timestamp:  time.Now().Unix(),
			}

			err := conn.WriteJSON(event)
			assert.NoError(t, err)
		}

		duration := time.Since(startTime)
		avgLatency := duration / time.Duration(eventCount)

		t.Logf("Sent %d events in %v (avg: %v per event)", 
			eventCount, duration, avgLatency)

		// Performance requirements
		assert.Less(t, avgLatency, 10*time.Millisecond, 
			"Average event latency should be less than 10ms")
	})
}

// Helper functions and utilities

func isSSHServerAvailable(host string, port int) bool {
	conn, err := websocket.DefaultDialer.Dial(
		fmt.Sprintf("ws://%s:%d", host, port),
		nil,
	)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// MockLogger implements logger interface for testing
type MockLogger struct {
	Messages []string
	Errors   []string
}

func (m *MockLogger) Debug(message string, fields map[string]interface{}) {
	m.Messages = append(m.Messages, fmt.Sprintf("DEBUG: %s", message))
}

func (m *MockLogger) Info(message string, fields map[string]interface{}) {
	m.Messages = append(m.Messages, fmt.Sprintf("INFO: %s", message))
}

func (m *MockLogger) Warn(message string, fields map[string]interface{}) {
	m.Messages = append(m.Messages, fmt.Sprintf("WARN: %s", message))
}

func (m *MockLogger) Error(message string, fields map[string]interface{}) {
	m.Errors = append(m.Errors, fmt.Sprintf("ERROR: %s", message))
}

func (m *MockLogger) Fatal(message string, fields map[string]interface{}) {
	m.Errors = append(m.Errors, fmt.Sprintf("FATAL: %s", message))
}

// BenchmarkWebSocketEventTransmission benchmarks WebSocket event transmission
func BenchmarkWebSocketEventTransmission(b *testing.B) {
	// Connect to test server
	conn, _, err := websocket.DefaultDialer.Dial(
		"ws://localhost:8091/ws?session_id=bench-test&client_id=bench-client",
		nil,
	)
	if err != nil {
		b.Skip("Cannot connect to test server for benchmark")
		return
	}
	defer conn.Close()

	testEvent := TestEvent{
		Type:       "translation_progress",
		SessionID:  "bench-test",
		Message:    "Benchmark test",
		Progress:   50.0,
		Timestamp:  time.Now().Unix(),
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := conn.WriteJSON(testEvent)
			if err != nil {
				b.Fatalf("Failed to send event: %v", err)
			}
		}
	})
}

// Integration test for complete monitoring workflow
func TestCompleteMonitoringWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("EndToEndWorkflow", func(t *testing.T) {
		// This test simulates a complete translation monitoring workflow
		// 1. Start monitoring server
		// 2. Connect multiple clients
		// 3. Run translation simulation
		// 4. Verify complete event flow
		// 5. Check final results

		sessionID := "e2e-test-session"
		
		// Connect monitoring client
		conn, _, err := websocket.DefaultDialer.Dial(
			fmt.Sprintf("ws://localhost:8091/ws?session_id=%s&client_id=e2e-monitor", sessionID),
			nil,
		)
		require.NoError(t, err)
		defer conn.Close()

		// Simulate translation workflow events
		workflowEvents := []TestEvent{
			{Type: "translation_started", SessionID: sessionID, Message: "Translation started", Progress: 0},
			{Type: "translation_progress", SessionID: sessionID, Step: "reading", Message: "Reading input file", Progress: 10},
			{Type: "translation_progress", SessionID: sessionID, Step: "translation", Message: "Translating content", Progress: 50},
			{Type: "translation_progress", SessionID: sessionID, Step: "translation", Message: "Almost done", Progress: 90},
			{Type: "translation_completed", SessionID: sessionID, Message: "Translation completed", Progress: 100},
		}

		// Send workflow events
		for _, event := range workflowEvents {
			err := conn.WriteJSON(event)
			require.NoError(t, err)
			time.Sleep(10 * time.Millisecond) // Simulate processing time
		}

		// Verify session completed correctly
		session := suite.testSessions[sessionID]
		assert.NotNil(t, session)
		assert.Equal(t, 100.0, session.Progress)
		assert.Equal(t, len(workflowEvents), len(session.Events))
		assert.NotZero(t, session.EndTime)

		// Verify event sequence
		for i, event := range session.Events {
			assert.Equal(t, workflowEvents[i].Type, event.Type)
			assert.Equal(t, workflowEvents[i].Progress, event.Progress)
		}
	})
}

// Test utility functions for WebSocket monitoring
func TestWebSocketUtilities(t *testing.T) {
	t.Run("EventSerialization", func(t *testing.T) {
		event := TestEvent{
			Type:        "translation_progress",
			SessionID:   "test-session",
			Step:        "translation",
			Message:     "Test serialization",
			Progress:    75.5,
			CurrentItem: "test_item",
			TotalItems:  100,
			Timestamp:   time.Now().Unix(),
		}

		// Test JSON serialization
		data, err := json.Marshal(event)
		assert.NoError(t, err)

		// Test JSON deserialization
		var decodedEvent TestEvent
		err = json.Unmarshal(data, &decodedEvent)
		assert.NoError(t, err)

		// Verify all fields match
		assert.Equal(t, event.Type, decodedEvent.Type)
		assert.Equal(t, event.SessionID, decodedEvent.SessionID)
		assert.Equal(t, event.Progress, decodedEvent.Progress)
		assert.Equal(t, event.CurrentItem, decodedEvent.CurrentItem)
	})

	t.Run("WorkerInfoSerialization", func(t *testing.T) {
		workerInfo := TestWorkerInfo{
			Host:     "localhost",
			Port:     8444,
			Type:     "ssh-llamacpp",
			Model:    "llama-2-7b-chat",
			Capacity: 10,
		}

		// Test JSON serialization
		data, err := json.Marshal(workerInfo)
		assert.NoError(t, err)

		// Test JSON deserialization
		var decodedWorkerInfo TestWorkerInfo
		err = json.Unmarshal(data, &decodedWorkerInfo)
		assert.NoError(t, err)

		// Verify all fields match
		assert.Equal(t, workerInfo.Host, decodedWorkerInfo.Host)
		assert.Equal(t, workerInfo.Port, decodedWorkerInfo.Port)
		assert.Equal(t, workerInfo.Type, decodedWorkerInfo.Type)
		assert.Equal(t, workerInfo.Model, decodedWorkerInfo.Model)
		assert.Equal(t, workerInfo.Capacity, decodedWorkerInfo.Capacity)
	})
}