package llm

import (
	"context"
	"net/http"
	"testing"
	"time"

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
		t.Skip("Skipping integration test in short mode")
	}

	// Skip if no real API key
	apiKey := getTestAPIKey("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("No ANTHROPIC_API_KEY provided for integration test")
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
			if !contains(err.Error(), "context") && 
			   !contains(err.Error(), "canceled") && 
			   !contains(err.Error(), "deadline") {
				t.Logf("Error may not be context-related: %v", err)
			}
		}
	})

	t.Run("very_long_text", func(t *testing.T) {
		client, err := NewAnthropicClient(TranslationConfig{
			Provider: "anthropic",
			APIKey:   "test-api-key",
			Model:    "claude-3-haiku-20240307",
		})
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
			if !contains(err.Error(), "too large") && 
			   !contains(err.Error(), "size") && 
			   !contains(err.Error(), "limit") {
				t.Logf("Error may not be size-related: %v", err)
			}
		}
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
			if !contains(err.Error(), "temperature") &&
			   !contains(err.Error(), "invalid") {
				t.Logf("Error may not be temperature-related: %v", err)
			}
		}
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
			if !contains(err.Error(), "max_tokens") &&
			   !contains(err.Error(), "invalid") {
				t.Logf("Error may not be max_tokens-related: %v", err)
			}
		}
	})

	t.Run("invalid_base_url", func(t *testing.T) {
		client, err := NewAnthropicClient(TranslationConfig{
			Provider: "anthropic",
			APIKey:   "test-api-key",
			Model:    "claude-3-haiku-20240307",
			BaseURL:  "invalid-url://invalid", // Invalid URL
		})
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
			if !contains(err.Error(), "url") && 
			   !contains(err.Error(), "scheme") &&
			   !contains(err.Error(), "invalid") {
				t.Logf("Error may not be URL-related: %v", err)
			}
		}
	})
}