package utils

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"digital.vasic.translator/pkg/api"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/logger"
	"digital.vasic.translator/pkg/websocket"
	"digital.vasic.translator/test/mocks"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/mock"
)

// TestHTTPServer provides a reusable HTTP server for testing
type TestHTTPServer struct {
	server    *httptest.Server
	apiServer *api.Server
	port      int
	logger    logger.Logger
}

// NewTestHTTPServer creates a new test HTTP server with mock dependencies
func NewTestHTTPServer(t *testing.T) *TestHTTPServer {
	// Create mock dependencies
	mockLogger := logger.NewLogger(logger.LoggerConfig{
		Level:  logger.DEBUG,
		Format: logger.FORMAT_TEXT,
	})
	
	mockTranslator := new(mocks.MockTranslator)
	mockTranslator.On("GetName").Return("test-translator")
	mockTranslator.On("Translate", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("translated text", nil)
	mockTranslator.On("GetStats").Return(api.TranslationStats{
		Total:        0,
		Cached:       0,
		Translated:   0,
		Errors:       0,
		CacheHitRate: 0.0,
	})

	// Create API server with dynamic port
	port := GetFreePort()
	apiServer := api.NewServer(api.ServerConfig{
		Port:  port,
		Logger: mockLogger,
		Security: &api.SecurityConfig{
			APIKey: "test-api-key-12345",
		},
	})
	apiServer.SetTranslator(mockTranslator)

	// Create HTTP test server
	testServer := httptest.NewServer(apiServer.GetRouter())

	return &TestHTTPServer{
		server:    testServer,
		apiServer: apiServer,
		port:      port,
		logger:    mockLogger,
	}
}

// GetURL returns the server URL
func (s *TestHTTPServer) GetURL() string {
	return s.server.URL
}

// GetPort returns the server port
func (s *TestHTTPServer) GetPort() int {
	return s.port
}

// Close shuts down the test server
func (s *TestHTTPServer) Close() {
	if s.server != nil {
		s.server.Close()
	}
	ReleasePort(s.port)
}

// GetAPIServer returns the underlying API server
func (s *TestHTTPServer) GetAPIServer() *api.Server {
	return s.apiServer
}

// TestWebSocketServer provides a WebSocket server for testing
type TestWebSocketServer struct {
	server    *http.Server
	hub       *websocket.Hub
	eventBus  *events.EventBus
	port      int
	clients   []*websocket.Conn
	mutex     sync.Mutex
}

// NewTestWebSocketServer creates a new test WebSocket server
func NewTestWebSocketServer(t *testing.T) *TestWebSocketServer {
	port := GetFreePort()
	
	// Create event bus and hub
	eventBus := events.NewEventBus()
	hub := websocket.NewHub()
	
	// Create HTTP server with WebSocket endpoint
	mux := http.NewServeMux()
	
	wsServer := &TestWebSocketServer{
		server:   &http.Server{Addr: GetLocalhostURL(port, "")[7:]}, // Remove http:// prefix
		hub:      hub,
		eventBus: eventBus,
		port:     port,
	}
	
	// WebSocket endpoint
	mux.HandleFunc("/ws", wsServer.handleWebSocket)
	
	// Status endpoint
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"running","clients":0}`))
	})
	
	wsServer.server.Handler = mux
	
	return wsServer
}

// Start starts the WebSocket server
func (s *TestWebSocketServer) Start(t *testing.T) {
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Logf("WebSocket server error: %v", err)
		}
	}()
	
	// Wait for server to start
	time.Sleep(50 * time.Millisecond)
}

// Close shuts down the WebSocket server
func (s *TestWebSocketServer) Close(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Close all client connections
	for _, client := range s.clients {
		client.Close()
	}
	s.clients = nil
	
	// Shut down server
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	
	ReleasePort(s.port)
	return nil
}

// handleWebSocket handles WebSocket connections
func (s *TestWebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	
	s.mutex.Lock()
	s.clients = append(s.clients, conn)
	s.mutex.Unlock()
	
	// Register with hub
	s.hub.RegisterClient(conn)
	
	// Handle messages
	go func() {
		defer func() {
			s.hub.UnregisterClient(conn)
			conn.Close()
		}()
		
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}

// GetHub returns the WebSocket hub
func (s *TestWebSocketServer) GetHub() *websocket.Hub {
	return s.hub
}

// GetEventBus returns the event bus
func (s *TestWebSocketServer) GetEventBus() *events.EventBus {
	return s.eventBus
}

// GetPort returns the server port
func (s *TestWebSocketServer) GetPort() int {
	return s.port
}

// ConnectClient creates a test WebSocket client connection
func ConnectClient(t *testing.T, port int) *websocket.Conn {
	wsURL := GetLocalhostWSURL(port, "/ws")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	return conn
}

// CreateTestContext creates a test context with timeout
func CreateTestContext(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// SetupTestEnvironment provides a complete test environment setup
type SetupTestEnvironment struct {
	HTTPServer  *TestHTTPServer
	WSServer    *TestWebSocketServer
	Context     context.Context
	Logger      logger.Logger
}

// SetupTestEnvironment creates a complete test environment with HTTP and WebSocket servers
func SetupTestEnvironment(t *testing.T) *SetupTestEnvironment {
	ctx := CreateTestContext(t)
	httpServer := NewTestHTTPServer(t)
	wsServer := NewTestWebSocketServer(t)
	
	// Start WebSocket server
	wsServer.Start(t)
	
	return &SetupTestEnvironment{
		HTTPServer: httpServer,
		WSServer:   wsServer,
		Context:    ctx,
		Logger: logger.NewLogger(logger.LoggerConfig{
			Level:  logger.DEBUG,
			Format: logger.FORMAT_TEXT,
		}),
	}
}

// Cleanup cleans up the test environment
func (env *SetupTestEnvironment) Cleanup() {
	if env.HTTPServer != nil {
		env.HTTPServer.Close()
	}
	if env.WSServer != nil {
		env.WSServer.Close(env.Context)
	}
}