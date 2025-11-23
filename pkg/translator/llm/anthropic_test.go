package llm

import (
	"context"
	"net/http"
	"testing"
	"time"

	"digital.vasic.translator/pkg/translator"
)

func TestAnthropicClient(t *testing.T) {
	tests := []struct {
		name    string
		config  translator.TranslationConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: translator.TranslationConfig{
				Provider: "anthropic",
				APIKey:   "test-key",
				Model:    "claude-3-sonnet-20240229",
				BaseURL:  "https://api.anthropic.com",
			},
			wantErr: false,
		},
		{
			name: "missing api key",
			config: translator.TranslationConfig{
				Provider: "anthropic",
				Model:    "claude-3-sonnet-20240229",
			},
			wantErr: true,
		},
		{
			name: "missing model",
			config: translator.TranslationConfig{
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

	client, err := NewAnthropicClient(translator.TranslationConfig{
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
		config: translator.TranslationConfig{
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
		config translator.TranslationConfig
	}{
		{
			name: "invalid api key",
			config: translator.TranslationConfig{
				Provider: "anthropic",
				APIKey:   "invalid-key",
				Model:    "claude-3-sonnet-20240229",
				BaseURL:  "https://api.anthropic.com",
			},
		},
		{
			name: "invalid base url",
			config: translator.TranslationConfig{
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
		config: translator.TranslationConfig{
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
			client, err := NewAnthropicClient(translator.TranslationConfig{
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
			client, err := NewAnthropicClient(translator.TranslationConfig{
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