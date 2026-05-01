package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"digital.vasic.translator/internal/verifier"
	"digital.vasic.translator/pkg/api"
	"digital.vasic.translator/pkg/grpc/proto"
	"digital.vasic.translator/pkg/logger"
	"digital.vasic.translator/pkg/websocket"

	"github.com/gin-gonic/gin"
	gorillaws "github.com/gorilla/websocket"
)

// APIServer represents the REST API server
type APIServer struct {
	// gRPC client
	grpcClient proto.TranslationServiceClient
	conn       *grpc.ClientConn

	// WebSocket hub
	wsHub *websocket.Hub

	// HTTP server
	server *http.Server
	router *gin.Engine

	// Configuration
	config *APIConfig

	// Logger
	logger logger.Logger

	// Verifier integration (CONST-034 SSOT)
	verifierHandler *api.VerifierHandler
}

// APIConfig holds API server configuration
type APIConfig struct {
	GRPCAddress   string
	HTTPAddress   string
	HTTPPort      int
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
	IdleTimeout   time.Duration
	EnableMetrics bool
	Debug         bool
}

// TranslationRequest represents a translation request from REST API
type TranslationRequest struct {
	SessionID      string                 `json:"session_id" binding:"required"`
	InputFile      string                 `json:"input_file" binding:"required"`
	OutputFile     string                 `json:"output_file"`
	SourceLang     string                 `json:"source_lang"`
	TargetLang     string                 `json:"target_lang"`
	Script         string                 `json:"script"`
	ProviderConfig ProviderRequestConfig  `json:"provider_config" binding:"required"`
	Options        map[string]interface{} `json:"options"`
}

// ProviderRequestConfig represents provider configuration from REST API
type ProviderRequestConfig struct {
	Type        string            `json:"type" binding:"required"`
	Model       string            `json:"model"`
	Temperature float64           `json:"temperature"`
	MaxTokens   int               `json:"max_tokens"`
	TimeoutSec  int               `json:"timeout_seconds"`
	APIKey      string            `json:"api_key"`
	BaseURL     string            `json:"base_url"`
	SSHHost     string            `json:"ssh_host"`
	SSHUser     string            `json:"ssh_user"`
	SSHPassword string            `json:"ssh_password"`
	SSHPort     int               `json:"ssh_port"`
	RemoteDir   string            `json:"remote_dir"`
	LlamaBinary string            `json:"llama_binary"`
	LlamaModel  string            `json:"llama_model"`
	ContextSize int               `json:"context_size"`
	Options     map[string]string `json:"options"`
}

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type      string                 `json:"type"`
	SessionID string                 `json:"session_id,omitempty"`
	Data      map[string]interface{} `json:"data"`
	Timestamp int64                  `json:"timestamp"`
}

// APIResponse represents a standard API response
type APIResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Error     string `json:"error"`
	ErrorCode int    `json:"error_code,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

func main() {
	// Load configuration
	config := loadAPIConfig()

	// Initialize logger
	logLevel := logger.INFO
	if config.Debug {
		logLevel = logger.DEBUG
	}

	logger := logger.NewLogger(logger.LoggerConfig{
		Level:  logLevel,
		Format: logger.FORMAT_JSON,
	})

	// Create API server
	server, err := NewAPIServer(config, logger)
	if err != nil {
		logger.Fatal("Failed to create API server", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Start server
	if err := server.Start(); err != nil {
		logger.Fatal("Failed to start API server", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

// NewAPIServer creates a new API server
func NewAPIServer(config *APIConfig, logger logger.Logger) (*APIServer, error) {
	// Connect to gRPC server
	conn, err := grpc.NewClient(config.GRPCAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	grpcClient := proto.NewTranslationServiceClient(conn)

	// Create WebSocket hub
	wsHub := websocket.NewHub(nil)

	// Create Gin router
	if config.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// Create verifier handler with default config (CONST-034 SSOT)
	verifierCfg := &verifier.Config{
		APIURL:   "http://localhost:8080",
		APIKey:   os.Getenv("LLMSVERIFIER_API_KEY"),
		CacheTTL: time.Hour,
	}
	verifierHandler := api.NewVerifierHandler(verifierCfg)

	// Create API server
	apiServer := &APIServer{
		grpcClient:      grpcClient,
		conn:            conn,
		wsHub:           wsHub,
		router:          router,
		config:          config,
		logger:          logger,
		verifierHandler: verifierHandler,
	}

	// Setup routes
	apiServer.setupRoutes()

	// Create HTTP server
	apiServer.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.HTTPAddress, config.HTTPPort),
		Handler:      router,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
	}

	// Start WebSocket hub
	go wsHub.Run()

	return apiServer, nil
}

// Start starts the API server
func (s *APIServer) Start() error {
	s.logger.Info("Starting API server", map[string]interface{}{
		"http_address": s.config.HTTPAddress,
		"http_port":    s.config.HTTPPort,
		"grpc_address": s.config.GRPCAddress,
	})

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		return err
	case <-quit:
		s.logger.Info("Shutting down API server...", nil)
		return s.Shutdown()
	}
}

// Shutdown gracefully shuts down the API server
func (s *APIServer) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := s.server.Shutdown(ctx); err != nil {
		s.logger.Error("Error during server shutdown", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Close gRPC connection
	if err := s.conn.Close(); err != nil {
		s.logger.Error("Error closing gRPC connection", map[string]interface{}{
			"error": err.Error(),
		})
	}

	s.logger.Info("API server shutdown complete", nil)
	return nil
}

// setupRoutes configures all API routes
func (s *APIServer) setupRoutes() {
	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Translation endpoints
		v1.POST("/translations", s.startTranslation)
		v1.GET("/translations/:session_id", s.getTranslationStatus)
		v1.GET("/translations", s.listTranslations)
		v1.DELETE("/translations/:session_id", s.cancelTranslation)
		v1.GET("/translations/:session_id/stream", s.streamTranslationProgress)

		// Provider endpoints
		v1.GET("/providers", s.getProviders)

		// System endpoints
		v1.GET("/health", s.healthCheck)
		v1.GET("/metrics", s.getMetrics)

		// Verifier endpoints (CONST-034 SSOT)
		if s.verifierHandler != nil {
			s.verifierHandler.RegisterVerifierRoutes(v1)
		}
	}

	// WebSocket endpoint
	s.router.GET("/ws", s.handleWebSocket)

	// Serve static files
	s.router.Static("/static", "./web/static")
	s.router.LoadHTMLGlob("web/templates/*")

	// Serve dashboard
	s.router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "dashboard.html", gin.H{
			"title": "Translation Dashboard",
		})
	})
}

// API Handlers

func (s *APIServer) startTranslation(c *gin.Context) {
	var req TranslationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.sendErrorResponse(c, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	// Convert to gRPC request
	grpcReq := s.convertToGRPCRequest(req)

	// Call gRPC service
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	resp, err := s.grpcClient.StartTranslation(ctx, grpcReq)
	if err != nil {
		s.logger.Error("Failed to start translation", map[string]interface{}{
			"session_id": req.SessionID,
			"error":      err.Error(),
		})
		s.sendErrorResponse(c, http.StatusInternalServerError, "Failed to start translation", err.Error())
		return
	}

	s.sendSuccessResponse(c, resp)
}

func (s *APIServer) getTranslationStatus(c *gin.Context) {
	sessionID := c.Param("session_id")

	req := &proto.TranslationStatusRequest{
		SessionId: sessionID,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	resp, err := s.grpcClient.GetTranslationStatus(ctx, req)
	if err != nil {
		s.logger.Error("Failed to get translation status", map[string]interface{}{
			"session_id": sessionID,
			"error":      err.Error(),
		})
		s.sendErrorResponse(c, http.StatusInternalServerError, "Failed to get translation status", err.Error())
		return
	}

	s.sendSuccessResponse(c, resp)
}

func (s *APIServer) listTranslations(c *gin.Context) {
	req := &emptypb.Empty{}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	resp, err := s.grpcClient.ListTranslations(ctx, req)
	if err != nil {
		s.logger.Error("Failed to list translations", map[string]interface{}{
			"error": err.Error(),
		})
		s.sendErrorResponse(c, http.StatusInternalServerError, "Failed to list translations", err.Error())
		return
	}

	s.sendSuccessResponse(c, resp)
}

func (s *APIServer) cancelTranslation(c *gin.Context) {
	sessionID := c.Param("session_id")

	var reason struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&reason)

	req := &proto.CancelTranslationRequest{
		SessionId: sessionID,
		Reason:    reason.Reason,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	resp, err := s.grpcClient.CancelTranslation(ctx, req)
	if err != nil {
		s.logger.Error("Failed to cancel translation", map[string]interface{}{
			"session_id": sessionID,
			"error":      err.Error(),
		})
		s.sendErrorResponse(c, http.StatusInternalServerError, "Failed to cancel translation", err.Error())
		return
	}

	s.sendSuccessResponse(c, resp)
}

func (s *APIServer) streamTranslationProgress(c *gin.Context) {
	sessionID := c.Param("session_id")
	clientID := c.Query("client_id")
	if clientID == "" {
		clientID = fmt.Sprintf("web-%d", time.Now().UnixNano())
	}

	// Upgrade to WebSocket
	upgrader := gorillaws.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for development
		},
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		s.logger.Error("Failed to upgrade to WebSocket", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	defer conn.Close()

	// Create gRPC stream request
	req := &proto.TranslationStreamRequest{
		SessionId: sessionID,
		ClientId:  clientID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Hour) // Long streaming timeout
	defer cancel()

	// Create gRPC stream
	stream, err := s.grpcClient.StreamTranslationProgress(ctx, req)
	if err != nil {
		s.logger.Error("Failed to create gRPC stream", map[string]interface{}{
			"session_id": sessionID,
			"error":      err.Error(),
		})
		return
	}

	// Forward events to WebSocket
	for {
		event, err := stream.Recv()
		if err != nil {
			s.logger.Debug("Stream ended", map[string]interface{}{
				"session_id": sessionID,
				"error":      err.Error(),
			})
			break
		}

		// Convert to WebSocket message
		wsMessage := WebSocketMessage{
			Type:      event.EventType,
			SessionID: event.SessionId,
			Data: map[string]interface{}{
				"progress_percentage": event.ProgressPercentage,
				"current_step":        event.StepName,
				"message":             event.Message,
				"metadata":            event.Metadata,
				"current_item":        event.CurrentItem,
				"total_items":         event.TotalItems,
				"current_operation":   event.CurrentOperation,
			},
			Timestamp: time.Now().Unix(),
		}

		if err := conn.WriteJSON(wsMessage); err != nil {
			s.logger.Debug("Failed to send WebSocket message", map[string]interface{}{
				"error": err.Error(),
			})
			break
		}
	}
}

func (s *APIServer) getProviders(c *gin.Context) {
	req := &emptypb.Empty{}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	resp, err := s.grpcClient.GetProviders(ctx, req)
	if err != nil {
		s.logger.Error("Failed to get providers", map[string]interface{}{
			"error": err.Error(),
		})
		s.sendErrorResponse(c, http.StatusInternalServerError, "Failed to get providers", err.Error())
		return
	}

	s.sendSuccessResponse(c, resp)
}

func (s *APIServer) healthCheck(c *gin.Context) {
	s.sendSuccessResponse(c, map[string]interface{}{
		"status":         "healthy",
		"timestamp":      time.Now().Unix(),
		"version":        "3.0.0",
		"grpc_connected": s.conn.GetState().String(),
	})
}

func (s *APIServer) getMetrics(c *gin.Context) {
	if !s.config.EnableMetrics {
		s.sendErrorResponse(c, http.StatusNotFound, "Metrics not enabled", "")
		return
	}

	// Basic metrics - this could be enhanced with proper metrics collection
	metrics := map[string]interface{}{
		"active_sessions":    0, // This would be tracked
		"total_translations": 0, // This would be tracked
		"success_rate":       0.0,
		"average_duration":   0,
		"grpc_connection":    s.conn.GetState().String(),
		"uptime_seconds":     0, // This would be tracked
		"timestamp":          time.Now().Unix(),
	}

	s.sendSuccessResponse(c, metrics)
}

func (s *APIServer) handleWebSocket(c *gin.Context) {
	// Upgrade to WebSocket
	upgrader := gorillaws.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for development
		},
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		s.logger.Error("Failed to upgrade to WebSocket", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// Get session ID from query
	sessionID := c.Query("session_id")
	if sessionID == "" {
		conn.WriteJSON(WebSocketMessage{
			Type:      "error",
			Data:      map[string]interface{}{"error": "session_id required"},
			Timestamp: time.Now().Unix(),
		})
		conn.Close()
		return
	}

	clientID := c.Query("client_id")
	if clientID == "" {
		clientID = fmt.Sprintf("ws-%d", time.Now().UnixNano())
	}

	// Create WebSocket client
	client := &websocket.Client{
		ID:        clientID,
		SessionID: sessionID,
		Conn:      conn,
		Send:      make(chan []byte, 256),
		Hub:       s.wsHub,
	}

	// Register client
	s.wsHub.Register(client)

	// Start client routines
	go client.WritePump()
	go client.ReadPump()
}

// Helper methods

func (s *APIServer) convertToGRPCRequest(req TranslationRequest) *proto.TranslationRequest {
	return &proto.TranslationRequest{
		SessionId:  req.SessionID,
		InputFile:  req.InputFile,
		OutputFile: req.OutputFile,
		SourceLang: req.SourceLang,
		TargetLang: req.TargetLang,
		Script:     req.Script,
		ProviderConfig: &proto.ProviderConfig{
			Type:              req.ProviderConfig.Type,
			Model:             req.ProviderConfig.Model,
			Temperature:       req.ProviderConfig.Temperature,
			MaxTokens:         int32(req.ProviderConfig.MaxTokens),
			TimeoutSeconds:    int32(req.ProviderConfig.TimeoutSec),
			ApiKey:            req.ProviderConfig.APIKey,
			BaseUrl:           req.ProviderConfig.BaseURL,
			SshHost:           req.ProviderConfig.SSHHost,
			SshUser:           req.ProviderConfig.SSHUser,
			SshPassword:       req.ProviderConfig.SSHPassword,
			SshPort:           int32(req.ProviderConfig.SSHPort),
			RemoteDir:         req.ProviderConfig.RemoteDir,
			LlamaBinary:       req.ProviderConfig.LlamaBinary,
			LlamaModel:        req.ProviderConfig.LlamaModel,
			ContextSize:       int32(req.ProviderConfig.ContextSize),
			AdditionalOptions: req.ProviderConfig.Options,
		},
		Options: &proto.TranslationOptions{
			Workers:          1, // Default values
			ChunkSize:        2000,
			Concurrency:      4,
			VerifyOutput:     true,
			Verbose:          false,
			EnableMonitoring: true,
		},
		CreatedAt: timestamppb.New(time.Now()),
		ClientId:  "rest-api",
	}
}

func (s *APIServer) sendSuccessResponse(c *gin.Context, data interface{}) {
	response := APIResponse{
		Success:   true,
		Message:   "Success",
		Data:      data,
		Timestamp: time.Now().Unix(),
	}
	c.JSON(http.StatusOK, response)
}

func (s *APIServer) sendErrorResponse(c *gin.Context, statusCode int, message, error string) {
	response := ErrorResponse{
		Success:   false,
		Message:   message,
		Error:     error,
		Timestamp: time.Now().Unix(),
	}
	c.JSON(statusCode, response)
}

func loadAPIConfig() *APIConfig {
	config := &APIConfig{
		GRPCAddress:   getEnv("GRPC_ADDRESS", "localhost:50051"),
		HTTPAddress:   getEnv("HTTP_ADDRESS", "0.0.0.0"),
		HTTPPort:      getEnvInt("HTTP_PORT", 8080),
		ReadTimeout:   getEnvDuration("HTTP_READ_TIMEOUT", 30*time.Second),
		WriteTimeout:  getEnvDuration("HTTP_WRITE_TIMEOUT", 30*time.Second),
		IdleTimeout:   getEnvDuration("HTTP_IDLE_TIMEOUT", 120*time.Second),
		EnableMetrics: getEnvBool("ENABLE_METRICS", true),
		Debug:         getEnvBool("DEBUG", false),
	}

	return config
}

// Environment variable helpers
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
