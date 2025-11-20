package websocket

import (
	"digital.vasic.translator/pkg/events"
	"encoding/json"
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

	// Subscribe to all events
	eventBus.SubscribeAll(hub.handleEvent)

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
			c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		w, err := c.Conn.NextWriter(websocket.TextMessage)
		if err != nil {
			return
		}
		w.Write(message)

		// Add queued messages to current websocket message
		n := len(c.Send)
		for i := 0; i < n; i++ {
			w.Write([]byte{'\n'})
			w.Write(<-c.Send)
		}

		if err := w.Close(); err != nil {
			return
		}
	}
}
