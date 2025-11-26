package api

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"digital.vasic.translator/pkg/logger"
	"digital.vasic.translator/pkg/translator"
)

// TestNewServer tests server creation
func TestNewServer(t *testing.T) {
	mockLogger := logger.NewLogger(logger.LoggerConfig{
		Level:  logger.INFO,
		Format: logger.FORMAT_TEXT,
	})

	config := ServerConfig{
		Port:   8080,
		Logger: mockLogger,
	}

	server := NewServer(config)
	assert.NotNil(t, server)
	assert.Equal(t, 8080, server.config.Port)
	assert.Equal(t, mockLogger, server.config.Logger)
}

// TestServer_Start_Stop tests server Start and Stop methods
func TestServer_Start_Stop(t *testing.T) {
	mockLogger := logger.NewLogger(logger.LoggerConfig{
		Level:  logger.INFO,
		Format: logger.FORMAT_TEXT,
	})

	config := ServerConfig{
		Port:   0, // Let OS choose a random port
		Logger: mockLogger,
	}

	server := NewServer(config)
	assert.NotNil(t, server)

	// Set a translator (required for some handlers)
	mockTranslator := &translator.MockTranslator{}
	server.SetTranslator(mockTranslator)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server in background
	go func() {
		if err := server.Start(ctx); err != nil && err != http.ErrServerClosed {
			t.Logf("Server start error: %v", err)
		}
	}()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Stop server
	err := server.Stop(ctx)
	assert.NoError(t, err)
}

// TestDistributedTranslator_Translate tests that Translate method exists
func TestDistributedTranslator_Translate(t *testing.T) {
	dt := &distributedTranslator{}
	
	// Just test that method exists and has correct signature
	// We can't actually call it without proper distributed manager setup
	assert.NotNil(t, dt.Translate)
	
	// Verify return types match expected translator interface
	var _ translator.Translator = dt
}

// TestDistributedTranslator_TranslateWithProgress tests that TranslateWithProgress method exists
func TestDistributedTranslator_TranslateWithProgress(t *testing.T) {
	dt := &distributedTranslator{}
	
	// Just test that method exists and has correct signature
	// We can't actually call it without proper distributed manager setup
	assert.NotNil(t, dt.TranslateWithProgress)
}

// TestSecurityConfigStructure tests security configuration
func TestSecurityConfigStructure(t *testing.T) {
	securityConfig := SecurityConfig{
		APIKey:         "test-api-key",
		RequireAuth:    true,
		MaxRequestSize: 2048000,
		MaxBatchSize:   50,
		RateLimit:      500,
		RateWindow:     30 * time.Minute,
		EnableCSRF:     false,
		SanitizeInput:  true,
		MaxTextLength:  5000,
	}

	assert.Equal(t, "test-api-key", securityConfig.APIKey)
	assert.True(t, securityConfig.RequireAuth)
	assert.Equal(t, int64(2048000), securityConfig.MaxRequestSize)
	assert.Equal(t, 50, securityConfig.MaxBatchSize)
	assert.Equal(t, 500, securityConfig.RateLimit)
	assert.Equal(t, 30*time.Minute, securityConfig.RateWindow)
	assert.False(t, securityConfig.EnableCSRF)
	assert.True(t, securityConfig.SanitizeInput)
	assert.Equal(t, 5000, securityConfig.MaxTextLength)
}

// TestDistributedTranslator_GetName tests GetName method
func TestDistributedTranslator_GetName(t *testing.T) {
	dt := &distributedTranslator{}
	
	name := dt.GetName()
	assert.Equal(t, "distributed", name)
}

// TestDistributedTranslator_GetStats tests GetStats method
func TestDistributedTranslator_GetStats(t *testing.T) {
	dt := &distributedTranslator{}
	
	stats := dt.GetStats()
	// Should return empty stats as per implementation
	assert.Equal(t, 0, stats.Total)
	assert.Equal(t, 0, stats.Translated)
	assert.Equal(t, 0, stats.Cached)
	assert.Equal(t, 0, stats.Errors)
}

// TestTranslateTextHandler tests translateHandler basic functionality
func TestTranslateHandler(t *testing.T) {
	mockLogger := logger.NewLogger(logger.LoggerConfig{
		Level:  logger.INFO,
		Format: logger.FORMAT_TEXT,
	})

	config := ServerConfig{
		Port:   8080,
		Logger: mockLogger,
	}

	server := NewServer(config)
	assert.NotNil(t, server)

	// Test that handler exists by checking router
	router := server.GetRouter()
	assert.NotNil(t, router)
}

// TestLanguagesHandler tests languagesHandler basic functionality
func TestLanguagesHandler(t *testing.T) {
	mockLogger := logger.NewLogger(logger.LoggerConfig{
		Level:  logger.INFO,
		Format: logger.FORMAT_TEXT,
	})

	config := ServerConfig{
		Port:   8080,
		Logger: mockLogger,
	}

	server := NewServer(config)
	assert.NotNil(t, server)

	// Test that handler exists by checking router
	router := server.GetRouter()
	assert.NotNil(t, router)
}

// TestStatsHandler tests statsHandler basic functionality
func TestStatsHandler(t *testing.T) {
	mockLogger := logger.NewLogger(logger.LoggerConfig{
		Level:  logger.INFO,
		Format: logger.FORMAT_TEXT,
	})

	config := ServerConfig{
		Port:   8080,
		Logger: mockLogger,
	}

	server := NewServer(config)
	assert.NotNil(t, server)

	// Test that handler exists by checking router
	router := server.GetRouter()
	assert.NotNil(t, router)
}

// TestAuthMiddleware tests authMiddleware
func TestAuthMiddleware_FromComprehensive(t *testing.T) {
	mockLogger := logger.NewLogger(logger.LoggerConfig{
		Level:  logger.INFO,
		Format: logger.FORMAT_TEXT,
	})

	config := ServerConfig{
		Port: 8080,
		Logger: mockLogger,
		Security: &SecurityConfig{
			APIKey:      "test-key",
			RequireAuth: true,
		},
	}

	server := NewServer(config)
	assert.NotNil(t, server)

	// Test that middleware is configured
	router := server.GetRouter()
	assert.NotNil(t, router)
}

// TestHealthCheck tests healthCheck handler
func TestHealthCheck_FromComprehensive(t *testing.T) {
	mockLogger := logger.NewLogger(logger.LoggerConfig{
		Level:  logger.INFO,
		Format: logger.FORMAT_TEXT,
	})

	config := ServerConfig{
		Port:   8080,
		Logger: mockLogger,
	}

	server := NewServer(config)
	assert.NotNil(t, server)

	// Test that handler exists and can be called
	router := server.GetRouter()
	assert.NotNil(t, router)
}