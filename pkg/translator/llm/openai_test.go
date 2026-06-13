package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newOpenAIMockServer returns an httptest server that answers /chat/completions
// with a valid OpenAI-shaped response echoing a fixed translation. It lets the
// error/success-path tests exercise the real request-build + response-parse
// logic deterministically and OFFLINE (§11.4.98) instead of dialing
// api.openai.com (which previously returned a live 401, masking the assertion).
func newOpenAIMockServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Assert the client actually built a well-formed request so the test
		// keeps its teeth: a parsing regression would change these.
		if r.URL.Path != "/chat/completions" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var req OpenAIRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		resp := OpenAIResponse{
			Choices: []Choice{{Message: Message{Role: "assistant", Content: "Привет"}}},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestOpenAITranslateErrorPaths tests error paths in OpenAI Translate function
func TestOpenAITranslateErrorPaths(t *testing.T) {
	// Test with invalid configuration to trigger error paths
	t.Run("invalid_api_key", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "openai",
			APIKey:   "", // Empty API key to trigger error
			Model:    "gpt-4",
		}

		client, err := NewOpenAIClient(config)
		require.Error(t, err)
		require.Nil(t, client)
		assert.Contains(t, err.Error(), "API key is required")
	})

	t.Run("empty_text_input", func(t *testing.T) {
		srv := newOpenAIMockServer(t)
		config := TranslationConfig{
			Provider: "openai",
			APIKey:   "test-api-key",
			Model:    "gpt-4",
			BaseURL:  srv.URL,
		}

		client, err := NewOpenAIClient(config)
		require.NoError(t, err)
		require.NotNil(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Empty input still produces a well-formed request the client parses
		// offline; the client does not short-circuit empty text.
		result, err := client.Translate(ctx, "", "Translate to Russian")
		require.NoError(t, err)
		assert.Equal(t, "Привет", result)
	})

	t.Run("context_cancellation", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "openai",
			APIKey:   "test-api-key",
			Model:    "gpt-4",
		}

		client, err := NewOpenAIClient(config)
		require.NoError(t, err)
		require.NotNil(t, client)

		// Create cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		result, err := client.Translate(ctx, "Hello world", "Translate to Russian")
		require.Error(t, err)
		assert.Empty(t, result)
	})

	t.Run("malformed_model_name", func(t *testing.T) {
		srv := newOpenAIMockServer(t)
		config := TranslationConfig{
			Provider: "openai",
			APIKey:   "test-api-key",
			Model:    "", // Empty model name to trigger default logic (-> gpt-4)
			BaseURL:  srv.URL,
		}

		client, err := NewOpenAIClient(config)
		require.NoError(t, err)
		require.NotNil(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Empty model defaults to gpt-4 inside Translate; request succeeds offline.
		result, err := client.Translate(ctx, "Hello", "Translate to Russian")
		require.NoError(t, err)
		assert.Equal(t, "Привет", result)
	})

	t.Run("temperature_options", func(t *testing.T) {
		srv := newOpenAIMockServer(t)
		config := TranslationConfig{
			Provider: "openai",
			APIKey:   "test-api-key",
			Model:    "gpt-4",
			BaseURL:  srv.URL,
			Options: map[string]interface{}{
				"temperature": 1.5, // High (but valid: 0.0-2.0) temperature
			},
		}

		client, err := NewOpenAIClient(config)
		require.NoError(t, err)
		require.NotNil(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Temperature is carried into the request; the client succeeds offline.
		result, err := client.Translate(ctx, "Hello", "Translate to Russian")
		require.NoError(t, err)
		assert.Equal(t, "Привет", result)
	})

	t.Run("max_tokens_override", func(t *testing.T) {
		srv := newOpenAIMockServer(t)
		config := TranslationConfig{
			Provider: "openai",
			APIKey:   "test-api-key",
			Model:    "gpt-4",
			BaseURL:  srv.URL,
			Options: map[string]interface{}{
				"max_tokens": -1, // Negative max_tokens override
			},
		}

		client, err := NewOpenAIClient(config)
		require.NoError(t, err)
		require.NotNil(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// The max_tokens override is applied (omitempty drops negative); request
		// is parsed by the mock and the response is returned offline.
		result, err := client.Translate(ctx, "Hello", "Translate to Russian")
		require.NoError(t, err)
		assert.Equal(t, "Привет", result)
	})

	// api_error_status keeps explicit coverage of the non-200 error path that the
	// old (network-dependent) tests relied on the live 401 to provide — now via a
	// local server returning 401, fully offline (§11.4.98).
	t.Run("api_error_status", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"message":"Invalid API key"}}`))
		}))
		t.Cleanup(srv.Close)

		client, err := NewOpenAIClient(TranslationConfig{
			Provider: "openai",
			APIKey:   "test-api-key",
			Model:    "gpt-4",
			BaseURL:  srv.URL,
		})
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err = client.Translate(ctx, "Hello", "Translate to Russian")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "status 401")
	})
}

// TestOpenAIClientCreation tests client creation error paths
func TestOpenAIClientCreation(t *testing.T) {
	t.Run("invalid_provider", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "invalid-provider",
			APIKey:   "test-key",
			Model:    "gpt-4",
		}

		client, err := NewOpenAIClient(config)
		// The function doesn't validate provider, it just creates a client
		if err != nil {
			t.Errorf("Unexpected error with invalid provider: %v", err)
		}
		if client == nil {
			t.Error("Client should not be nil even with invalid provider")
		}

		// The provider name should still be "openai" regardless of config
		if client != nil && client.GetProviderName() != "openai" {
			t.Errorf("Expected provider 'openai', got: %s", client.GetProviderName())
		}
	})

	t.Run("minimal_valid_config", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "openai",
			APIKey:   "test-key",
			// Model and options are optional
		}

		client, err := NewOpenAIClient(config)
		if err != nil {
			t.Errorf("Unexpected error with minimal config: %v", err)
		}
		if client == nil {
			t.Error("Client should not be nil with minimal valid config")
		}

		if client != nil {
			provider := client.GetProviderName()
			if provider != "openai" {
				t.Errorf("Expected provider 'openai', got: %s", provider)
			}
		}
	})

	t.Run("full_config", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "openai",
			APIKey:   "test-key",
			Model:    "gpt-4-turbo",
			Options: map[string]interface{}{
				"temperature": 0.7,
				"max_tokens":  2000,
				"timeout":     60 * time.Second,
			},
		}

		client, err := NewOpenAIClient(config)
		if err != nil {
			t.Errorf("Unexpected error with full config: %v", err)
		}
		if client == nil {
			t.Error("Client should not be nil with full valid config")
		}

		if client != nil {
			provider := client.GetProviderName()
			if provider != "openai" {
				t.Errorf("Expected provider 'openai', got: %s", provider)
			}
		}
	})
}

// Helper function to check if error is context-related
func containsContextError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "context") ||
		strings.Contains(errStr, "canceled") ||
		strings.Contains(errStr, "deadline")
}
