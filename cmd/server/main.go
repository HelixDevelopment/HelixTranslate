package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"digital.vasic.translator/internal/cache"
	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/pkg/api"
	"digital.vasic.translator/pkg/coordination"
	"digital.vasic.translator/pkg/deployment"
	"digital.vasic.translator/pkg/distributed"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/models"
	"digital.vasic.translator/pkg/security"
	versionpkg "digital.vasic.translator/pkg/version"
	"digital.vasic.translator/pkg/websocket"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/quic-go/quic-go/http3"
)

// version is sourced from the single authoritative versionpkg.AppVersion (== VERSION file).
const version = versionpkg.AppVersion

func main() {
	// Parse command-line flags
	configFile := flag.String("config", "config.json", "Configuration file path")
	showVersion := flag.Bool("version", false, "Show version")
	generateCerts := flag.Bool("generate-certs", false, "Generate self-signed TLS certificates")
	flag.Parse()

	if *showVersion {
		fmt.Printf("Universal Multi-Format Multi-Language Ebook Translation Server v%s\n", version)
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
	userRepo := models.NewInMemoryUserRepository()
	authService := security.NewUserAuthService(cfg.Security.JWTSecret, 24*time.Hour, userRepo)
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

	// Register LLMsVerifier routes if enabled
	if cfg.LLMsVerifier.Enabled {
		verifierHandler := api.InitVerifierFromConfig(cfg)
		verifierHandler.RegisterVerifierRoutes(router.Group("/api/v1"))
		log.Println("LLMsVerifier integration enabled")
	}

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
	// Load TLS certificates
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	cert, err := tls.LoadX509KeyPair(cfg.Server.TLSCertFile, cfg.Server.TLSKeyFile)
	if err != nil {
		return fmt.Errorf("failed to load TLS certificates: %w", err)
	}
	tlsConfig.Certificates = []tls.Certificate{cert}

	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		TLSConfig:    tlsConfig,
	}

	// Handle graceful shutdown
	go handleShutdown(nil, server)

	log.Printf("Server started successfully!")
	log.Printf("HTTP/2 (TLS): https://%s", addr)

	return server.ListenAndServeTLS("", "")
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
	// Pre-resolve config once. A literal "*" entry means public/wildcard; any
	// other entry is a specific allowlisted origin.
	wildcard := false
	allow := make(map[string]bool, len(origins))
	for _, o := range origins {
		if o == "*" {
			wildcard = true
		} else if o != "" {
			allow[o] = true
		}
	}
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		switch {
		case origin != "" && allow[origin]:
			// Specific allowlisted origin → safe to reflect AND permit credentials
			// (only the configured origin, never arbitrary).
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Add("Vary", "Origin")
			setCORSAllowHeaders(c)
		case wildcard:
			// Public wildcard → emit a LITERAL "*" and DO NOT set Allow-Credentials
			// (D12 security fix). Reflecting the request's arbitrary origin TOGETHER
			// with Allow-Credentials:true is a CORS auth-bypass — it lets any site
			// make credentialed cross-origin requests and read responses, defeating
			// the browser rule that forbids "*" with credentials. A literal "*"
			// (no credentials) is the correct public-API posture.
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			setCORSAllowHeaders(c)
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// setCORSAllowHeaders sets the shared Allow-Headers/Allow-Methods for a granted
// CORS response.
func setCORSAllowHeaders(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
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
	certDir := "certs"
	if err := os.MkdirAll(certDir, 0700); err != nil {
		return fmt.Errorf("failed to create certs directory: %w", err)
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"HelixTranslate"},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	certFile := filepath.Join(certDir, "server.crt")
	certOut, err := os.Create(certFile)
	if err != nil {
		return fmt.Errorf("failed to create cert file: %w", err)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		certOut.Close()
		return fmt.Errorf("failed to write cert PEM: %w", err)
	}
	certOut.Close()

	keyFile := filepath.Join(certDir, "server.key")
	// Private key MUST NOT be world/group readable (credential-leak class
	// defect): open with 0600 instead of os.Create's default 0666 (umask-masked
	// to 0644). See §11.4.10 credentials-handling mandate.
	keyOut, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create key file: %w", err)
	}
	// Enforce 0600 even when the file pre-existed (O_CREATE leaves an existing
	// file's permissions untouched).
	if err := keyOut.Chmod(0600); err != nil {
		keyOut.Close()
		return fmt.Errorf("failed to chmod key file: %w", err)
	}
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		keyOut.Close()
		return fmt.Errorf("failed to marshal private key: %w", err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyBytes}); err != nil {
		keyOut.Close()
		return fmt.Errorf("failed to write key PEM: %w", err)
	}
	keyOut.Close()

	fmt.Printf("Generated TLS certificates:\n  Certificate: %s\n  Key: %s\n", certFile, keyFile)
	return nil
}
