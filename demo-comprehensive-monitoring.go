package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"digital.vasic.translator/pkg/events"
	"github.com/gorilla/websocket"
)

type TranslationEvent struct {
	Type         string  `json:"type"`
	SessionID    string  `json:"session_id"`
	Step         string  `json:"step,omitempty"`
	Message      string  `json:"message,omitempty"`
	Progress     float64 `json:"progress,omitempty"`
	Error        string  `json:"error,omitempty"`
	CurrentItem  string  `json:"current_item,omitempty"`
	TotalItems   int     `json:"total_items,omitempty"`
	Timestamp    int64    `json:"timestamp"`
}

func main() {
	sessionID := "comprehensive-demo-" + fmt.Sprintf("%d", time.Now().Unix())
	inputFile := "test/fixtures/ebooks/russian_sample.txt"
	outputFile := "demo_comprehensive_output.md"

	fmt.Printf("🚀 Starting Comprehensive WebSocket Monitoring Demo\n")
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

	// Create event bus for internal event management
	eventBus := events.NewEventBus()

	// Subscribe to events for logging
	eventBus.Subscribe(events.EventTranslationProgress, func(event events.Event) {
		log.Printf("📊 Progress: %s - %.1f%%", event.Message, getProgressFromData(event.Data))
	})

	eventBus.Subscribe(events.EventTranslationCompleted, func(event events.Event) {
		log.Printf("🎉 Translation completed: %s", event.Message)
	})

	// Emit translation started event
	emitEvent(ws, TranslationEvent{
		Type:      "translation_started",
		SessionID: sessionID,
		Message:   "Comprehensive translation demo started",
		Progress:  0,
		Timestamp:  time.Now().Unix(),
	})

	// Emit internal event
		emitInternalEvent(eventBus, sessionID, "Demo started internally", map[string]interface{}{
			"progress": 0,
			"step":     "initialization",
		})

	// Read input file
	emitEvent(ws, TranslationEvent{
		Type:       "translation_progress",
		SessionID:  sessionID,
		Step:       "reading",
		Message:    "Reading input file...",
		Progress:   5,
		Timestamp:  time.Now().Unix(),
	})

	content, err := os.ReadFile(inputFile)
	if err != nil {
		emitErrorEvent(ws, sessionID, "reading", fmt.Sprintf("Failed to read input file: %v", err))
		log.Fatalf("Failed to read input file: %v", err)
	}

	text := string(content)
	lines := strings.Split(text, "\n")
	totalLines := len(lines)

	emitInternalEvent(eventBus, sessionID, "File read successfully", map[string]interface{}{
		"progress":    10,
		"step":        "reading",
		"current_item": "file_read",
		"total_items":  totalLines,
	})

	emitEvent(ws, TranslationEvent{
		Type:        "step_completed",
		SessionID:   sessionID,
		Step:        "reading",
		Message:     fmt.Sprintf("File read successfully (%d lines)", totalLines),
		Progress:    10,
		CurrentItem: "file_read",
		TotalItems:  totalLines,
		Timestamp:   time.Now().Unix(),
	})

	// Simulate content preparation
	emitEvent(ws, TranslationEvent{
		Type:       "translation_progress",
		SessionID:  sessionID,
		Step:       "preparation",
		Message:    "Preparing content for translation...",
		Progress:   15,
		Timestamp:  time.Now().Unix(),
	})

	time.Sleep(1 * time.Second) // Simulate processing time

	emitInternalEvent(eventBus, sessionID, "Content prepared", map[string]interface{}{
		"progress": 20,
		"step":     "preparation",
	})

	emitEvent(ws, TranslationEvent{
		Type:        "step_completed",
		SessionID:   sessionID,
		Step:        "preparation",
		Message:     "Content preparation completed",
		Progress:    20,
		CurrentItem: "content_prepared",
		TotalItems:  totalLines,
		Timestamp:   time.Now().Unix(),
	})

	// Main translation phase with different strategies
	strategies := []string{"demo", "mock-llm", "ssh-simulation"}
	
	for _, strategy := range strategies {
		fmt.Printf("\n🔧 Running translation strategy: %s\n", strategy)
		runTranslationStrategy(ws, eventBus, sessionID, lines, strategy, totalLines)
		
		if strategy != strategies[len(strategies)-1] {
			emitEvent(ws, TranslationEvent{
				Type:       "translation_progress",
				SessionID:  sessionID,
				Step:       "strategy_switch",
				Message:    fmt.Sprintf("Switching to next strategy: %s", strategy),
				Progress:   25 + (float64(len(strategies)) * 20),
				Timestamp:  time.Now().Unix(),
			})
			time.Sleep(1 * time.Second)
		}
	}

	// Generate final output
	emitEvent(ws, TranslationEvent{
		Type:       "translation_progress",
		SessionID:  sessionID,
		Step:       "generation",
		Message:    "Generating comprehensive output...",
		Progress:   90,
		Timestamp:  time.Now().Unix(),
	})

	// Create comprehensive output with all strategies
	outputContent := generateComprehensiveOutput(text, strategies)
	err = os.WriteFile(outputFile, []byte(outputContent), 0644)
	if err != nil {
		emitErrorEvent(ws, sessionID, "generation", fmt.Sprintf("Failed to write output: %v", err))
		log.Fatalf("Failed to write output: %v", err)
	}

	time.Sleep(1 * time.Second)

	// Emit completion events
	emitInternalEvent(eventBus, sessionID, "Demo completed internally", map[string]interface{}{
		"progress":    100,
		"strategies":   strategies,
		"total_lines":  totalLines,
	})

	emitEvent(ws, TranslationEvent{
		Type:        "translation_completed",
		SessionID:   sessionID,
		Message:     fmt.Sprintf("Comprehensive demo completed! Output saved to %s", outputFile),
		Progress:    100,
		CurrentItem: "output_generated",
		TotalItems:  totalLines,
		Timestamp:   time.Now().Unix(),
	})

	fmt.Printf("\n🎉 Comprehensive WebSocket Monitoring Demo completed!\n")
	fmt.Printf("📁 Output file: %s\n", outputFile)
	fmt.Printf("📊 View progress at: http://localhost:8090/monitor\n")
	fmt.Printf("🔗 Monitor this session: ws://localhost:8090/ws?session_id=%s\n", sessionID)
	fmt.Printf("🧪 Tested strategies: %v\n", strategies)

	// Keep running to allow monitoring
	fmt.Println("\nPress Ctrl+C to exit...")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("\n👋 Comprehensive demo completed!")
}

func runTranslationStrategy(ws *websocket.Conn, eventBus *events.EventBus, sessionID string, lines []string, strategy string, totalLines int) {
	translatedLines := make([]string, 0, len(lines))
	
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			translatedLines = append(translatedLines, "")
			continue
		}

		// Calculate progress
		baseProgress := 25.0
		strategyProgress := 10.0
		totalProgress := baseProgress + strategyProgress + (float64(i+1)/float64(len(lines)) * 50.0)

		emitEvent(ws, TranslationEvent{
			Type:        "translation_progress",
			SessionID:   sessionID,
			Step:        fmt.Sprintf("translation_%s", strategy),
			Message:     fmt.Sprintf("Translating line %d/%d using %s", i+1, len(lines), strategy),
			Progress:    totalProgress,
			CurrentItem: fmt.Sprintf("line_%d_%s", i+1, strategy),
			TotalItems:  len(lines),
			Timestamp:   time.Now().Unix(),
		})

		// Translate using strategy
		var translatedLine string
		switch strategy {
		case "demo":
			time.Sleep(200 * time.Millisecond)
			translatedLine = translateDemoText(line)
		case "mock-llm":
			time.Sleep(400 * time.Millisecond)
			translatedLine = translateMockLLM(line)
		case "ssh-simulation":
			time.Sleep(600 * time.Millisecond)
			translatedLine = translateSSHSimulation(line)
		}

		translatedLines = append(translatedLines, translatedLine)
		
		// Emit internal event
		emitInternalEvent(eventBus, sessionID, 
			fmt.Sprintf("Line %d translated with %s", i+1, strategy), 
			map[string]interface{}{
				"progress":    totalProgress,
				"step":        fmt.Sprintf("translation_%s", strategy),
				"current_item": fmt.Sprintf("line_%d_%s", i+1, strategy),
				"total_items": len(lines),
				"strategy":    strategy,
			})

		fmt.Printf("🔄 [%s] Line %d/%d: %s\n", strategy, i+1, len(lines), translatedLine)
	}
}

func generateComprehensiveOutput(originalText string, strategies []string) string {
	lines := strings.Split(originalText, "\n")
	
	output := "# Comprehensive Translation Demo Results\n\n"
	output += fmt.Sprintf("**Original Text:**\n%s\n\n", originalText)
	output += fmt.Sprintf("**Strategies Tested:** %v\n\n", strategies)
	
	for _, strategy := range strategies {
		output += fmt.Sprintf("## %s Translation Results\n\n", strings.ToUpper(strategy))
		
		for i, line := range lines {
			var translated string
			switch strategy {
			case "demo":
				translated = translateDemoText(line)
			case "mock-llm":
				translated = translateMockLLM(line)
			case "ssh-simulation":
				translated = translateSSHSimulation(line)
			}
			
			if translated != "" {
				output += fmt.Sprintf("%d. %s → %s\n", i+1, line, translated)
			}
		}
		output += "\n"
	}
	
	output += "---\n"
	output += fmt.Sprintf("Generated at: %s\n", time.Now().Format(time.RFC3339))
	output += "WebSocket Monitoring System Demo\n"
	
	return output
}

func connectWebSocket(sessionID string) (*websocket.Conn, error) {
	u := fmt.Sprintf("ws://localhost:8090/ws?session_id=%s&client_id=comprehensive-demo", sessionID)
	
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

func emitEvent(ws *websocket.Conn, event TranslationEvent) {
	if ws == nil {
		return
	}
	
	event.Timestamp = time.Now().Unix()
	
	if err := ws.WriteJSON(event); err != nil {
		log.Printf("Failed to emit event: %v", err)
	}
	
	fmt.Printf("📤 Event: %s (%.1f%%) - %s\n", event.Type, event.Progress, event.Message)
}

func emitErrorEvent(ws *websocket.Conn, sessionID, step, message string) {
	event := TranslationEvent{
		Type:      "translation_error",
		SessionID: sessionID,
		Step:      step,
		Error:     message,
		Timestamp:  time.Now().Unix(),
	}
	emitEvent(ws, event)
}

func emitInternalEvent(eventBus *events.EventBus, sessionID, message string, data map[string]interface{}) {
	if eventBus == nil {
		return
	}
	
	event := events.NewEvent(events.EventTranslationProgress, message, data)
	event.SessionID = sessionID
	eventBus.Publish(event)
}

func getProgressFromData(data map[string]interface{}) float64 {
	if progress, ok := data["progress"].(float64); ok {
		return progress
	}
	return 0
}

// Different translation strategies for demonstration

func translateDemoText(text string) string {
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

func translateMockLLM(text string) string {
	// Simulate more sophisticated LLM-like translation
	prefixes := map[string]string{
		"Это":    "[LLM] Ово је",
		"Он":     "[LLM] Он представља",
		"Текст":   "[LLM] Текст садржи",
		"образец": "[LLM] пример",
	}
	
	result := text
	for original, llmVersion := range prefixes {
		if strings.Contains(result, original) {
			result = strings.Replace(result, original, llmVersion, 1)
		}
	}
	
	return result + " [LLM Enhanced]"
}

func translateSSHSimulation(text string) string {
	// Simulate SSH worker translation with remote processing
	return fmt.Sprintf("[SSH] %s [Remote Processed]", translateDemoText(text))
}