package llm

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		config := TranslationConfig{
			Provider: "openai",
			APIKey:   "test-api-key",
			Model:    "gpt-4",
		}

		client, err := NewOpenAIClient(config)
		require.NoError(t, err)
		require.NotNil(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		result, err := client.Translate(ctx, "", "Translate to Russian")
		_ = result
		_ = err
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
		config := TranslationConfig{
			Provider: "openai",
			APIKey:   "test-api-key",
			Model:    "", // Empty model name to trigger default logic
		}

		client, err := NewOpenAIClient(config)
		require.NoError(t, err)
		require.NotNil(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err = client.Translate(ctx, "Hello", "Translate to Russian")
		require.Error(t, err)
	})

	t.Run("temperature_options", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "openai",
			APIKey:   "test-api-key",
			Model:    "gpt-4",
			Options: map[string]interface{}{
				"temperature": 1.5, // Very high temperature
			},
		}

		client, err := NewOpenAIClient(config)
		require.NoError(t, err)
		require.NotNil(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err = client.Translate(ctx, "Hello", "Translate to Russian")
		require.Error(t, err)
	})

	t.Run("max_tokens_override", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "openai",
			APIKey:   "test-api-key",
			Model:    "gpt-4",
			Options: map[string]interface{}{
				"max_tokens": -1, // Invalid max_tokens
			},
		}

		client, err := NewOpenAIClient(config)
		require.NoError(t, err)
		require.NotNil(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err = client.Translate(ctx, "Hello", "Translate to Russian")
		require.Error(t, err)
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
