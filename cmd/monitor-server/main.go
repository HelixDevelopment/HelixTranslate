package main

import (
	"fmt"
	"net/http"
	"os"

	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/websocket"

	"github.com/gin-gonic/gin"
	gorillaws "github.com/gorilla/websocket"
)

// serverRunner abstracts the blocking server-start call so it can be injected
// in tests. *gin.Engine.Run satisfies this signature.
type serverRunner interface {
	Run(addr ...string) error
}

// runServer starts the monitoring server on addr and PROPAGATES any startup
// error (e.g. "address already in use") to the caller. Previously main()
// called router.Run(...) and discarded its error return, so a failed bind was
// silently swallowed — the process produced no diagnostic. Extracted as a pure
// helper so the error-propagation contract is unit-testable without binding a
// privileged port.
func runServer(r serverRunner, addr string) error {
	return r.Run(addr)
}

func main() {
	// Initialize event bus
	eventBus := events.NewEventBus()

	// Initialize WebSocket hub
	wsHub := websocket.NewHub(eventBus)
	go wsHub.Run()

	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// WebSocket endpoint
	router.GET("/ws", func(c *gin.Context) {
		upgrader := gorillaws.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}

		sessionID := c.Query("session_id")
		client := &websocket.Client{
			ID:        sessionID,
			SessionID: sessionID,
			Conn:      conn,
			Send:      make(chan []byte, 256),
			Hub:       wsHub,
		}

		wsHub.Register(client)
		go client.WritePump()
		go client.ReadPump()
	})

	// Serve static monitoring page
	router.StaticFile("/monitor", "./monitor.html")
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/monitor")
	})

	// Simple API endpoint for session status
	router.GET("/api/v1/status/:session_id", func(c *gin.Context) {
		sessionID := c.Param("session_id")
		c.JSON(http.StatusOK, gin.H{
			"session_id": sessionID,
			"status":     "monitoring_active",
			"message":    "WebSocket monitoring is available for this session",
		})
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":     "healthy",
			"component":  "ssh-monitor-server",
			"websockets": wsHub.GetClientCount(),
		})
	})

	// Start server
	port := 8090

	fmt.Printf("🚀 SSH Translation Monitoring Server Started\n")
	fmt.Printf("📊 Monitoring Dashboard: http://localhost:%d/monitor\n", port)
	fmt.Printf("🔗 WebSocket Endpoint: ws://localhost:%d/ws\n", port)
	fmt.Printf("🏥 Health Check: http://localhost:%d/health\n", port)

	if err := runServer(router, fmt.Sprintf(":%d", port)); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Monitoring server failed to start on :%d: %v\n", port, err)
		os.Exit(1)
	}
}
