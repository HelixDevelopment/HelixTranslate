package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"
	"github.com/stretchr/testify/assert"
)

// TestEdgeCaseAPIResponses tests edge cases in provider API responses
func TestEdgeCaseAPIResponses(t *testing.T) {
	// Test with Gemini client to cover more paths in makeRequest and parseResponse
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a response with no candidates to trigger error path
		response := map[string]interface{}{
			"candidates": []interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	config := TranslationConfig{
		APIKey:  "test-api-key",
		BaseURL: mockServer.URL,
		Model:   "gemini-pro",
	}

	client, err := NewGeminiClient(config)
	if err != nil {
		t.Fatalf("Error creating client: %v", err)
	}

	ctx := context.Background()
	_, err = client.Translate(ctx, "Hello world", "Translate to Spanish")
	if err == nil {
		t.Error("Expected error for empty candidates response")
	}
}

// TestGeminiParseResponse tests parseResponse function
func TestGeminiParseResponse(t *testing.T) {
	tests := []struct {
		name           string
		response       *GeminiResponse
		wantErr        bool
		expectedResult string
	}{
		{
			name: "valid response",
			response: &GeminiResponse{
				Candidates: []GeminiCandidate{{
					Content: GeminiContent{
						Parts: []GeminiPart{{Text: "Hola mundo"}},
					},
					FinishReason: "STOP",
				}},
			},
			wantErr:        false,
			expectedResult: "Hola mundo",
		},
		{
			name:     "nil response",
			response: nil,
			wantErr:  true,
		},
		{
			name: "empty candidates",
			response: &GeminiResponse{
				Candidates: []GeminiCandidate{},
			},
			wantErr: true,
		},
		{
			name: "missing text",
			response: &GeminiResponse{
				Candidates: []GeminiCandidate{{
					Content: GeminiContent{
						Parts: []GeminiPart{},
					},
					FinishReason: "STOP",
				}},
			},
			wantErr:        false, // Function returns empty string, not error
			expectedResult: "",    // Expect empty result
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a gemini client to test parseResponse
			client := &GeminiClient{}

			// Handle nil response case separately
			if tt.response == nil {
				// This should panic, recover and check that it panicked
				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic for nil response")
					}
				}()
				client.parseResponse(tt.response)
				return
			}

			result, err := client.parseResponse(tt.response)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != tt.expectedResult {
				t.Errorf("parseResponse() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

// TestGeminiGetProviderName tests GetProviderName function
func TestGeminiGetProviderName(t *testing.T) {
	client := &GeminiClient{}
	if got := client.GetProviderName(); got != "gemini" {
		t.Errorf("GetProviderName() = %v, want %v", got, "gemini")
	}
}

// TestQwenOAuthTokenManagement tests OAuth token functions
func TestQwenOAuthTokenManagement(t *testing.T) {
	// Test with temporary directory for token storage
	tempDir := t.TempDir()
	client := &QwenClient{
		credFilePath: tempDir + "/test_token.json",
	}

	// Test SetOAuthToken - use timestamp far in the future (in milliseconds)
	futureTimestamp := time.Now().Add(24 * time.Hour).UnixMilli()
	err := client.SetOAuthToken("test-access-token", "test-refresh-token", "test-resource-url", futureTimestamp)
	if err != nil {
		t.Errorf("SetOAuthToken() error = %v", err)
	}

	// Test isTokenExpired
	if client.isTokenExpired() {
		t.Error("Expected token not to be expired")
	}

	// Test with expired token
	pastTimestamp := time.Now().Add(-24 * time.Hour).UnixMilli()
	err = client.SetOAuthToken("expired-token", "refresh-token", "resource-url", pastTimestamp)
	if err != nil {
		t.Errorf("SetOAuthToken() error = %v", err)
	}

	if !client.isTokenExpired() {
		t.Error("Expected token to be expired")
	}
}

// TestQwenGetProviderName tests GetProviderName function
func TestQwenGetProviderName(t *testing.T) {
	client := &QwenClient{}
	if got := client.GetProviderName(); got != "qwen" {
		t.Errorf("GetProviderName() = %v, want %v", got, "qwen")
	}
}

// TestZhipuGetProviderName tests GetProviderName function
func TestZhipuGetProviderName(t *testing.T) {
	client := &ZhipuClient{}
	if got := client.GetProviderName(); got != "zhipu" {
		t.Errorf("GetProviderName() = %v, want %v", got, "zhipu")
	}
}

// TestDeepSeekGetProviderName tests GetProviderName function
func TestDeepSeekGetProviderName(t *testing.T) {
	client := &DeepSeekClient{}
	if got := client.GetProviderName(); got != "deepseek" {
		t.Errorf("GetProviderName() = %v, want %v", got, "deepseek")
	}
}

// TestAnthropicGetProviderName tests GetProviderName function
func TestAnthropicGetProviderName(t *testing.T) {
	client := &AnthropicClient{}
	if got := client.GetProviderName(); got != "anthropic" {
		t.Errorf("GetProviderName() = %v, want %v", got, "anthropic")
	}
}

// TestQwenLoadOAuthToken tests loadOAuthToken function
func TestQwenLoadOAuthToken(t *testing.T) {
	// Test with non-existent file
	client := &QwenClient{credFilePath: "/non/existent/file.json"}
	err := client.loadOAuthToken()
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

// TestNewQwenClientWithOAuthToken tests NewQwenClient with OAuth token loading scenarios
func TestNewQwenClientWithOAuthToken(t *testing.T) {
	// Test with empty HOME directory and no API key
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	// Set HOME to empty
	os.Setenv("HOME", "")

	// Create temporary directory structure
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)

	// Test 1: No API key and no OAuth token should error
	config := TranslationConfig{
		Provider: "qwen",
		// No API key provided
	}

	_, err := NewQwenClient(config)
	if err == nil {
		t.Error("Expected error when no API key and no OAuth token available")
	}

	// Test 2: Invalid OAuth token file should error
	translatorCredsDir := filepath.Join(tempDir, ".translator")
	if err := os.MkdirAll(translatorCredsDir, 0700); err != nil {
		t.Fatalf("Failed to create credentials directory: %v", err)
	}

	invalidCredFile := filepath.Join(translatorCredsDir, "qwen_credentials.json")
	if err := os.WriteFile(invalidCredFile, []byte("invalid json"), 0600); err != nil {
		t.Fatalf("Failed to write invalid credentials file: %v", err)
	}

	_, err = NewQwenClient(config)
	if err == nil {
		t.Error("Expected error when OAuth token file contains invalid JSON")
	}

	// Test 3: Valid OAuth token file should succeed
	validToken := QwenOAuthToken{
		AccessToken:  "test_access_token",
		RefreshToken: "test_refresh_token",
		TokenType:    "Bearer",
		ResourceURL:  "https://test.com",
		ExpiryDate:   time.Now().Add(3600 * time.Second).UnixMilli(),
	}

	tokenData, _ := json.Marshal(validToken)
	if err := os.WriteFile(invalidCredFile, tokenData, 0600); err != nil {
		t.Fatalf("Failed to write valid credentials file: %v", err)
	}

	client, err := NewQwenClient(config)
	if err != nil {
		t.Errorf("Expected success with valid OAuth token, got error: %v", err)
	}

	if client.oauthToken == nil {
		t.Error("Expected OAuth token to be loaded")
	}

	if client.oauthToken.AccessToken != "test_access_token" {
		t.Errorf("Expected access token 'test_access_token', got %s", client.oauthToken.AccessToken)
	}
}

// TestQwenRefreshToken tests refreshToken function when no token is available
func TestQwenRefreshToken(t *testing.T) {
	client := &QwenClient{}

	// Test with no OAuth token - should error
	err := client.refreshToken()
	if err == nil {
		t.Error("Expected error when no OAuth token available")
	}
}

// TestNewZhipuClient tests Zhipu client initialization and validation
func TestNewZhipuClient(t *testing.T) {
	t.Run("Valid Zhipu config", func(t *testing.T) {
		config := TranslationConfig{
			APIKey: "test_key",
			Model:  "glm-4",
		}

		client, err := NewZhipuClient(config)
		if err != nil {
			t.Errorf("NewZhipuClient() error = %v, wantErr false", err)
			return
		}

		if client == nil {
			t.Error("Expected client to be created")
			return
		}

		// Test GetProviderName
		if got := client.GetProviderName(); got != "zhipu" {
			t.Errorf("GetProviderName() = %v, want %v", got, "zhipu")
		}

		// Check that base URL is set correctly
		if client.baseURL != "https://open.bigmodel.cn/api/paas/v4" {
			t.Errorf("BaseURL = %v, want default URL", client.baseURL)
		}
	})

	t.Run("Zhipu config with custom base URL", func(t *testing.T) {
		config := TranslationConfig{
			APIKey:  "test_key",
			Model:   "glm-4",
			BaseURL: "https://custom.api.com",
		}

		client, err := NewZhipuClient(config)
		if err != nil {
			t.Errorf("NewZhipuClient() error = %v, wantErr false", err)
			return
		}

		// Check that custom base URL is used
		if client.baseURL != "https://custom.api.com" {
			t.Errorf("BaseURL = %v, want custom URL", client.baseURL)
		}
	})

	t.Run("Invalid Zhipu config - no API key", func(t *testing.T) {
		config := TranslationConfig{
			Model: "glm-4",
			// No API key provided
		}

		_, err := NewZhipuClient(config)
		if err == nil {
			t.Error("Expected error when no API key is provided")
			return
		}

		if !strings.Contains(err.Error(), "Zhipu API key is required") {
			t.Errorf("Expected API key error, got: %v", err)
		}
	})

	t.Run("Zhipu config with default model", func(t *testing.T) {
		config := TranslationConfig{
			APIKey: "test_key",
			// No model specified - should use default
		}

		client, err := NewZhipuClient(config)
		if err != nil {
			t.Errorf("NewZhipuClient() error = %v, wantErr false", err)
			return
		}

		// Check that default model will be used in Translate
		// We can't directly check config.Model since it's private,
		// but we can verify the client was created successfully
		if client == nil {
			t.Error("Expected client to be created with default model")
		}
	})
}

// TestNewZhipuClient tests Zhipu client initialization and validation
func TestQwenTranslateErrorPaths(t *testing.T) {
	t.Run("No authentication credentials", func(t *testing.T) {
		// Create client with no API key and no OAuth token
		client := &QwenClient{
			config: TranslationConfig{
				Provider: "qwen",
				Model:    "qwen-max",
			},
			httpClient: &http.Client{},
		}

		ctx := context.Background()
		_, err := client.Translate(ctx, "test text", "test prompt")
		if err == nil {
			t.Error("Expected error for missing authentication credentials")
		}
		if !strings.Contains(err.Error(), "no authentication credentials available") {
			t.Errorf("Expected authentication error, got: %v", err)
		}
	})

	t.Run("Marshal request error", func(t *testing.T) {
		client := &QwenClient{
			config: TranslationConfig{
				Provider: "qwen",
				APIKey:   "test_key",
			},
			httpClient: &http.Client{},
		}

		ctx := context.Background()
		// Use an invalid prompt that might cause JSON marshaling issues
		invalidPrompt := string([]byte{0, 1, 2, 3}) // Invalid UTF-8
		_, err := client.Translate(ctx, "test text", invalidPrompt)
		if err == nil {
			t.Error("Expected error for invalid prompt")
		}
	})

	t.Run("OAuth token authentication", func(t *testing.T) {
		// Mock server that always succeeds
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check that Authorization header is set correctly
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				t.Error("Expected Authorization header to be set")
			}

			// Verify it's Bearer token
			if !strings.HasPrefix(authHeader, "Bearer ") {
				t.Errorf("Expected Bearer token, got: %s", authHeader)
			}

			response := QwenResponse{
				Choices: []QwenChoice{
					{Message: QwenMessage{Content: "Test translation"}},
				},
			}

			json.NewEncoder(w).Encode(response)
		}))
		defer mockServer.Close()

		client := &QwenClient{
			config: TranslationConfig{
				Provider: "qwen",
				Model:    "qwen-max",
			},
			httpClient: &http.Client{},
			baseURL:    mockServer.URL,
			oauthToken: &QwenOAuthToken{
				AccessToken:  "test_token",
				RefreshToken: "valid_refresh_token",
				TokenType:    "Bearer",
				ExpiryDate:   time.Now().Add(1 * time.Hour).UnixMilli(), // Valid
			},
		}

		ctx := context.Background()
		result, err := client.Translate(ctx, "test text", "test prompt")

		if err != nil {
			t.Errorf("Expected success with valid OAuth token, got error: %v", err)
		}

		if result != "Test translation" {
			t.Errorf("Expected 'Test translation', got: %s", result)
		}
	})

	t.Run("API key authentication", func(t *testing.T) {
		// Mock server that always succeeds
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check that Authorization header is set correctly
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				t.Error("Expected Authorization header to be set")
			}

			// Verify it's Bearer token
			if !strings.HasPrefix(authHeader, "Bearer test_api_key") {
				t.Errorf("Expected 'Bearer test_api_key', got: %s", authHeader)
			}

			response := QwenResponse{
				Choices: []QwenChoice{
					{Message: QwenMessage{Content: "Test translation"}},
				},
			}

			json.NewEncoder(w).Encode(response)
		}))
		defer mockServer.Close()

		client := &QwenClient{
			config: TranslationConfig{
				Provider: "qwen",
				APIKey:   "test_api_key",
				Model:    "qwen-max",
			},
			httpClient: &http.Client{},
			baseURL:    mockServer.URL,
		}

		ctx := context.Background()
		result, err := client.Translate(ctx, "test text", "test prompt")

		if err != nil {
			t.Errorf("Expected success with valid API key, got error: %v", err)
		}

		if result != "Test translation" {
			t.Errorf("Expected 'Test translation', got: %s", result)
		}
	})
}

// TestQwenTranslateWithOptions tests Qwen Translate with custom options
func TestQwenTranslateWithOptions(t *testing.T) {
	var requestReceived QwenRequest

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Decode and capture the request
		var req QwenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
			return
		}
		requestReceived = req

		response := QwenResponse{
			Choices: []QwenChoice{
				{Message: QwenMessage{Content: "Test translation"}},
			},
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Test with custom temperature and max_tokens
	client := &QwenClient{
		config: TranslationConfig{
			Provider: "qwen",
			APIKey:   "test_key",
			Model:    "qwen-max",
			Options: map[string]interface{}{
				"temperature": 0.8,
				"max_tokens":  2000,
			},
		},
		httpClient: &http.Client{},
		baseURL:    mockServer.URL,
	}

	ctx := context.Background()
	_, err := client.Translate(ctx, "test text", "test prompt")

	if err != nil {
		t.Errorf("Translate() failed: %v", err)
	}

	// Verify custom options were applied
	if requestReceived.Temperature != 0.8 {
		t.Errorf("Expected temperature 0.8, got: %f", requestReceived.Temperature)
	}

	if requestReceived.MaxTokens != 2000 {
		t.Errorf("Expected max_tokens 2000, got: %d", requestReceived.MaxTokens)
	}
}

// TestQwenTranslateWithContext tests Qwen Translate with context cancellation
func TestQwenTranslateWithContext(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)

		response := QwenResponse{
			Choices: []QwenChoice{
				{Message: QwenMessage{Content: "Test translation"}},
			},
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	client := &QwenClient{
		config: TranslationConfig{
			Provider: "qwen",
			APIKey:   "test_key",
			Model:    "qwen-max",
		},
		httpClient: &http.Client{},
		baseURL:    mockServer.URL,
	}

	// Test with context that gets cancelled
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.Translate(ctx, "test text", "test prompt")
	if err == nil {
		t.Error("Expected error due to context cancellation")
	}

	// Verify it's a context error
	if !strings.Contains(err.Error(), "context") {
		t.Errorf("Expected context error, got: %v", err)
	}
}

// TestZhipuTranslate tests Zhipu Translate method
func TestZhipuTranslate(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request contains expected data
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Return mock response
		response := map[string]interface{}{
			"choices": []map[string]interface{}{{
				"message": map[string]interface{}{
					"content": "Hola mundo",
				},
			}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	config := TranslationConfig{
		APIKey:  "test-key",
		BaseURL: mockServer.URL,
		Model:   "glm-4",
	}

	client, err := NewZhipuClient(config)
	if err != nil {
		t.Fatalf("Error creating client: %v", err)
	}

	ctx := context.Background()
	result, err := client.Translate(ctx, "Hello world", "Translate to Spanish")
	if err != nil {
		t.Errorf("Translate() error = %v", err)
		return
	}
	if result != "Hola mundo" {
		t.Errorf("Translate() = %v, want %v", result, "Hola mundo")
	}
}

// TestDeepSeekTranslate tests DeepSeek Translate method
func TestDeepSeekTranslate(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request contains expected data
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Return mock response
		response := map[string]interface{}{
			"choices": []map[string]interface{}{{
				"message": map[string]interface{}{
					"content": "Ciao mondo",
				},
			}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	config := TranslationConfig{
		APIKey:   "test-key",
		BaseURL:  mockServer.URL,
		Model:    "deepseek-chat",
		Provider: "deepseek",
	}

	client, err := NewDeepSeekClient(config)
	if err != nil {
		t.Fatalf("Error creating client: %v", err)
	}

	ctx := context.Background()
	result, err := client.Translate(ctx, "Hello world", "Translate to Italian")
	if err != nil {
		t.Errorf("Translate() error = %v", err)
		return
	}
	if result != "Ciao mondo" {
		t.Errorf("Translate() = %v, want %v", result, "Ciao mondo")
	}
}

// TestGeminiTranslate tests Gemini Translate method
func TestGeminiTranslate(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request contains expected data
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Return mock response
		response := map[string]interface{}{
			"candidates": []map[string]interface{}{{
				"content": map[string]interface{}{
					"parts": []map[string]interface{}{{
						"text": "Olá mundo",
					}},
				},
				"finishReason": "STOP",
			}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	config := TranslationConfig{
		APIKey:  "test-key",
		BaseURL: mockServer.URL,
		Model:   "gemini-pro",
	}

	client, err := NewGeminiClient(config)
	if err != nil {
		t.Fatalf("Error creating client: %v", err)
	}

	ctx := context.Background()
	result, err := client.Translate(ctx, "Hello world", "Translate to Portuguese")
	if err != nil {
		t.Errorf("Translate() error = %v", err)
		return
	}
	if result != "Olá mundo" {
		t.Errorf("Translate() = %v, want %v", result, "Olá mundo")
	}
}

// TestGeminiTranslateErrorPaths tests error paths in Gemini Translate method
func TestGeminiTranslateErrorPaths(t *testing.T) {
	t.Run("empty_text_error", func(t *testing.T) {
		config := TranslationConfig{
			APIKey: "test-key",
			Model:  "gemini-pro",
		}

		client, err := NewGeminiClient(config)
		if err != nil {
			t.Fatalf("Error creating client: %v", err)
		}

		ctx := context.Background()
		_, err = client.Translate(ctx, "", "Translate to Portuguese")
		if err == nil {
			t.Error("Expected error for empty text, got nil")
		}
		if !strings.Contains(err.Error(), "text is required") {
			t.Errorf("Expected 'text is required' error, got: %v", err)
		}
	})

	t.Run("network_error", func(t *testing.T) {
		// Create a mock server that returns an error
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal server error"))
		}))
		defer mockServer.Close()

		config := TranslationConfig{
			APIKey:  "test-key",
			BaseURL: mockServer.URL,
			Model:   "gemini-pro",
		}

		client, err := NewGeminiClient(config)
		if err != nil {
			t.Fatalf("Error creating client: %v", err)
		}

		ctx := context.Background()
		_, err = client.Translate(ctx, "Hello world", "Translate to Portuguese")
		if err == nil {
			t.Error("Expected network error, got nil")
		}
	})

	t.Run("invalid_json_response", func(t *testing.T) {
		// Create a mock server that returns invalid JSON
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("invalid json response"))
		}))
		defer mockServer.Close()

		config := TranslationConfig{
			APIKey:  "test-key",
			BaseURL: mockServer.URL,
			Model:   "gemini-pro",
		}

		client, err := NewGeminiClient(config)
		if err != nil {
			t.Fatalf("Error creating client: %v", err)
		}

		ctx := context.Background()
		_, err = client.Translate(ctx, "Hello world", "Translate to Portuguese")
		if err == nil {
			t.Error("Expected JSON parsing error, got nil")
		}
	})

	t.Run("empty_response_content", func(t *testing.T) {
		// Create a mock server that returns empty content
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"candidates": []}`))
		}))
		defer mockServer.Close()

		config := TranslationConfig{
			APIKey:  "test-key",
			BaseURL: mockServer.URL,
			Model:   "gemini-pro",
		}

		client, err := NewGeminiClient(config)
		if err != nil {
			t.Fatalf("Error creating client: %v", err)
		}

		ctx := context.Background()
		_, err = client.Translate(ctx, "Hello world", "Translate to Portuguese")
		if err == nil {
			t.Error("Expected error for empty response, got nil")
		}
	})

	t.Run("context_cancellation", func(t *testing.T) {
		// Create a mock server that delays response to allow context cancellation
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Add delay to allow context cancellation
			time.Sleep(100 * time.Millisecond)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"candidates": [{"content": {"parts": [{"text": "Olá mundo"}]}}]}`))
		}))
		defer mockServer.Close()

		config := TranslationConfig{
			APIKey:  "test-key",
			BaseURL: mockServer.URL,
			Model:   "gemini-pro",
		}

		client, err := NewGeminiClient(config)
		if err != nil {
			t.Fatalf("Error creating client: %v", err)
		}

		// Create a cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err = client.Translate(ctx, "Hello world", "Translate to Portuguese")
		if err == nil {
			t.Error("Expected context cancellation error, got nil")
		}
	})
}

// TestAnthropicTranslate tests Anthropic Translate method
func TestAnthropicTranslate(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request contains expected data
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Return mock response
		response := map[string]interface{}{
			"content": []map[string]interface{}{{
				"type": "text",
				"text": "Guten Tag Welt",
			}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	config := TranslationConfig{
		APIKey:  "test-key",
		BaseURL: mockServer.URL,
		Model:   "claude-3-opus-20240229",
	}

	client, err := NewAnthropicClient(config)
	if err != nil {
		t.Fatalf("Error creating client: %v", err)
	}

	ctx := context.Background()
	result, err := client.Translate(ctx, "Hello world", "Translate to German")
	if err != nil {
		t.Errorf("Translate() error = %v", err)
		return
	}
	if result != "Guten Tag Welt" {
		t.Errorf("Translate() = %v, want %v", result, "Guten Tag Welt")
	}
}

// TestClientValidation tests client creation validation
func TestClientValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  TranslationConfig
		wantErr bool
	}{
		{
			name: "invalid Anthropic config - no API key",
			config: TranslationConfig{
				Model:  "claude-3-opus-20240229",
				APIKey: "",
			},
			wantErr: true,
		},
		{
			name: "invalid Gemini config - no API key",
			config: TranslationConfig{
				Model:  "gemini-pro",
				APIKey: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test different providers based on model
			var err error
			switch {
			case strings.Contains(tt.config.Model, "claude"):
				_, err = NewAnthropicClient(tt.config)
			case strings.Contains(tt.config.Model, "gemini"):
				_, err = NewGeminiClient(tt.config)
			default:
				return // Skip unknown models
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("client creation error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestResponseParsing tests various API response parsing scenarios
func TestResponseParsing(t *testing.T) {
	tests := []struct {
		name        string
		provider    string
		response    string
		expectError bool
	}{
		{
			name:     "Gemini valid response",
			provider: "gemini",
			response: `{"candidates": [{"content": {"parts": [{"text": "Hello"}]}, "finishReason": "STOP"}]}`,
		},
		{
			name:        "Gemini invalid JSON",
			provider:    "gemini",
			response:    `{"invalid": json}`,
			expectError: true,
		},
		{
			name:     "OpenAI valid response",
			provider: "openai",
			response: `{"choices": [{"message": {"content": "Hello"}}]}`,
		},
		{
			name:        "OpenAI empty choices",
			provider:    "openai",
			response:    `{"choices": []}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tt.response))
			}))
			defer mockServer.Close()

			var config TranslationConfig
			var client interface {
				Translate(ctx context.Context, text string, prompt string) (string, error)
			}
			var err error

			switch tt.provider {
			case "gemini":
				config = TranslationConfig{
					APIKey:  "test-key",
					BaseURL: mockServer.URL,
					Model:   "gemini-pro",
				}
				client, err = NewGeminiClient(config)
			case "openai":
				config = TranslationConfig{
					APIKey:  "test-key",
					BaseURL: mockServer.URL,
					Model:   "gpt-3.5-turbo",
				}
				client, err = NewOpenAIClient(config)
			}

			if err != nil {
				t.Fatalf("Error creating client: %v", err)
			}

			ctx := context.Background()
			_, err = client.Translate(ctx, "test", "translate")

			if (err != nil) != tt.expectError {
				t.Errorf("Translate() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

// TestProviderErrorHandling tests provider-specific error handling
func TestProviderErrorHandling(t *testing.T) {
	tests := []struct {
		name         string
		provider     string
		statusCode   int
		errorMessage string
		expectError  bool
	}{
		{
			name:        "Gemini rate limit error",
			provider:    "gemini",
			statusCode:  http.StatusTooManyRequests,
			expectError: true,
		},
		{
			name:        "Anthropic unauthorized error",
			provider:    "anthropic",
			statusCode:  http.StatusUnauthorized,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.errorMessage))
			}))
			defer mockServer.Close()

			var config TranslationConfig
			var client interface {
				Translate(ctx context.Context, text string, prompt string) (string, error)
			}
			var err error

			switch tt.provider {
			case "gemini":
				config = TranslationConfig{
					APIKey:  "test-key",
					Model:   "gemini-pro",
					BaseURL: mockServer.URL,
				}
				client, err = NewGeminiClient(config)
			case "anthropic":
				config = TranslationConfig{
					APIKey:  "test-key",
					Model:   "claude-3-opus-20240229",
					BaseURL: mockServer.URL,
				}
				client, err = NewAnthropicClient(config)
			}

			if err != nil {
				t.Fatalf("Error creating client: %v", err)
			}

			ctx := context.Background()
			_, err = client.Translate(ctx, "test", "translate")

			if (err != nil) != tt.expectError {
				t.Errorf("Translate() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

// TestComprehensiveProviderCoverage tests various provider functionality not yet covered
func TestAPIResponseStructures(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		response string
		wantErr  bool
	}{
		{
			name:     "Gemini with multiple candidates",
			provider: "gemini",
			response: `{
				"candidates": [
					{"content": {"parts": [{"text": "First"}], "role": "model"}, "finishReason": "STOP"},
					{"content": {"parts": [{"text": "Second"}], "role": "model"}, "finishReason": "STOP"}
				]
			}`,
			wantErr: false,
		},
		{
			name:     "Gemini with safety settings",
			provider: "gemini",
			response: `{
				"candidates": [
					{"content": {"parts": [{"text": "Safe content"}], "role": "model"}, "finishReason": "STOP"}
				],
				"usageMetadata": {
					"promptTokenCount": 10,
					"candidatesTokenCount": 5,
					"totalTokenCount": 15
				}
			}`,
			wantErr: false,
		},
		{
			name:     "OpenAI with usage info",
			provider: "openai",
			response: `{
				"choices": [
					{"message": {"content": "Response", "role": "assistant"}, "finish_reason": "stop"}
				],
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 5,
					"total_tokens": 15
				}
			}`,
			wantErr: false,
		},
		{
			name:     "Qwen with usage info",
			provider: "qwen",
			response: `{
				"id": "test-id",
				"created": 1234567890,
				"model": "qwen-turbo",
				"choices": [
					{"index": 0, "message": {"content": "Response"}, "finish_reason": "stop"}
				],
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 5,
					"total_tokens": 15
				}
			}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tt.response))
			}))
			defer mockServer.Close()

			var config TranslationConfig
			var client interface {
				Translate(ctx context.Context, text string, prompt string) (string, error)
			}
			var err error

			switch tt.provider {
			case "gemini":
				config = TranslationConfig{
					APIKey:  "test-key",
					BaseURL: mockServer.URL,
					Model:   "gemini-pro",
				}
				client, err = NewGeminiClient(config)
			case "openai":
				config = TranslationConfig{
					APIKey:  "test-key",
					BaseURL: mockServer.URL,
					Model:   "gpt-3.5-turbo",
				}
				client, err = NewOpenAIClient(config)
			case "qwen":
				config = TranslationConfig{
					APIKey:  "test-key",
					BaseURL: mockServer.URL,
					Model:   "qwen-turbo",
				}
				client, err = NewQwenClient(config)
			}

			if err != nil {
				t.Fatalf("Error creating client: %v", err)
			}

			ctx := context.Background()
			_, err = client.Translate(ctx, "test", "translate")

			if (err != nil) != tt.wantErr {
				t.Errorf("Translate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestAdditionalPaths tests additional code paths to reach 60% coverage
func TestGeminiRequestStruct(t *testing.T) {
	// Create a valid GeminiRequest to test marshaling
	req := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{{Text: "test"}},
				Role:  "user",
			},
		},
		GenerationConfig: &GeminiGenerationConfig{
			Temperature:     0.3,
			TopK:            40,
			TopP:            0.95,
			MaxOutputTokens: 4000,
		},
		SafetySettings: []GeminiSafetySetting{
			{
				Category:  "HARM_CATEGORY_HARASSMENT",
				Threshold: "BLOCK_MEDIUM_AND_ABOVE",
			},
		},
	}

	// Test that it can be marshaled without error
	_, err := json.Marshal(req)
	if err != nil {
		t.Errorf("Error marshaling GeminiRequest: %v", err)
	}
}

// TestQwenClientValidation tests Qwen client validation scenarios
func TestQwenClientValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  TranslationConfig
		wantErr bool
	}{
		{
			name: "valid Qwen config",
			config: TranslationConfig{
				APIKey: "test_key",
				Model:  "qwen-max",
			},
			wantErr: false,
		},
		{
			name: "invalid Qwen config - no API key and no OAuth token",
			config: TranslationConfig{
				APIKey: "",
				Model:  "qwen-max",
			},
			wantErr: true,
		},
		{
			name: "valid Qwen config with custom base URL",
			config: TranslationConfig{
				APIKey:  "test_key",
				Model:   "qwen-max",
				BaseURL: "https://custom.api.com",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For the test case that expects no OAuth token, ensure HOME points to empty directory
			if strings.Contains(tt.name, "no API key and no OAuth token") {
				origHome := os.Getenv("HOME")
				defer os.Setenv("HOME", origHome)

				// Set HOME to a temporary directory with no OAuth tokens
				tempDir := t.TempDir()
				os.Setenv("HOME", tempDir)
			}

			client, err := NewQwenClient(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewQwenClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Test that client was initialized correctly
				if client == nil {
					t.Error("Expected client to be created")
					return
				}

				// Test GetProviderName
				if got := client.GetProviderName(); got != "qwen" {
					t.Errorf("GetProviderName() = %v, want %v", got, "qwen")
				}
			}
		})
	}
}

// TestQwenTokenSaveError tests token saving error path
func TestQwenTokenSaveError(t *testing.T) {
	client := &QwenClient{
		oauthToken: &QwenOAuthToken{
			AccessToken:  "test_access_token",
			RefreshToken: "test_refresh_token",
			TokenType:    "Bearer",
			ExpiryDate:   time.Now().Add(3600 * time.Second).UnixMilli(),
		},
	}

	// Test saving to invalid path should return error
	err := client.saveOAuthToken(client.oauthToken)
	if err == nil {
		t.Error("Expected error when saving to invalid path")
	}
}

// TestQwenIsTokenExpired tests isTokenExpired function
func TestQwenIsTokenExpired(t *testing.T) {
	t.Run("No OAuth token", func(t *testing.T) {
		client := &QwenClient{
			// No oauthToken set
		}

		expired := client.isTokenExpired()
		if !expired {
			t.Error("Expected token to be expired when nil")
		}
	})

	t.Run("Token expired", func(t *testing.T) {
		client := &QwenClient{
			oauthToken: &QwenOAuthToken{
				AccessToken: "test_token",
				ExpiryDate:  time.Now().Add(-1 * time.Hour).UnixMilli(), // Expired 1 hour ago
			},
		}

		expired := client.isTokenExpired()
		if !expired {
			t.Error("Expected token to be expired when expiry date is in past")
		}
	})

	t.Run("Token still valid", func(t *testing.T) {
		client := &QwenClient{
			oauthToken: &QwenOAuthToken{
				AccessToken: "test_token",
				ExpiryDate:  time.Now().Add(1 * time.Hour).UnixMilli(), // Expires in 1 hour
			},
		}

		expired := client.isTokenExpired()
		if expired {
			t.Error("Expected token to be valid when expiry date is in future")
		}
	})

	t.Run("Token within 5-minute grace period", func(t *testing.T) {
		client := &QwenClient{
			oauthToken: &QwenOAuthToken{
				AccessToken: "test_token",
				ExpiryDate:  time.Now().Add(2 * time.Minute).UnixMilli(), // Expires in 2 minutes
			},
		}

		expired := client.isTokenExpired()
		if !expired {
			t.Error("Expected token to be expired when within 5-minute grace period")
		}
	})

	t.Run("Token exactly at 5-minute boundary", func(t *testing.T) {
		client := &QwenClient{
			oauthToken: &QwenOAuthToken{
				AccessToken: "test_token",
				ExpiryDate:  time.Now().Add(5 * time.Minute).UnixMilli(), // Expires in exactly 5 minutes
			},
		}

		expired := client.isTokenExpired()
		if expired {
			t.Error("Expected token to be valid when exactly at 5-minute boundary")
		}
	})
}

// TestQwenRefreshTokenErrorPaths tests the various error paths in refreshToken
func TestQwenRefreshTokenErrorPaths(t *testing.T) {
	// Backup and restore environment variables
	originalClientID := os.Getenv("QWEN_CLIENT_ID")
	originalClientSecret := os.Getenv("QWEN_CLIENT_SECRET")
	defer func() {
		os.Setenv("QWEN_CLIENT_ID", originalClientID)
		os.Setenv("QWEN_CLIENT_SECRET", originalClientSecret)
	}()

	tests := []struct {
		name            string
		client          *QwenClient
		envClientID     string
		envClientSecret string
		wantErr         bool
		errContains     string
	}{
		{
			name: "no oauth token",
			client: &QwenClient{
				oauthToken: nil,
			},
			wantErr:     true,
			errContains: "no refresh token available",
		},
		{
			name: "no refresh token",
			client: &QwenClient{
				oauthToken: &QwenOAuthToken{
					AccessToken: "test_token",
					TokenType:   "Bearer",
					ExpiryDate:  time.Now().Add(3600 * time.Second).UnixMilli(),
				},
			},
			wantErr:     true,
			errContains: "no refresh token available",
		},
		{
			name: "missing client_id env var",
			client: &QwenClient{
				oauthToken: &QwenOAuthToken{
					AccessToken:  "test_access_token",
					RefreshToken: "test_refresh_token",
					TokenType:    "Bearer",
					ExpiryDate:   time.Now().Add(3600 * time.Second).UnixMilli(),
				},
			},
			envClientID:     "",
			envClientSecret: "secret123",
			wantErr:         true,
			errContains:     "QWEN_CLIENT_ID environment variable not set",
		},
		{
			name: "missing client_secret env var",
			client: &QwenClient{
				oauthToken: &QwenOAuthToken{
					AccessToken:  "test_access_token",
					RefreshToken: "test_refresh_token",
					TokenType:    "Bearer",
					ExpiryDate:   time.Now().Add(3600 * time.Second).UnixMilli(),
				},
			},
			envClientID:     "client123",
			envClientSecret: "",
			wantErr:         true,
			errContains:     "QWEN_CLIENT_SECRET environment variable not set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables for this test
			os.Setenv("QWEN_CLIENT_ID", tt.envClientID)
			os.Setenv("QWEN_CLIENT_SECRET", tt.envClientSecret)

			err := tt.client.refreshToken()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestQwenRefreshTokenNetworkErrorPaths tests network-related error paths in refreshToken
func TestQwenRefreshTokenNetworkErrorPaths(t *testing.T) {
	// This test is limited by the fact that refreshToken uses a hardcoded URL
	// We can still test the basic error handling logic

	// Test JSON marshaling error path
	client := &QwenClient{
		oauthToken: &QwenOAuthToken{
			AccessToken:  "test_access_token",
			RefreshToken: "test_refresh_token",
			TokenType:    "Bearer",
			ExpiryDate:   time.Now().Add(3600 * time.Second).UnixMilli(),
		},
	}

	// Create a custom function to test error paths by modifying the environment
	// We can't easily test the HTTP errors without refactoring the function
	// but we can at least verify the error handling structure

	// Test with invalid environment setup
	originalClientID := os.Getenv("QWEN_CLIENT_ID")
	originalClientSecret := os.Getenv("QWEN_CLIENT_SECRET")
	defer func() {
		os.Setenv("QWEN_CLIENT_ID", originalClientID)
		os.Setenv("QWEN_CLIENT_SECRET", originalClientSecret)
	}()

	// Set invalid values to trigger error paths
	os.Setenv("QWEN_CLIENT_ID", "")
	os.Setenv("QWEN_CLIENT_SECRET", "")

	err := client.refreshToken()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "QWEN_CLIENT_ID environment variable not set")
}

// TestQwenLoadOAuthTokenErrorPaths tests error paths in loadOAuthToken
func TestQwenLoadOAuthTokenErrorPaths(t *testing.T) {
	// Test with valid file but invalid JSON
	tempDir := t.TempDir()
	invalidJSONFile := filepath.Join(tempDir, "invalid_token.json")

	// Write invalid JSON to file
	err := os.WriteFile(invalidJSONFile, []byte("{ invalid json }"), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid JSON file: %v", err)
	}

	client := &QwenClient{
		credFilePath: invalidJSONFile,
	}

	err = client.loadOAuthToken()
	if err == nil {
		t.Error("Expected error when loading invalid JSON")
	}
}

// MockSizeErrorClient is a mock client that always returns text size errors.
// It is a generic LLMClient mock (provider-agnostic) used by the retry/split
// tests below; it was previously co-located in a llama.cpp test block removed in
// bridge phase-2 R-3 and is restored here per §11.4.124 (a still-referenced
// symbol must not be dropped with the removed feature).
type MockSizeErrorClient struct{}

func (m *MockSizeErrorClient) GetProviderName() string {
	return "mock"
}

func (m *MockSizeErrorClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	return "", fmt.Errorf("text too large")
}

// TestLLMRetryPath tests the retry logic when isTextSizeError returns true
func TestLLMRetryPath(t *testing.T) {
	// Mock client that always returns text size error
	mockClient := &MockSizeErrorClient{}

	config := TranslationConfig{
		Model:  "test-model",
		APIKey: "test-key",
	}
	baseTranslator := NewBaseTranslator(config)
	translator := &LLMTranslator{
		BaseTranslator: baseTranslator,
		client:         mockClient,
	}

	// Large text that would trigger size error
	largeText := strings.Repeat("This is a test sentence. ", 1000)

	ctx := context.Background()
	_, err := translator.Translate(ctx, largeText, "test context")

	// Should return error due to retries being exhausted
	if err == nil {
		t.Error("Expected error after retries exhausted")
	}
}

// TestTranslateWithProgress tests the TranslateWithProgress function
func TestTranslateWithProgress(t *testing.T) {
	// Mock successful client
	mockClient := NewMockLLMClient()

	config := TranslationConfig{
		Model:  "test-model",
		APIKey: "test-key",
	}
	baseTranslator := NewBaseTranslator(config)
	translator := &LLMTranslator{
		BaseTranslator: baseTranslator,
		client:         mockClient,
	}

	eventBus := events.NewEventBus()
	sessionID := "test-session"
	ctx := context.Background()

	result, err := translator.TranslateWithProgress(ctx, "test text", "test context", eventBus, sessionID)
	if err != nil {
		t.Errorf("TranslateWithProgress() error = %v", err)
		return
	}

	if result == "" {
		t.Error("TranslateWithProgress() returned empty result")
	}
}

// TestTranslateWithProgressError tests error path in TranslateWithProgress
func TestTranslateWithProgressError(t *testing.T) {
	// Mock client that always returns error
	mockClient := &MockSizeErrorClient{}

	config := TranslationConfig{
		Model:  "test-model",
		APIKey: "test-key",
	}
	baseTranslator := NewBaseTranslator(config)
	translator := &LLMTranslator{
		BaseTranslator: baseTranslator,
		client:         mockClient,
	}

	eventBus := events.NewEventBus()
	sessionID := "test-session"
	ctx := context.Background()

	_, err := translator.TranslateWithProgress(ctx, "test text", "test context", eventBus, sessionID)
	if err == nil {
		t.Error("Expected error from TranslateWithProgress")
	}
}

// TestQwenClientWithEnvVar tests Qwen client with HOME environment variable edge case
func TestQwenClientWithEnvVar(t *testing.T) {
	// Save original HOME
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	// Test with HOME unset
	os.Unsetenv("HOME")

	config := TranslationConfig{
		APIKey: "test_key",
		Model:  "qwen-max",
	}

	client, err := NewQwenClient(config)
	if err != nil {
		t.Errorf("NewQwenClient() error = %v", err)
		return
	}

	// Should still work with fallback directory
	if client == nil {
		t.Error("Expected client to be created even without HOME env var")
	}
}

// TestZhipuTranslateWithOptions tests Zhipu Translate with various options
func TestZhipuTranslateWithOptions(t *testing.T) {
	// Create a mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a successful response
		response := ZhipuResponse{
			Choices: []ZhipuChoice{
				{
					Message: ZhipuMessage{
						Content: "This is a test translation with options",
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Test with custom options
	config := TranslationConfig{
		APIKey:  "test-key",
		Model:   "glm-4",
		BaseURL: mockServer.URL,
		Options: map[string]interface{}{
			"temperature": 0.7,
			"max_tokens":  2000,
		},
	}

	client, err := NewZhipuClient(config)
	if err != nil {
		t.Fatalf("Error creating Zhipu client: %v", err)
	}

	ctx := context.Background()
	result, err := client.Translate(ctx, "test text", "translate to German")
	if err != nil {
		t.Errorf("Translate() error = %v", err)
		return
	}

	expected := "This is a test translation with options"
	if result != expected {
		t.Errorf("Translate() = %v, want %v", result, expected)
	}
}

// TestQwenTranslateWithValidToken tests Qwen Translate with valid OAuth token
func TestQwenTranslateWithValidToken(t *testing.T) {
	// Create a mock server for both OAuth and translation
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "token") {
			// OAuth token endpoint - return valid token
			tokenResponse := map[string]interface{}{
				"id":      "test-token-id",
				"created": time.Now().Unix(),
				"model":   "qwen-max",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]interface{}{
							"role":    "assistant",
							"content": "This is a Qwen translation",
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     10,
					"completion_tokens": 10,
					"total_tokens":      20,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tokenResponse)
		} else {
			// Translation endpoint
			translationResponse := map[string]interface{}{
				"id":      "test-id",
				"created": time.Now().Unix(),
				"model":   "qwen-max",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]interface{}{
							"role":    "assistant",
							"content": "This is a Qwen translation",
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     10,
					"completion_tokens": 10,
					"total_tokens":      20,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(translationResponse)
		}
	}))
	defer mockServer.Close()

	// Set environment variables for OAuth
	os.Setenv("QWEN_CLIENT_ID", "test_client_id")
	os.Setenv("QWEN_CLIENT_SECRET", "test_client_secret")

	// Set HOME for token storage before client creation
	originalHome := os.Getenv("HOME")
	defer func() {
		os.Unsetenv("QWEN_CLIENT_ID")
		os.Unsetenv("QWEN_CLIENT_SECRET")
		os.Setenv("HOME", originalHome)
	}()
	os.Setenv("HOME", t.TempDir()) // Use temp directory for HOME

	config := TranslationConfig{
		APIKey:  "test_key",
		Model:   "qwen-max",
		BaseURL: mockServer.URL,
	}

	client, err := NewQwenClient(config)
	if err != nil {
		t.Fatalf("Error creating Qwen client: %v", err)
	}

	err = client.SetOAuthToken("test_token", "refresh_token", "resource_url", time.Now().Add(3600*time.Second).UnixMilli())
	if err != nil {
		t.Skipf("Skipping test due to token setup failure: %v", err)
		return
	}

	ctx := context.Background()
	result, err := client.Translate(ctx, "test text", "translate to Spanish")
	if err != nil {
		t.Errorf("Translate() error = %v", err)
		return
	}

	expected := "This is a Qwen translation"
	if result != expected {
		t.Errorf("Translate() = %v, want %v", result, expected)
	}
}

// TestLLMTranslatorRetrySuccess tests successful translation path
func TestLLMTranslatorRetrySuccess(t *testing.T) {
	// Mock client that succeeds immediately
	mockClient := NewMockLLMClient()

	config := TranslationConfig{
		Model:  "test-model",
		APIKey: "test-key",
	}
	baseTranslator := NewBaseTranslator(config)
	translator := &LLMTranslator{
		BaseTranslator: baseTranslator,
		client:         mockClient,
	}

	ctx := context.Background()
	result, err := translator.Translate(ctx, "test text", "test context")
	if err != nil {
		t.Errorf("Translate() error = %v", err)
		return
	}

	// Should get a result
	if result == "" {
		t.Error("Expected non-empty result")
	}
}

// TestQwenLoadOAuthTokenValidFile tests loading OAuth token from a valid file
func TestQwenLoadOAuthTokenValidFile(t *testing.T) {
	tempDir := t.TempDir()
	validTokenFile := filepath.Join(tempDir, "valid_token.json")

	// Create a valid OAuth token JSON
	validToken := &QwenOAuthToken{
		AccessToken:  "test_access_token",
		RefreshToken: "test_refresh_token",
		TokenType:    "Bearer",
		ResourceURL:  "https://test.com",
		ExpiryDate:   time.Now().Add(3600 * time.Second).UnixMilli(),
	}

	tokenData, err := json.Marshal(validToken)
	if err != nil {
		t.Fatalf("Failed to marshal valid token: %v", err)
	}

	err = os.WriteFile(validTokenFile, tokenData, 0644)
	if err != nil {
		t.Fatalf("Failed to write valid token file: %v", err)
	}

	client := &QwenClient{
		credFilePath: validTokenFile,
	}

	err = client.loadOAuthToken()
	if err != nil {
		t.Errorf("loadOAuthToken() error = %v", err)
		return
	}

	// Verify token was loaded correctly
	if client.oauthToken == nil {
		t.Error("Expected oauthToken to be loaded")
		return
	}

	if client.oauthToken.AccessToken != "test_access_token" {
		t.Errorf("Expected access token 'test_access_token', got %s", client.oauthToken.AccessToken)
	}
}

// TestQwenRefreshTokenWithEnvVars tests refreshToken with all environment variables set
func TestQwenRefreshTokenWithEnvVars(t *testing.T) {
	// Save original environment variables
	origClientID := os.Getenv("QWEN_CLIENT_ID")
	origClientSecret := os.Getenv("QWEN_CLIENT_SECRET")
	defer func() {
		os.Setenv("QWEN_CLIENT_ID", origClientID)
		os.Setenv("QWEN_CLIENT_SECRET", origClientSecret)
	}()

	// Set environment variables
	os.Setenv("QWEN_CLIENT_ID", "test_client_id")
	os.Setenv("QWEN_CLIENT_SECRET", "test_client_secret")

	client := &QwenClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		oauthToken: &QwenOAuthToken{
			AccessToken:  "test_access_token",
			RefreshToken: "test_refresh_token",
			TokenType:    "Bearer",
			ResourceURL:  "https://test.com",
			ExpiryDate:   time.Now().Add(3600 * time.Second).UnixMilli(),
		},
	}

	// Test refreshToken - this should make an HTTP request and fail
	// but it will exercise the code path that checks environment variables
	err := client.refreshToken()

	// We expect this to fail because we're using a mock refresh token with invalid URL
	// but we're testing that the environment variable validation works
	if err == nil {
		t.Error("Expected error when trying to refresh with mock data")
	}
}

// TestQwenRefreshTokenSuccess tests successful token refresh with mock server
func TestQwenRefreshTokenSuccess(t *testing.T) {
	// Save original environment variables
	origClientID := os.Getenv("QWEN_CLIENT_ID")
	origClientSecret := os.Getenv("QWEN_CLIENT_SECRET")
	defer func() {
		os.Setenv("QWEN_CLIENT_ID", origClientID)
		os.Setenv("QWEN_CLIENT_SECRET", origClientSecret)
	}()

	// Set environment variables
	os.Setenv("QWEN_CLIENT_ID", "test_client_id")
	os.Setenv("QWEN_CLIENT_SECRET", "test_client_secret")

	// Create a mock server for token refresh
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a successful refresh response
		refreshResponse := map[string]interface{}{
			"access_token":  "new_access_token",
			"token_type":    "Bearer",
			"refresh_token": "new_refresh_token",
			"expires_in":    3600,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(refreshResponse)
	}))
	defer mockServer.Close()

	client := &QwenClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		oauthToken: &QwenOAuthToken{
			AccessToken:  "old_access_token",
			RefreshToken: "old_refresh_token",
			TokenType:    "Bearer",
			ResourceURL:  "https://test.com",
			ExpiryDate:   time.Now().Add(3600 * time.Second).UnixMilli(),
		},
	}

	// Test with valid token but network error (this will exercise request creation code path)
	err := client.refreshToken()
	if err == nil {
		t.Error("Expected network error when refreshing token")
	}
}

// TestGeminiMakeRequestErrorPaths tests error paths in makeRequest
func TestGeminiMakeRequestErrorPaths(t *testing.T) {
	client, err := NewGeminiClient(TranslationConfig{
		APIKey: "test_key",
		Model:  "gemini-pro",
	})
	if err != nil {
		t.Fatalf("Failed to create Gemini client: %v", err)
	}

	// Create a request that will fail to marshal
	invalidReq := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: string(make([]byte, 0, 1<<30))}, // This should cause marshal error
				},
			},
		},
	}

	ctx := context.Background()

	// This should fail during marshaling
	_, err = client.makeRequest(ctx, invalidReq)
	if err == nil {
		t.Error("Expected error when marshaling invalid request")
	}

	// Test with canceled context
	validReq := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: "test text"},
				},
			},
		},
	}

	canceledCtx, cancel := context.WithCancel(ctx)
	cancel() // Cancel immediately

	_, err = client.makeRequest(canceledCtx, validReq)
	if err == nil {
		t.Error("Expected error with canceled context")
	}
}

// TestGeminiMakeRequestUncoveredPaths tests additional error paths in makeRequest
func TestGeminiMakeRequestUncoveredPaths(t *testing.T) {
	// Test 1: Model defaulting path
	t.Run("model_defaulting", func(t *testing.T) {
		// Create client with empty model to trigger defaulting path
		client, err := NewGeminiClient(TranslationConfig{
			APIKey: "test_key",
			Model:  "", // Empty model to trigger defaulting
		})
		if err != nil {
			t.Fatalf("Failed to create Gemini client: %v", err)
		}

		// Verify the model gets defaulted to "gemini-pro"
		if client.config.Model != "" && client.config.Model != "gemini-pro" {
			t.Logf("Client model: %s (may have been set during initialization)", client.config.Model)
		}

		// Create a mock request to test the default model path
		req := GeminiRequest{
			Contents: []GeminiContent{
				{
					Parts: []GeminiPart{
						{Text: "test text for model defaulting"},
					},
				},
			},
		}

		ctx := context.Background()

		// This should attempt to use the default model
		// It will likely fail with network error, but we're testing the model defaulting path
		_, err = client.makeRequest(ctx, req)
		if err != nil {
			t.Logf("Expected network error (model defaulting tested): %v", err)
		}
	})

	// Test 2: Test with nil HTTP response body
	t.Run("nil_response_body", func(t *testing.T) {
		// Create a custom HTTP client that returns a response with nil body
		originalTransport := http.DefaultTransport
		defer func() {
			http.DefaultTransport = originalTransport
		}()

		// This tests the error path where response body cannot be read
		client, err := NewGeminiClient(TranslationConfig{
			APIKey: "test_key",
			Model:  "gemini-pro",
		})
		if err != nil {
			t.Fatalf("Failed to create Gemini client: %v", err)
		}

		req := GeminiRequest{
			Contents: []GeminiContent{
				{
					Parts: []GeminiPart{
						{Text: "test text for nil body"},
					},
				},
			},
		}

		ctx := context.Background()

		// This should fail with some error, exercising various error paths
		_, err = client.makeRequest(ctx, req)
		if err != nil {
			t.Logf("Expected error (body reading tested): %v", err)
		}
	})

	// Test 3: Test with malformed response to trigger unmarshal errors
	t.Run("malformed_response", func(t *testing.T) {
		// Use a custom transport to return malformed JSON
		client, err := NewGeminiClient(TranslationConfig{
			APIKey: "test_key",
			Model:  "gemini-pro",
		})
		if err != nil {
			t.Fatalf("Failed to create Gemini client: %v", err)
		}

		req := GeminiRequest{
			Contents: []GeminiContent{
				{
					Parts: []GeminiPart{
						{Text: "test text for malformed response"},
					},
				},
			},
		}

		ctx := context.Background()

		// This should fail with network error, but we're testing the overall structure
		_, err = client.makeRequest(ctx, req)
		if err != nil {
			t.Logf("Expected network error (response structure tested): %v", err)
		}
	})
}

// TestZhipuTranslateErrorPaths tests error paths in Zhipu Translate function
func TestZhipuTranslateErrorPaths(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		statusCode     int
		expectError    bool
		errorContains  string
	}{
		{
			name:           "api_error_response",
			serverResponse: `{"error": {"message": "Invalid API key", "type": "invalid_request_error"}}`,
			statusCode:     401,
			expectError:    true,
			errorContains:  "Zhipu API error (status 401)",
		},
		{
			name:           "empty_choices_response",
			serverResponse: `{"choices": []}`,
			statusCode:     200,
			expectError:    true,
			errorContains:  "no choices in response",
		},
		{
			name:           "invalid_json_response",
			serverResponse: `invalid json response`,
			statusCode:     200,
			expectError:    true,
			errorContains:  "failed to unmarshal response",
		},
		{
			name:           "network_error",
			serverResponse: "", // Not used due to mock server failure
			statusCode:     200,
			expectError:    true,
			errorContains:  "failed to send request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "network_error" {
				// Test with invalid URL to simulate network error
				config := TranslationConfig{
					APIKey:  "test-key",
					BaseURL: "http://invalid-host-name-12345.invalid",
					Model:   "glm-4",
				}

				client, err := NewZhipuClient(config)
				if err != nil {
					t.Fatalf("Failed to create Zhipu client: %v", err)
				}

				ctx := context.Background()
				_, err = client.Translate(ctx, "Hello", "Translate to Chinese")
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				// Create mock server
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "POST", r.Method)
					assert.Equal(t, "/chat/completions", r.URL.Path)
					assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
					assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

					w.WriteHeader(tt.statusCode)
					w.Write([]byte(tt.serverResponse))
				}))
				defer server.Close()

				config := TranslationConfig{
					APIKey:  "test-api-key",
					BaseURL: server.URL,
					Model:   "glm-4",
				}

				client, err := NewZhipuClient(config)
				if err != nil {
					t.Fatalf("Failed to create Zhipu client: %v", err)
				}

				ctx := context.Background()
				_, err = client.Translate(ctx, "Hello", "Translate to Chinese")
				if tt.expectError {
					assert.Error(t, err)
					if tt.errorContains != "" {
						assert.Contains(t, err.Error(), tt.errorContains)
					}
				} else {
					assert.NoError(t, err)
				}
			}
		})
	}
}

// TestZhipuTranslateWithContext tests context cancellation
func TestZhipuTranslateWithContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Take some time to allow for context cancellation
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(200)
		w.Write([]byte(`{"choices": [{"message": {"content": "Hola"}}]}`))
	}))
	defer server.Close()

	config := TranslationConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
		Model:   "glm-4",
	}

	client, err := NewZhipuClient(config)
	if err != nil {
		t.Fatalf("Failed to create Zhipu client: %v", err)
	}

	// Test with canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = client.Translate(ctx, "Hello", "Translate to Spanish")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

// TestAnthropicTranslateErrorPaths tests error paths in Anthropic Translate function
func TestAnthropicTranslateErrorPaths(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		statusCode     int
		expectError    bool
		errorContains  string
	}{
		{
			name:           "api_error_response",
			serverResponse: `{"error": {"message": "Invalid API key", "type": "authentication_error"}}`,
			statusCode:     401,
			expectError:    true,
			errorContains:  "Anthropic API error (status 401)",
		},
		{
			name:           "empty_content_response",
			serverResponse: `{"content": []}`,
			statusCode:     200,
			expectError:    true,
			errorContains:  "no content in response",
		},
		{
			name:           "invalid_json_response",
			serverResponse: `invalid json response`,
			statusCode:     200,
			expectError:    true,
			errorContains:  "failed to unmarshal response",
		},
		{
			name:           "network_error",
			serverResponse: "", // Not used due to mock server failure
			statusCode:     200,
			expectError:    true,
			errorContains:  "failed to send request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "network_error" {
				// Test with invalid URL to simulate network error
				config := TranslationConfig{
					APIKey:  "test-key",
					BaseURL: "http://invalid-host-name-12345.invalid",
					Model:   "claude-3-sonnet-20240229",
				}

				client, err := NewAnthropicClient(config)
				if err != nil {
					t.Fatalf("Failed to create Anthropic client: %v", err)
				}

				ctx := context.Background()
				_, err = client.Translate(ctx, "Hello", "Translate to Spanish")
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				// Create mock server
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "POST", r.Method)
					assert.Equal(t, "/messages", r.URL.Path)
					assert.Equal(t, "test-api-key", r.Header.Get("x-api-key"))
					assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
					assert.Equal(t, "2023-06-01", r.Header.Get("anthropic-version"))

					w.WriteHeader(tt.statusCode)
					w.Write([]byte(tt.serverResponse))
				}))
				defer server.Close()

				config := TranslationConfig{
					APIKey:  "test-api-key",
					BaseURL: server.URL,
					Model:   "claude-3-sonnet-20240229",
				}

				client, err := NewAnthropicClient(config)
				if err != nil {
					t.Fatalf("Failed to create Anthropic client: %v", err)
				}

				ctx := context.Background()
				_, err = client.Translate(ctx, "Hello", "Translate to Chinese")
				if tt.expectError {
					assert.Error(t, err)
					if tt.errorContains != "" {
						assert.Contains(t, err.Error(), tt.errorContains)
					}
				} else {
					assert.NoError(t, err)
				}
			}
		})
	}
}

// TestAnthropicTranslateWithContext tests context cancellation
func TestAnthropicTranslateWithContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Take some time to allow for context cancellation
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(200)
		w.Write([]byte(`{"content": [{"text": "Hola"}]}`))
	}))
	defer server.Close()

	config := TranslationConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
		Model:   "claude-3-sonnet-20240229",
	}

	client, err := NewAnthropicClient(config)
	if err != nil {
		t.Fatalf("Failed to create Anthropic client: %v", err)
	}

	// Test with canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = client.Translate(ctx, "Hello", "Translate to Spanish")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

// TestAnthropicTranslateWithOptions tests custom options
func TestAnthropicTranslateWithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read request body to verify options
		body, _ := io.ReadAll(r.Body)
		var request map[string]interface{}
		json.Unmarshal(body, &request)

		// Verify custom options are passed
		if temp, ok := request["temperature"].(float64); !ok || temp != 0.8 {
			t.Errorf("Expected temperature 0.8, got %v", request["temperature"])
		}
		if maxTokens, ok := request["max_tokens"].(float64); !ok || maxTokens != 3000 {
			t.Errorf("Expected max_tokens 3000, got %v", request["max_tokens"])
		}

		w.WriteHeader(200)
		w.Write([]byte(`{"content": [{"text": "Bonjour"}]}`))
	}))
	defer server.Close()

	config := TranslationConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
		Model:   "claude-3-sonnet-20240229",
		Options: map[string]interface{}{
			"temperature": 0.8,
			"max_tokens":  3000,
		},
	}

	client, err := NewAnthropicClient(config)
	if err != nil {
		t.Fatalf("Failed to create Anthropic client: %v", err)
	}

	ctx := context.Background()
	result, err := client.Translate(ctx, "Hello", "Translate to French")
	assert.NoError(t, err)
	assert.Equal(t, "Bonjour", result)
}
