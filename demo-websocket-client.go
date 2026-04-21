//go:build demo

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

type EventMessage struct {
	Type      string                 `json:"type"`
	SessionID string                 `json:"session_id,omitempty"`
	Step      string                 `json:"step,omitempty"`
	Message   string                 `json:"message,omitempty"`
	Progress  float64                `json:"progress,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp int64                  `json:"timestamp,omitempty"`
}

func main() {
	sessionID := "demo-session-" + fmt.Sprintf("%d", time.Now().Unix())
	clientID := "demo-client"

	// Connect to WebSocket
	u := url.URL{Scheme: "ws", Host: "localhost:8090", Path: "/ws"}
	q := u.Query()
	q.Set("session_id", sessionID)
	q.Set("client_id", clientID)
	u.RawQuery = q.Encode()

	log.Printf("Connecting to WebSocket: %s", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("WebSocket dial error:", err)
	}
	defer conn.Close()

	log.Printf("🔗 Connected to WebSocket monitoring server")
	log.Printf("📊 Session ID: %s", sessionID)
	log.Printf("🖥️  Client ID: %s", clientID)

	// Send a demo message to test connection
	helloMsg := EventMessage{
		Type:      "hello",
		Message:   "Demo client connected",
		Timestamp: time.Now().Unix(),
	}

	if err := conn.WriteJSON(helloMsg); err != nil {
		log.Printf("Error sending hello message: %v", err)
	}

	// Simulate translation progress events
	go simulateTranslationProgress(conn, sessionID)

	// Listen for incoming messages
	for {
		var msg EventMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}

		prettyMsg, _ := json.MarshalIndent(msg, "  ", "")
		log.Printf("📩 Received event:\n%s", string(prettyMsg))
	}
}

func simulateTranslationProgress(conn *websocket.Conn, sessionID string) {
	log.Printf("🎬 Starting translation progress simulation...")

	steps := []EventMessage{
		{Type: "translation_started", SessionID: sessionID, Message: "Translation job started", Timestamp: time.Now().Unix()},
		{Type: "step_completed", SessionID: sessionID, Step: "parsing", Message: "Input file parsed successfully", Progress: 10, Timestamp: time.Now().Unix()},
		{Type: "translation_progress", SessionID: sessionID, Step: "conversion", Message: "Converting to markdown format...", Progress: 25, Timestamp: time.Now().Unix()},
		{Type: "step_completed", SessionID: sessionID, Step: "conversion", Message: "Markdown conversion completed", Progress: 30, Timestamp: time.Now().Unix()},
		{Type: "translation_progress", SessionID: sessionID, Step: "translation", Message: "Translating content with LLM...", Progress: 45, Timestamp: time.Now().Unix()},
		{Type: "translation_progress", SessionID: sessionID, Step: "translation", Message: "Applying language-specific rules...", Progress: 70, Timestamp: time.Now().Unix()},
		{Type: "step_completed", SessionID: sessionID, Step: "translation", Message: "Translation completed", Progress: 85, Timestamp: time.Now().Unix()},
		{Type: "translation_progress", SessionID: sessionID, Step: "generation", Message: "Generating EPUB output...", Progress: 95, Timestamp: time.Now().Unix()},
		{Type: "translation_completed", SessionID: sessionID, Message: "Translation job completed successfully!", Progress: 100, Timestamp: time.Now().Unix()},
	}

	for i, step := range steps {
		time.Sleep(2 * time.Second)

		log.Printf("📤 Sending step %d/%d: %s (progress: %.0f%%)", i+1, len(steps), step.Type, step.Progress)

		if err := conn.WriteJSON(step); err != nil {
			log.Printf("Error sending step %d: %v", i+1, err)
			return
		}
	}

	log.Printf("🎉 Translation simulation completed!")
}
