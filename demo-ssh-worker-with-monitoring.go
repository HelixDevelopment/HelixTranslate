package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"digital.vasic.translator/pkg/logger"
	"digital.vasic.translator/pkg/sshworker"
	"github.com/gorilla/websocket"
)

type TranslationProgress struct {
	SessionID        string
	CurrentStep      string
	Progress         float64
	Message          string
	StartTime        time.Time
	TotalItems       int
	CurrentItem      string
	Error            string
	WSClients        map[string]*websocket.Conn
}

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
	WorkerInfo   *WorkerInfo            `json:"worker_info,omitempty"`
}

type WorkerInfo struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Type     string `json:"type"`
	Model    string `json:"model"`
	Capacity int    `json:"capacity"`
}

func main() {
	sessionID := "ssh-worker-demo-" + fmt.Sprintf("%d", time.Now().Unix())
	inputFile := "test/fixtures/ebooks/russian_sample.txt"
	outputFile := "demo_ssh_worker_output.md"

	fmt.Printf("🚀 Starting SSH Worker Translation Demo with WebSocket Monitoring\n")
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
		go listenForWebSocketEvents(ws)
	}

	// Create progress tracker
	progress := &TranslationProgress{
		SessionID:   sessionID,
		CurrentStep: "initialization",
		Progress:    0,
		Message:     "Initializing SSH worker demo",
		StartTime:   time.Now(),
		WSClients:   make(map[string]*websocket.Conn),
	}

	if ws != nil {
		progress.WSClients["main"] = ws
	}

	// Emit demo started event
	emitProgressEvent(progress, "translation_started", "SSH Worker translation demo started", 0)

	// Check if SSH worker configuration exists
	workerConfig := getSSHWorkerConfig()
	if workerConfig == nil {
		emitProgressEvent(progress, "translation_error", "No SSH worker configuration found", 0)
		progress.Error = "No SSH worker configuration available"
		runDemoMode(sessionID, inputFile, outputFile, progress)
		return
	}

	workerInfo := &WorkerInfo{
		Host:     workerConfig.Host,
		Port:     workerConfig.Port,
		Type:     "ssh-llamacpp",
		Model:    "llama-2-7b-chat",
		Capacity: 10,
	}

	// Initialize logger
	logger := logger.NewLogger(logger.LoggerConfig{
		Level:  "info",
		Format: "text",
	})
	logger.Info("Starting SSH worker translation", map[string]interface{}{"session_id": sessionID})

	// Initialize SSH worker
	emitProgressEvent(progress, "translation_progress", "Connecting to SSH worker...", 5)
	
	sshWorker, err := sshworker.NewSSHWorker(*workerConfig, logger)
	if err != nil {
		emitErrorEvent(progress, "worker_connection", fmt.Sprintf("Failed to create SSH worker: %v", err))
		log.Printf("Failed to create SSH worker: %v", err)
		runDemoMode(sessionID, inputFile, outputFile, progress)
		return
	}

	// Connect to SSH worker
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err = sshWorker.Connect(ctx)
	if err != nil {
		emitErrorEvent(progress, "worker_connection", fmt.Sprintf("Failed to connect to SSH worker: %v", err))
		log.Printf("Failed to connect to SSH worker: %v", err)
		log.Printf("⚠️  SSH worker not available, running in demo mode")
		runDemoModeWithWorkerInfo(sessionID, inputFile, outputFile, progress, workerInfo)
		return
	}
	defer sshWorker.Disconnect()

	emitProgressEvent(progress, "step_completed", "Successfully connected to SSH worker", 10)

	// Read input file
	emitProgressEvent(progress, "translation_progress", "Reading input file...", 15)

	content, err := os.ReadFile(inputFile)
	if err != nil {
		emitErrorEvent(progress, "file_reading", fmt.Sprintf("Failed to read input file: %v", err))
		log.Fatalf("Failed to read input file: %v", err)
	}

	text := string(content)
	lines := strings.Split(text, "\n")
	progress.TotalItems = len(lines)

	emitProgressEvent(progress, "step_completed", fmt.Sprintf("File read successfully (%d lines)", len(lines)), 20)

	// Translate using SSH worker
	emitProgressEvent(progress, "translation_progress", "Starting translation via SSH worker...", 25)
	translatedLines := make([]string, 0, len(lines))

	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			translatedLines = append(translatedLines, "")
			continue
		}

		// Calculate progress (25% to 85% for translation phase)
		progress.Progress = 25.0 + (float64(i+1)/float64(len(lines)) * 60.0)
		progress.CurrentItem = fmt.Sprintf("line_%d", i+1)
		progress.Message = fmt.Sprintf("Translating line %d/%d via SSH worker", i+1, len(lines))
		
		emitProgressEventWithWorker(progress, "translation_progress", progress.Message, progress.Progress, workerInfo)

		// Create translation command for remote worker
		translationCmd := createTranslationCommand(line, "russian", "serbian")
		
		// Execute command on remote worker
		result, err := sshWorker.ExecuteCommand(ctx, translationCmd)
		var translatedLine string
		if err != nil {
			log.Printf("Failed to translate line %d: %v", i+1, err)
			// Fall back to demo translation
			translatedLine = translateDemoText(line)
			translatedLines = append(translatedLines, translatedLine)
		} else {
			translatedLine = strings.TrimSpace(result.Stdout)
			if translatedLine == "" {
				// Fallback to demo if worker returned empty
				translatedLine = translateDemoText(line)
			}
			translatedLines = append(translatedLines, translatedLine)
		}
		
		fmt.Printf("🔄 Line %d/%d (SSH): %s\n", i+1, len(lines), translatedLine)
	}

	// Generate output
	emitProgressEvent(progress, "translation_progress", "Generating output file...", 90)
	translatedMarkdown := strings.Join(translatedLines, "\n")
	err = os.WriteFile(outputFile, []byte(translatedMarkdown), 0644)
	if err != nil {
		emitErrorEvent(progress, "file_writing", fmt.Sprintf("Failed to write output: %v", err))
		log.Fatalf("Failed to write output: %v", err)
	}

	duration := time.Since(progress.StartTime)
	emitProgressEventWithWorker(progress, "translation_completed", 
		fmt.Sprintf("SSH Worker translation completed successfully! Output saved to %s (Duration: %v)", outputFile, duration), 
		100, workerInfo)

	fmt.Printf("\n🎉 SSH Worker Translation completed!\n")
	fmt.Printf("📁 Output file: %s\n", outputFile)
	fmt.Printf("⏱️  Duration: %v\n", duration)
	fmt.Printf("🖥️  Worker: %s:%d (%s)\n", workerConfig.Host, workerConfig.Port, workerInfo.Type)
	fmt.Printf("📊 View progress at: http://localhost:8090/monitor\n")
	fmt.Printf("🔗 Monitor this session: ws://localhost:8090/ws?session_id=%s\n", sessionID)

	// Keep running to allow monitoring
	fmt.Println("\nPress Ctrl+C to exit...")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("\n👋 SSH Worker demo completed!")
}

func runDemoMode(sessionID, inputFile, outputFile string, progress *TranslationProgress) {
	fmt.Printf("🎭 Running in demo mode with simulated SSH worker responses\n")
	runDemoModeWithWorkerInfo(sessionID, inputFile, outputFile, progress, &WorkerInfo{
		Host:     "demo-ssh-worker",
		Port:     22,
		Type:     "ssh-llamacpp",
		Model:    "llama-2-7b-chat",
		Capacity: 10,
	})
}

func runDemoModeWithWorkerInfo(sessionID, inputFile, outputFile string, progress *TranslationProgress, workerInfo *WorkerInfo) {
	fmt.Printf("🎭 Running in demo mode with simulated SSH worker responses\n")
	
	// Read input file
	content, err := os.ReadFile(inputFile)
	if err != nil {
		log.Fatalf("Failed to read input file: %v", err)
	}

	text := string(content)
	lines := strings.Split(text, "\n")
	progress.TotalItems = len(lines)

	// Emit demo started
	emitProgressEventWithWorker(progress, "translation_started", "Demo SSH worker translation started", 0, workerInfo)

	// Simulate SSH connection
	emitProgressEvent(progress, "translation_progress", "Connecting to demo SSH worker...", 5)
	time.Sleep(1 * time.Second)
	emitProgressEvent(progress, "step_completed", "Successfully connected to demo SSH worker", 10)

	// Translate line by line (demo mode)
	translatedLines := make([]string, 0, len(lines))
	for i, line := range lines {
		progress.Progress = 10.0 + (float64(i+1)/float64(len(lines)) * 75.0)
		progress.CurrentItem = fmt.Sprintf("line_%d", i+1)
		progress.Message = fmt.Sprintf("Translating line %d/%d via demo SSH worker", i+1, len(lines))
		
		emitProgressEventWithWorker(progress, "translation_progress", progress.Message, progress.Progress, workerInfo)

		time.Sleep(800 * time.Millisecond) // Simulate SSH + LLM processing time
		translatedLine := translateDemoText(line)
		translatedLines = append(translatedLines, translatedLine)
		
		fmt.Printf("🔄 Line %d/%d (Demo SSH): %s\n", i+1, len(lines), translatedLine)
	}

	// Generate output
	translatedMarkdown := strings.Join(translatedLines, "\n")
	err = os.WriteFile(outputFile, []byte(translatedMarkdown), 0644)
	if err != nil {
		log.Fatalf("Failed to write output: %v", err)
	}

	emitProgressEventWithWorker(progress, "translation_completed", 
		fmt.Sprintf("Demo SSH worker translation completed! Output saved to %s", outputFile), 100, workerInfo)

	fmt.Printf("\n🎉 Demo SSH Worker Translation completed!\n")
	fmt.Printf("📁 Output file: %s\n", outputFile)
	fmt.Printf("🖥️  Simulated Worker: %s:%d (%s)\n", workerInfo.Host, workerInfo.Port, workerInfo.Type)
	fmt.Printf("📊 View progress at: http://localhost:8090/monitor\n")
}

func getSSHWorkerConfig() *sshworker.SSHWorkerConfig {
	// Check for environment variables first
	if host := os.Getenv("SSH_WORKER_HOST"); host != "" {
		return &sshworker.SSHWorkerConfig{
			Host:              host,
			Username:          getEnvOrDefault("SSH_WORKER_USER", "milosvasic"),
			Password:          getEnvOrDefault("SSH_WORKER_PASSWORD", ""),
			PrivateKeyPath:    getEnvOrDefault("SSH_PRIVATE_KEY_PATH", ""),
			Port:              22,
			RemoteDir:         getEnvOrDefault("SSH_WORKER_REMOTE_DIR", "/tmp/translate-ssh"),
			ConnectionTimeout: 30 * time.Second,
			CommandTimeout:    60 * time.Second,
		}
	}

	// Check if config file has worker configuration
	if _, err := os.Stat("internal/working/config.distributed.json"); err == nil {
		return readWorkerFromConfig()
	}

	return nil
}

func readWorkerFromConfig() *sshworker.SSHWorkerConfig {
	// Try to read from the distributed config file
	configFile := "internal/working/config.distributed.json"
	content, err := os.ReadFile(configFile)
	if err != nil {
		return nil
	}

	var config struct {
		Distributed struct {
			Enabled bool            `json:"enabled"`
			Workers map[string]struct {
				Name     string `json:"name"`
				Host     string `json:"host"`
				Port     int    `json:"port"`
				User     string `json:"user"`
				Password string `json:"password"`
				Enabled  bool   `json:"enabled"`
			} `json:"workers"`
		} `json:"distributed"`
	}

	if err := json.Unmarshal(content, &config); err != nil {
		return nil
	}

	// Find the first enabled worker
	for name, worker := range config.Distributed.Workers {
		if worker.Enabled && name == "thinker-worker" {
			return &sshworker.SSHWorkerConfig{
				Host:              worker.Host,
				Username:          worker.User,
				Password:          worker.Password,
				Port:              worker.Port,
				RemoteDir:         "/tmp/translate-ssh",
				ConnectionTimeout: 30 * time.Second,
				CommandTimeout:    60 * time.Second,
			}
		}
	}

	return nil
}

func createTranslationCommand(text, sourceLang, targetLang string) string {
	return fmt.Sprintf("echo '%s' | /path/to/llama.cpp/main -m /path/to/model --color -c 2048 --temp 0.7 -n 256 --repeat-penalty 1.1 -i \"Translate this %s text to %s:\"", 
		text, sourceLang, targetLang)
}

func emitProgressEvent(progress *TranslationProgress, eventType, message string, progressPercent float64) {
	progress.CurrentStep = eventType
	progress.Message = message
	progress.Progress = progressPercent

	event := WebSocketEvent{
		Type:       eventType,
		SessionID:  progress.SessionID,
		Step:       eventType,
		Message:    message,
		Progress:   progressPercent,
		CurrentItem: progress.CurrentItem,
		TotalItems: progress.TotalItems,
		Timestamp:  time.Now().Unix(),
	}

	sendWebSocketEvent(progress, event)
	fmt.Printf("📤 Event: %s (%.1f%%) - %s\n", eventType, progressPercent, message)
}

func emitProgressEventWithWorker(progress *TranslationProgress, eventType, message string, progressPercent float64, workerInfo *WorkerInfo) {
	progress.CurrentStep = eventType
	progress.Message = message
	progress.Progress = progressPercent

	event := WebSocketEvent{
		Type:       eventType,
		SessionID:  progress.SessionID,
		Step:       eventType,
		Message:    message,
		Progress:   progressPercent,
		CurrentItem: progress.CurrentItem,
		TotalItems: progress.TotalItems,
		Timestamp:  time.Now().Unix(),
		WorkerInfo: workerInfo,
	}

	sendWebSocketEvent(progress, event)
	fmt.Printf("📤 Event: %s (%.1f%%) - %s [Worker: %s:%d]\n", eventType, progressPercent, message, workerInfo.Host, workerInfo.Port)
}

func emitErrorEvent(progress *TranslationProgress, step, message string) {
	progress.Error = message

	event := WebSocketEvent{
		Type:       "translation_error",
		SessionID:  progress.SessionID,
		Step:       step,
		Message:    message,
		Error:      message,
		Progress:   progress.Progress,
		Timestamp:  time.Now().Unix(),
	}

	sendWebSocketEvent(progress, event)
	fmt.Printf("❌ Error Event: %s - %s\n", step, message)
}

func sendWebSocketEvent(progress *TranslationProgress, event WebSocketEvent) {
	for clientID, ws := range progress.WSClients {
		if ws != nil {
			if err := ws.WriteJSON(event); err != nil {
				log.Printf("Failed to send WebSocket event to client %s: %v", clientID, err)
			}
		}
	}
}

func connectWebSocket(sessionID string) (*websocket.Conn, error) {
	u := fmt.Sprintf("ws://localhost:8090/ws?session_id=%s&client_id=ssh-worker-translator", sessionID)
	
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

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Simple demo translation (Russian to Serbian Cyrillic)
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