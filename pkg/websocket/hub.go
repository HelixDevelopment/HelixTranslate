package websocket

import (
	"digital.vasic.translator/pkg/events"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// Client represents a WebSocket client
type Client struct {
	ID        string
	SessionID string
	Conn      *websocket.Conn
	Send      chan []byte
	Hub       *Hub
}

// Hub manages WebSocket connections
type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	eventBus   *events.EventBus
}

// NewHub creates a new WebSocket hub
func NewHub(eventBus *events.EventBus) *Hub {
	hub := &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		eventBus:   eventBus,
	}

	// Subscribe to all events (if event bus is provided)
	if eventBus != nil {
		eventBus.SubscribeAll(hub.handleEvent)
	}

	return hub
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
		}
	}
}

// Register registers a new client
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister unregisters a client
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// handleEvent handles events from the event bus
func (h *Hub) handleEvent(event events.Event) {
	// Convert event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	// Send to all clients (or filter by session ID)
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		// Filter by session ID if specified
		if event.SessionID != "" && client.SessionID != "" && client.SessionID != event.SessionID {
			continue
		}

		select {
		case client.Send <- data:
		default:
			// Client's send channel is full, skip
		}
	}
}

// Broadcast sends a message to all clients
func (h *Hub) Broadcast(message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		select {
		case client.Send <- message:
		default:
			// Client's send channel is full, skip
		}
	}
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// ReadPump handles reading messages from the client
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister(c)
		c.Conn.Close()
	}()

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
		// We don't expect messages from clients in this implementation
		// But we need to read to detect disconnections
	}
}

// WritePump handles writing messages to the client
func (c *Client) WritePump() {
	defer func() {
		c.Conn.Close()
	}()

	for {
		message, ok := <-c.Send
		if !ok {
			// Hub closed the channel
			_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		w, err := c.Conn.NextWriter(websocket.TextMessage)
		if err != nil {
			return
		}
		if _, err := w.Write(message); err != nil {
			return
		}

		// Add queued messages to current websocket message
		n := len(c.Send)
		for i := 0; i < n; i++ {
			if _, err := w.Write([]byte{'\n'}); err != nil {
				return
			}
			if _, err := w.Write(<-c.Send); err != nil {
				return
			}
		}

		if err := w.Close(); err != nil {
			return
		}
	}
}

// wsHandler upgrades an HTTP request to a WebSocket connection, builds the
// Client, registers it with the hub, and starts its pumps. It reads BOTH
// client_id AND session_id from the query string: session_id populates
// Client.SessionID, which the hub's per-session fan-out filter (handleEvent)
// relies on. Without it every dashboard client receives every session's
// events instead of only its own.
func (h *Hub) wsHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		ID:        r.URL.Query().Get("client_id"),
		SessionID: r.URL.Query().Get("session_id"),
		Conn:      conn,
		Send:      make(chan []byte, 256),
		Hub:       h,
	}
	h.Register(client)

	go client.WritePump()
	go client.ReadPump()
}

// StartServer starts an HTTP server with WebSocket endpoint on the given address.
// It registers the handler on a private ServeMux rather than the global
// http.DefaultServeMux so a second StartServer call (or any other consumer of
// the default mux in the same process) does not panic with a duplicate "/ws"
// pattern registration.
func (h *Hub) StartServer(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", h.wsHandler)
	return http.ListenAndServe(addr, mux)
}
