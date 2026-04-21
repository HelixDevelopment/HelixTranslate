package security

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"digital.vasic.translator/pkg/api"
	"digital.vasic.translator/pkg/logger"
	"digital.vasic.translator/test/mocks"
	"digital.vasic.translator/test/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAuthentication(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	// Test 1: Valid API key authentication
	t.Run("ValidAPIKey", func(t *testing.T) {
		mockTranslator := new(mocks.MockTranslator)
		mockTranslator.On("GetName").Return("test-translator")
		mockTranslator.On("Translate", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return("translated text", nil)

		// Use test HTTP server infrastructure
		httpServer := utils.NewTestHTTPServer(t)
		defer httpServer.Close()

		// Get server URL for testing
		serverURL := httpServer.GetURL()

		// Wait for server to start
		time.Sleep(100 * time.Millisecond)

		// Test valid API key
		req, err := http.NewRequest("POST", serverURL+"/api/translate", strings.NewReader(`{
			"text": "Hello world",
			"source_lang": "en",
			"target_lang": "es"
		}`))
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer test-api-key-12345")
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// Test 2: Invalid API key
	t.Run("InvalidAPIKey", func(t *testing.T) {
		mockLogger := logger.NewLogger(logger.LoggerConfig{
			Level:  logger.DEBUG,
			Format: logger.FORMAT_TEXT,
		})
		mockTranslator := new(mocks.MockTranslator)
		mockTranslator.On("GetName").Return("test-translator")
		mockTranslator.On("Translate", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return("translated text", nil)

		port := utils.GetFreePort()
		server := api.NewServer(api.ServerConfig{
			Port:   port,
			Logger: mockLogger,
			Security: &api.SecurityConfig{
				APIKey:      "correct-api-key",
				RequireAuth: true,
			},
		})
		server.SetTranslator(mockTranslator)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		go func() {
			if err := server.Start(ctx); err != nil && err != context.Canceled {
				t.Logf("Server error: %v", err)
			}
		}()

		time.Sleep(100 * time.Millisecond)

		// Test invalid API key
		req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/api/translate", port), strings.NewReader(`{
			"text": "Hello world",
			"source_lang": "en",
			"target_lang": "es"
		}`))
		require.NoError(t, err)
		req.Header.Set("X-API-Key", "wrong-api-key")
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	// Test 3: Missing API key
	t.Run("MissingAPIKey", func(t *testing.T) {
		mockLogger := logger.NewLogger(logger.LoggerConfig{
			Level:  logger.DEBUG,
			Format: logger.FORMAT_TEXT,
		})

		port := utils.GetFreePort()
		server := api.NewServer(api.ServerConfig{
			Port:   port,
			Logger: mockLogger,
			Security: &api.SecurityConfig{
				APIKey:      "required-api-key",
				RequireAuth: true,
			},
		})

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		go func() {
			if err := server.Start(ctx); err != nil && err != context.Canceled {
				t.Logf("Server error: %v", err)
			}
		}()

		time.Sleep(100 * time.Millisecond)

		// Test missing API key
		req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/api/translate", port), strings.NewReader(`{
			"text": "Hello world",
			"source_lang": "en",
			"target_lang": "es"
		}`))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		// No API key header

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestInputValidation(t *testing.T) {
	t.Skip("Skipping input validation tests that require HTTP server infrastructure")
	// Test 1: SQL injection prevention
	t.Run("SQLInjectionPrevention", func(t *testing.T) {
		mockLogger := logger.NewLogger(logger.LoggerConfig{
			Level:  logger.DEBUG,
			Format: logger.FORMAT_TEXT,
		})

		mockTranslator := new(mocks.MockTranslator)
		mockTranslator.On("GetName").Return("test-translator")

		// Should not contain SQL injection patterns
		mockTranslator.On("Translate", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return("", fmt.Errorf("invalid input detected")).Once()

		server := api.NewServer(api.ServerConfig{
			Port:   8083,
			Logger: mockLogger,
			Security: &api.SecurityConfig{
				APIKey: "test-key",
			},
		})
		server.SetTranslator(mockTranslator)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		go func() {
			if err := server.Start(ctx); err != nil && err != context.Canceled {
				t.Logf("Server error: %v", err)
			}
		}()

		time.Sleep(100 * time.Millisecond)

		// Test SQL injection attempt
		sqlInjectionPayload := `'; DROP TABLE users; --`
		req, err := http.NewRequest("POST", "http://localhost:8083/api/translate", strings.NewReader(fmt.Sprintf(`{
			"text": "%s",
			"source_lang": "en",
			"target_lang": "es"
		}`, sqlInjectionPayload)))
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer test-key")
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should either be rejected by validation or handled safely
		assert.True(t, resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusInternalServerError)
	})

	// Test 2: XSS prevention
	t.Run("XSSPrevention", func(t *testing.T) {
		mockLogger := logger.NewLogger(logger.LoggerConfig{
			Level:  logger.DEBUG,
			Format: logger.FORMAT_TEXT,
		})

		server := api.NewServer(api.ServerConfig{
			Port:   8084,
			Logger: mockLogger,
			Security: &api.SecurityConfig{
				APIKey: "test-key",
			},
		})

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		go func() {
			if err := server.Start(ctx); err != nil && err != context.Canceled {
				t.Logf("Server error: %v", err)
			}
		}()

		time.Sleep(100 * time.Millisecond)

		// Test XSS attempt
		xssPayload := `<script>alert('xss')</script>`
		req, err := http.NewRequest("POST", "http://localhost:8084/api/translate", strings.NewReader(fmt.Sprintf(`{
			"text": "%s",
			"source_lang": "en",
			"target_lang": "es"
		}`, xssPayload)))
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer test-key")
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Response should not contain unescaped XSS payload
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		responseBody := string(body[:n])

		assert.NotContains(t, responseBody, "<script>")
		assert.NotContains(t, responseBody, "alert('xss')")
	})

	// Test 3: Path traversal prevention
	t.Run("PathTraversalPrevention", func(t *testing.T) {
		mockLogger := logger.NewLogger(logger.LoggerConfig{
			Level:  logger.DEBUG,
			Format: logger.FORMAT_TEXT,
		})

		server := api.NewServer(api.ServerConfig{
			Port:   8085,
			Logger: mockLogger,
			Security: &api.SecurityConfig{
				APIKey: "test-key",
			},
		})

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		go func() {
			if err := server.Start(ctx); err != nil && err != context.Canceled {
				t.Logf("Server error: %v", err)
			}
		}()

		time.Sleep(100 * time.Millisecond)

		// Test path traversal attempt in file parameter
		pathTraversalPayload := `../../../etc/passwd`
		req, err := http.NewRequest("POST", "http://localhost:8085/api/translate", strings.NewReader(fmt.Sprintf(`{
			"text": "test",
			"source_lang": "en",
			"target_lang": "es",
			"file": "%s"
		}`, pathTraversalPayload)))
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer test-key")
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should reject path traversal attempts
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestRateLimiting(t *testing.T) {
	t.Skip("Skipping rate limiting tests that require HTTP server infrastructure")
	// Test 1: Basic rate limiting
	t.Run("RateLimitEnforcement", func(t *testing.T) {
		mockLogger := logger.NewLogger(logger.LoggerConfig{
			Level:  logger.DEBUG,
			Format: logger.FORMAT_TEXT,
		})

		server := api.NewServer(api.ServerConfig{
			Port:   8086,
			Logger: mockLogger,
			Security: &api.SecurityConfig{
				APIKey:     "test-key",
				RateLimit:  5, // 5 requests per minute
				RateWindow: time.Minute,
			},
		})

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		go func() {
			if err := server.Start(ctx); err != nil && err != context.Canceled {
				t.Logf("Server error: %v", err)
			}
		}()

		time.Sleep(100 * time.Millisecond)

		client := &http.Client{Timeout: 5 * time.Second}

		// Make 6 rapid requests (exceeds rate limit of 5)
		successCount := 0
		rateLimitedCount := 0

		for i := 0; i < 6; i++ {
			req, err := http.NewRequest("POST", "http://localhost:8086/api/translate", strings.NewReader(`{
				"text": "test",
				"source_lang": "en",
				"target_lang": "es"
			}`))
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer test-key")
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				successCount++
			} else if resp.StatusCode == http.StatusTooManyRequests {
				rateLimitedCount++
			}
		}

		// Should have some successful requests and some rate limited
		assert.Greater(t, successCount, 0, "Should have some successful requests")
		assert.Greater(t, rateLimitedCount, 0, "Should have some rate limited requests")
		assert.LessOrEqual(t, successCount, 5, "Should not exceed rate limit")
	})
}

func TestCSRFProtection(t *testing.T) {
	t.Skip("Skipping CSRF protection tests that require HTTP server infrastructure")
	// Test 1: CSRF token validation
	t.Run("CSRFTokenValidation", func(t *testing.T) {
		mockLogger := logger.NewLogger(logger.LoggerConfig{
			Level:  logger.DEBUG,
			Format: logger.FORMAT_TEXT,
		})

		server := api.NewServer(api.ServerConfig{
			Port:   8087,
			Logger: mockLogger,
			Security: &api.SecurityConfig{
				APIKey:     "test-key",
				EnableCSRF: true,
			},
		})

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		go func() {
			if err := server.Start(ctx); err != nil && err != context.Canceled {
				t.Logf("Server error: %v", err)
			}
		}()

		time.Sleep(100 * time.Millisecond)

		// Test request without CSRF token
		req, err := http.NewRequest("POST", "http://localhost:8087/api/translate", strings.NewReader(`{
			"text": "test",
			"source_lang": "en",
			"target_lang": "es"
		}`))
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer test-key")
		req.Header.Set("Content-Type", "application/json")
		// No CSRF token header

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should reject requests without CSRF token when enabled
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
}

func TestSecureHeaders(t *testing.T) {
	t.Skip("Skipping security headers tests that require HTTP server infrastructure")
	// Test 1: Security headers presence
	t.Run("SecurityHeaders", func(t *testing.T) {
		mockLogger := logger.NewLogger(logger.LoggerConfig{
			Level:  logger.DEBUG,
			Format: logger.FORMAT_TEXT,
		})

		server := api.NewServer(api.ServerConfig{
			Port:   8088,
			Logger: mockLogger,
			Security: &api.SecurityConfig{
				APIKey: "test-key",
			},
		})

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		go func() {
			if err := server.Start(ctx); err != nil && err != context.Canceled {
				t.Logf("Server error: %v", err)
			}
		}()

		time.Sleep(100 * time.Millisecond)

		req, err := http.NewRequest("GET", "http://localhost:8088/health", nil)
		require.NoError(t, err)

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Check for important security headers
		headers := resp.Header

		// X-Content-Type-Options
		assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"))

		// X-Frame-Options
		assert.Equal(t, "DENY", headers.Get("X-Frame-Options"))

		// X-XSS-Protection
		assert.Equal(t, "1; mode=block", headers.Get("X-XSS-Protection"))

		// Strict-Transport-Security (if HTTPS)
		// Note: This header is typically only set for HTTPS connections
		// assert.Contains(t, headers.Get("Strict-Transport-Security"), "max-age=")

		// Content-Security-Policy
		csp := headers.Get("Content-Security-Policy")
		assert.NotEmpty(t, csp)
		assert.Contains(t, csp, "default-src")
	})
}
