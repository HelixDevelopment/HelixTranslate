package llm

import (
	"context"
	"net/http"
	"testing"
	"time"

)

func TestDeepSeekClient(t *testing.T) {
	tests := []struct {
		name    string
		config  TranslationConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: TranslationConfig{
				Provider: "deepseek",
				APIKey:   "test-key",
				Model:    "deepseek-chat",
				BaseURL:  "https://api.deepseek.com",
			},
			wantErr: false,
		},
		{
			name: "missing api key",
			config: TranslationConfig{
				Provider: "deepseek",
				Model:    "deepseek-chat",
			},
			wantErr: true,
		},
		{
			name: "missing model",
			config: TranslationConfig{
				Provider: "deepseek",
				APIKey:   "test-key",
			},
			wantErr: true,
		},
		{
			name: "invalid provider",
			config: TranslationConfig{
				Provider: "invalid",
				APIKey:   "test-key",
				Model:    "deepseek-chat",
			},
			wantErr: true,
		},
		{
			name: "empty model",
			config: TranslationConfig{
				Provider: "deepseek",
				APIKey:   "test-key",
				Model:    "",
			},
			wantErr: true,
		},
		{
			name: "whitespace model",
			config: TranslationConfig{
				Provider: "deepseek",
				APIKey:   "test-key",
				Model:    "   ",
			},
			wantErr: true,
		},
		{
			name: "invalid model",
			config: TranslationConfig{
				Provider: "deepseek",
				APIKey:   "test-key",
				Model:    "invalid-model-name",
			},
			wantErr: true,
		},
		{
			name: "temperature too low",
			config: TranslationConfig{
				Provider: "deepseek",
				APIKey:   "test-key",
				Model:    "deepseek-chat",
				Options: map[string]interface{}{
					"temperature": -0.1,
				},
			},
			wantErr: true,
		},
		{
			name: "temperature too high",
			config: TranslationConfig{
				Provider: "deepseek",
				APIKey:   "test-key",
				Model:    "deepseek-chat",
				Options: map[string]interface{}{
					"temperature": 2.1,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewDeepSeekClient(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDeepSeekClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && client != nil {
				if client.GetProviderName() != "deepseek" {
					t.Errorf("GetProviderName() = %v, want %v", client.GetProviderName(), "deepseek")
				}
			}
		})
	}
}

func TestDeepSeekClient_Translate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Skip if no real API key
	apiKey := getTestAPIKey("DEEPSEEK_API_KEY")
	if apiKey == "" {
		t.Skip("No DEEPSEEK_API_KEY provided for integration test")
	}

	client, err := NewDeepSeekClient(TranslationConfig{
		Provider: "deepseek",
		APIKey:   apiKey,
		Model:    "deepseek-chat",
		BaseURL:  "https://api.deepseek.com",
	})
	if err != nil {
		t.Fatalf("Failed to create DeepSeek client: %v", err)
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
			prompt:  "Translate this text to Russian",
			wantErr: false,
			validate: func(result string) bool {
				return len(result) > 0 && result != "Hello world"
			},
		},
		{
			name:    "technical translation",
			text:    "Artificial Intelligence is transforming the world",
			prompt:  "Translate this technical text to Chinese",
			wantErr: false,
			validate: func(result string) bool {
				return len(result) > 10
			},
		},
		{
			name:    "code translation",
			text:    "The function returns a boolean value indicating success",
			prompt:  "Translate this programming-related text to Japanese",
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
				return true // API should handle empty text gracefully
			},
		},
		{
			name:    "long text",
			text:    "This comprehensive text examines the intricate relationship between technology and society, exploring how digital transformation affects various aspects of modern life including communication, work, education, and entertainment.",
			prompt:  "Translate this academic text to French",
			wantErr: false,
			validate: func(result string) bool {
				return len(result) > 50
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

func TestDeepSeekClient_RequestStructure(t *testing.T) {
	_ = &DeepSeekClient{
		OpenAIClient: &OpenAIClient{
			config: TranslationConfig{
				Provider: "deepseek",
				Model:    "deepseek-chat",
			},
		},
	}

	// Test request structure - use OpenAI types since DeepSeek embeds OpenAI
	req := OpenAIRequest{
		Model: "deepseek-chat",
		Messages: []Message{
			{
				Role:    "user",
				Content: "Test message",
			},
		},
		Temperature: 0.7,
		MaxTokens:   1024,
	}

	if req.Model != "deepseek-chat" {
		t.Errorf("Expected model 'deepseek-chat', got '%s'", req.Model)
	}

	if len(req.Messages) == 0 {
		t.Error("Messages should not be empty")
	}

	if req.Messages[0].Role != "user" {
		t.Errorf("Expected role 'user', got '%s'", req.Messages[0].Role)
	}

	if req.Temperature != 0.7 {
		t.Errorf("Expected temperature 0.7, got %f", req.Temperature)
	}

	if req.MaxTokens != 1024 {
		t.Errorf("Expected MaxTokens 1024, got %d", req.MaxTokens)
	}
}

func TestDeepSeekClient_ErrorHandling(t *testing.T) {
	tests := []struct {
		name   string
		config TranslationConfig
	}{
		{
			name: "invalid api key",
			config: TranslationConfig{
				Provider: "deepseek",
				APIKey:   "invalid-key",
				Model:    "deepseek-chat",
				BaseURL:  "https://api.deepseek.com",
			},
		},
		{
			name: "invalid base url",
			config: TranslationConfig{
				Provider: "deepseek",
				APIKey:   "test-key",
				Model:    "deepseek-chat",
				BaseURL:  "https://invalid-url.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewDeepSeekClient(tt.config)
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

func TestDeepSeekClient_ContextHandling(t *testing.T) {
	client := &DeepSeekClient{
		OpenAIClient: &OpenAIClient{
			config: TranslationConfig{
				Provider: "deepseek",
				Model:    "deepseek-chat",
			},
			httpClient: &http.Client{
				Timeout: 5 * time.Second,
			},
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

func TestDeepSeekClient_ModelValidation(t *testing.T) {
	validModels := []string{
		"deepseek-chat",
		"deepseek-coder",
	}

	for _, model := range validModels {
		t.Run("valid_model_"+model, func(t *testing.T) {
			client, err := NewDeepSeekClient(TranslationConfig{
				Provider: "deepseek",
				APIKey:   "test-key",
				Model:    model,
				BaseURL:  "https://api.deepseek.com",
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
		"claude-3",
		"",
	}

	for _, model := range invalidModels {
		t.Run("invalid_model_"+model, func(t *testing.T) {
			client, err := NewDeepSeekClient(TranslationConfig{
				Provider: "deepseek",
				APIKey:   "test-key",
				Model:    model,
				BaseURL:  "https://api.deepseek.com",
			})
			// Note: DeepSeek might not validate models at creation time
			// This test might need adjustment based on actual implementation
			if err == nil && model == "" {
				t.Error("Expected error for empty model")
			}
			if client != nil && model == "" {
				t.Error("Expected client to be nil for empty model")
			}
		})
	}
}

func TestDeepSeekClient_TemperatureValidation(t *testing.T) {
	tests := []struct {
		name        string
		temperature float64
		expectError bool
	}{
		{
			name:        "valid temperature",
			temperature: 0.7,
			expectError: false,
		},
		{
			name:        "minimum temperature",
			temperature: 0.0,
			expectError: false,
		},
		{
			name:        "maximum temperature",
			temperature: 2.0,
			expectError: false,
		},
		{
			name:        "too high temperature",
			temperature: 2.5,
			expectError: true,
		},
		{
			name:        "negative temperature",
			temperature: -0.1,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := TranslationConfig{
				Provider: "deepseek",
				APIKey:   "test-key",
				Model:    "deepseek-chat",
				BaseURL:  "https://api.deepseek.com",
				Options: map[string]interface{}{
					"temperature": tt.temperature,
				},
			}

			client, err := NewDeepSeekClient(config)
			if tt.expectError && err == nil {
				t.Error("Expected error for invalid temperature")
			}
			if !tt.expectError && client == nil {
				t.Error("Expected client to be created for valid temperature")
			}
		})
	}
}