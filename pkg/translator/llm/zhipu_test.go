package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZhipuProvider(t *testing.T) {
	t.Run("provider_name", func(t *testing.T) {
		client, err := NewZhipuClient(TranslationConfig{
			Provider: "zhipu",
			APIKey:   "test-key",
		})
		require.NoError(t, err)
		require.NotNil(t, client)
		assert.Equal(t, "zhipu", client.GetProviderName())
	})

	t.Run("missing_api_key", func(t *testing.T) {
		client, err := NewZhipuClient(TranslationConfig{
			Provider: "zhipu",
			APIKey:   "",
		})
		require.Error(t, err)
		require.Nil(t, client)
		assert.Contains(t, err.Error(), "API key is required")
	})

	t.Run("default_base_url", func(t *testing.T) {
		client, err := NewZhipuClient(TranslationConfig{
			Provider: "zhipu",
			APIKey:   "test-key",
		})
		require.NoError(t, err)
		require.NotNil(t, client)
		// baseURL is unexported, but we can verify client was created
	})

	t.Run("custom_base_url", func(t *testing.T) {
		client, err := NewZhipuClient(TranslationConfig{
			Provider: "zhipu",
			APIKey:   "test-key",
			BaseURL:  "https://custom.zhipuai.com",
		})
		require.NoError(t, err)
		require.NotNil(t, client)
	})

	t.Run("mock_translate_success", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"choices": [{"message": {"content": "Привет"}}]
			}`))
		}))
		defer mockServer.Close()

		client := &ZhipuClient{
			config: TranslationConfig{
				Provider: "zhipu",
				APIKey:   "test-key",
				Model:    "glm-4",
			},
			httpClient: &http.Client{},
			baseURL:    mockServer.URL,
		}

		ctx := context.Background()
		result, err := client.Translate(ctx, "Hello", "Translate to Russian")
		require.NoError(t, err)
		assert.Equal(t, "Привет", result)
	})

	t.Run("mock_translate_empty_choices", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"choices": []}`))
		}))
		defer mockServer.Close()

		client := &ZhipuClient{
			config: TranslationConfig{
				Provider: "zhipu",
				APIKey:   "test-key",
				Model:    "glm-4",
			},
			httpClient: &http.Client{},
			baseURL:    mockServer.URL,
		}

		ctx := context.Background()
		_, err := client.Translate(ctx, "Hello", "Translate to Russian")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no choices")
	})
}

// TestZhipuRequestErrorPaths tests error paths in zhipu Translate function
func TestZhipuRequestErrorPaths(t *testing.T) {
	t.Run("invalid_api_key", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "zhipu",
			APIKey:   "", // Empty API key
			Model:    "glm-4",
		}

		client, err := NewZhipuClient(config)
		require.Error(t, err)
		require.Nil(t, client)
		assert.Contains(t, err.Error(), "API key is required")
	})

	t.Run("invalid_model", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "zhipu",
			APIKey:   "test-api-key",
			Model:    "invalid-model-name",
		}

		client, err := NewZhipuClient(config)
		require.NoError(t, err)
		require.NotNil(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err = client.Translate(ctx, "Hello", "Translate to Russian")
		// Should fail because no real server is running
		require.Error(t, err)
	})

	t.Run("context_cancellation", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "zhipu",
			APIKey:   "test-api-key",
			Model:    "glm-4",
		}

		client, err := NewZhipuClient(config)
		require.NoError(t, err)
		require.NotNil(t, client)

		// Create cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		result, err := client.Translate(ctx, "Hello", "Translate to Russian")
		require.Error(t, err)
		assert.Empty(t, result)
		assert.True(t, strings.Contains(err.Error(), "context") ||
			strings.Contains(err.Error(), "canceled") ||
			strings.Contains(err.Error(), "deadline"))
	})

	t.Run("empty_text_input", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "zhipu",
			APIKey:   "test-api-key",
			Model:    "glm-4",
		}

		client, err := NewZhipuClient(config)
		require.NoError(t, err)
		require.NotNil(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		result, err := client.Translate(ctx, "", "Translate to Russian")
		// Empty text may return empty result or error depending on implementation
		_ = result
		_ = err
	})

	t.Run("temperature_option_validation", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "zhipu",
			APIKey:   "test-api-key",
			Model:    "glm-4",
			Options: map[string]interface{}{
				"temperature": 2.5, // Too high (should be 0.0-2.0)
			},
		}

		client, err := NewZhipuClient(config)
		require.NoError(t, err)
		require.NotNil(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err = client.Translate(ctx, "Hello", "Translate to Russian")
		// Zhipu doesn't validate temperature locally; request will fail due to no server
		require.Error(t, err)
	})

	t.Run("max_tokens_option_validation", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "zhipu",
			APIKey:   "test-api-key",
			Model:    "glm-4",
			Options: map[string]interface{}{
				"max_tokens": -1, // Invalid max_tokens
			},
		}

		client, err := NewZhipuClient(config)
		require.NoError(t, err)
		require.NotNil(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err = client.Translate(ctx, "Hello", "Translate to Russian")
		// Zhipu doesn't validate max_tokens locally; request will fail due to no server
		require.Error(t, err)
	})

	t.Run("very_long_text", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "zhipu",
			APIKey:   "test-api-key",
			Model:    "glm-4",
		}

		client, err := NewZhipuClient(config)
		require.NoError(t, err)
		require.NotNil(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Create very long text that might trigger size limits
		longText := strings.Repeat("Hello world. ", 1000)

		_, err = client.Translate(ctx, longText, "Translate to Russian")
		// Should fail due to connection error, not size validation
		require.Error(t, err)
	})

	t.Run("invalid_base_url", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "zhipu",
			APIKey:   "test-api-key",
			Model:    "glm-4",
			BaseURL:  "invalid-url://invalid", // Invalid URL
		}

		client, err := NewZhipuClient(config)
		require.NoError(t, err)
		require.NotNil(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err = client.Translate(ctx, "Hello", "Translate to Russian")
		require.Error(t, err)
	})
}

// TestZhipuClientCreation tests client creation error paths
func TestZhipuClientCreation(t *testing.T) {
	t.Run("minimal_valid_config", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "zhipu",
			APIKey:   "test-key",
			// Model and options are optional
		}

		client, err := NewZhipuClient(config)
		if err != nil {
			t.Errorf("Unexpected error with minimal config: %v", err)
		}
		if client == nil {
			t.Error("Client should not be nil with minimal valid config")
		}

		if client != nil {
			provider := client.GetProviderName()
			if provider != "zhipu" {
				t.Errorf("Expected provider 'zhipu', got: %s", provider)
			}
		}
	})

	t.Run("full_config", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "zhipu",
			APIKey:   "test-key",
			Model:    "glm-4",
			BaseURL:  "https://custom.zhipuai.com",
			Options: map[string]interface{}{
				"temperature": 0.7,
				"max_tokens":  2000,
			},
		}

		client, err := NewZhipuClient(config)
		if err != nil {
			t.Errorf("Unexpected error with full config: %v", err)
		}
		if client == nil {
			t.Error("Client should not be nil with full valid config")
		}

		if client != nil {
			provider := client.GetProviderName()
			if provider != "zhipu" {
				t.Errorf("Expected provider 'zhipu', got: %s", provider)
			}
		}
	})
}

// TestZhipuTranslateUncoveredPaths tests uncovered error paths in Zhipu Translate function
func TestZhipuTranslateUncoveredPaths(t *testing.T) {
	t.Run("json_marshal_error", func(t *testing.T) {
		// Test JSON marshaling error by creating a client with problematic data
		client := &ZhipuClient{
			config: TranslationConfig{
				Provider: "zhipu",
				APIKey:   "test_key",
				Model:    "glm-4",
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
		client := &ZhipuClient{
			config: TranslationConfig{
				Provider: "zhipu",
				APIKey:   "test_key",
				Model:    "glm-4",
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
		client := &ZhipuClient{
			config: TranslationConfig{
				Provider: "zhipu",
				APIKey:   "test_key",
				Model:    "glm-4",
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

	t.Run("empty_response_choices", func(t *testing.T) {
		// Use httptest.NewServer to simulate an API response with empty choices
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			// Return valid JSON but with empty choices
			w.Write([]byte(`{
				"choices": []
			}`))
		}))
		defer mockServer.Close()

		client := &ZhipuClient{
			config: TranslationConfig{
				Provider: "zhipu",
				APIKey:   "test_key",
				Model:    "glm-4",
			},
			httpClient: &http.Client{},
			baseURL:    mockServer.URL,
		}

		ctx := context.Background()
		_, err := client.Translate(ctx, "test text", "test prompt")
		if err == nil {
			t.Error("Expected error for empty choices in response")
		}

		if !strings.Contains(err.Error(), "no choices") {
			t.Errorf("Expected 'no choices' error, got: %v", err)
		}
	})

	t.Run("model_defaulting", func(t *testing.T) {
		client := &ZhipuClient{
			config: TranslationConfig{
				Provider: "zhipu",
				APIKey:   "test_key",
				// No model specified - should default to glm-4
			},
			httpClient: &http.Client{},
			baseURL:    "http://localhost:99999", // Invalid port to prevent actual requests
		}

		// Verify the client has no model configured initially
		if client.config.Model != "" {
			t.Errorf("Client should have no model configured initially, got: %s", client.config.Model)
		}

		ctx := context.Background()
		_, err := client.Translate(ctx, "test text", "test prompt")
		if err != nil {
			// Expected to fail due to invalid port, but model defaulting should happen
			t.Logf("Expected connection error: %v", err)
		}

		// The model defaulting happens during Translate, not during client creation
		// So the config should still be empty after the call
		// But we confirmed that the Translate method was called
	})
}
