package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"
	"digital.vasic.translator/pkg/translator/llm"
	"github.com/gorilla/websocket"
)

type WebSocketEvent struct {
	Type         string                 `json:"type"`
	SessionID    string                 `json:"session_id"`
	Step         string                 `json:"step,omitempty"`
	Message      string                 `json:"message,omitempty"`
	Progress     float64                `json:"progress,omitempty"`
	Error        string                 `json:"error,omitempty"`
	CurrentItem  string                 `json:"current_item,omitempty"`
	TotalItems   int                    `json:"total_items,omitempty"`
	Timestamp    int64                  `json:"timestamp"`
	Data         map[string]interface{} `json:"data,omitempty"`
}

// WebSocketEventBus wraps an EventBus and forwards events to WebSocket
type WebSocketEventBus struct {
	*events.EventBus
	wsConn     *websocket.Conn
	sessionID  string
}

func NewWebSocketEventBus(wsConn *websocket.Conn, sessionID string) *WebSocketEventBus {
	eventBus := events.NewEventBus()
	return &WebSocketEventBus{
		EventBus:  eventBus,
		wsConn:    wsConn,
		sessionID: sessionID,
	}
}

func (wseb *WebSocketEventBus) Publish(event events.Event) {
	// Publish to internal event bus first
	wseb.EventBus.Publish(event)
	
	// Forward to WebSocket if connected
	if wseb.wsConn != nil {
		wsEvent := WebSocketEvent{
			Type:       string(event.Type),
			SessionID:  wseb.sessionID,
			Step:       getStepFromEvent(event),
			Message:    event.Message,
			Progress:   getProgressFromEvent(event),
			Data:       event.Data,
			Timestamp:  time.Now().Unix(),
		}
		
		// Extract additional fields from data
		if event.Data != nil {
			if step, ok := event.Data["step"].(string); ok {
				wsEvent.Step = step
			}
			if currentItem, ok := event.Data["current_item"].(string); ok {
				wsEvent.CurrentItem = currentItem
			}
			if totalItems, ok := event.Data["total_items"].(int); ok {
				wsEvent.TotalItems = totalItems
			}
			if progress, ok := event.Data["progress"].(float64); ok {
				wsEvent.Progress = progress
			}
			if errMsg, ok := event.Data["error"].(string); ok {
				wsEvent.Error = errMsg
			}
		}
		
		if err := wseb.wsConn.WriteJSON(wsEvent); err != nil {
			log.Printf("Failed to send WebSocket event: %v", err)
		}
		
		fmt.Printf("📤 Event: %s (%.1f%%) - %s\n", wsEvent.Type, wsEvent.Progress, wsEvent.Message)
	}
}

func getStepFromEvent(event events.Event) string {
	if event.Data != nil {
		if step, ok := event.Data["step"].(string); ok {
			return step
		}
	}
	return ""
}

func getProgressFromEvent(event events.Event) float64 {
	if event.Data != nil {
		if progress, ok := event.Data["progress"].(float64); ok {
			return progress
		}
	}
	return 0
}

func main() {
	sessionID := "real-llm-demo-" + fmt.Sprintf("%d", time.Now().Unix())
	inputFile := "test/fixtures/ebooks/russian_sample.txt"
	outputFile := "demo_real_llm_output.md"

	fmt.Printf("🚀 Starting Real LLM Translation Demo with WebSocket Monitoring\n")
	fmt.Printf("📊 Session ID: %s\n", sessionID)
	fmt.Printf("📄 Input: %s\n", inputFile)
	fmt.Printf("📄 Output: %s\n", outputFile)
	fmt.Printf("🔗 WebSocket: ws://localhost:8090/ws?session_id=%s\n\n", sessionID)

	// Connect to WebSocket monitoring
	ws, err := connectWebSocket(sessionID)
	if err != nil {
		log.Printf("Warning: Could not connect to monitoring: %v", err)
	} else {
		defer ws.Close()
		fmt.Printf("✅ Connected to monitoring server\n")
		
		// Start listening for messages in background
		go listenForWebSocketEvents(ws)
	}

	// Create WebSocket-enabled event bus
	var eventBus *events.EventBus
	baseEventBus := events.NewEventBus()
	
	if ws != nil {
		wsEventBus := NewWebSocketEventBus(ws, sessionID)
		// Use the wrapped event bus for publishing
		baseEventBus = wsEventBus.EventBus
		eventBus = wsEventBus
	} else {
		eventBus = baseEventBus
	}

	// Initialize real LLM translator
	fmt.Printf("🤖 Initializing LLM translator...\n")
	
	// Check for API key in environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Printf("⚠️  No OPENAI_API_KEY found, using demo mode\n")
		runDemoMode(sessionID, inputFile, outputFile, eventBus)
		return
	}

	// Configure LLM translator (OpenAI GPT-4)
	config := translator.TranslationConfig{
		SourceLang:  "russian",
		TargetLang:  "serbian",
		Provider:    "openai",
		Model:       "gpt-4",
		Temperature: 0.3,
		MaxTokens:   2000,
		Timeout:     60 * time.Second,
		APIKey:      apiKey,
		Options:     make(map[string]interface{}),
	}

	translator, err := llm.NewLLMTranslator(config)
	if err != nil {
		log.Fatalf("Failed to create LLM translator: %v", err)
	}

	fmt.Printf("✅ LLM translator initialized: %s\n", translator.GetName())

	// Emit translation started event
	if eventBus != nil {
		data := map[string]interface{}{
			"progress": 0,
			"step":     "initialization",
		}
		event := events.NewEvent(events.EventTranslationStarted, "Translation job started with LLM", data)
		event.SessionID = sessionID
		baseEventBus.Publish(event)
	}

	// Read input file
	if baseEventBus != nil {
		data := map[string]interface{}{
			"progress": 5,
			"step":     "reading",
		}
		event := events.NewEvent(events.EventTranslationProgress, "Reading input file...", data)
		event.SessionID = sessionID
		baseEventBus.Publish(event)
	}

	content, err := os.ReadFile(inputFile)
	if err != nil {
		if eventBus != nil {
			data := map[string]interface{}{
				"step":  "reading",
				"error": fmt.Sprintf("Failed to read input file: %v", err),
			}
			event := events.NewEvent(events.EventTranslationError, "Failed to read input file", data)
			event.SessionID = sessionID
			eventBus.Publish(event)
		}
		log.Fatalf("Failed to read input file: %v", err)
	}

	text := string(content)
	lines := strings.Split(text, "\n")
	totalLines := len(lines)

	if eventBus != nil {
		data := map[string]interface{}{
			"progress":    10,
			"step":        "reading",
			"current_item": "file_read",
			"total_items": totalLines,
		}
		event := events.NewEvent(events.EventStepCompleted, fmt.Sprintf("File read successfully (%d lines)", totalLines), data)
		event.SessionID = sessionID
		eventBus.Publish(event)
	}

	// Translate line by line with real LLM
	if eventBus != nil {
		data := map[string]interface{}{
			"progress": 15,
			"step":     "translation",
		}
		event := events.NewEvent(events.EventTranslationProgress, "Starting LLM translation...", data)
		event.SessionID = sessionID
		eventBus.Publish(event)
	}

	ctx := context.Background()
	translatedLines := make([]string, 0, len(lines))
	
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			translatedLines = append(translatedLines, "")
			continue
		}

		// Calculate progress (15% to 85% for translation phase)
		progress := 15.0 + (float64(i+1)/float64(totalLines) * 70.0)
		
		if eventBus != nil {
			data := map[string]interface{}{
				"progress":    progress,
				"step":        "translation",
				"current_item": fmt.Sprintf("line_%d", i+1),
				"total_items": totalLines,
			}
			event := events.NewEvent(events.EventTranslationProgress, fmt.Sprintf("Translating line %d/%d", i+1, totalLines), data)
			event.SessionID = sessionID
			eventBus.Publish(event)
		}

		// Translate using real LLM
		translatedLine, err := translator.Translate(ctx, line, "Translate this Russian text to Serbian Cyrillic")
		if err != nil {
			log.Printf("Failed to translate line %d: %v", i+1, err)
			// Fall back to demo translation
			translatedLine = translateDemoText(line)
		}
		
		translatedLines = append(translatedLines, translatedLine)
		
		fmt.Printf("🔄 Line %d/%d: %s\n", i+1, totalLines, translatedLine)
	}

	// Generate output
	if eventBus != nil {
		data := map[string]interface{}{
			"progress": 90,
			"step":     "generation",
		}
		event := events.NewEvent(events.EventTranslationProgress, "Generating output file...", data)
		event.SessionID = sessionID
		eventBus.Publish(event)
	}

	translatedMarkdown := strings.Join(translatedLines, "\n")
	err = os.WriteFile(outputFile, []byte(translatedMarkdown), 0644)
	if err != nil {
		if eventBus != nil {
			data := map[string]interface{}{
				"step":  "generation",
				"error": fmt.Sprintf("Failed to write output: %v", err),
			}
			event := events.NewEvent(events.EventTranslationError, "Failed to write output", data)
			event.SessionID = sessionID
			eventBus.Publish(event)
		}
		log.Fatalf("Failed to write output: %v", err)
	}

	// Show final stats
	stats := translator.GetStats()
	if eventBus != nil {
		data := map[string]interface{}{
			"progress":    100,
			"current_item": "output_generated",
			"total_items": totalLines,
			"stats":       stats,
		}
		event := events.NewEvent(events.EventTranslationCompleted, fmt.Sprintf("Translation completed successfully! Output saved to %s", outputFile), data)
		event.SessionID = sessionID
		eventBus.Publish(event)
	}

	fmt.Printf("\n🎉 Real LLM Translation completed!\n")
	fmt.Printf("📁 Output file: %s\n", outputFile)
	fmt.Printf("📊 View progress at: http://localhost:8090/monitor\n")
	fmt.Printf("🔗 Monitor this session: ws://localhost:8090/ws?session_id=%s\n", sessionID)
	fmt.Printf("📈 Stats: Total=%d, Translated=%d, Cached=%d, Errors=%d\n", 
		stats.Total, stats.Translated, stats.Cached, stats.Errors)

	// Keep running to allow monitoring
	fmt.Println("\nPress Ctrl+C to exit...")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("\n👋 Demo completed!")
}

func runDemoMode(sessionID, inputFile, outputFile string, eventBus *events.EventBus) {
	fmt.Printf("🎭 Running in demo mode with simulated LLM responses\n")
	
	// Read input file
	content, err := os.ReadFile(inputFile)
	if err != nil {
		log.Fatalf("Failed to read input file: %v", err)
	}

	text := string(content)
	lines := strings.Split(text, "\n")
	totalLines := len(lines)

	// Emit translation started
	if eventBus != nil {
		data := map[string]interface{}{"progress": 0, "step": "initialization"}
		event := events.NewEvent(events.EventTranslationStarted, "Demo translation started", data)
		event.SessionID = sessionID
		eventBus.Publish(event)
	}

	// Simulate reading
	if eventBus != nil {
		data := map[string]interface{}{"progress": 10, "step": "reading"}
		event := events.NewEvent(events.EventTranslationProgress, "Reading input file...", data)
		event.SessionID = sessionID
		eventBus.Publish(event)
	}

	time.Sleep(1 * time.Second)

	// Translate line by line (demo mode)
	translatedLines := make([]string, 0, len(lines))
	for i, line := range lines {
		progress := 10.0 + (float64(i+1)/float64(totalLines) * 80.0)
		
		if eventBus != nil {
			data := map[string]interface{}{
				"progress":    progress,
				"step":        "translation",
				"current_item": fmt.Sprintf("line_%d", i+1),
				"total_items": totalLines,
			}
			event := events.NewEvent(events.EventTranslationProgress, fmt.Sprintf("Translating line %d/%d (demo)", i+1, totalLines), data)
			event.SessionID = sessionID
			eventBus.Publish(event)
		}

		time.Sleep(500 * time.Millisecond) // Simulate LLM processing time
		translatedLine := translateDemoText(line)
		translatedLines = append(translatedLines, translatedLine)
		
		fmt.Printf("🔄 Line %d/%d (demo): %s\n", i+1, totalLines, translatedLine)
	}

	// Generate output
	translatedMarkdown := strings.Join(translatedLines, "\n")
	err = os.WriteFile(outputFile, []byte(translatedMarkdown), 0644)
	if err != nil {
		log.Fatalf("Failed to write output: %v", err)
	}

	if eventBus != nil {
		data := map[string]interface{}{
			"progress":    100,
			"current_item": "output_generated",
			"total_items": totalLines,
		}
		event := events.NewEvent(events.EventTranslationCompleted, fmt.Sprintf("Demo translation completed! Output saved to %s", outputFile), data)
		event.SessionID = sessionID
		eventBus.Publish(event)
	}

	fmt.Printf("\n🎉 Demo translation completed!\n")
	fmt.Printf("📁 Output file: %s\n", outputFile)
	fmt.Printf("📊 View progress at: http://localhost:8090/monitor\n")
}

func connectWebSocket(sessionID string) (*websocket.Conn, error) {
	u := fmt.Sprintf("ws://localhost:8090/ws?session_id=%s&client_id=real-llm-translator", sessionID)
	
	conn, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket: %w", err)
	}
	
	return conn, nil
}

func listenForWebSocketEvents(ws *websocket.Conn) {
	for {
		var msg map[string]interface{}
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			return
		}
		
		log.Printf("📩 Received WebSocket message: %+v", msg)
	}
}

// Simple demo translation (Russian to Serbian Cyrillic)
func translateDemoText(text string) string {
	// This is a simplified demo - in reality this would use LLM APIs
	replacements := map[string]string{
		"Это":         "Ово",
		"образец":     "пример",
		"русского":    "руског",
		"текста":      "текста",
		"для":         "за",
		"проверки":    "проверу",
		"функции":     "функције",
		"перевода":    "превода",
		"Он":          "Он",
		"содержит":    "садржи",
		"несколько":   "неколико",
		"предложений": "реченица",
		"и":           "и",
		"подходит":    "одговара",
		"тестирования": "тестирања",
		"Текст":       "Текст",
		"включает":    "укључује",
		"различные":   "различите",
		"знаки":      "знакове",
		"препинания":  "знаке",
		"структуры":   "структуре",
		".":           ".",
	}
	
	result := text
	for russian, serbian := range replacements {
		result = strings.ReplaceAll(result, russian, serbian)
	}
	
	return result
}