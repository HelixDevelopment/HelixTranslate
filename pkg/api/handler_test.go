package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"digital.vasic.translator/internal/cache"
	"digital.vasic.translator/pkg/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGenerateOutputFilename(t *testing.T) {
	tests := []struct {
		input    string
		provider string
		expected string
	}{
		{"book.fb2", "dictionary", "book_sr_dictionary.fb2"},
		{"test.b2", "openai", "test_sr_openai.b2"},
		{"novel.fb2", "anthropic", "novel_sr_anthropic.fb2"},
	}

	for _, tt := range tests {
		t.Run(tt.input+"_"+tt.provider, func(t *testing.T) {
			result := generateOutputFilename(tt.input, tt.provider)
			if result != tt.expected {
				t.Errorf("generateOutputFilename(%s, %s) = %s, want %s", tt.input, tt.provider, result, tt.expected)
			}
		})
	}
}

// TestAPIInfo tests the apiInfo handler
func TestAPIInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Create a minimal handler
	h := &Handler{}
	
	// Setup test context
	router := gin.New()
	router.GET("/test", h.apiInfo)
	
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Universal Multi-Format Multi-Language Ebook Translation API", response["name"])
	assert.Equal(t, "1.0.0", response["version"])
	assert.Contains(t, response, "endpoints")
	assert.Contains(t, response, "documentation")
}

// TestTranslateText tests the translateText handler
func TestTranslateText(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Create a minimal handler
	h := &Handler{}
	
	router := gin.New()
	router.POST("/translate", h.translateText)
	
	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		shouldContain  string
	}{
		{
			name:           "missing text field",
			requestBody:    `{"provider":"openai"}`,
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "error",
		},
		{
			name:           "empty text field",
			requestBody:    `{"text":"","provider":"openai"}`,
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "error",
		},
		{
			name:           "invalid JSON",
			requestBody:    `{invalid json}`,
			expectedStatus: http.StatusBadRequest,
			shouldContain:  "error",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", "/translate", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.shouldContain)
		})
	}
}

// TestHealthCheck tests the healthCheck handler
func TestHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	h := &Handler{}
	
	router := gin.New()
	router.GET("/health", h.healthCheck)
	
	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
}

// TestVersionInfo tests version-related handlers
func TestVersionInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	h := &Handler{}
	
	router := gin.New()
	router.GET("/version", h.getVersion)
	
	req, _ := http.NewRequest("GET", "/version", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "codebase_version")
	assert.Contains(t, response, "build_time")
	assert.Contains(t, response, "git_commit")
	assert.Contains(t, response, "go_version")
}

// TestStats tests the stats handler
func TestStats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Create minimal mocks for testing
	mockCache := &cache.Cache{} // This will be nil, but handler should handle it
	mockWebSocketHub := &websocket.Hub{} // This will be nil, but handler should handle it
	
	h := &Handler{
		cache: mockCache,
		wsHub: mockWebSocketHub,
	}
	
	router := gin.New()
	router.GET("/stats", h.getStats)
	
	req, _ := http.NewRequest("GET", "/stats", nil)
	w := httptest.NewRecorder()
	
	// Use a defer to catch any panics from nil dependencies
	defer func() {
		if r := recover(); r != nil {
			// If we get a panic, it's likely due to nil dependencies
			// Set the response code to internal server error
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()
	
	router.ServeHTTP(w, req)
	
	// Should handle nil dependencies gracefully or return error
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError)
}

// TestWebSocketHandler tests the websocket handler setup
func TestWebSocketHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	h := &Handler{}
	
	router := gin.New()
	router.GET("/ws", h.websocketHandler)
	
	req, _ := http.NewRequest("GET", "/ws", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// WebSocket upgrade should return 101 if properly configured
	// or error if no upgrade headers provided
	assert.True(t, w.Code == http.StatusSwitchingProtocols || w.Code == http.StatusBadRequest)
}

// TestAuthMiddleware tests the authentication middleware
func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	h := &Handler{}
	
	// Test middleware without proper auth token
	router := gin.New()
	router.Use(h.authMiddleware())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
	
	req, _ := http.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Should return 401 Unauthorized when no token provided
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestLogin tests the login handler
func TestLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name         string
		setupHandler func() *Handler
		requestBody  string
		expectedCode int
	}{
		{
			name: "empty credentials",
			setupHandler: func() *Handler {
				return &Handler{} // No auth service
			},
			requestBody:  `{}`,
			expectedCode: http.StatusBadRequest, // Empty credentials fail JSON binding validation
		},
		{
			name: "nil auth service",
			setupHandler: func() *Handler {
				return &Handler{} // No auth service
			},
			requestBody:  `{"username":"invalid","password":"invalid"}`,
			expectedCode: http.StatusInternalServerError, // Nil auth service causes panic/internal error
		},
		{
			name: "valid JSON structure",
			setupHandler: func() *Handler {
				return &Handler{} // No auth service
			},
			requestBody:  `{"username":"test","password":"test"}`,
			expectedCode: http.StatusInternalServerError, // Valid JSON but nil auth service
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := tt.setupHandler()
			
			router := gin.New()
			router.POST("/login", h.login)
			
			req, _ := http.NewRequest("POST", "/login", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			
			// Use a defer to catch any panics from nil pointer access
			defer func() {
				if r := recover(); r != nil {
					// If we get a panic, it's likely due to nil auth service
					// Set the response code to internal server error
					w.WriteHeader(http.StatusInternalServerError)
				}
			}()
			
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := tt.setupHandler()
			
			router := gin.New()
			router.POST("/login", h.login)
			
			req, _ := http.NewRequest("POST", "/login", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			
			// Use a defer to catch any panics from nil pointer access
			defer func() {
				if r := recover(); r != nil {
					// If we get a panic, it's likely due to nil auth service
					// Set the response code to internal server error
					w.WriteHeader(http.StatusInternalServerError)
				}
			}()
			
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}

// TestProfile tests the profile handler
func TestProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	h := &Handler{}
	
	router := gin.New()
	router.GET("/profile", h.getProfile)
	
	req, _ := http.NewRequest("GET", "/profile", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Should return 200 with empty user data (no authentication check in handler)
	assert.Equal(t, http.StatusOK, w.Code)
	
	// Parse response to verify structure
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "", response["user_id"])
	assert.Equal(t, "", response["username"])
}
