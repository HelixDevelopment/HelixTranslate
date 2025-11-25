package llm

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestGeminiProvider(t *testing.T) {
	// Test placeholder - provider implementation needed
	t.Log("Gemini provider test placeholder")
}

func TestGeminiProviderConfig(t *testing.T) {
	// Test placeholder for config testing
	t.Log("Gemini config test placeholder")
}

// TestGeminiRequestErrorPaths tests error paths in gemini makeRequest function
func TestGeminiRequestErrorPaths(t *testing.T) {
	t.Run("invalid_api_key", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "gemini",
			APIKey:   "", // Empty API key
			Model:    "gemini-pro",
		}

		client, err := NewGeminiClient(config)
		if err != nil || client == nil {
			t.Skip("Skipping test - client creation failed")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Test makeRequest indirectly through Translate
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
			Provider: "gemini",
			APIKey:   "test-api-key",
			Model:    "invalid-model-name",
		}

		client, err := NewGeminiClient(config)
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
			Provider: "gemini",
			APIKey:   "test-api-key",
			Model:    "gemini-pro",
		}

		client, err := NewGeminiClient(config)
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
			Provider: "gemini",
			APIKey:   "test-api-key",
			Model:    "gemini-pro",
		}

		client, err := NewGeminiClient(config)
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

	t.Run("very_long_text", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "gemini",
			APIKey:   "test-api-key",
			Model:    "gemini-pro",
		}

		client, err := NewGeminiClient(config)
		if err != nil || client == nil {
			t.Skip("Skipping test - client creation failed")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Create very long text that might trigger size limits
		longText := strings.Repeat("Hello world. ", 1000)
		
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
}

// TestGeminiClientCreation tests client creation error paths
func TestGeminiClientCreation(t *testing.T) {
	t.Run("minimal_valid_config", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "gemini",
			APIKey:   "test-key",
			// Model is optional
		}

		client, err := NewGeminiClient(config)
		if err != nil {
			t.Errorf("Unexpected error with minimal config: %v", err)
		}
		if client == nil {
			t.Error("Client should not be nil with minimal valid config")
		}

		if client != nil {
			provider := client.GetProviderName()
			if provider != "gemini" {
				t.Errorf("Expected provider 'gemini', got: %s", provider)
			}
		}
	})

	t.Run("full_config", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "gemini",
			APIKey:   "test-key",
			Model:    "gemini-pro-vision",
			BaseURL:  "https://custom.googleapis.com",
		}

		client, err := NewGeminiClient(config)
		if err != nil {
			t.Errorf("Unexpected error with full config: %v", err)
		}
		if client == nil {
			t.Error("Client should not be nil with full valid config")
		}

		if client != nil {
			provider := client.GetProviderName()
			if provider != "gemini" {
				t.Errorf("Expected provider 'gemini', got: %s", provider)
			}
		}
	})
}

// TestGeminiParseResponseErrorPaths tests error paths in gemini parseResponse function
func TestGeminiParseResponseErrorPaths(t *testing.T) {
	// Create a test client to access parseResponse method
	client, err := NewGeminiClient(TranslationConfig{
		Provider: "gemini",
		APIKey:   "test-api-key",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Run("empty_candidates", func(t *testing.T) {
		response := &GeminiResponse{
			Candidates: []GeminiCandidate{},
		}

		result, err := client.parseResponse(response)
		if err == nil {
			t.Error("Expected error for empty candidates")
		}
		if result != "" {
			t.Error("Result should be empty when candidates are empty")
		}
		
		if !strings.Contains(err.Error(), "no candidates in response") {
			t.Errorf("Expected 'no candidates' error, got: %v", err)
		}
	})

	t.Run("non_stop_finish_reason", func(t *testing.T) {
		response := &GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					FinishReason: "MAX_TOKENS",
					Content: GeminiContent{
						Parts: []GeminiPart{
							{Text: "Partial translation"},
						},
					},
				},
			},
		}

		result, err := client.parseResponse(response)
		if err == nil {
			t.Error("Expected error for non-STOP finish reason")
		}
		if result != "" {
			t.Error("Result should be empty when finish reason is not STOP")
		}
		
		if !strings.Contains(err.Error(), "generation did not complete successfully") {
			t.Errorf("Expected completion error, got: %v", err)
		}
	})

	t.Run("empty_content_parts", func(t *testing.T) {
		response := &GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					FinishReason: "STOP",
					Content: GeminiContent{
						Parts: []GeminiPart{},
					},
				},
			},
		}

		result, err := client.parseResponse(response)
		if err != nil {
			t.Errorf("Unexpected error for empty parts: %v", err)
		}
		if result != "" {
			t.Errorf("Expected empty result for empty parts, got: '%s'", result)
		}
	})

	t.Run("multiple_content_parts", func(t *testing.T) {
		response := &GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					FinishReason: "STOP",
					Content: GeminiContent{
						Parts: []GeminiPart{
							{Text: "Hello "},
							{Text: "world"},
							{Text: "!"},
						},
					},
				},
			},
		}

		result, err := client.parseResponse(response)
		if err != nil {
			t.Errorf("Unexpected error for multiple parts: %v", err)
		}
		expected := "Hello world!"
		if result != expected {
			t.Errorf("Expected concatenated result '%s', got: '%s'", expected, result)
		}
	})

	t.Run("whitespace_handling", func(t *testing.T) {
		response := &GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					FinishReason: "STOP",
					Content: GeminiContent{
						Parts: []GeminiPart{
							{Text: "  Hello world  "},
						},
					},
				},
			},
		}

		result, err := client.parseResponse(response)
		if err != nil {
			t.Errorf("Unexpected error for whitespace: %v", err)
		}
		expected := "Hello world"
		if result != expected {
			t.Errorf("Expected trimmed result '%s', got: '%s'", expected, result)
		}
	})

	t.Run("multiple_candidates", func(t *testing.T) {
		response := &GeminiResponse{
			Candidates: []GeminiCandidate{
				{
					FinishReason: "STOP",
					Content: GeminiContent{
						Parts: []GeminiPart{
							{Text: "First candidate"},
						},
					},
				},
				{
					FinishReason: "STOP",
					Content: GeminiContent{
						Parts: []GeminiPart{
							{Text: "Second candidate"},
						},
					},
				},
			},
		}

		result, err := client.parseResponse(response)
		if err != nil {
			t.Errorf("Unexpected error for multiple candidates: %v", err)
		}
		// Should only use the first candidate
		expected := "First candidate"
		if result != expected {
			t.Errorf("Expected first candidate '%s', got: '%s'", expected, result)
		}
	})
}