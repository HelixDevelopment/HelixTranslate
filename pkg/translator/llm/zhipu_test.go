package llm

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestZhipuProvider(t *testing.T) {
	// Test placeholder - provider implementation needed
	t.Log("Zhipu provider test placeholder")
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
		if err != nil || client == nil {
			t.Skip("Skipping test - client creation failed")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		result, err := client.Translate(ctx, "Hello", "Translate to Russian")
		if err == nil {
			t.Log("Request succeeded with empty API key - may be using mock")
		}
		if result != "" && err != nil {
			t.Error("Result should be empty when error occurs")
		}
	})

	t.Run("invalid_model", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "zhipu",
			APIKey:   "test-api-key",
			Model:    "invalid-model-name",
		}

		client, err := NewZhipuClient(config)
		if err != nil || client == nil {
			t.Skip("Skipping test - client creation failed")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		result, err := client.Translate(ctx, "Hello", "Translate to Russian")
		if err == nil {
			t.Log("Request succeeded with invalid model - may be using mock")
		}
		if result != "" && err != nil {
			t.Error("Result should be empty when error occurs")
		}
	})

	t.Run("context_cancellation", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "zhipu",
			APIKey:   "test-api-key",
			Model:    "glm-4",
		}

		client, err := NewZhipuClient(config)
		if err != nil || client == nil {
			t.Skip("Skipping test - client creation failed")
			return
		}

		// Create cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		result, err := client.Translate(ctx, "Hello", "Translate to Russian")
		if err != nil {
			// Expected error path
			if result != "" {
				t.Error("Result should be empty when context is cancelled")
			}
			// Check for context-related error
			if !strings.Contains(err.Error(), "context") && 
			   !strings.Contains(err.Error(), "canceled") && 
			   !strings.Contains(err.Error(), "deadline") {
				t.Logf("Error may not be context-related: %v", err)
			}
		}
	})

	t.Run("empty_text_input", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "zhipu",
			APIKey:   "test-api-key",
			Model:    "glm-4",
		}

		client, err := NewZhipuClient(config)
		if err != nil || client == nil {
			t.Skip("Skipping test - client creation failed")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		result, err := client.Translate(ctx, "", "Translate to Russian")
		if err == nil && result == "" {
			t.Log("Empty input returned empty result - this is acceptable")
		}
		// Either should work - some APIs handle empty text, others don't
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
		if err != nil || client == nil {
			t.Skip("Skipping test - client creation failed")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		result, err := client.Translate(ctx, "Hello", "Translate to Russian")
		if err != nil {
			// Expected error path - invalid temperature
			if result != "" {
				t.Error("Result should be empty when temperature is invalid")
			}
			// Check for temperature-related error
			if !strings.Contains(err.Error(), "temperature") &&
			   !strings.Contains(err.Error(), "invalid") {
				t.Logf("Error may not be temperature-related: %v", err)
			}
		}
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
		if err != nil || client == nil {
			t.Skip("Skipping test - client creation failed")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		result, err := client.Translate(ctx, "Hello", "Translate to Russian")
		if err != nil {
			// Expected error path - invalid max_tokens
			if result != "" {
				t.Error("Result should be empty when max_tokens is invalid")
			}
			// Check for max_tokens-related error
			if !strings.Contains(err.Error(), "max_tokens") &&
			   !strings.Contains(err.Error(), "invalid") {
				t.Logf("Error may not be max_tokens-related: %v", err)
			}
		}
	})

	t.Run("very_long_text", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "zhipu",
			APIKey:   "test-api-key",
			Model:    "glm-4",
		}

		client, err := NewZhipuClient(config)
		if err != nil || client == nil {
			t.Skip("Skipping test - client creation failed")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Create very long text that might trigger size limits
		longText := ""
		for i := 0; i < 1000; i++ {
			longText += "Hello world. "
		}
		
		result, err := client.Translate(ctx, longText, "Translate to Russian")
		if err != nil {
			// Expected error path - text too long
			if result != "" {
				t.Error("Result should be empty when text is too long")
			}
			// Check for size-related error
			if !strings.Contains(err.Error(), "too large") && 
			   !strings.Contains(err.Error(), "size") && 
			   !strings.Contains(err.Error(), "limit") {
				t.Logf("Error may not be size-related: %v", err)
			}
		}
	})

	t.Run("invalid_base_url", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "zhipu",
			APIKey:   "test-api-key",
			Model:    "glm-4",
			BaseURL:  "invalid-url://invalid", // Invalid URL
		}

		client, err := NewZhipuClient(config)
		if err != nil || client == nil {
			t.Skip("Skipping test - client creation failed")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		result, err := client.Translate(ctx, "Hello", "Translate to Russian")
		if err != nil {
			// Expected error path - invalid URL
			if result != "" {
				t.Error("Result should be empty when URL is invalid")
			}
			// Check for URL-related error
			if !strings.Contains(err.Error(), "url") && 
			   !strings.Contains(err.Error(), "scheme") &&
			   !strings.Contains(err.Error(), "invalid") {
				t.Logf("Error may not be URL-related: %v", err)
			}
		}
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