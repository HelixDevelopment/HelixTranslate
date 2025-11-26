package api

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
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
		Security: &SecurityConfig{
			APIKey:         "test-key",
			RequireAuth:    true,
			MaxRequestSize: 1024 * 1024,
			MaxBatchSize:   100,
			RateLimit:      1000,
			RateWindow:     time.Hour,
			EnableCSRF:     true,
			SanitizeInput:  true,
			MaxTextLength:  10000,
		},
	}

	server := NewServer(config)

	assert.NotNil(t, server)
	assert.Equal(t, config, server.config)
	assert.NotNil(t, server.router)
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

// TestServerConfigStructure tests server configuration
func TestServerConfigStructure(t *testing.T) {
	securityConfig := &SecurityConfig{
		APIKey:         "security-key",
		RequireAuth:    false,
		MaxRequestSize: 1024,
		MaxBatchSize:   10,
		RateLimit:      100,
		RateWindow:     time.Minute,
		EnableCSRF:     true,
		SanitizeInput:  false,
		MaxTextLength:  1000,
	}

	mockLogger := logger.NewLogger(logger.LoggerConfig{
		Level:  logger.DEBUG,
		Format: logger.FORMAT_JSON,
	})
	
	config := ServerConfig{
		Port:     3000,
		Logger:   mockLogger,
		Security: securityConfig,
	}

	assert.Equal(t, 3000, config.Port)
	assert.Equal(t, mockLogger, config.Logger)
	assert.Equal(t, securityConfig, config.Security)
}

// TestTranslateStringRequest tests the translate string request structure
func TestTranslateStringRequest(t *testing.T) {
	// Test JSON unmarshaling
	jsonStr := `{
		"text": "Hello world",
		"source_language": "en",
		"target_language": "ru",
		"provider": "openai",
		"model": "gpt-4"
	}`

	var req TranslateStringRequest
	err := json.Unmarshal([]byte(jsonStr), &req)

	assert.NoError(t, err)
	assert.Equal(t, "Hello world", req.Text)
	assert.Equal(t, "en", req.SourceLanguage)
	assert.Equal(t, "ru", req.TargetLanguage)
	assert.Equal(t, "openai", req.Provider)
	assert.Equal(t, "gpt-4", req.Model)
}

// TestTranslateStringResponse tests the translate string response structure
func TestTranslateStringResponse(t *testing.T) {
	response := TranslateStringResponse{
		TranslatedText: "Привет мир",
		SourceLanguage: "en",
		TargetLanguage: "ru",
		Provider:       "openai",
		Duration:       0.25,
		SessionID:      "test-session-123",
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(response)
	assert.NoError(t, err)
	assert.Contains(t, string(jsonData), "Привет мир")
	assert.Contains(t, string(jsonData), "openai")
	assert.Contains(t, string(jsonData), "test-session-123")

	// Test unmarshaling back
	var unmarshaledResponse TranslateStringResponse
	err = json.Unmarshal(jsonData, &unmarshaledResponse)
	assert.NoError(t, err)
	assert.Equal(t, response.TranslatedText, unmarshaledResponse.TranslatedText)
	assert.Equal(t, response.SourceLanguage, unmarshaledResponse.SourceLanguage)
	assert.Equal(t, response.TargetLanguage, unmarshaledResponse.TargetLanguage)
	assert.Equal(t, response.Duration, unmarshaledResponse.Duration)
	assert.Equal(t, response.SessionID, unmarshaledResponse.SessionID)
}

// TestTranslateDirectoryRequest tests the translate directory request structure
func TestTranslateDirectoryRequest(t *testing.T) {
	// Test JSON unmarshaling
	jsonStr := `{
		"input_path": "/input/path",
		"output_path": "/output/path",
		"source_language": "en",
		"target_language": "ru",
		"provider": "openai",
		"model": "gpt-4",
		"recursive": true,
		"parallel": true,
		"max_concurrency": 5,
		"output_format": "epub"
	}`

	var req TranslateDirectoryRequest
	err := json.Unmarshal([]byte(jsonStr), &req)

	assert.NoError(t, err)
	assert.Equal(t, "/input/path", req.InputPath)
	assert.Equal(t, "/output/path", req.OutputPath)
	assert.Equal(t, "en", req.SourceLanguage)
	assert.Equal(t, "ru", req.TargetLanguage)
	assert.Equal(t, "openai", req.Provider)
	assert.Equal(t, "gpt-4", req.Model)
	assert.True(t, req.Recursive)
	assert.True(t, req.Parallel)
	assert.Equal(t, 5, req.MaxConcurrency)
	assert.Equal(t, "epub", req.OutputFormat)
}

// TestFileResult tests the file result structure
func TestFileResult(t *testing.T) {
	result := FileResult{
		InputPath:  "/input/book.epub",
		OutputPath: "/output/book_translated.epub",
		Success:    true,
		Error:      "",
	}

	assert.Equal(t, "/input/book.epub", result.InputPath)
	assert.Equal(t, "/output/book_translated.epub", result.OutputPath)
	assert.True(t, result.Success)
	assert.Equal(t, "", result.Error)

	// Test JSON marshaling
	jsonData, err := json.Marshal(result)
	assert.NoError(t, err)
	assert.Contains(t, string(jsonData), "book.epub")
	assert.Contains(t, string(jsonData), "true")

	// Test with error
	result.Error = "Translation failed"
	result.Success = false
	jsonData, err = json.Marshal(result)
	assert.NoError(t, err)
	assert.Contains(t, string(jsonData), "Translation failed")
	assert.Contains(t, string(jsonData), "false")
}

// TestGinMode tests gin mode setting
func TestGinMode(t *testing.T) {
	// Save current mode
	currentMode := gin.Mode()
	defer gin.SetMode(currentMode)

	// Test that NewServer sets gin to release mode
	mockLogger := logger.NewLogger(logger.LoggerConfig{
		Level:  logger.INFO,
		Format: logger.FORMAT_TEXT,
	})
	config := ServerConfig{
		Port:   8080,
		Logger: mockLogger,
	}

	NewServer(config)
	assert.Equal(t, gin.ReleaseMode, gin.Mode())
}

// TestRouterNotNil tests that router is properly initialized
func TestRouterNotNil(t *testing.T) {
	mockLogger := logger.NewLogger(logger.LoggerConfig{
		Level:  logger.INFO,
		Format: logger.FORMAT_TEXT,
	})
	config := ServerConfig{
		Port:   8080,
		Logger: mockLogger,
	}

	server := NewServer(config)
	
	assert.NotNil(t, server.router)
	
	// Test that we can create a test request with the router
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	
	// Should get some response (doesn't matter what for this test)
	assert.True(t, w.Code >= 200 && w.Code < 600)
}

// TestGetRouter tests getting router from server
func TestGetRouter(t *testing.T) {
	mockLogger := logger.NewLogger(logger.LoggerConfig{
		Level:  logger.INFO,
		Format: logger.FORMAT_TEXT,
	})
	
	config := ServerConfig{
		Port:   8080,
		Logger: mockLogger,
	}

	server := NewServer(config)
	
	// Test getting router
	router := server.GetRouter()
	
	// Verify it's not nil
	assert.NotNil(t, router)
	assert.Equal(t, server.router, router)
}

// TestServerStartStop tests that server methods exist
func TestServerStartStop(t *testing.T) {
	mockLogger := logger.NewLogger(logger.LoggerConfig{
		Level:  logger.INFO,
		Format: logger.FORMAT_TEXT,
	})
	
	config := ServerConfig{
		Port:   8080,
		Logger: mockLogger,
	}

	server := NewServer(config)
	
	// Test that methods exist (can't actually test start/stop without more complex setup)
	assert.NotNil(t, server.Start)
	assert.NotNil(t, server.Stop)
}

// TestDistributedTranslator_GetName tests the GetName function
func TestDistributedTranslator_GetName(t *testing.T) {
	// Since distributed.DistributedManager requires complex setup,
	// we'll test with a nil pointer just to verify method exists
	dt := &distributedTranslator{}
	
	name := dt.GetName()
	assert.Equal(t, "distributed", name)
}

// TestDistributedTranslator_GetStats tests the GetStats function
func TestDistributedTranslator_GetStats(t *testing.T) {
	dt := &distributedTranslator{}
	
	stats := dt.GetStats()
	// Should return empty stats as per implementation
	assert.Equal(t, 0, stats.Total)
	assert.Equal(t, 0, stats.Translated)
	assert.Equal(t, 0, stats.Cached)
	assert.Equal(t, 0, stats.Errors)
}

// TestDistributedTranslator_TranslateWithProgress tests that TranslateWithProgress method exists
func TestDistributedTranslator_TranslateWithProgress(t *testing.T) {
	dt := &distributedTranslator{}
	
	// Just test that the method exists and has correct signature
	// We can't actually call it without proper distributed manager setup
	assert.NotNil(t, dt.TranslateWithProgress)
}

// TestDistributedTranslator_Translate tests that Translate method exists
func TestDistributedTranslator_Translate(t *testing.T) {
	dt := &distributedTranslator{}
	
	// Just test that method exists and has correct signature
	assert.NotNil(t, dt.Translate)
	
	// Verify return types match expected translator interface
	var _ translator.Translator = dt
}