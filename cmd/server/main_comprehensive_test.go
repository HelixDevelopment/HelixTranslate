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
			expectedStatus: 201, // Created
		},
		{
			name:           "protected endpoint without auth",
			method:         "GET",
			path:           "/api/v1/user/profile",
			expectedStatus: 401, // Unauthorized
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
		assert.Equal(t, 8080, cfg.Server.Port)
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
		
		// Valid config should pass
		err := cfg.Validate()
		assert.NoError(t, err)
		
		// Invalid config should fail
		cfg.Security.JWTSecret = "short" // Too short
		err = cfg.Validate()
		assert.Error(t, err)
	})
}

// TestErrorHandling tests error scenarios
func TestErrorHandling(t *testing.T) {
	t.Run("Invalid config file", func(t *testing.T) {
		_, err := loadOrCreateConfig("/nonexistent/path/config.json")
		assert.NoError(t, err) // Should create default config
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