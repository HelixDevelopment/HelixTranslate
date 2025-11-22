package api

import (
	"context"
	"digital.vasic.translator/internal/cache"
	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/pkg/distributed"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/fb2"
	"digital.vasic.translator/pkg/script"
	"digital.vasic.translator/pkg/security"
	"digital.vasic.translator/pkg/translator"
	"digital.vasic.translator/pkg/translator/dictionary"
	"digital.vasic.translator/pkg/translator/llm"
	"digital.vasic.translator/pkg/websocket"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	gorillaws "github.com/gorilla/websocket"
)

// Handler handles API requests
type Handler struct {
	config             *config.Config
	eventBus           *events.EventBus
	cache              *cache.Cache
	authService        *security.AuthService
	wsHub              *websocket.Hub
	distributedManager interface{} // Will be *distributed.DistributedManager
}

// NewHandler creates a new API handler
func NewHandler(
	cfg *config.Config,
	eventBus *events.EventBus,
	cache *cache.Cache,
	authService *security.AuthService,
	wsHub *websocket.Hub,
	distributedManager interface{},
) *Handler {
	return &Handler{
		config:             cfg,
		eventBus:           eventBus,
		cache:              cache,
		authService:        authService,
		wsHub:              wsHub,
		distributedManager: distributedManager,
	}
}

// RegisterRoutes registers all API routes
func (h *Handler) RegisterRoutes(router *gin.Engine) {
	// Health check
	router.GET("/health", h.healthCheck)
	router.GET("/", h.apiInfo)

	// WebSocket endpoint
	router.GET("/ws", h.websocketHandler)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Translation endpoints
		v1.POST("/translate", h.translateText)
		v1.POST("/translate/fb2", h.translateFB2)
		v1.POST("/translate/batch", h.batchTranslate)

		// Script conversion
		v1.POST("/convert/script", h.convertScript)

		// Status and info
		v1.GET("/status/:session_id", h.getStatus)
		v1.GET("/version", h.getVersion)
		v1.GET("/providers", h.listProviders)
		v1.GET("/stats", h.getStats)

		// Distributed work endpoints
		if h.config.Distributed.Enabled {
			v1.GET("/distributed/status", h.getDistributedStatus)
			v1.POST("/distributed/workers/discover", h.discoverWorkers)
			v1.POST("/distributed/workers/:worker_id/pair", h.pairWorker)
			v1.DELETE("/distributed/workers/:worker_id/pair", h.unpairWorker)
			v1.POST("/distributed/translate", h.translateDistributed)

			// Update endpoints for workers
			v1.POST("/update/upload", h.uploadUpdate)
			v1.POST("/update/apply", h.applyUpdate)
			v1.POST("/update/rollback", h.rollbackUpdate)

			// Version management monitoring endpoints
			v1.GET("/monitoring/version/metrics", h.getVersionMetrics)
			v1.GET("/monitoring/version/alerts", h.getVersionAlerts)
			v1.GET("/monitoring/version/health", h.getVersionHealth)
			v1.GET("/monitoring/version/dashboard", h.getVersionDashboard)
			v1.POST("/monitoring/version/drift-check", h.triggerVersionDriftCheck)
			v1.GET("/monitoring/version/alerts/history", h.getAlertHistory)
			v1.POST("/monitoring/version/alerts/:alert_id/acknowledge", h.acknowledgeAlert)
			v1.POST("/monitoring/version/alerts/channels/email", h.addEmailAlertChannel)
			v1.POST("/monitoring/version/alerts/channels/webhook", h.addWebhookAlertChannel)
			v1.POST("/monitoring/version/alerts/channels/slack", h.addSlackAlertChannel)
			v1.GET("/monitoring/version/dashboard.html", h.serveDashboard)
		}

		// Register batch processing routes
		h.RegisterBatchRoutes(v1)

		// Authentication (if enabled)
		if h.config.Security.EnableAuth {
			v1.POST("/auth/login", h.login)
			v1.POST("/auth/token", h.generateToken)

			// Protected routes
			protected := v1.Group("/")
			protected.Use(h.authMiddleware())
			{
				protected.GET("/profile", h.getProfile)
			}
		}
	}
}

// healthCheck handles health check requests
func (h *Handler) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"version": "1.0.0",
		"time":    time.Now().UTC(),
	})
}

// apiInfo provides API information
func (h *Handler) apiInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"name":        "Russian-Serbian FB2 Translator API",
		"version":     "1.0.0",
		"description": "High-quality Russian to Serbian translation service with multiple LLM providers",
		"endpoints": gin.H{
			"health":       "GET /health",
			"websocket":    "GET /ws",
			"translate":    "POST /api/v1/translate",
			"translateFB2": "POST /api/v1/translate/fb2",
			"providers":    "GET /api/v1/providers",
		},
		"documentation": "/api/docs",
	})
}

// translateText handles text translation requests
func (h *Handler) translateText(c *gin.Context) {
	var req struct {
		Text     string `json:"text" binding:"required"`
		Provider string `json:"provider"`
		Model    string `json:"model"`
		Context  string `json:"context"`
		Script   string `json:"script"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create translator
	trans, err := h.createTranslator(req.Provider, req.Model)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate session ID
	sessionID := uuid.New().String()

	// Translate - use distributed coordinator if available
	ctx := context.Background()
	var translated string

	if h.distributedManager != nil {
		// Try distributed translation first
		if dm, ok := h.distributedManager.(*distributed.DistributedManager); ok {
			distributedResult, distributedErr := dm.TranslateDistributed(ctx, req.Text, req.Context)
			if distributedErr == nil {
				translated = distributedResult
			} else {
				// Fallback to local translation
				h.eventBus.Publish(events.Event{
					Type:      "distributed_fallback",
					SessionID: sessionID,
					Message:   "Distributed translation failed, using local translator",
					Data: map[string]interface{}{
						"error": distributedErr.Error(),
					},
				})
				localResult, localErr := trans.TranslateWithProgress(ctx, req.Text, req.Context, h.eventBus, sessionID)
				if localErr != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": localErr.Error()})
					return
				}
				translated = localResult
			}
		} else {
			// Type assertion failed, use local translator
			localResult, localErr := trans.TranslateWithProgress(ctx, req.Text, req.Context, h.eventBus, sessionID)
			if localErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": localErr.Error()})
				return
			}
			translated = localResult
		}
	} else {
		// Use local translator
		localResult, localErr := trans.TranslateWithProgress(ctx, req.Text, req.Context, h.eventBus, sessionID)
		if localErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": localErr.Error()})
			return
		}
		translated = localResult
	}

	// Convert script if requested
	if req.Script == "latin" {
		converter := script.NewConverter()
		translated = converter.ToLatin(translated)
	}

	c.JSON(http.StatusOK, gin.H{
		"original":   req.Text,
		"translated": translated,
		"provider":   trans.GetName(),
		"session_id": sessionID,
		"stats":      trans.GetStats(),
	})
}

// translateFB2 handles FB2 file translation
func (h *Handler) translateFB2(c *gin.Context) {
	// Parse multipart form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
		return
	}
	defer file.Close()

	provider := c.PostForm("provider")
	if provider == "" {
		provider = "dictionary"
	}

	model := c.PostForm("model")
	scriptType := c.PostForm("script")
	if scriptType == "" {
		scriptType = "cyrillic"
	}

	// Generate session ID
	sessionID := uuid.New().String()

	// Emit start event
	startEvent := events.NewEvent(
		events.EventTranslationStarted,
		"FB2 translation started",
		map[string]interface{}{
			"filename": header.Filename,
			"provider": provider,
		},
	)
	startEvent.SessionID = sessionID
	h.eventBus.Publish(startEvent)

	// Parse FB2
	parser := fb2.NewParser()
	book, err := parser.ParseReader(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to parse FB2: %v", err)})
		return
	}

	// Create translator
	trans, err := h.createTranslator(provider, model)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Translate
	ctx := context.Background()
	if err := h.translateBook(ctx, book, trans, sessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert script if needed
	if scriptType == "latin" {
		converter := script.NewConverter()
		h.convertBookToLatin(book, converter)
	}

	// Update metadata
	book.SetLanguage("sr")

	// Generate output filename
	outputFilename := generateOutputFilename(header.Filename, provider)

	// Set headers for file download
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", outputFilename))
	c.Header("Content-Type", "application/xml")

	// Write FB2 to response
	if err := parser.WriteToWriter(c.Writer, book); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to write FB2: %v", err)})
		return
	}

	// Emit completion event
	completeEvent := events.NewEvent(
		events.EventTranslationCompleted,
		"FB2 translation completed",
		map[string]interface{}{
			"filename": outputFilename,
			"stats":    trans.GetStats(),
		},
	)
	completeEvent.SessionID = sessionID
	h.eventBus.Publish(completeEvent)
}

// batchTranslate handles batch translation requests
func (h *Handler) batchTranslate(c *gin.Context) {
	var req struct {
		Texts    []string `json:"texts" binding:"required"`
		Provider string   `json:"provider"`
		Model    string   `json:"model"`
		Context  string   `json:"context"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create translator
	trans, err := h.createTranslator(req.Provider, req.Model)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate session ID
	sessionID := uuid.New().String()

	// Translate all texts
	ctx := context.Background()
	results := make([]string, len(req.Texts))

	for i, text := range req.Texts {
		translated, err := trans.TranslateWithProgress(ctx, text, req.Context, h.eventBus, sessionID)
		if err != nil {
			results[i] = fmt.Sprintf("[ERROR: %v]", err)
		} else {
			results[i] = translated
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"originals":  req.Texts,
		"translated": results,
		"provider":   trans.GetName(),
		"session_id": sessionID,
		"stats":      trans.GetStats(),
	})
}

// convertScript handles script conversion
func (h *Handler) convertScript(c *gin.Context) {
	var req struct {
		Text   string `json:"text" binding:"required"`
		Target string `json:"target" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	converter := script.NewConverter()
	var result string

	switch req.Target {
	case "latin":
		result = converter.ToLatin(req.Text)
	case "cyrillic":
		result = converter.ToCyrillic(req.Text)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target script"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"original":  req.Text,
		"converted": result,
		"target":    req.Target,
	})
}

// getStatus returns translation status for a session
func (h *Handler) getStatus(c *gin.Context) {
	sessionID := c.Param("session_id")

	c.JSON(http.StatusOK, gin.H{
		"session_id": sessionID,
		"status":     "completed", // In a real implementation, track session status
	})
}

// listProviders lists available translation providers
func (h *Handler) listProviders(c *gin.Context) {
	providers := []gin.H{
		{
			"name":             "dictionary",
			"description":      "Simple dictionary-based translation",
			"requires_api_key": false,
		},
		{
			"name":             "openai",
			"description":      "OpenAI GPT models",
			"requires_api_key": true,
			"models":           []string{"gpt-4", "gpt-3.5-turbo"},
		},
		{
			"name":             "anthropic",
			"description":      "Anthropic Claude models",
			"requires_api_key": true,
			"models":           []string{"claude-3-sonnet-20240229", "claude-3-opus-20240229"},
		},
		{
			"name":             "zhipu",
			"description":      "Zhipu AI GLM models",
			"requires_api_key": true,
			"models":           []string{"glm-4"},
		},
		{
			"name":             "deepseek",
			"description":      "DeepSeek Chat models",
			"requires_api_key": true,
			"models":           []string{"deepseek-chat"},
		},
		{
			"name":             "ollama",
			"description":      "Local Ollama models",
			"requires_api_key": false,
			"models":           []string{"llama3:8b", "llama2:13b"},
		},
		{
			"name":             "llamacpp",
			"description":      "Local Llama.cpp models",
			"requires_api_key": false,
			"models":           []string{"llama-3.2-3b-instruct"},
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"providers": providers,
	})
}

// getVersion returns version information
func (h *Handler) getVersion(c *gin.Context) {
	version := distributed.VersionInfo{
		CodebaseVersion: getCodebaseVersion(),
		BuildTime:       getBuildTime(),
		GitCommit:       getGitCommit(),
		GoVersion:       getGoVersion(),
		Components:      make(map[string]string),
		LastUpdated:     time.Now(),
	}

	// Add component versions
	version.Components["translator"] = version.CodebaseVersion
	version.Components["api"] = "1.0.0"
	version.Components["distributed"] = "1.0.0"
	version.Components["deployment"] = "1.0.0"

	c.JSON(http.StatusOK, version)
}

// Helper functions for version information

// getCodebaseVersion returns the current codebase version
func getCodebaseVersion() string {
	// Try to read from version file first
	if version, err := readVersionFile("VERSION"); err == nil {
		return strings.TrimSpace(version)
	}

	// Try git describe
	if version, err := runCommand("git", "describe", "--tags", "--abbrev=0"); err == nil {
		return strings.TrimSpace(version)
	}

	// Try git rev-parse
	if commit, err := runCommand("git", "rev-parse", "--short", "HEAD"); err == nil {
		return fmt.Sprintf("dev-%s", strings.TrimSpace(commit))
	}

	return "unknown"
}

// getBuildTime returns the build timestamp
func getBuildTime() string {
	if buildTime, err := runCommand("date", "-u", "+%Y-%m-%dT%H:%M:%SZ"); err == nil {
		return strings.TrimSpace(buildTime)
	}
	return time.Now().UTC().Format(time.RFC3339)
}

// getGitCommit returns the current git commit hash
func getGitCommit() string {
	if commit, err := runCommand("git", "rev-parse", "HEAD"); err == nil {
		return strings.TrimSpace(commit)
	}
	return "unknown"
}

// getGoVersion returns the Go version used to build
func getGoVersion() string {
	if version, err := runCommand("go", "version"); err == nil {
		parts := strings.Split(version, " ")
		if len(parts) >= 3 {
			return parts[2]
		}
	}
	return "unknown"
}

// readVersionFile reads version from a file
func readVersionFile(filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// runCommand executes a shell command and returns its output
func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// getStats returns API statistics
func (h *Handler) getStats(c *gin.Context) {
	cacheStats := h.cache.Stats()

	c.JSON(http.StatusOK, gin.H{
		"cache": cacheStats,
		"websocket": gin.H{
			"connected_clients": h.wsHub.GetClientCount(),
		},
	})
}

// websocketHandler handles WebSocket connections
func (h *Handler) websocketHandler(c *gin.Context) {
	upgrader := gorillaws.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Configure properly in production
		},
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	sessionID := c.Query("session_id")
	client := &websocket.Client{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Conn:      conn,
		Send:      make(chan []byte, 256),
		Hub:       h.wsHub,
	}

	h.wsHub.Register(client)

	go client.WritePump()
	go client.ReadPump()
}

// Helper methods

func (h *Handler) createTranslator(providerName, model string) (translator.Translator, error) {
	if providerName == "" {
		providerName = h.config.Translation.DefaultProvider
	}

	// Handle distributed provider specially
	if providerName == "distributed" {
		if h.distributedManager == nil {
			return nil, fmt.Errorf("distributed translation not available")
		}
		// Return a special distributed translator wrapper
		return &distributedTranslator{dm: h.distributedManager.(*distributed.DistributedManager)}, nil
	}

	config := translator.TranslationConfig{
		SourceLang: "ru",
		TargetLang: "sr",
		Provider:   providerName,
		Model:      model,
		Options:    make(map[string]interface{}),
	}

	// Load provider config
	if providerCfg, ok := h.config.Translation.Providers[providerName]; ok {
		config.APIKey = providerCfg.APIKey
		config.BaseURL = providerCfg.BaseURL
		if model == "" {
			config.Model = providerCfg.Model
		}
		config.Options = providerCfg.Options
	}

	if providerName == "dictionary" {
		return dictionary.NewDictionaryTranslator(config), nil
	}

	return llm.NewLLMTranslator(config)
}

// distributedTranslator wraps the distributed manager to implement translator.Translator interface
type distributedTranslator struct {
	dm *distributed.DistributedManager
}

func (dt *distributedTranslator) Translate(ctx context.Context, text, contextHint string) (string, error) {
	return dt.dm.TranslateDistributed(ctx, text, contextHint)
}

func (dt *distributedTranslator) TranslateWithProgress(ctx context.Context, text, contextHint string, eventBus *events.EventBus, sessionID string) (string, error) {
	// For now, just call Translate - progress events could be added later
	return dt.Translate(ctx, text, contextHint)
}

func (dt *distributedTranslator) GetStats() translator.TranslationStats {
	// Return empty stats for now
	return translator.TranslationStats{}
}

func (dt *distributedTranslator) GetName() string {
	return "distributed"
}

func (h *Handler) translateBook(ctx context.Context, book *fb2.FictionBook, trans translator.Translator, sessionID string) error {
	// Translate title
	if book.Description.TitleInfo.BookTitle != "" {
		translated, err := trans.TranslateWithProgress(
			ctx,
			book.Description.TitleInfo.BookTitle,
			"Book title",
			h.eventBus,
			sessionID,
		)
		if err == nil {
			book.Description.TitleInfo.BookTitle = translated
		}
	}

	// Translate body sections
	for i := range book.Body {
		for j := range book.Body[i].Section {
			if err := h.translateSection(ctx, &book.Body[i].Section[j], trans, sessionID); err != nil {
				return err
			}
		}
	}

	return nil
}

func (h *Handler) translateSection(ctx context.Context, section *fb2.Section, trans translator.Translator, sessionID string) error {
	// Translate title
	for i := range section.Title.Paragraphs {
		if section.Title.Paragraphs[i].Text != "" {
			translated, _ := trans.TranslateWithProgress(
				ctx,
				section.Title.Paragraphs[i].Text,
				"Section title",
				h.eventBus,
				sessionID,
			)
			section.Title.Paragraphs[i].Text = translated
		}
	}

	// Translate paragraphs
	for i := range section.Paragraph {
		if section.Paragraph[i].Text != "" {
			translated, _ := trans.TranslateWithProgress(
				ctx,
				section.Paragraph[i].Text,
				"Paragraph",
				h.eventBus,
				sessionID,
			)
			section.Paragraph[i].Text = translated
		}
	}

	// Recursively translate subsections
	for i := range section.Section {
		if err := h.translateSection(ctx, &section.Section[i], trans, sessionID); err != nil {
			return err
		}
	}

	return nil
}

func (h *Handler) convertBookToLatin(book *fb2.FictionBook, converter *script.Converter) {
	book.Description.TitleInfo.BookTitle = converter.ToLatin(book.Description.TitleInfo.BookTitle)

	for i := range book.Body {
		for j := range book.Body[i].Section {
			h.convertSectionToLatin(&book.Body[i].Section[j], converter)
		}
	}
}

func (h *Handler) convertSectionToLatin(section *fb2.Section, converter *script.Converter) {
	for i := range section.Title.Paragraphs {
		section.Title.Paragraphs[i].Text = converter.ToLatin(section.Title.Paragraphs[i].Text)
	}

	for i := range section.Paragraph {
		section.Paragraph[i].Text = converter.ToLatin(section.Paragraph[i].Text)
	}

	for i := range section.Section {
		h.convertSectionToLatin(&section.Section[i], converter)
	}
}

func generateOutputFilename(inputFilename, provider string) string {
	ext := filepath.Ext(inputFilename)
	base := inputFilename[:len(inputFilename)-len(ext)]
	return fmt.Sprintf("%s_sr_%s%s", base, provider, ext)
}

// Authentication middleware
func (h *Handler) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "No authorization header"})
			c.Abort()
			return
		}

		// Extract token
		token := authHeader
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}

		// Validate token
		claims, err := h.authService.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("roles", claims.Roles)

		c.Next()
	}
}

// Authentication handlers
func (h *Handler) login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// In a real implementation, validate credentials against database
	// This is a placeholder
	userID := uuid.New().String()

	token, err := h.authService.GenerateToken(userID, req.Username, []string{"user"})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":    token,
		"user_id":  userID,
		"username": req.Username,
	})
}

func (h *Handler) generateToken(c *gin.Context) {
	var req struct {
		UserID   string   `json:"user_id" binding:"required"`
		Username string   `json:"username" binding:"required"`
		Roles    []string `json:"roles"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.authService.GenerateToken(req.UserID, req.Username, req.Roles)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
	})
}

func (h *Handler) getProfile(c *gin.Context) {
	userID := c.GetString("user_id")
	username := c.GetString("username")
	roles, _ := c.Get("roles")

	c.JSON(http.StatusOK, gin.H{
		"user_id":  userID,
		"username": username,
		"roles":    roles,
	})
}

// Distributed work handlers

func (h *Handler) getDistributedStatus(c *gin.Context) {
	if h.distributedManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Distributed work not available"})
		return
	}

	// Type assertion to access methods
	dm, ok := h.distributedManager.(*distributed.DistributedManager)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid distributed manager"})
		return
	}

	status := dm.GetStatus()
	c.JSON(http.StatusOK, status)
}

func (h *Handler) discoverWorkers(c *gin.Context) {
	if h.distributedManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Distributed work not available"})
		return
	}

	dm, ok := h.distributedManager.(*distributed.DistributedManager)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid distributed manager"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := dm.DiscoverAndPairWorkers(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Worker discovery completed"})
}

func (h *Handler) pairWorker(c *gin.Context) {
	if h.distributedManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Distributed work not available"})
		return
	}

	workerID := c.Param("worker_id")
	if workerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Worker ID is required"})
		return
	}

	dm, ok := h.distributedManager.(*distributed.DistributedManager)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid distributed manager"})
		return
	}

	if err := dm.PairWorker(workerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Successfully paired with worker %s", workerID)})
}

func (h *Handler) unpairWorker(c *gin.Context) {
	if h.distributedManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Distributed work not available"})
		return
	}

	workerID := c.Param("worker_id")
	if workerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Worker ID is required"})
		return
	}

	dm, ok := h.distributedManager.(*distributed.DistributedManager)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid distributed manager"})
		return
	}

	if err := dm.UnpairWorker(workerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Successfully unpaired from worker %s", workerID)})
}

func (h *Handler) translateDistributed(c *gin.Context) {
	if h.distributedManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Distributed work not available"})
		return
	}

	var req struct {
		Text        string `json:"text" binding:"required"`
		ContextHint string `json:"context_hint,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dm, ok := h.distributedManager.(*distributed.DistributedManager)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid distributed manager"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	sessionID := c.GetHeader("X-Session-ID")
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	translated, err := dm.TranslateDistributed(ctx, req.Text, req.ContextHint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"translated_text": translated,
		"session_id":      sessionID,
	})
}

// uploadUpdate handles update package uploads
func (h *Handler) uploadUpdate(c *gin.Context) {
	// Get the uploaded file
	file, err := c.FormFile("update_package")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No update package provided"})
		return
	}

	// Get version from header
	version := c.GetHeader("X-Update-Version")
	if version == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Update version not specified"})
		return
	}

	// Save the update package
	updateDir := "/tmp/translator-updates"
	if err := os.MkdirAll(updateDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create update directory"})
		return
	}

	updatePath := filepath.Join(updateDir, fmt.Sprintf("update-%s.tar.gz", version))
	if err := c.SaveUploadedFile(file, updatePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save update package"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Update package uploaded successfully",
		"version": version,
		"path":    updatePath,
	})
}

// applyUpdate applies a previously uploaded update
func (h *Handler) applyUpdate(c *gin.Context) {
	// Get version from header
	version := c.GetHeader("X-Update-Version")
	if version == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Update version not specified"})
		return
	}

	// For security, this should be a very controlled process
	// In a real implementation, you'd want extensive validation

	updatePath := filepath.Join("/tmp/translator-updates", fmt.Sprintf("update-%s.tar.gz", version))

	// Check if update package exists
	if _, err := os.Stat(updatePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Update package not found"})
		return
	}

	// Extract and apply the update
	// This is a simplified version - in production you'd want rollback capabilities
	if err := applyUpdatePackage(updatePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to apply update: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Update applied successfully",
		"version": version,
	})
}

// applyUpdatePackage extracts and applies an update package
func applyUpdatePackage(updatePath string) error {
	// Create backup of current binary
	backupPath := "/tmp/translator-server.backup"
	if _, err := runCommand("cp", "/usr/local/bin/translator-server", backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %v", err)
	}

	// Extract update package
	extractDir := "/tmp/translator-update-extract"
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return fmt.Errorf("failed to create extract directory: %v", err)
	}

	if _, err := runCommand("tar", "-xzf", updatePath, "-C", extractDir); err != nil {
		return fmt.Errorf("failed to extract update package: %v", err)
	}

	// Find and install new binary
	newBinary := filepath.Join(extractDir, "translator-server")
	if _, err := os.Stat(newBinary); os.IsNotExist(err) {
		return fmt.Errorf("new binary not found in update package")
	}

	// Install new binary
	if _, err := runCommand("cp", newBinary, "/usr/local/bin/translator-server"); err != nil {
		return fmt.Errorf("failed to install new binary: %v", err)
	}

	// Make sure it's executable
	if _, err := runCommand("chmod", "+x", "/usr/local/bin/translator-server"); err != nil {
		return fmt.Errorf("failed to make binary executable: %v", err)
	}

	// Clean up
	os.RemoveAll(extractDir)

	return nil
}

// rollbackUpdate handles manual rollback requests
func (h *Handler) rollbackUpdate(c *gin.Context) {
	if h.distributedManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Distributed work not available"})
		return
	}

	dm, ok := h.distributedManager.(*distributed.DistributedManager)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid distributed manager"})
		return
	}

	// Get worker ID from header or query param
	workerID := c.GetHeader("X-Worker-ID")
	if workerID == "" {
		workerID = c.Query("worker_id")
	}
	if workerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Worker ID is required"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Get the worker service
	worker := dm.GetWorkerByID(workerID)
	if worker == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Worker not found"})
		return
	}

	// Perform rollback
	if err := dm.RollbackWorker(ctx, worker); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Worker rollback completed successfully",
		"worker_id": workerID,
	})
}

// Version Management Monitoring Handlers

// getVersionMetrics returns comprehensive version management metrics
func (h *Handler) getVersionMetrics(c *gin.Context) {
	if h.distributedManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Distributed work not available"})
		return
	}

	dm, ok := h.distributedManager.(*distributed.DistributedManager)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid distributed manager"})
		return
	}

	metrics := dm.GetVersionMetrics()
	c.JSON(http.StatusOK, metrics)
}

// getVersionAlerts returns current version drift alerts
func (h *Handler) getVersionAlerts(c *gin.Context) {
	if h.distributedManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Distributed work not available"})
		return
	}

	dm, ok := h.distributedManager.(*distributed.DistributedManager)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid distributed manager"})
		return
	}

	alerts := dm.GetVersionAlerts()

	// Filter alerts by severity if requested
	severity := c.Query("severity")
	if severity != "" {
		filtered := make([]*distributed.DriftAlert, 0)
		for _, alert := range alerts {
			if alert.Severity == severity {
				filtered = append(filtered, alert)
			}
		}
		alerts = filtered
	}

	c.JSON(http.StatusOK, gin.H{
		"alerts": alerts,
		"count":  len(alerts),
	})
}

// getVersionHealth returns overall version management health status
func (h *Handler) getVersionHealth(c *gin.Context) {
	if h.distributedManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Distributed work not available"})
		return
	}

	dm, ok := h.distributedManager.(*distributed.DistributedManager)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid distributed manager"})
		return
	}

	health := dm.GetVersionHealth()
	c.JSON(http.StatusOK, health)
}

// getVersionDashboard returns dashboard data for version management visualization
func (h *Handler) getVersionDashboard(c *gin.Context) {
	if h.distributedManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Distributed work not available"})
		return
	}

	dm, ok := h.distributedManager.(*distributed.DistributedManager)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid distributed manager"})
		return
	}

	// Get all dashboard data
	metrics := dm.GetVersionMetrics()
	alerts := dm.GetVersionAlerts()
	health := dm.GetVersionHealth()
	status := dm.GetStatus()

	// Get worker version details
	workers := make([]gin.H, 0)
	if pairedServices := dm.GetPairedServices(); pairedServices != nil {
		for workerID, service := range pairedServices {
			workers = append(workers, gin.H{
				"worker_id":      workerID,
				"host":           service.Host,
				"port":           service.Port,
				"protocol":       service.Protocol,
				"status":         service.Status,
				"version":        service.Version.CodebaseVersion,
				"last_seen":      service.LastSeen,
				"last_updated":   service.Version.LastUpdated,
				"drift_duration": time.Since(service.Version.LastUpdated),
			})
		}
	}

	// Calculate summary statistics
	totalWorkers := len(workers)
	upToDateWorkers := 0
	outdatedWorkers := 0
	unhealthyWorkers := 0

	for _, worker := range workers {
		status := worker["status"].(string)
		switch status {
		case "paired":
			upToDateWorkers++
		case "outdated":
			outdatedWorkers++
		default:
			unhealthyWorkers++
		}
	}

	dashboard := gin.H{
		"summary": gin.H{
			"total_workers":      totalWorkers,
			"up_to_date_workers": upToDateWorkers,
			"outdated_workers":   outdatedWorkers,
			"unhealthy_workers":  unhealthyWorkers,
			"active_alerts":      len(alerts),
			"health_score":       health["health_score"],
			"last_drift_check":   metrics.LastDriftCheck,
		},
		"metrics":   metrics,
		"alerts":    alerts,
		"health":    health,
		"workers":   workers,
		"status":    status,
		"timestamp": time.Now().UTC(),
	}

	c.JSON(http.StatusOK, dashboard)
}

// triggerVersionDriftCheck manually triggers a version drift check
func (h *Handler) triggerVersionDriftCheck(c *gin.Context) {
	if h.distributedManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Distributed work not available"})
		return
	}

	dm, ok := h.distributedManager.(*distributed.DistributedManager)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid distributed manager"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	alerts := dm.CheckVersionDrift(ctx)

	c.JSON(http.StatusOK, gin.H{
		"message":          "Version drift check completed",
		"alerts_generated": len(alerts),
		"alerts":           alerts,
	})
}

// getAlertHistory returns alert history
func (h *Handler) getAlertHistory(c *gin.Context) {
	if h.distributedManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Distributed work not available"})
		return
	}

	dm, ok := h.distributedManager.(*distributed.DistributedManager)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid distributed manager"})
		return
	}

	limitStr := c.Query("limit")
	limit := 50 // default limit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	alerts := dm.GetAlertHistory(limit)

	c.JSON(http.StatusOK, gin.H{
		"alerts": alerts,
		"count":  len(alerts),
		"limit":  limit,
	})
}

// acknowledgeAlert marks an alert as acknowledged
func (h *Handler) acknowledgeAlert(c *gin.Context) {
	if h.distributedManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Distributed work not available"})
		return
	}

	alertID := c.Param("alert_id")
	if alertID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Alert ID is required"})
		return
	}

	var req struct {
		AcknowledgedBy string `json:"acknowledged_by" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dm, ok := h.distributedManager.(*distributed.DistributedManager)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid distributed manager"})
		return
	}

	if acknowledged := dm.AcknowledgeAlert(alertID, req.AcknowledgedBy); !acknowledged {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found or already acknowledged"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Alert acknowledged successfully",
		"alert_id": alertID,
	})
}

// addEmailAlertChannel adds an email alert channel
func (h *Handler) addEmailAlertChannel(c *gin.Context) {
	if h.distributedManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Distributed work not available"})
		return
	}

	var req struct {
		SMTPHost    string   `json:"smtp_host" binding:"required"`
		SMTPPort    int      `json:"smtp_port" binding:"required"`
		Username    string   `json:"username" binding:"required"`
		Password    string   `json:"password" binding:"required"`
		FromAddress string   `json:"from_address" binding:"required"`
		ToAddresses []string `json:"to_addresses" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	channel := &distributed.EmailAlertChannel{
		SMTPHost:    req.SMTPHost,
		SMTPPort:    req.SMTPPort,
		Username:    req.Username,
		Password:    req.Password,
		FromAddress: req.FromAddress,
		ToAddresses: req.ToAddresses,
	}

	dm, ok := h.distributedManager.(*distributed.DistributedManager)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid distributed manager"})
		return
	}

	dm.AddAlertChannel(channel)

	c.JSON(http.StatusOK, gin.H{
		"message":      "Email alert channel added successfully",
		"channel_type": "email",
		"recipients":   len(req.ToAddresses),
	})
}

// addWebhookAlertChannel adds a webhook alert channel
func (h *Handler) addWebhookAlertChannel(c *gin.Context) {
	if h.distributedManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Distributed work not available"})
		return
	}

	var req struct {
		URL     string            `json:"url" binding:"required"`
		Method  string            `json:"method"`
		Headers map[string]string `json:"headers"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Method == "" {
		req.Method = "POST"
	}

	channel := &distributed.WebhookAlertChannel{
		URL:     req.URL,
		Method:  req.Method,
		Headers: req.Headers,
	}

	dm, ok := h.distributedManager.(*distributed.DistributedManager)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid distributed manager"})
		return
	}

	dm.AddAlertChannel(channel)

	c.JSON(http.StatusOK, gin.H{
		"message":      "Webhook alert channel added successfully",
		"channel_type": "webhook",
		"url":          req.URL,
		"method":       req.Method,
	})
}

// addSlackAlertChannel adds a Slack alert channel
func (h *Handler) addSlackAlertChannel(c *gin.Context) {
	if h.distributedManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Distributed work not available"})
		return
	}

	var req struct {
		WebhookURL string `json:"webhook_url" binding:"required"`
		Channel    string `json:"channel"`
		Username   string `json:"username"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Username == "" {
		req.Username = "Version Monitor"
	}

	channel := &distributed.SlackAlertChannel{
		WebhookURL: req.WebhookURL,
		Channel:    req.Channel,
		Username:   req.Username,
	}

	dm, ok := h.distributedManager.(*distributed.DistributedManager)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid distributed manager"})
		return
	}

	dm.AddAlertChannel(channel)

	c.JSON(http.StatusOK, gin.H{
		"message":      "Slack alert channel added successfully",
		"channel_type": "slack",
		"channel":      req.Channel,
		"username":     req.Username,
	})
}

// serveDashboard serves the HTML dashboard
func (h *Handler) serveDashboard(c *gin.Context) {
	dashboardPath := "pkg/api/dashboard.html"

	// Read the dashboard HTML file
	htmlContent, err := os.ReadFile(dashboardPath)
	if err != nil {
		// Fallback: serve embedded dashboard if file not found
		htmlContent = []byte(h.getEmbeddedDashboardHTML())
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, string(htmlContent))
}

// getEmbeddedDashboardHTML returns the embedded dashboard HTML
func (h *Handler) getEmbeddedDashboardHTML() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Version Management Dashboard</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; }
        .header { background: #007bff; color: white; padding: 20px; border-radius: 8px; text-align: center; margin-bottom: 20px; }
        .stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; margin-bottom: 20px; }
        .stat-card { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); text-align: center; }
        .stat-value { font-size: 2em; font-weight: bold; }
        .stat-label { color: #666; margin-top: 5px; }
        .alert { background: #fff3cd; border: 1px solid #ffeaa7; color: #856404; padding: 15px; border-radius: 4px; margin: 10px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔄 Version Management Dashboard</h1>
            <p>Dashboard file not found. This is a fallback version.</p>
        </div>

        <div class="alert">
            <strong>Note:</strong> The full dashboard HTML file was not found. Please ensure the dashboard.html file is properly deployed.
        </div>

        <div class="stats">
            <div class="stat-card">
                <div class="stat-value" id="total-workers">-</div>
                <div class="stat-label">Total Workers</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="up-to-date">-</div>
                <div class="stat-label">Up to Date</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="outdated">-</div>
                <div class="stat-label">Outdated</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="health-score">-</div>
                <div class="stat-label">Health Score</div>
            </div>
        </div>

        <button onclick="loadData()" style="padding: 10px 20px; background: #007bff; color: white; border: none; border-radius: 4px; cursor: pointer;">Load Data</button>

        <div id="data" style="margin-top: 20px; background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1);"></div>
    </div>

    <script>
        async function loadData() {
            try {
                const response = await fetch('/api/v1/monitoring/version/dashboard');
                const data = await response.json();

                document.getElementById('total-workers').textContent = data.summary.total_workers;
                document.getElementById('up-to-date').textContent = data.summary.up_to_date_workers;
                document.getElementById('outdated').textContent = data.summary.outdated_workers;
                document.getElementById('health-score').textContent = Math.round(data.summary.health_score);

                document.getElementById('data').innerHTML = '<pre>' + JSON.stringify(data, null, 2) + '</pre>';
            } catch (error) {
                document.getElementById('data').innerHTML = '<p style="color: red;">Error loading data: ' + error.message + '</p>';
            }
        }
    </script>
</body>
</html>`
}
