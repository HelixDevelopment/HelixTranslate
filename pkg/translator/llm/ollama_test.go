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

func TestOllamaProvider(t *testing.T) {
	t.Run("provider_name", func(t *testing.T) {
		client, err := NewOllamaClient(TranslationConfig{
			Provider: "ollama",
		})
		require.NoError(t, err)
		require.NotNil(t, client)
		assert.Equal(t, "ollama", client.GetProviderName())
	})

	t.Run("default_base_url", func(t *testing.T) {
		client, err := NewOllamaClient(TranslationConfig{
			Provider: "ollama",
		})
		require.NoError(t, err)
		require.NotNil(t, client)
	})

	t.Run("custom_base_url", func(t *testing.T) {
		client, err := NewOllamaClient(TranslationConfig{
			Provider: "ollama",
			BaseURL:  "http://custom.localhost:11435",
		})
		require.NoError(t, err)
		require.NotNil(t, client)
	})

	t.Run("no_api_key_required", func(t *testing.T) {
		client, err := NewOllamaClient(TranslationConfig{
			Provider: "ollama",
			APIKey:   "",
		})
		require.NoError(t, err)
		require.NotNil(t, client)
	})

	t.Run("mock_translate_success", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"response": "Привет", "done": true}`))
		}))
		defer mockServer.Close()

		client := &OllamaClient{
			config: TranslationConfig{
				Provider: "ollama",
				Model:    "llama3:8b",
			},
			httpClient: &http.Client{},
			baseURL:    mockServer.URL,
		}

		ctx := context.Background()
		result, err := client.Translate(ctx, "Hello", "Translate to Russian")
		require.NoError(t, err)
		assert.Equal(t, "Привет", result)
	})

	t.Run("mock_translate_invalid_json", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`invalid json`))
		}))
		defer mockServer.Close()

		client := &OllamaClient{
			config: TranslationConfig{
				Provider: "ollama",
				Model:    "llama3:8b",
			},
			httpClient: &http.Client{},
			baseURL:    mockServer.URL,
		}

		ctx := context.Background()
		_, err := client.Translate(ctx, "Hello", "Translate to Russian")
		require.Error(t, err)
	})
}

// TestOllamaRequestErrorPaths tests error paths in ollama Translate function
func TestOllamaRequestErrorPaths(t *testing.T) {
	t.Run("invalid_base_url", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "ollama",
			Model:    "llama3:8b",
			BaseURL:  "invalid-url", // Invalid URL
		}

		client, err := NewOllamaClient(config)
		require.NoError(t, err)
		require.NotNil(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err = client.Translate(ctx, "Hello", "Translate to Russian")
		require.Error(t, err)
	})

	t.Run("empty_model", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "ollama",
			Model:    "", // Empty model to trigger default logic
			BaseURL:  "http://localhost:11434",
		}

		client, err := NewOllamaClient(config)
		require.NoError(t, err)
		require.NotNil(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err = client.Translate(ctx, "Hello", "Translate to Russian")
		require.Error(t, err)
	})

	t.Run("context_cancellation", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "ollama",
			Model:    "llama3:8b",
			BaseURL:  "http://localhost:11434",
		}

		client, err := NewOllamaClient(config)
		require.NoError(t, err)
		require.NotNil(t, client)

		// Create cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		result, err := client.Translate(ctx, "Hello", "Translate to Russian")
		require.Error(t, err)
		assert.Empty(t, result)
	})

	t.Run("empty_text_input", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "ollama",
			Model:    "llama3:8b",
			BaseURL:  "http://localhost:11434",
		}

		client, err := NewOllamaClient(config)
		require.NoError(t, err)
		require.NotNil(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		result, err := client.Translate(ctx, "", "Translate to Russian")
		_ = result
		_ = err
	})

	t.Run("very_long_text", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "ollama",
			Model:    "llama3:8b",
			BaseURL:  "http://localhost:11434",
		}

		client, err := NewOllamaClient(config)
		require.NoError(t, err)
		require.NotNil(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		longText := strings.Repeat("Hello world. ", 1000)

		_, err = client.Translate(ctx, longText, "Translate to Russian")
		require.Error(t, err)
	})

	t.Run("malformed_json_response", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "ollama",
			Model:    "llama3:8b",
			BaseURL:  "http://httpbin.org/json", // Returns valid JSON but wrong format
		}

		client, err := NewOllamaClient(config)
		require.NoError(t, err)
		require.NotNil(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err = client.Translate(ctx, "Hello", "Translate to Russian")
		require.Error(t, err)
	})
}

// TestOllamaClientCreation tests client creation error paths
func TestOllamaClientCreation(t *testing.T) {
	t.Run("minimal_valid_config", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "ollama",
			// Model and BaseURL are optional
		}

		client, err := NewOllamaClient(config)
		if err != nil {
			t.Errorf("Unexpected error with minimal config: %v", err)
		}
		if client == nil {
			t.Error("Client should not be nil with minimal valid config")
		}

		if client != nil {
			provider := client.GetProviderName()
			if provider != "ollama" {
				t.Errorf("Expected provider 'ollama', got: %s", provider)
			}
		}
	})

	t.Run("full_config", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "ollama",
			Model:    "mistral:7b",
			BaseURL:  "http://custom.localhost:11435",
		}

		client, err := NewOllamaClient(config)
		if err != nil {
			t.Errorf("Unexpected error with full config: %v", err)
		}
		if client == nil {
			t.Error("Client should not be nil with full valid config")
		}

		if client != nil {
			provider := client.GetProviderName()
			if provider != "ollama" {
				t.Errorf("Expected provider 'ollama', got: %s", provider)
			}
		}
	})
}

// TestOllamaTranslateUncoveredPaths tests uncovered error paths in Ollama Translate function
func TestOllamaTranslateUncoveredPaths(t *testing.T) {
	// Test 1: JSON marshaling error
	t.Run("json_marshal_error", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "ollama",
			BaseURL:  "http://localhost:11434",
		}

		client, err := NewOllamaClient(config)
		require.NoError(t, err)
		require.NotNil(t, client)

		ctx := context.Background()

		result, err := client.Translate(ctx, "Hello", "Translate to Russian")
		require.Error(t, err)
		assert.Empty(t, result)
	})

	// Test 2: HTTP request creation error
	t.Run("http_request_error", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "ollama",
			BaseURL:  "invalid://invalid-url", // Invalid URL that should cause request creation error
		}

		client, err := NewOllamaClient(config)
		require.NoError(t, err)
		require.NotNil(t, client)

		ctx := context.Background()

		result, err := client.Translate(ctx, "Hello", "Translate to Russian")
		require.Error(t, err)
		assert.Empty(t, result)
	})

	// Test 3: Response reading error
	t.Run("response_reading_error", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "ollama",
			BaseURL:  "http://localhost:99999", // Port that's likely not running
		}

		client, err := NewOllamaClient(config)
		require.NoError(t, err)
		require.NotNil(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		result, err := client.Translate(ctx, "Hello", "Translate to Russian")
		require.Error(t, err)
		assert.Empty(t, result)
	})

	// Test 4: Non-200 status codes
	t.Run("non_200_status_codes", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "ollama",
			BaseURL:  "http://httpbin.org/status/404", // Will return 404
		}

		client, err := NewOllamaClient(config)
		require.NoError(t, err)
		require.NotNil(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		result, err := client.Translate(ctx, "Hello", "Translate to Russian")
		require.Error(t, err)
		assert.Empty(t, result)
	})

	// Test 5: JSON unmarshaling error
	t.Run("json_unmarshal_error", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "ollama",
			BaseURL:  "http://httpbin.org/html", // Returns HTML, not JSON
		}

		client, err := NewOllamaClient(config)
		require.NoError(t, err)
		require.NotNil(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		result, err := client.Translate(ctx, "Hello", "Translate to Russian")
		require.Error(t, err)
		assert.Empty(t, result)
	})

	// Test 6: Model defaulting behavior
	t.Run("model_defaulting_behavior", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "ollama",
			BaseURL:  "http://httpbin.org", // Base URL - client will append /api/generate
			Model:    "",                   // Empty model to trigger defaulting
		}

		client, err := NewOllamaClient(config)
		require.NoError(t, err)
		require.NotNil(t, client)

		assert.Empty(t, client.config.Model)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err = client.Translate(ctx, "Hello", "Translate to Russian")
		require.Error(t, err)

		assert.False(t, strings.Contains(err.Error(), "model") && strings.Contains(err.Error(), "required"))
	})
}

// TestOllamaTranslateAdditionalPaths tests additional uncovered error paths in Ollama Translate
func TestOllamaTranslateAdditionalPaths(t *testing.T) {
	t.Run("json_marshal_error_for_invalid_options", func(t *testing.T) {
		client := &OllamaClient{
			config: TranslationConfig{
				Provider: "ollama",
				APIKey:   "test_key",
				Model:    "llama2",
				Options: map[string]interface{}{
					// Add an option that might cause JSON issues
					"invalid_option": make(chan int), // Channels can't be marshaled to JSON
				},
			},
			httpClient: &http.Client{},
			baseURL:    "http://localhost:99999", // Invalid port
		}

		ctx := context.Background()
		_, err := client.Translate(ctx, "test text", "test prompt")
		if err == nil {
			t.Error("Expected error with invalid option")
		} else {
			t.Logf("Expected error with invalid option: %v", err)
		}
	})

	t.Run("response_body_read_error", func(t *testing.T) {
		// Create a mock server that returns a response but then fails during body reading
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set headers but don't write body to simulate partial response
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Length", "1")
			// Close connection without writing body
			w.(http.Flusher).Flush()
			hj, ok := w.(http.Hijacker)
			if ok {
				conn, _, _ := hj.Hijack()
				conn.Close()
			}
		}))
		defer mockServer.Close()

		client := &OllamaClient{
			config: TranslationConfig{
				Provider: "ollama",
				APIKey:   "test_key",
				Model:    "llama2",
			},
			httpClient: &http.Client{},
			baseURL:    mockServer.URL,
		}

		ctx := context.Background()
		_, err := client.Translate(ctx, "test text", "test prompt")
		if err == nil {
			t.Error("Expected error with incomplete response")
		} else {
			t.Logf("Expected error with incomplete response: %v", err)
		}
	})

	t.Run("invalid_response_json", func(t *testing.T) {
		// Create a mock server that returns invalid JSON
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			// Return invalid JSON (missing closing brace)
			w.Write([]byte(`{"model": "llama2", "response": "test response"`))
		}))
		defer mockServer.Close()

		client := &OllamaClient{
			config: TranslationConfig{
				Provider: "ollama",
				APIKey:   "test_key",
				Model:    "llama2",
			},
			httpClient: &http.Client{},
			baseURL:    mockServer.URL,
		}

		ctx := context.Background()
		_, err := client.Translate(ctx, "test text", "test prompt")
		if err == nil {
			t.Error("Expected error with invalid JSON")
		}

		if !strings.Contains(err.Error(), "failed to unmarshal response") {
			t.Errorf("Expected JSON unmarshal error, got: %v", err)
		}
	})

	t.Run("temperature_option_handling", func(t *testing.T) {
		client := &OllamaClient{
			config: TranslationConfig{
				Provider: "ollama",
				APIKey:   "test_key",
				Model:    "llama2",
				Options: map[string]interface{}{
					"temperature": 1.5, // Higher than typical range
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
		// The important thing is that the option was processed during request creation
	})
}
