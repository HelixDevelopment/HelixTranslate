package main

import (
	"crypto/tls"
	"digital.vasic.translator/internal/config"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestRateLimitMiddleware tests rate limiting functionality
func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Create a mock rate limiter
	limiter := &MockRateLimiter{
		allowCount:  0,
		maxRequests: 5,
	}
	
	// Create middleware
	middleware := func(c *gin.Context) {
		if !limiter.Allow(c.ClientIP()) {
			c.JSON(429, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}
		c.Next()
	}
	
	// Create test router
	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "test"})
	})
	
	// Test requests within limit
	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
	}
	
	// Test request exceeding limit
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

// MockRateLimiter is a mock rate limiter for testing
type MockRateLimiter struct {
	allowCount  int
	maxRequests int
}

func (m *MockRateLimiter) Allow(key string) bool {
	m.allowCount++
	return m.allowCount <= m.maxRequests
}

// TestCORSMiddleware tests CORS middleware functionality
func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Allowed origins
	allowedOrigins := []string{"http://localhost:3000", "https://example.com"}
	
	// Create middleware
	middleware := func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// Check if origin is allowed
		allowed := false
		for _, o := range allowedOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}
		
		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		
		// Set other CORS headers
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")
		
		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		
		c.Next()
	}
	
	// Create test router
	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "test"})
	})
	
	// Test with allowed origin
	req1, _ := http.NewRequest("GET", "/test", nil)
	req1.Header.Set("Origin", "http://localhost:3000")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, "http://localhost:3000", w1.Header().Get("Access-Control-Allow-Origin"))
	
	// Test with disallowed origin
	req2, _ := http.NewRequest("GET", "/test", nil)
	req2.Header.Set("Origin", "http://evil.com")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Empty(t, w2.Header().Get("Access-Control-Allow-Origin"))
	
	// Test preflight request
	req3, _ := http.NewRequest("OPTIONS", "/test", nil)
	req3.Header.Set("Origin", "http://localhost:3000")
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)
	
	assert.Equal(t, http.StatusNoContent, w3.Code)
}

// TestTLSCertificateLoading tests TLS certificate loading
func TestTLSCertificateLoading(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			TLSCertFile: "nonexistent.pem",
			TLSKeyFile:  "nonexistent.key",
		},
	}
	
	// Test loading non-existent certificates
	_, err := tls.LoadX509KeyPair(cfg.Server.TLSCertFile, cfg.Server.TLSKeyFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
}

// TestServerConfig tests server configuration
func TestServerConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	
	// Test default values
	assert.NotEmpty(t, cfg.Server.Host)
	assert.Greater(t, cfg.Server.Port, 0)
	assert.NotEmpty(t, cfg.Server.TLSCertFile)
	assert.NotEmpty(t, cfg.Server.TLSKeyFile)
}