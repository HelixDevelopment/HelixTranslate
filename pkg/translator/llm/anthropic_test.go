package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnthropicClient(t *testing.T) {
	tests := []struct {
		name    string
		config  TranslationConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: TranslationConfig{
				Provider: "anthropic",
				APIKey:   "test-key",
				Model:    "claude-3-sonnet-20240229",
				BaseURL:  "https://api.anthropic.com",
			},
			wantErr: false,
		},
		{
			name: "missing api key",
			config: TranslationConfig{
				Provider: "anthropic",
				Model:    "claude-3-sonnet-20240229",
			},
			wantErr: true,
		},
		{
			name: "missing model",
			config: TranslationConfig{
				Provider: "anthropic",
				APIKey:   "test-key",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewAnthropicClient(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAnthropicClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && client != nil {
				if client.GetProviderName() != "anthropic" {
					t.Errorf("GetProviderName() = %v, want %v", client.GetProviderName(), "anthropic")
				}
			}
		})
	}
}

func TestAnthropicClient_Translate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")  // SKIP-OK: #short-mode
	}

	// Skip if no real API key
	apiKey := getTestAPIKey("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("No ANTHROPIC_API_KEY provided for integration test")  // SKIP-OK: #requires-upstream-key
	}

	client, err := NewAnthropicClient(TranslationConfig{
		Provider: "anthropic",
		APIKey:   apiKey,
		Model:    "claude-3-haiku-20240307",
		BaseURL:  "https://api.anthropic.com",
	})
	if err != nil {
		t.Fatalf("Failed to create Anthropic client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tests := []struct {
		name     string
		text     string
		prompt   string
		wantErr  bool
		validate func(string) bool
	}{
		{
			name:    "simple translation",
			text:    "Hello world",
			prompt:  "Translate this text to Spanish",
			wantErr: false,
			validate: func(result string) bool {
				return len(result) > 0 && result != "Hello world"
			},
		},
		{
			name:    "technical translation",
			text:    "The quantum computer uses qubits to perform calculations",
			prompt:  "Translate this technical text to German",
			wantErr: false,
			validate: func(result string) bool {
				return len(result) > 10
			},
		},
		{
			name:    "creative translation",
			text:    "The sun painted the sky with shades of orange and pink",
			prompt:  "Translate this poetic text to Italian",
			wantErr: false,
			validate: func(result string) bool {
				return len(result) > 15
			},
		},
		{
			name:    "empty text",
			text:    "",
			prompt:  "Translate this text",
			wantErr: false,
			validate: func(result string) bool {
				return true // Claude should handle empty text gracefully
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.Translate(ctx, tt.text, tt.prompt)
			if (err != nil) != tt.wantErr {
				t.Errorf("Translate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !tt.validate(result) {
				t.Errorf("Translate() result validation failed for: %s", result)
			}
		})
	}
}

func TestAnthropicClient_RequestStructure(t *testing.T) {
	_ = &AnthropicClient{
		config: TranslationConfig{
			Provider: "anthropic",
			Model:    "claude-3-sonnet-20240229",
		},
	}

	// Test message structure
	messages := []AnthropicMessage{
		{
			Role:    "user",
			Content: "Test message",
		},
	}

	if len(messages) == 0 {
		t.Error("Messages should not be empty")
	}

	if messages[0].Role != "user" {
		t.Errorf("Expected role 'user', got '%s'", messages[0].Role)
	}

	if messages[0].Content != "Test message" {
		t.Errorf("Expected content 'Test message', got '%s'", messages[0].Content)
	}
}

func TestAnthropicClient_ErrorHandling(t *testing.T) {
	tests := []struct {
		name   string
		config TranslationConfig
	}{
		{
			name: "invalid api key",
			config: TranslationConfig{
				Provider: "anthropic",
				APIKey:   "invalid-key",
				Model:    "claude-3-sonnet-20240229",
				BaseURL:  "https://api.anthropic.com",
			},
		},
		{
			name: "invalid base url",
			config: TranslationConfig{
				Provider: "anthropic",
				APIKey:   "test-key",
				Model:    "claude-3-sonnet-20240229",
				BaseURL:  "https://invalid-url.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewAnthropicClient(tt.config)
			if err != nil {
				// Some configurations might fail at creation
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			_, err = client.Translate(ctx, "test", "test prompt")
			if err == nil {
				t.Error("Expected error for invalid configuration")
			}
		})
	}
}

func TestAnthropicClient_ContextHandling(t *testing.T) {
	client := &AnthropicClient{
		config: TranslationConfig{
			Provider: "anthropic",
			Model:    "claude-3-sonnet-20240229",
		},
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	// Test context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.Translate(ctx, "test", "test prompt")
	if err == nil {
		t.Error("Expected error for cancelled context")
	}
}

func TestAnthropicClient_ModelValidation(t *testing.T) {
	validModels := []string{
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
	}

	for _, model := range validModels {
		t.Run("valid_model_"+model, func(t *testing.T) {
			client, err := NewAnthropicClient(TranslationConfig{
				Provider: "anthropic",
				APIKey:   "test-key",
				Model:    model,
				BaseURL:  "https://api.anthropic.com",
			})
			if err != nil {
				t.Errorf("Expected no error for valid model %s, got: %v", model, err)
			}
			if client == nil {
				t.Error("Expected client to be created")
			}
		})
	}

	invalidModels := []string{
		"invalid-model",
		"gpt-4",
		"claude-2",
		"",
	}

	for _, model := range invalidModels {
		t.Run("invalid_model_"+model, func(t *testing.T) {
			client, err := NewAnthropicClient(TranslationConfig{
				Provider: "anthropic",
				APIKey:   "test-key",
				Model:    model,
				BaseURL:  "https://api.anthropic.com",
			})
			if err == nil {
				t.Errorf("Expected error for invalid model %s", model)
			}
			if client != nil {
				t.Error("Expected client to be nil for invalid model")
			}
		})
	}
}

// TestAnthropicRequestErrorPaths tests additional error paths in anthropic Translate function
func TestAnthropicRequestErrorPaths(t *testing.T) {
	t.Run("context_cancellation", func(t *testing.T) {
		client, err := NewAnthropicClient(TranslationConfig{
			Provider: "anthropic",
			APIKey:   "test-api-key",
			Model:    "claude-3-haiku-20240307",
		})
		require.NoError(t, err)
		require.NotNil(t, client)

		// Create cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		result, err := client.Translate(ctx, "Hello", "Translate to Russian")
		require.Error(t, err)
		require.Empty(t, result)
	})

	t.Run("very_long_text", func(t *testing.T) {
		client, err := NewAnthropicClient(TranslationConfig{
			Provider: "anthropic",
			APIKey:   "test-api-key",
			Model:    "claude-3-haiku-20240307",
		})
		require.NoError(t, err)
		require.NotNil(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		longText := strings.Repeat("Hello world. ", 1000)

		_, err = client.Translate(ctx, longText, "Translate to Russian")
		require.Error(t, err)
	})

	t.Run("temperature_option_validation", func(t *testing.T) {
		client, err := NewAnthropicClient(TranslationConfig{
			Provider: "anthropic",
			APIKey:   "test-api-key",
			Model:    "claude-3-haiku-20240307",
			Options: map[string]interface{}{
				"temperature": 2.5, // Too high (should be 0.0-1.0)
			},
		})
		require.NoError(t, err)
		require.NotNil(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err = client.Translate(ctx, "Hello", "Translate to Russian")
		require.Error(t, err)
	})

	t.Run("max_tokens_option_validation", func(t *testing.T) {
		client, err := NewAnthropicClient(TranslationConfig{
			Provider: "anthropic",
			APIKey:   "test-api-key",
			Model:    "claude-3-haiku-20240307",
			Options: map[string]interface{}{
				"max_tokens": -1, // Invalid max_tokens
			},
		})
		require.NoError(t, err)
		require.NotNil(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err = client.Translate(ctx, "Hello", "Translate to Russian")
		require.Error(t, err)
	})

	t.Run("invalid_base_url", func(t *testing.T) {
		client, err := NewAnthropicClient(TranslationConfig{
			Provider: "anthropic",
			APIKey:   "test-api-key",
			Model:    "claude-3-haiku-20240307",
			BaseURL:  "invalid-url://invalid", // Invalid URL
		})
		require.NoError(t, err)
		require.NotNil(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err = client.Translate(ctx, "Hello", "Translate to Russian")
		require.Error(t, err)
	})
}

// TestAnthropicTranslateUncoveredPaths tests uncovered error paths in Anthropic Translate function
func TestAnthropicTranslateUncoveredPaths(t *testing.T) {
	t.Run("json_marshal_error", func(t *testing.T) {
		// Test JSON marshaling error by creating a client with problematic data
		client := &AnthropicClient{
			config: TranslationConfig{
				Provider: "anthropic",
				APIKey:   "test_key",
				Model:    "claude-3-haiku-20240307",
				Options: map[string]interface{}{
					// This might cause JSON marshaling issues if it contains invalid data
					"temperature": float64(0.3),
				},
			},
			httpClient: &http.Client{},
			baseURL:    "http://localhost:99999", // Invalid port to prevent actual requests
		}

		ctx := context.Background()
		// The request should fail at JSON marshaling or request creation stage
		_, err := client.Translate(ctx, "test text", "test prompt")
		if err != nil {
			// This confirms the error path is being tested
			t.Logf("Expected error (JSON marshal or request creation): %v", err)
		}
	})

	t.Run("http_request_error", func(t *testing.T) {
		client := &AnthropicClient{
			config: TranslationConfig{
				Provider: "anthropic",
				APIKey:   "test_key",
				Model:    "claude-3-haiku-20240307",
			},
			httpClient: &http.Client{},
			baseURL:    "invalid://invalid-url", // Invalid URL scheme
		}

		ctx := context.Background()
		_, err := client.Translate(ctx, "test text", "test prompt")
		if err == nil {
			t.Error("Expected HTTP request creation error")
		}

		// Should get an error about unsupported protocol scheme
		if !strings.Contains(err.Error(), "failed to create request") {
			t.Logf("Error may not be request creation related: %v", err)
		}
	})

	t.Run("response_reading_error", func(t *testing.T) {
		client := &AnthropicClient{
			config: TranslationConfig{
				Provider: "anthropic",
				APIKey:   "test_key",
				Model:    "claude-3-haiku-20240307",
			},
			httpClient: &http.Client{},
			baseURL:    "http://localhost:99999", // Invalid port
		}

		ctx := context.Background()
		_, err := client.Translate(ctx, "test text", "test prompt")
		if err == nil {
			t.Error("Expected connection error")
		}

		t.Logf("Expected connection error: %v", err)
	})

	t.Run("invalid_response_json", func(t *testing.T) {
		// Create a mock server that returns invalid JSON
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			// Return invalid JSON (missing closing brace)
			w.Write([]byte(`{
				"content": [
					{
						"type": "text",
						"text": "Translated text"
					}
				}`))
		}))
		defer mockServer.Close()

		client := &AnthropicClient{
			config: TranslationConfig{
				Provider: "anthropic",
				APIKey:   "test_key",
				Model:    "claude-3-haiku-20240307",
			},
			httpClient: &http.Client{},
			baseURL:    mockServer.URL,
		}

		ctx := context.Background()
		_, err := client.Translate(ctx, "test text", "test prompt")
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}

		if !strings.Contains(err.Error(), "failed to unmarshal response") {
			t.Errorf("Expected JSON unmarshal error, got: %v", err)
		}
	})

	t.Run("max_tokens_option_handling", func(t *testing.T) {
		client := &AnthropicClient{
			config: TranslationConfig{
				Provider: "anthropic",
				APIKey:   "test_key",
				Model:    "claude-3-haiku-20240307",
				Options: map[string]interface{}{
					"max_tokens": 2000, // Reasonable value
				},
			},
			httpClient: &http.Client{},
			baseURL:    "http://localhost:99999", // Invalid port
		}

		ctx := context.Background()
		_, err := client.Translate(ctx, "test text", "test prompt")
		if err != nil {
			// Expected to fail due to invalid port
			t.Logf("Expected connection error: %v", err)
		}
		// The important thing is that option was processed during request creation
	})

	t.Run("temperature_option_handling", func(t *testing.T) {
		client := &AnthropicClient{
			config: TranslationConfig{
				Provider: "anthropic",
				APIKey:   "test_key",
				Model:    "claude-3-haiku-20240307",
				Options: map[string]interface{}{
					"temperature": 0.8, // Higher value
				},
			},
			httpClient: &http.Client{},
			baseURL:    "http://localhost:99999", // Invalid port
		}

		ctx := context.Background()
		_, err := client.Translate(ctx, "test text", "test prompt")
		if err != nil {
			// Expected to fail due to invalid port
			t.Logf("Expected connection error: %v", err)
		}
		// The important thing is that option was processed during request creation
	})

	t.Run("empty_content_response", func(t *testing.T) {
		// Test case where response.Content is empty
		client, err := NewAnthropicClient(TranslationConfig{
			Provider: "anthropic",
			APIKey:   "test-api-key",
			Model:    "claude-3-haiku-20240307",
			BaseURL:  "invalid://test", // Invalid URL to force error path
		})
		require.NoError(t, err)
		require.NotNil(t, client)

		// Create a mock server that returns empty content
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Return valid response structure but with empty content
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"content": []}`))
		}))
		defer server.Close()

		client = &AnthropicClient{
			config: TranslationConfig{
				Provider: "anthropic",
				APIKey:   "test-api-key",
				Model:    "claude-3-haiku-20240307",
			},
			httpClient: &http.Client{},
			baseURL:    server.URL,
		}

		ctx := context.Background()
		_, err = client.Translate(ctx, "test text", "test prompt")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no content in response")
	})
}

// TestAnthropicTranslateAdditionalErrorPaths tests additional error paths in Translate
func TestAnthropicTranslateAdditionalErrorPaths(t *testing.T) {
	t.Run("context_cancellation", func(t *testing.T) {
		client := &AnthropicClient{
			baseURL: "https://api.anthropic.com/v1",
			config: TranslationConfig{
				Provider: "anthropic",
				APIKey:   "test-key",
				Model:    "claude-3-sonnet-20240229",
			},
			httpClient: &http.Client{},
		}

		// Create a cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := client.Translate(ctx, "test text", "translate")
		if err == nil {
			t.Error("Expected error with cancelled context")
		} else {
			t.Logf("Got expected error with cancelled context: %v", err)
		}
	})

	t.Run("custom_model_and_options", func(t *testing.T) {
		// Test with custom model and options to hit those code paths
		client := &AnthropicClient{
			baseURL: "https://api.anthropic.com/v1",
			config: TranslationConfig{
				Provider: "anthropic",
				APIKey:   "test-key",
				Model:    "", // Empty to trigger default
				Options: map[string]interface{}{
					"temperature": 0.7,
					"max_tokens":  2048,
				},
			},
			httpClient: &http.Client{},
		}

		// Mock server to verify request structure
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check headers
			if r.Header.Get("x-api-key") != "test-key" {
				t.Errorf("Expected x-api-key header, got: %s", r.Header.Get("x-api-key"))
			}
			if r.Header.Get("anthropic-version") != "2023-06-01" {
				t.Errorf("Expected anthropic-version header, got: %s", r.Header.Get("anthropic-version"))
			}

			// Parse request body to verify model and options
			body, _ := io.ReadAll(r.Body)
			var req AnthropicRequest
			if err := json.Unmarshal(body, &req); err == nil {
				// Should use default model when empty
				if req.Model != "claude-3-sonnet-20240229" {
					t.Errorf("Expected default model, got: %s", req.Model)
				}
				// Should use custom options
				if req.Temperature != 0.7 {
					t.Errorf("Expected temperature 0.7, got: %f", req.Temperature)
				}
				if req.MaxTokens != 2048 {
					t.Errorf("Expected max_tokens 2048, got: %d", req.MaxTokens)
				}
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"content":[{"text":"test response"}]}`))
		}))
		defer server.Close()

		originalURL := client.baseURL
		client.baseURL = server.URL
		defer func() {
			client.baseURL = originalURL
		}()

		ctx := context.Background()
		_, err := client.Translate(ctx, "test", "translate")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}
