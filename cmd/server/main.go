package main

import (
	"context"
	"crypto/tls"
	"digital.vasic.translator/internal/cache"
	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/pkg/api"
	"digital.vasic.translator/pkg/coordination"
	"digital.vasic.translator/pkg/deployment"
	"digital.vasic.translator/pkg/distributed"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/security"
	"digital.vasic.translator/pkg/websocket"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/quic-go/quic-go/http3"
)

const version = "1.0.0"

func main() {
	// Parse command-line flags
	configFile := flag.String("config", "config.json", "Configuration file path")
	showVersion := flag.Bool("version", false, "Show version")
	generateCerts := flag.Bool("generate-certs", false, "Generate self-signed TLS certificates")
	flag.Parse()

	if *showVersion {
		fmt.Printf("Russian-Serbian FB2 Translator Server v%s\n", version)
		os.Exit(0)
	}

	if *generateCerts {
		if err := generateTLSCertificates(); err != nil {
			log.Fatalf("Failed to generate certificates: %v", err)
		}
		fmt.Println("TLS certificates generated successfully")
		os.Exit(0)
	}

	// Load configuration
	cfg, err := loadOrCreateConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Initialize components
	eventBus := events.NewEventBus()
	translationCache := cache.NewCache(time.Duration(cfg.Translation.CacheTTL)*time.Second, cfg.Translation.CacheEnabled)
	authService := security.NewAuthService(cfg.Security.JWTSecret, 24*time.Hour)
	rateLimiter := security.NewRateLimiter(cfg.Security.RateLimitRPS, cfg.Security.RateLimitBurst)
	wsHub := websocket.NewHub(eventBus)

	// Initialize local coordinator
	localCoordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
		EventBus: eventBus,
	})

	// Initialize API communication logger for distributed operations
	var apiLogger *deployment.APICommunicationLogger
	if cfg.Distributed.Enabled {
		var err error
		apiLogger, err = deployment.NewAPICommunicationLogger("workers_api_communication.log")
		if err != nil {
			log.Printf("Warning: failed to initialize API logger: %v", err)
		}
	}

	// Initialize distributed manager if enabled
	var distributedManager interface{}
	if cfg.Distributed.Enabled {
		distributedManager = distributed.NewDistributedManager(cfg, eventBus, apiLogger)
		// Initialize with local coordinator
		if dm, ok := distributedManager.(*distributed.DistributedManager); ok {
			if err := dm.Initialize(localCoordinator); err != nil {
				log.Printf("Failed to initialize distributed manager: %v", err)
				distributedManager = nil
			}
		}
	}

	// Start WebSocket hub
	go wsHub.Run()

	// Create Gin router
	if cfg.Logging.Level != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Setup middleware
	router.Use(corsMiddleware(cfg.Security.CORSOrigins))
	router.Use(rateLimitMiddleware(rateLimiter))

	// Create API handler
	apiHandler := api.NewHandler(cfg, eventBus, translationCache, authService, wsHub, distributedManager)
	apiHandler.RegisterRoutes(router)

	// Server configuration
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	// Create HTTP/3 server if enabled
	if cfg.Server.EnableHTTP3 {
		log.Printf("Starting HTTP/3 server on %s", addr)
		if err := startHTTP3Server(addr, cfg, router); err != nil {
			log.Fatalf("HTTP/3 server failed: %v", err)
		}
	} else {
		log.Printf("Starting HTTP/2 server on %s", addr)
		if err := startHTTP2Server(addr, cfg, router); err != nil {
			log.Fatalf("HTTP/2 server failed: %v", err)
		}
	}
}

func loadOrCreateConfig(filename string) (*config.Config, error) {
	// Check if config exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		log.Printf("Config file not found, creating default: %s", filename)
		cfg := config.DefaultConfig()

		if err := config.SaveConfig(filename, cfg); err != nil {
			return nil, fmt.Errorf("failed to save default config: %w", err)
		}

		return cfg, nil
	}

	return config.LoadConfig(filename)
}

func startHTTP3Server(addr string, cfg *config.Config, handler http.Handler) error {
	// Load TLS certificates
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
		NextProtos: []string{"h3"},
	}

	cert, err := tls.LoadX509KeyPair(cfg.Server.TLSCertFile, cfg.Server.TLSKeyFile)
	if err != nil {
		return fmt.Errorf("failed to load TLS certificates: %w", err)
	}
	tlsConfig.Certificates = []tls.Certificate{cert}

	// Create HTTP/3 server
	server := &http3.Server{
		Addr:      addr,
		Handler:   handler,
		TLSConfig: tlsConfig,
	}

	// Create HTTP/2 fallback server
	fallbackServer := &http.Server{
		Addr:         addr,
		Handler:      handler,
		TLSConfig:    tlsConfig,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	// Start HTTP/2 fallback in goroutine
	go func() {
		log.Printf("Starting HTTP/2 fallback server on %s", addr)
		if err := fallbackServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP/2 fallback server error: %v", err)
		}
	}()

	// Handle graceful shutdown
	go handleShutdown(server, fallbackServer)

	log.Printf("Server started successfully!")
	log.Printf("HTTP/3 (QUIC): https://%s", addr)
	log.Printf("HTTP/2 (TLS): https://%s", addr)
	log.Printf("WebSocket: wss://%s/ws", addr)

	// Start HTTP/3 server
	return server.ListenAndServeTLS(cfg.Server.TLSCertFile, cfg.Server.TLSKeyFile)
}

func startHTTP2Server(addr string, cfg *config.Config, handler http.Handler) error {
	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	// Handle graceful shutdown
	go handleShutdown(nil, server)

	log.Printf("Server started successfully!")
	log.Printf("HTTP: http://%s", addr)

	return server.ListenAndServe()
}

func handleShutdown(http3Server *http3.Server, http2Server *http.Server) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	log.Println("Shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if http3Server != nil {
		if err := http3Server.Close(); err != nil {
			log.Printf("HTTP/3 server shutdown error: %v", err)
		}
	}

	if http2Server != nil {
		if err := http2Server.Shutdown(ctx); err != nil {
			log.Printf("HTTP/2 server shutdown error: %v", err)
		}
	}

	log.Println("Server stopped")
	os.Exit(0)
}

func corsMiddleware(origins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		allowed := false
		for _, o := range origins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func rateLimitMiddleware(limiter *security.RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Use IP address as key
		key := c.ClientIP()

		if !limiter.Allow(key) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func generateTLSCertificates() error {
	// This is a placeholder - in production, use proper certificate generation
	// or obtain certificates from Let's Encrypt
	fmt.Println("Please generate TLS certificates using:")
	fmt.Println("  openssl req -x509 -newkey rsa:4096 -keyout certs/server.key -out certs/server.crt -days 365 -nodes")
	return nil
}
