package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"google.golang.org/grpc/reflection"

	"digital.vasic.translator/pkg/events"
	translatorgrpc "digital.vasic.translator/pkg/grpc"
	"digital.vasic.translator/pkg/grpc/proto"
	"digital.vasic.translator/pkg/logger"
)

const (
	appVersion = "3.0.0"
)

// ServerConfig holds configuration for the gRPC server
type ServerConfig struct {
	Address          string
	Port             int
	MaxConnections   int
	EnableReflection bool
	EnableMetrics    bool
	LogLevel         string
}

func main() {
	// Parse command line flags
	config := parseFlags()

	// Initialize logger
	logLevel := parseLogLevel(config.LogLevel)
	log := logger.NewLogger(logger.LoggerConfig{
		Level:  logLevel,
		Format: logger.FORMAT_JSON,
	})

	log.Info("Starting gRPC Translation Server", map[string]interface{}{
		"version": appVersion,
		"address": fmt.Sprintf("%s:%d", config.Address, config.Port),
	})

	// Initialize event bus
	eventBus := events.NewEventBus()

	// Initialize core translator
	coreTranslator := translatorgrpc.NewCoreTranslator(log)

	// Initialize server configuration
	serverConfig := &translatorgrpc.ServerConfig{
		MaxConcurrentTranslations: 50,
		SessionTimeout:            24 * time.Hour,
		StreamBufferSize:          1000,
		EnableMetrics:             config.EnableMetrics,
	}

	// Create gRPC server
	grpcServer := translatorgrpc.NewServer(eventBus, log, coreTranslator, serverConfig)

	// Create listener
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", config.Address, config.Port))
	if err != nil {
		log.Fatal("Failed to listen", map[string]interface{}{
			"address": fmt.Sprintf("%s:%d", config.Address, config.Port),
			"error":   err.Error(),
		})
	}

	// Register reflection if enabled
	if config.EnableReflection {
		reflection.Register(grpcServer.GetGRPCServer())
		log.Info("gRPC reflection enabled", nil)
	}

	// Register translation service
	proto.RegisterTranslationServiceServer(grpcServer.GetGRPCServer(), grpcServer)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		log.Info("gRPC server starting", map[string]interface{}{
			"address": lis.Addr().String(),
		})

		if err := grpcServer.GetGRPCServer().Serve(lis); err != nil {
			errChan <- err
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		log.Fatal("Server failed", map[string]interface{}{
			"error": err.Error(),
		})
	case <-quit:
		log.Info("Shutting down gRPC server...", nil)
		grpcServer.Shutdown()
		log.Info("gRPC server shutdown complete", nil)
	}
}

// parseFlags parses command line flags
func parseFlags() *ServerConfig {
	config := &ServerConfig{}

	flag.StringVar(&config.Address, "address", "0.0.0.0", "Server address")
	flag.IntVar(&config.Port, "port", 50051, "Server port")
	flag.IntVar(&config.MaxConnections, "max-connections", 1000, "Maximum concurrent connections")
	flag.BoolVar(&config.EnableReflection, "reflection", true, "Enable gRPC reflection")
	flag.BoolVar(&config.EnableMetrics, "metrics", true, "Enable metrics collection")
	flag.StringVar(&config.LogLevel, "log-level", "info", "Log level: debug, info, warn, error")

	versionFlag := flag.Bool("version", false, "Show version information")
	help := flag.Bool("help", false, "Show help information")

	flag.Parse()

	// Apply environment-variable overrides as documented in printHelp().
	applyEnvOverrides(config, os.Getenv)

	if *versionFlag {
		fmt.Printf("gRPC Translation Server v%s\n", appVersion)
		os.Exit(0)
	}

	if *help {
		printHelp()
		os.Exit(0)
	}

	return config
}

// applyEnvOverrides applies the environment-variable overrides documented in
// printHelp() (GRPC_ADDRESS, GRPC_PORT, LOG_LEVEL, ENABLE_METRICS,
// ENABLE_REFLECTION). When a variable is set and well-formed it takes
// precedence over the corresponding command-line flag, as the help text
// promises. getenv is injected to keep this unit-testable.
func applyEnvOverrides(config *ServerConfig, getenv func(string) string) {
	if v := getenv("GRPC_ADDRESS"); v != "" {
		config.Address = v
	}
	if v := getenv("GRPC_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			config.Port = p
		}
	}
	if v := getenv("LOG_LEVEL"); v != "" {
		config.LogLevel = v
	}
	if v := getenv("ENABLE_METRICS"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			config.EnableMetrics = b
		}
	}
	if v := getenv("ENABLE_REFLECTION"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			config.EnableReflection = b
		}
	}
}

// printHelp displays usage information
func printHelp() {
	fmt.Printf(`gRPC Translation Server v%s

Usage:
  grpc-server [options]

Options:
  -address <addr>          Server address (default: 0.0.0.0)
  -port <port>              Server port (default: 50051)
  -max-connections <num>    Maximum concurrent connections (default: 1000)
  -reflection               Enable gRPC reflection (default: true)
  -metrics                  Enable metrics collection (default: true)
  -log-level <level>        Log level: debug, info, warn, error (default: info)
  -version                  Show version information
  -help                     Show this help

Examples:
  grpc-server -port 50051 -address 127.0.0.1
  grpc-server -log-level debug -metrics false
  grpc-server -reflection -max-connections 500

Features:
  - Multi-provider translation support (OpenAI, Anthropic, SSH, etc.)
  - Event-driven architecture with real-time progress
  - Streaming translation progress
  - Session management and cancellation
  - Provider registry and status monitoring
  - WebSocket support for web dashboards

Environment Variables:
  GRPC_ADDRESS     Server address (overrides -address)
  GRPC_PORT        Server port (overrides -port)
  LOG_LEVEL        Log level (overrides -log-level)
  ENABLE_METRICS   Enable metrics (overrides -metrics)
  ENABLE_REFLECTION Enable reflection (overrides -reflection)

Services:
  TranslationService:
    - StartTranslation: Start new translation job
    - GetTranslationStatus: Get translation status
    - ListTranslations: List all sessions
    - CancelTranslation: Cancel translation
    - StreamTranslationProgress: Stream progress events
    - GetProviders: Get available providers
    - SubscribeEvents: Subscribe to system events

Monitoring:
  - Health check: Available through service calls
  - Metrics: Available if enabled
  - Event streaming: Real-time progress and system events
  - Provider status: Available through GetProviders API

Configuration:
  - Server runs with default configuration for development
  - Production deployment should set appropriate limits
  - TLS/SSL can be configured through gRPC server options
  - Authentication and authorization can be added through interceptors

For more information, see the project documentation.
`, appVersion)
}

// parseLogLevel converts string to logger level string
func parseLogLevel(level string) string {
	switch level {
	case "debug":
		return logger.DEBUG
	case "info":
		return logger.INFO
	case "warn":
		return logger.WARN
	case "error":
		return logger.ERROR
	default:
		return logger.INFO
	}
}
