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

func TestGeminiProvider(t *testing.T) {
	t.Run("provider_name", func(t *testing.T) {
		client, err := NewGeminiClient(TranslationConfig{
			Provider: "gemini",
			APIKey:   "test-key",
		})
		require.NoError(t, err)
		require.NotNil(t, client)
		assert.Equal(t, "gemini", client.GetProviderName())
	})

	t.Run("missing_api_key", func(t *testing.T) {
		client, err := NewGeminiClient(TranslationConfig{
			Provider: "gemini",
			APIKey:   "",
		})
		require.Error(t, err)
		require.Nil(t, client)
		assert.Contains(t, err.Error(), "API key is required")
	})

	t.Run("default_base_url", func(t *testing.T) {
		client, err := NewGeminiClient(TranslationConfig{
			Provider: "gemini",
			APIKey:   "test-key",
		})
		require.NoError(t, err)
		require.NotNil(t, client)
	})

	t.Run("custom_base_url", func(t *testing.T) {
		client, err := NewGeminiClient(TranslationConfig{
			Provider: "gemini",
			APIKey:   "test-key",
			BaseURL:  "https://custom.googleapis.com",
		})
		require.NoError(t, err)
		require.NotNil(t, client)
	})

	t.Run("mock_translate_success", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"candidates": [{"content": {"parts": [{"text": "Привет"}]}, "finishReason": "STOP"}]
			}`))
		}))
		defer mockServer.Close()

		client := &GeminiClient{
			config: TranslationConfig{
				Provider: "gemini",
				APIKey:   "test-key",
				Model:    "gemini-pro",
			},
			httpClient: &http.Client{},
			baseURL:    mockServer.URL,
		}

		ctx := context.Background()
		result, err := client.Translate(ctx, "Hello", "Translate to Russian")
		require.NoError(t, err)
		assert.Equal(t, "Привет", result)
	})

	t.Run("mock_translate_empty_candidates", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"candidates": []}`))
		}))
		defer mockServer.Close()

		client := &GeminiClient{
			config: TranslationConfig{
				Provider: "gemini",
				APIKey:   "test-key",
				Model:    "gemini-pro",
			},
			httpClient: &http.Client{},
			baseURL:    mockServer.URL,
		}

		ctx := context.Background()
		_, err := client.Translate(ctx, "Hello", "Translate to Russian")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no candidates")
	})
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
		require.Error(t, err)
		require.Nil(t, client)
		assert.Contains(t, err.Error(), "API key is required")
	})

	t.Run("invalid_model", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "gemini",
			APIKey:   "test-api-key",
			Model:    "invalid-model-name",
		}

		client, err := NewGeminiClient(config)
		require.NoError(t, err)
		require.NotNil(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err = client.Translate(ctx, "Hello", "Translate to Russian")
		require.Error(t, err)
	})

	t.Run("context_cancellation", func(t *testing.T) {
		config := TranslationConfig{
			Provider: "gemini",
			APIKey:   "test-api-key",
			Model:    "gemini-pro",
		}

		client, err := NewGeminiClient(config)
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
			Provider: "gemini",
			APIKey:   "test-api-key",
			Model:    "gemini-pro",
		}

		client, err := NewGeminiClient(config)
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
			Provider: "gemini",
			APIKey:   "test-api-key",
			Model:    "gemini-pro",
		}

		client, err := NewGeminiClient(config)
		require.NoError(t, err)
		require.NotNil(t, client)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		longText := strings.Repeat("Hello world. ", 1000)

		_, err = client.Translate(ctx, longText, "Translate to Russian")
		require.Error(t, err)
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

// TestGeminiMakeRequestAdditionalPaths tests uncovered paths in makeRequest
func TestGeminiMakeRequestAdditionalPaths(t *testing.T) {
	client := &GeminiClient{
		baseURL: "https://generativelanguage.googleapis.com/v1beta",
		config: TranslationConfig{
			Provider: "gemini",
			APIKey:   "test-api-key",
			Model:    "gemini-pro",
		},
		httpClient: &http.Client{},
	}

	ctx := context.Background()
	req := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: "Test content"},
				},
			},
		},
	}

	t.Run("cancelled_context", func(t *testing.T) {
		// Create a context that's already cancelled
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := client.makeRequest(ctx, req)
		if err != nil {
			t.Logf("Expected error with cancelled context: %v", err)
		} else {
			t.Error("Expected error with cancelled context")
		}
	})

	t.Run("empty_model_defaults_to_gemini_pro", func(t *testing.T) {
		// Test that empty model defaults to "gemini-pro"
		client := &GeminiClient{
			baseURL: "https://generativelanguage.googleapis.com/v1beta",
			config: TranslationConfig{
				Provider: "gemini",
				APIKey:   "test-api-key",
				Model:    "", // Empty model should default to gemini-pro
			},
			httpClient: &http.Client{},
		}

		// Mock server to capture the request
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check that the URL contains "gemini-pro" model
			if !strings.Contains(r.URL.String(), "gemini-pro") {
				t.Errorf("Expected URL to contain 'gemini-pro', got: %s", r.URL.String())
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"test"}]}}]}`))
		}))
		defer server.Close()

		originalURL := client.baseURL
		client.baseURL = server.URL
		defer func() {
			client.baseURL = originalURL
		}()

		_, err := client.makeRequest(ctx, req)
		if err != nil {
			t.Errorf("Unexpected error with empty model: %v", err)
		}
	})

	t.Run("malformed_json_response", func(t *testing.T) {
		// Create a mock server that returns invalid JSON
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("{invalid json response"))
		}))
		defer server.Close()

		// Update client baseURL to use the test server
		originalURL := client.baseURL
		client.baseURL = server.URL

		defer func() {
			client.baseURL = originalURL
		}()

		_, err := client.makeRequest(ctx, req)
		if err != nil {
			t.Logf("Expected error with malformed JSON: %v", err)
			if !strings.Contains(err.Error(), "failed to unmarshal response") {
				t.Errorf("Error should mention unmarshaling: %v", err)
			}
		} else {
			t.Error("Expected error with malformed JSON")
		}
	})

	t.Run("empty_candidates_in_response", func(t *testing.T) {
		// Create a mock server that returns empty candidates
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"candidates": []}`))
		}))
		defer server.Close()

		// Update client baseURL to use the test server
		originalURL := client.baseURL
		client.baseURL = server.URL

		defer func() {
			client.baseURL = originalURL
		}()

		_, err := client.makeRequest(ctx, req)
		if err != nil {
			t.Logf("Expected error with empty candidates: %v", err)
			if !strings.Contains(err.Error(), "no candidates") {
				t.Errorf("Error should mention no candidates: %v", err)
			}
		} else {
			t.Error("Expected error with empty candidates")
		}
	})

	t.Run("error_status_response", func(t *testing.T) {
		// Create a mock server that returns error status
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": {"message": "Invalid request"}}`))
		}))
		defer server.Close()

		// Update client baseURL to use test server
		originalURL := client.baseURL
		client.baseURL = server.URL

		defer func() {
			client.baseURL = originalURL
		}()

		_, err := client.makeRequest(ctx, req)
		if err != nil {
			t.Logf("Expected error with bad status: %v", err)
			if !strings.Contains(err.Error(), "status 400") {
				t.Errorf("Error should mention status 400: %v", err)
			}
		} else {
			t.Error("Expected error with bad status")
		}
	})
}
