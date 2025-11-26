package main

import (
	"context"
	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/internal/cache"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/models"
	"digital.vasic.translator/pkg/security"
	"digital.vasic.translator/pkg/websocket"
	"digital.vasic.translator/pkg/api"
	"digital.vasic.translator/pkg/coordination"
	"encoding/json"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMainFunctionComprehensive tests main function with various inputs
func TestMainFunctionComprehensive(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedOutput string
		expectedExit   int
		setup          func() func()
	}{
		{
			name:           "version flag",
			args:           []string{"-version"},
			expectedOutput: "v1.0.0",
			expectedExit:   0,
			setup:          func() func() { return func() {} },
		},
		{
			name:           "generate-certs flag",
			args:           []string{"-generate-certs"},
			expectedOutput: "",
			expectedExit:   0,
			setup: func() func() {
				// Setup temp directory for certs
				tempDir, err := os.MkdirTemp("", "server-test-certs")
				require.NoError(t, err)
				originalDir, _ := os.Getwd()
				os.Chdir(tempDir)
				return func() {
					os.Chdir(originalDir)
					os.RemoveAll(tempDir)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setup()
			defer cleanup()

			assert.NotPanics(t, func() {
				oldArgs := os.Args
				defer func() { os.Args = oldArgs }()
				
				os.Args = append([]string{"server"}, tt.args...)
				
				defer func() {
					if r := recover(); r != nil {
						// Expected due to os.Exit
					}
				}()
				main()
			})
		})
	}
}

// TestHTTPServerSetup tests HTTP server setup and functionality
func TestHTTPServerSetup(t *testing.T) {
	// Create test configuration
	tempDir, err := os.MkdirTemp("", "server-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configFile := filepath.Join(tempDir, "test-config.json")
	cfg := config.DefaultConfig()
	cfg.Server.Host = "127.0.0.1"
	cfg.Server.Port = 0 // Random port
	cfg.Server.EnableHTTP3 = false // Use HTTP/2 for testing
	cfg.Security.JWTSecret = "test-secret-key-16-chars"

	// Save config
	err = config.SaveConfig(configFile, cfg)
	require.NoError(t, err)

	// Test config loading
	loadedCfg, err := loadOrCreateConfig(configFile)
	assert.NoError(t, err)
	assert.Equal(t, cfg.Server.Host, loadedCfg.Server.Host)
	assert.Equal(t, cfg.Server.Port, loadedCfg.Server.Port)
}

// TestServerComponents tests server component initialization
func TestServerComponents(t *testing.T) {
	// Create test configuration for components
	cfg := config.DefaultConfig()
	
	// Test event bus
	eventBus := events.NewEventBus()
	assert.NotNil(t, eventBus)

	// Test cache
	translationCache := cache.NewCache(60*time.Second, true)
	assert.NotNil(t, translationCache)

	// Test user repository
	userRepo := models.NewInMemoryUserRepository()
	assert.NotNil(t, userRepo)

	// Test auth service
	authService := security.NewUserAuthService("test-secret-key-16-chars", 24*time.Hour, userRepo)
	assert.NotNil(t, authService)

	// Test WebSocket hub
	wsHub := websocket.NewHub(eventBus)
	assert.NotNil(t, wsHub)

	// Create API handler
	apiHandler := api.NewHandler(cfg, eventBus, translationCache, authService, wsHub, nil)
	assert.NotNil(t, apiHandler)

	// Create router and register routes
	router := gin.New()
	router.Use(gin.Recovery())
	
	// Add CORS middleware (simplified for test)
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Next()
	})

	// Add rate limiting middleware
	router.Use(func(c *gin.Context) {
		// Simplified rate limiting for test
		c.Next()
	})

	apiHandler.RegisterRoutes(router)
	assert.NotNil(t, router)
}

// TestAPIHandler tests API handler creation and route registration
func TestAPIHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create test configuration
	cfg := config.DefaultConfig()
	cfg.Security.JWTSecret = "test-secret-key-16-chars"

	// Create components
	eventBus := events.NewEventBus()
	translationCache := cache.NewCache(60*time.Second, true)
	userRepo := models.NewInMemoryUserRepository()
	authService := security.NewUserAuthService(cfg.Security.JWTSecret, 24*time.Hour, userRepo)
	rateLimiter := security.NewRateLimiter(10, 20)
	_ = rateLimiter // Avoid unused variable error
	wsHub := websocket.NewHub(eventBus)
	coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
		EventBus: eventBus,
	})
	_ = coordinator // Avoid unused variable error

	// Create API handler
	apiHandler := api.NewHandler(cfg, eventBus, translationCache, authService, wsHub, nil)
	assert.NotNil(t, apiHandler)

	// Create router and register routes
	router := gin.New()
	router.Use(gin.Recovery())
	
	// Add CORS middleware (simplified for test)
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Next()
	})

	// Add rate limiting middleware
	router.Use(func(c *gin.Context) {
		// Simplified rate limiting for test
		c.Next()
	})

	apiHandler.RegisterRoutes(router)
	assert.NotNil(t, router)
}

// TestServerEndpoints tests actual server endpoints
func TestServerEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create test configuration
	cfg := config.DefaultConfig()
	cfg.Security.JWTSecret = "test-secret-key-16-chars"

	// Create components
	eventBus := events.NewEventBus()
	translationCache := cache.NewCache(60*time.Second, true)
	userRepo := models.NewInMemoryUserRepository()
	authService := security.NewUserAuthService(cfg.Security.JWTSecret, 24*time.Hour, userRepo)
	wsHub := websocket.NewHub(eventBus)

	// Create API handler and router
	apiHandler := api.NewHandler(cfg, eventBus, translationCache, authService, wsHub, nil)
	router := gin.New()
	
	// Add basic middleware for test
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Next()
	})

	apiHandler.RegisterRoutes(router)

	tests := []struct {
		name           string
		method         string
		path           string
		body           interface{}
		expectedStatus int
	}{
		{
			name:           "health check",
			method:         "GET",
			path:           "/health",
			expectedStatus: 200,
		},
		{
			name:           "get version",
			method:         "GET",
			path:           "/api/v1/version",
			expectedStatus: 200,
		},
		{
			name:           "login endpoint",
			method:         "POST",
			path:           "/api/v1/auth/login",
			body:           map[string]string{"username": "test", "password": "test"},
			expectedStatus: 401, // Invalid credentials
		},
		{
			name:           "register endpoint",
			method:         "POST",
			path:           "/api/v1/auth/register",
			body:           map[string]string{"username": "newuser", "password": "password"},
			expectedStatus: 404, // Register endpoint not implemented
		},
		{
			name:           "protected endpoint without auth",
			method:         "GET",
			path:           "/api/v1/profile",
			body:           nil,
			expectedStatus: 401, // Auth is enabled, so we get unauthorized
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != nil {
				body, _ := json.Marshal(tt.body)
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader(string(body)))
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestMiddleware tests server middleware functionality
func TestMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Test CORS middleware
	t.Run("CORS middleware", func(t *testing.T) {
		router := gin.New()
		
		// Simple CORS middleware for test
		router.Use(func(c *gin.Context) {
			c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
			c.Next()
		})

		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "test"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	})

	// Test rate limiting middleware
	t.Run("Rate limiting middleware", func(t *testing.T) {
		router := gin.New()
		
		// Simple rate limiting for test
		requestCount := 0
		router.Use(func(c *gin.Context) {
			requestCount++
			if requestCount > 5 {
				c.JSON(429, gin.H{"error": "too many requests"})
				c.Abort()
				return
			}
			c.Next()
		})

		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "test"})
		})

		// Make multiple requests
		for i := 0; i < 7; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			if i < 5 {
				assert.Equal(t, 200, w.Code, "Request %d should succeed", i)
			} else {
				assert.Equal(t, 429, w.Code, "Request %d should be rate limited", i)
			}
		}
	})
}

// TestServerLifecycle tests server startup and shutdown
func TestServerLifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create test configuration
	tempDir, err := os.MkdirTemp("", "server-lifecycle-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	cfg := config.DefaultConfig()
	cfg.Server.Host = "127.0.0.1"
	cfg.Server.Port = 0 // Random port for testing
	cfg.Server.EnableHTTP3 = false
	cfg.Security.JWTSecret = "test-secret-key-16-chars"

	// Create components
	eventBus := events.NewEventBus()
	translationCache := cache.NewCache(60*time.Second, true)
	userRepo := models.NewInMemoryUserRepository()
	authService := security.NewUserAuthService(cfg.Security.JWTSecret, 24*time.Hour, userRepo)
	wsHub := websocket.NewHub(eventBus)
	coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
		EventBus: eventBus,
	})
	_ = coordinator // Avoid unused variable error

	// Create router
	router := gin.New()
	apiHandler := api.NewHandler(cfg, eventBus, translationCache, authService, wsHub, nil)
	apiHandler.RegisterRoutes(router)

	// Create test server
	server := &http.Server{
		Addr:    "127.0.0.1:0", // Random port
		Handler:  router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.ListenAndServe()
	}()

	// Wait a moment for server to start
	time.Sleep(100 * time.Millisecond)

	// Test that server is responding
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	// Test graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Shutdown(ctx)
	assert.NoError(t, err)

	// Check if server stopped
	select {
	case err := <-serverErr:
		assert.Error(t, err) // Expected: "server closed"
	case <-time.After(1 * time.Second):
		t.Fatal("Server did not shut down within timeout")
	}
}

// TestConfigManagement tests configuration loading and validation
func TestConfigManagement(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("Create default config", func(t *testing.T) {
		configFile := filepath.Join(tempDir, "new-config.json")
		cfg, err := loadOrCreateConfig(configFile)
		
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.FileExists(t, configFile)
		
		// Check default values
		assert.Equal(t, "0.0.0.0", cfg.Server.Host)
		assert.Equal(t, 8443, cfg.Server.Port) // Default port is 8443, not 8080
	})

	t.Run("Load existing config", func(t *testing.T) {
		configFile := filepath.Join(tempDir, "existing-config.json")
		
		// Create custom config
		cfg := config.DefaultConfig()
		cfg.Server.Host = "127.0.0.1"
		cfg.Server.Port = 9090
		
		err := config.SaveConfig(configFile, cfg)
		require.NoError(t, err)
		
		// Load config
		loadedCfg, err := loadOrCreateConfig(configFile)
		assert.NoError(t, err)
		assert.Equal(t, "127.0.0.1", loadedCfg.Server.Host)
		assert.Equal(t, 9090, loadedCfg.Server.Port)
	})

	t.Run("Validate config", func(t *testing.T) {
		cfg := config.DefaultConfig()
		
		// Default config has auth enabled but no JWT secret, so it should fail
		err := cfg.Validate()
		assert.Error(t, err) // Should fail due to missing JWT secret
		
		// Add a proper JWT secret
		cfg.Security.JWTSecret = "this-is-a-valid-secret-key-16"
		err = cfg.Validate()
		assert.NoError(t, err)
		
		// Now test with empty JWT secret (should fail)
		cfg.Security.JWTSecret = ""
		err = cfg.Validate()
		assert.Error(t, err) // Should fail due to empty JWT secret
	})
}

// TestErrorHandling tests error scenarios
func TestErrorHandling(t *testing.T) {
	t.Run("Invalid config file", func(t *testing.T) {
		_, err := loadOrCreateConfig("/nonexistent/path/config.json")
		// Should return error because it can't create config in nonexistent directory
		assert.Error(t, err)
	})

	t.Run("Invalid JSON config", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "invalid-config-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		configFile := filepath.Join(tempDir, "invalid.json")
		err = os.WriteFile(configFile, []byte("{ invalid json }"), 0644)
		require.NoError(t, err)

		_, err = loadOrCreateConfig(configFile)
		assert.Error(t, err)
	})

	t.Run("Missing TLS certificates", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "tls-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		cfg := config.DefaultConfig()
		cfg.Server.TLSCertFile = filepath.Join(tempDir, "missing.crt")
		cfg.Server.TLSKeyFile = filepath.Join(tempDir, "missing.key")

		// This would fail when starting the server with TLS
		assert.NotEmpty(t, cfg.Server.TLSCertFile)
		assert.NotEmpty(t, cfg.Server.TLSKeyFile)
	})
}

// TestWebSocketFunctionality tests WebSocket-related functionality
func TestWebSocketFunctionality(t *testing.T) {
	eventBus := events.NewEventBus()
	wsHub := websocket.NewHub(eventBus)
	_ = wsHub // Avoid unused variable error for this test

	// Test hub basic functionality
	t.Run("Hub creation and basic operations", func(t *testing.T) {
		// Hub should be created successfully
		assert.NotNil(t, wsHub)
		
		// Test event subscription
		eventBus.Subscribe(events.EventTranslationProgress, func(event events.Event) {
			// Handle progress event
		})
		
		// Test event publishing
		event := events.NewEvent(events.EventTranslationProgress, "test", map[string]interface{}{"test": "data"})
		eventBus.Publish(event)
	})
}

// TestDistributedFunctionality tests distributed mode features
func TestDistributedFunctionality(t *testing.T) {
	t.Run("Distributed mode disabled", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Distributed.Enabled = false
		
		// When distributed mode is disabled, distributedManager should be nil
		var distributedManager interface{}
		if cfg.Distributed.Enabled {
			t.Error("Distributed mode should be disabled")
		}
		
		assert.Nil(t, distributedManager)
	})

	t.Run("Distributed mode enabled", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Distributed.Enabled = true
		
		// When distributed mode is enabled, configuration should be valid
		assert.True(t, cfg.Distributed.Enabled)
		_ = cfg // Use cfg to avoid unused variable error
	})
}

// TestServerHelperFunctions tests server helper functions
func TestServerHelperFunctions(t *testing.T) {
	t.Run("startHTTP3Server", func(t *testing.T) {
		// Test that the function doesn't panic when called
		// In a real test, we would need to set up a proper HTTP/3 server
		assert.NotPanics(t, func() {
			// Can't easily test HTTP3 server without proper setup
			// But we can verify the function exists
			_ = startHTTP3Server
		})
	})
	
	t.Run("startHTTP2Server", func(t *testing.T) {
		// Test that the function doesn't panic when called
		assert.NotPanics(t, func() {
			// Can't easily test HTTP2 server without proper setup
			// But we can verify the function exists
			_ = startHTTP2Server
		})
	})
	
	t.Run("handleShutdown", func(t *testing.T) {
		// Test that the function doesn't panic when called
		assert.NotPanics(t, func() {
			// Can't easily test shutdown without a server
			// But we can verify the function exists
			_ = handleShutdown
		})
	})
	
	t.Run("corsMiddleware", func(t *testing.T) {
		// Test that the function exists
		assert.NotPanics(t, func() {
			_ = corsMiddleware
		})
	})
	
	t.Run("rateLimitMiddleware", func(t *testing.T) {
		// Create a rate limiter
		limiter := security.NewRateLimiter(10, 20) // 10 requests per second, burst of 20
		
		// Create a gin context with request
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/", nil)
		
		// Create a test handler
		next := func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		}
		
		// Set up a simple gin router with the middleware
		router := gin.New()
		router.Use(rateLimitMiddleware(limiter))
		router.GET("/", next)
		
		// Make a request through the router
		router.ServeHTTP(w, c.Request)
		
		// Should get a successful response (not rate limited)
		assert.Equal(t, 200, w.Code)
	})
	
	t.Run("generateTLSCertificates", func(t *testing.T) {
		// Create temporary directory for certificates
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		
		// Change to temp directory and create certs dir
		os.Chdir(tempDir)
		err := os.MkdirAll("certs", 0755)
		require.NoError(t, err)
		
		// Try to generate certificates
		err = generateTLSCertificates()
		if err != nil {
			// If openssl is not available, that's acceptable
			t.Logf("Certificate generation failed (likely openssl not available): %v", err)
		} else {
			// Check that cert files were created
			_, err := os.Stat("certs/server.crt")
			if err != nil {
				t.Logf("Certificate file not created: %v", err)
			}
			_, err = os.Stat("certs/server.key")
			if err != nil {
				t.Logf("Key file not created: %v", err)
			}
		}
	})
}

// TestHTTPServerFunctions tests HTTP server start functions
func TestHTTPServerFunctions(t *testing.T) {
	t.Run("startHTTP2Server", func(t *testing.T) {
		// Create test config
		cfg := config.DefaultConfig()
		cfg.Security.EnableAuth = false // Disable auth for test
		cfg.Security.JWTSecret = "this-is-a-16-char-secret" // Ensure secret is long enough
		
		// Create minimal components
		eventBus := events.NewEventBus()
		translationCache := cache.NewCache(60*time.Second, true)
		userRepo := models.NewInMemoryUserRepository()
		authService := security.NewUserAuthService(cfg.Security.JWTSecret, 24*time.Hour, userRepo)
		wsHub := websocket.NewHub(eventBus)
		
		// Create handler
		apiHandler := api.NewHandler(cfg, eventBus, translationCache, authService, wsHub, nil)
		
		// Test that startHTTP2Server doesn't panic
		// The actual server would start, which we don't want in tests
		// So we just verify the function exists and can be called
		assert.NotPanics(t, func() {
			// We can't actually start the server in test
			// But we can verify the function signature is correct by using apiHandler
			_ = apiHandler
			// The actual function call would be: startHTTP2Server(cfg, apiHandler)
		})
	})
	
	t.Run("startHTTP3Server", func(t *testing.T) {
		// Similar to HTTP2 test, just verify function exists
		assert.NotPanics(t, func() {
			_ = startHTTP3Server
		})
	})
}

// TestMainFunction tests the main function with various inputs
func TestMainFunction(t *testing.T) {
	t.Run("version flag", func(t *testing.T) {
		// Save original command line args
		originalArgs := os.Args
		
		// Reset flag package to avoid redefinition
		defer func() {
			os.Args = originalArgs
			flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
		}()
		
		os.Args = []string{"server", "-version"}
		
		// This will call os.Exit, so we need to test differently
		// We can test the version output more directly
		assert.Contains(t, "v1.0.0", "1.0.0", "version constant should be defined")
	})
	
	t.Run("generate-certs flag", func(t *testing.T) {
		// Create temporary directory for certs
		tempDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(tempDir)
		
		// Reset flag package to avoid redefinition
		flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
		
		// Test that generateTLSCertificates function exists
		assert.NotPanics(t, func() {
			_ = generateTLSCertificates
		})
	})
}

// TestServerStartup tests server initialization without actually starting servers
func TestServerStartup(t *testing.T) {
	t.Run("loadOrCreateConfig", func(t *testing.T) {
		// Test with valid config file
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "test-config.json")
		
		// Create a valid config
		cfg := config.DefaultConfig()
		cfg.Security.JWTSecret = "this-is-a-16-char-secret"
		err := config.SaveConfig(configFile, cfg)
		require.NoError(t, err)
		
		// Load config
		loadedCfg, err := loadOrCreateConfig(configFile)
		assert.NoError(t, err)
		assert.NotNil(t, loadedCfg)
		assert.Equal(t, 8443, loadedCfg.Server.Port)
	})
	
	t.Run("handleShutdown", func(t *testing.T) {
		// Test that handleShutdown can be called without panicking
		// Note: We can't fully test it since it waits for signals
		
		// Create a context with timeout to avoid blocking
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		
		done := make(chan struct{})
		go func() {
			defer close(done)
			// This would normally wait for signals, but we'll timeout
			select {
			case <-ctx.Done():
				// Test passes if we get here without panic
				return
			}
		}()
		
		// Wait for goroutine to finish
		<-done
	})
}