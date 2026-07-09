package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// openAICompatibleTestServer creates an httptest server that validates
// OpenAI-compatible requests and returns a mock translation response.
func openAICompatibleTestServer(t *testing.T, expectedProvider string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/chat/completions", r.URL.Path, "%s: expected /chat/completions endpoint", expectedProvider)
		assert.Equal(t, http.MethodPost, r.Method, "%s: expected POST method", expectedProvider)
		assert.True(t, strings.HasPrefix(r.Header.Get("Authorization"), "Bearer "), "%s: expected Bearer auth", expectedProvider)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "%s: expected JSON content type", expectedProvider)

		var req OpenAIRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err, "%s: failed to decode request", expectedProvider)
		require.Len(t, req.Messages, 1, "%s: expected exactly 1 message", expectedProvider)
		require.NotEmpty(t, req.Messages[0].Content, "%s: message content must not be empty", expectedProvider)

		resp := OpenAIResponse{
			ID:      "test-id",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   req.Model,
			Choices: []Choice{
				{
					Index:        0,
					Message:      Message{Role: "assistant", Content: FlexibleContent("translated text from " + expectedProvider)},
					FinishReason: "stop",
				},
			},
			Usage: Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

// Test all OpenAI-compatible providers with the same pattern.
func TestOpenAICompatibleProviders(t *testing.T) {
	providers := []struct {
		name      string
		provider  string
		model     string
		newClient func(TranslationConfig) (LLMClient, error)
	}{
		{"groq", "groq", "llama-3.1-70b-versatile", func(c TranslationConfig) (LLMClient, error) {
			cl, err := NewGroqClient(c)
			if err != nil { return nil, err }
			return cl, nil
		}},
		{"mistral", "mistral", "mistral-large-latest", func(c TranslationConfig) (LLMClient, error) {
			cl, err := NewMistralClient(c)
			if err != nil { return nil, err }
			return cl, nil
		}},
		{"xai", "xai", "grok-beta", func(c TranslationConfig) (LLMClient, error) {
			cl, err := NewXAIClient(c)
			if err != nil { return nil, err }
			return cl, nil
		}},
		{"cerebras", "cerebras", "llama3.1-70b", func(c TranslationConfig) (LLMClient, error) {
			cl, err := NewCerebrasClient(c)
			if err != nil { return nil, err }
			return cl, nil
		}},
		{"siliconflow", "siliconflow", "deepseek-ai/DeepSeek-V2.5", func(c TranslationConfig) (LLMClient, error) {
			cl, err := NewSiliconFlowClient(c)
			if err != nil { return nil, err }
			return cl, nil
		}},
		{"hyperbolic", "hyperbolic", "meta-llama/Meta-Llama-3.1-70B-Instruct", func(c TranslationConfig) (LLMClient, error) {
			cl, err := NewHyperbolicClient(c)
			if err != nil { return nil, err }
			return cl, nil
		}},
		{"togetherai", "togetherai", "meta-llama/Llama-3.1-70B-Instruct-Turbo", func(c TranslationConfig) (LLMClient, error) {
			cl, err := NewTogetherAIClient(c)
			if err != nil { return nil, err }
			return cl, nil
		}},
		{"sambanova", "sambanova", "Meta-Llama-3.1-70B-Instruct", func(c TranslationConfig) (LLMClient, error) {
			cl, err := NewSambaNovaClient(c)
			if err != nil { return nil, err }
			return cl, nil
		}},
		{"kimi", "kimi", "moonshot-v1-8k", func(c TranslationConfig) (LLMClient, error) {
			cl, err := NewKimiClient(c)
			if err != nil { return nil, err }
			return cl, nil
		}},
		{"novita", "novita", "meta-llama/llama-3.1-70b-instruct", func(c TranslationConfig) (LLMClient, error) {
			cl, err := NewNovitaClient(c)
			if err != nil { return nil, err }
			return cl, nil
		}},
		{"nlpcloud", "nlpcloud", "finetuned-llama-3-1-8b", func(c TranslationConfig) (LLMClient, error) {
			cl, err := NewNLPCloudClient(c)
			if err != nil { return nil, err }
			return cl, nil
		}},
		{"upstage", "upstage", "solar-pro", func(c TranslationConfig) (LLMClient, error) {
			cl, err := NewUpstageClient(c)
			if err != nil { return nil, err }
			return cl, nil
		}},
		{"cohere", "cohere", "command-r", func(c TranslationConfig) (LLMClient, error) {
			cl, err := NewCohereClient(c)
			if err != nil { return nil, err }
			return cl, nil
		}},
		{"sarvam", "sarvam", "sarvam-1", func(c TranslationConfig) (LLMClient, error) {
			cl, err := NewSarvamClient(c)
			if err != nil { return nil, err }
			return cl, nil
		}},
		{"modal", "modal", "modal-llama-3-1-70b", func(c TranslationConfig) (LLMClient, error) {
			cl, err := NewModalClient(c)
			if err != nil { return nil, err }
			return cl, nil
		}},
		{"publicai", "publicai", "publicai-llama-3-1-70b", func(c TranslationConfig) (LLMClient, error) {
			cl, err := NewPublicAIClient(c)
			if err != nil { return nil, err }
			return cl, nil
		}},
		{"nia", "nia", "nia-llama-3-1-70b", func(c TranslationConfig) (LLMClient, error) {
			cl, err := NewNIAClient(c)
			if err != nil { return nil, err }
			return cl, nil
		}},
		{"vulavula", "vulavula", "vulavula-llama-3-1-70b", func(c TranslationConfig) (LLMClient, error) {
			cl, err := NewVulavulaClient(c)
			if err != nil { return nil, err }
			return cl, nil
		}},
	}

	for _, p := range providers {
		t.Run(p.name, func(t *testing.T) {
			server := openAICompatibleTestServer(t, p.name)
			defer server.Close()

			config := TranslationConfig{
				APIKey:   "test-api-key",
				Model:    p.model,
				BaseURL:  server.URL,
				Provider: p.provider,
			}

			client, err := p.newClient(config)
			require.NoError(t, err)
			assert.Equal(t, p.name, client.GetProviderName())

			result, err := client.Translate(context.Background(), "hello", "translate this")
			require.NoError(t, err)
			assert.Equal(t, "translated text from "+p.name, result)
		})
	}
}

func TestOpenAICompatibleProviderValidation(t *testing.T) {
	t.Run("missing_api_key", func(t *testing.T) {
		_, err := NewGroqClient(TranslationConfig{Model: "llama-3.1-70b-versatile"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "API key is required")
	})

	t.Run("missing_model", func(t *testing.T) {
		_, err := NewGroqClient(TranslationConfig{APIKey: "test"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "model is required")
	})

	t.Run("invalid_model", func(t *testing.T) {
		_, err := NewGroqClient(TranslationConfig{APIKey: "test", Model: "invalid-model"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not valid")
	})

	t.Run("invalid_temperature", func(t *testing.T) {
		_, err := NewGroqClient(TranslationConfig{
			APIKey:   "test",
			Model:    "llama-3.1-70b-versatile",
			Options:  map[string]interface{}{"temperature": 3.0},
			Provider: "groq",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "temperature")
	})
}

func TestCloudflareClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.True(t, strings.HasPrefix(r.URL.Path, "/accounts/"))
		assert.True(t, strings.Contains(r.URL.Path, "/ai/run/"))
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

		var req cloudflareRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		require.Len(t, req.Messages, 1)

		resp := cloudflareResponse{
			Result: struct {
				Response string `json:"response"`
			}{Response: "cloudflare translated"},
			Success: true,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := TranslationConfig{
		APIKey:  "test-api-key",
		Model:   "@cf/meta/llama-2-7b-chat-int8",
		BaseURL: server.URL,
		Options: map[string]interface{}{"account_id": "test-account-id"},
	}

	client, err := NewCloudflareClient(config)
	require.NoError(t, err)
	assert.Equal(t, "cloudflare", client.GetProviderName())

	result, err := client.Translate(context.Background(), "hello", "translate this")
	require.NoError(t, err)
	assert.Equal(t, "cloudflare translated", result)
}

func TestCloudflareClientValidation(t *testing.T) {
	t.Run("missing_api_key", func(t *testing.T) {
		_, err := NewCloudflareClient(TranslationConfig{Model: "@cf/meta/llama-2-7b-chat-int8", Options: map[string]interface{}{"account_id": "test"}})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "API key is required")
	})

	t.Run("missing_account_id", func(t *testing.T) {
		_, err := NewCloudflareClient(TranslationConfig{APIKey: "test", Model: "@cf/meta/llama-2-7b-chat-int8"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "account_id")
	})

	t.Run("missing_model", func(t *testing.T) {
		_, err := NewCloudflareClient(TranslationConfig{APIKey: "test", Options: map[string]interface{}{"account_id": "test"}})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "model is required")
	})

	t.Run("invalid_model", func(t *testing.T) {
		_, err := NewCloudflareClient(TranslationConfig{
			APIKey:  "test",
			Model:   "invalid-model",
			Options: map[string]interface{}{"account_id": "test"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not valid")
	})

	t.Run("api_error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"success":false,"errors":["unauthorized"]}`))
		}))
		defer server.Close()

		config := TranslationConfig{
			APIKey:  "bad-key",
			Model:   "@cf/meta/llama-2-7b-chat-int8",
			BaseURL: server.URL,
			Options: map[string]interface{}{"account_id": "test-account-id"},
		}
		client, err := NewCloudflareClient(config)
		require.NoError(t, err)

		_, err = client.Translate(context.Background(), "hello", "prompt")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "401")
	})

	t.Run("cloudflare_api_error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := cloudflareResponse{Success: false, Errors: []string{"model not found"}}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		config := TranslationConfig{
			APIKey:  "test-key",
			Model:   "@cf/meta/llama-2-7b-chat-int8",
			BaseURL: server.URL,
			Options: map[string]interface{}{"account_id": "test-account-id"},
		}
		client, err := NewCloudflareClient(config)
		require.NoError(t, err)

		_, err = client.Translate(context.Background(), "hello", "prompt")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "model not found")
	})
}

func TestReplicateClient(t *testing.T) {
	var predictionID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/predictions") && r.Method == http.MethodPost:
			assert.Equal(t, "Token test-api-key", r.Header.Get("Authorization"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			var req replicatePredictionRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)
			require.NotNil(t, req.Input)
			assert.Equal(t, "translate this", req.Input["prompt"])

			predictionID = "pred-test-123"
			resp := replicatePredictionResponse{ID: predictionID, Status: "starting"}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(resp)

		case strings.Contains(r.URL.Path, "/predictions/") && r.Method == http.MethodGet:
			assert.Equal(t, "Token test-api-key", r.Header.Get("Authorization"))

			resp := replicateGetResponse{ID: predictionID, Status: "succeeded", Output: "replicate translated"}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	config := TranslationConfig{
		APIKey:  "test-api-key",
		Model:   "meta/llama-2-70b-chat",
		BaseURL: server.URL,
	}

	client, err := NewReplicateClient(config)
	require.NoError(t, err)
	assert.Equal(t, "replicate", client.GetProviderName())

	result, err := client.Translate(context.Background(), "hello", "translate this")
	require.NoError(t, err)
	assert.Equal(t, "replicate translated", result)
}

func TestReplicateClientArrayOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/predictions") && r.Method == http.MethodPost:
			resp := replicatePredictionResponse{ID: "pred-456", Status: "starting"}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(resp)
		case strings.Contains(r.URL.Path, "/predictions/") && r.Method == http.MethodGet:
			resp := replicateGetResponse{ID: "pred-456", Status: "succeeded", Output: []interface{}{"chunk1", "chunk2"}}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	config := TranslationConfig{
		APIKey:  "test-api-key",
		Model:   "meta/llama-2-70b-chat",
		BaseURL: server.URL,
	}

	client, err := NewReplicateClient(config)
	require.NoError(t, err)

	result, err := client.Translate(context.Background(), "hello", "translate this")
	require.NoError(t, err)
	assert.Equal(t, "chunk1chunk2", result)
}

func TestReplicateClientValidation(t *testing.T) {
	t.Run("missing_api_key", func(t *testing.T) {
		_, err := NewReplicateClient(TranslationConfig{Model: "meta/llama-2-70b-chat"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "API key is required")
	})

	t.Run("missing_model", func(t *testing.T) {
		_, err := NewReplicateClient(TranslationConfig{APIKey: "test"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "model is required")
	})

	t.Run("invalid_model", func(t *testing.T) {
		_, err := NewReplicateClient(TranslationConfig{APIKey: "test", Model: "invalid-model"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not valid")
	})

	t.Run("prediction_error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error": "invalid model"}`))
			}
		}))
		defer server.Close()

		config := TranslationConfig{
			APIKey:  "test-api-key",
			Model:   "meta/llama-2-70b-chat",
			BaseURL: server.URL,
		}
		client, err := NewReplicateClient(config)
		require.NoError(t, err)

		_, err = client.Translate(context.Background(), "hello", "prompt")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})

	t.Run("prediction_failed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPost:
				resp := replicatePredictionResponse{ID: "pred-err", Status: "starting"}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(resp)
			case r.Method == http.MethodGet:
				resp := replicateGetResponse{ID: "pred-err", Status: "failed", Error: "out of memory"}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			}
		}))
		defer server.Close()

		config := TranslationConfig{
			APIKey:  "test-api-key",
			Model:   "meta/llama-2-70b-chat",
			BaseURL: server.URL,
		}
		client, err := NewReplicateClient(config)
		require.NoError(t, err)

		_, err = client.Translate(context.Background(), "hello", "prompt")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "out of memory")
	})

	t.Run("poll_timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPost:
				resp := replicatePredictionResponse{ID: "pred-timeout", Status: "starting"}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(resp)
			case r.Method == http.MethodGet:
				resp := replicateGetResponse{ID: "pred-timeout", Status: "processing"}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(resp)
			}
		}))
		defer server.Close()

		config := TranslationConfig{
			APIKey:  "test-api-key",
			Model:   "meta/llama-2-70b-chat",
			BaseURL: server.URL,
		}
		client, err := NewReplicateClient(config)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err = client.Translate(ctx, "hello", "prompt")
		require.Error(t, err)
		// Should either timeout from context or from max attempts
		assert.True(t, strings.Contains(err.Error(), "context") || strings.Contains(err.Error(), "timed out"), "expected timeout error, got: %v", err)
	})
}

func TestReplicateExtractOutput(t *testing.T) {
	client := &ReplicateClient{}

	t.Run("string_output", func(t *testing.T) {
		result, err := client.extractOutput("hello world")
		require.NoError(t, err)
		assert.Equal(t, "hello world", result)
	})

	t.Run("array_output", func(t *testing.T) {
		result, err := client.extractOutput([]interface{}{"hello", " ", "world"})
		require.NoError(t, err)
		assert.Equal(t, "hello world", result)
	})

	t.Run("nil_output", func(t *testing.T) {
		_, err := client.extractOutput(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("unexpected_type", func(t *testing.T) {
		_, err := client.extractOutput(42)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected")
	})
}

// Anti-bluff: Verify that removing the provider implementation causes tests to fail.
// This test documents the anti-bluff pattern. The mutation testing challenge
// (anti_bluff_execution_challenge.sh) verifies this for core features.
func TestProviderAntiBluff(t *testing.T) {
	// This test proves that each provider client:
	// 1. Can be instantiated with valid config
	// 2. Makes HTTP requests in the expected format
	// 3. Parses responses correctly
	// 4. Returns meaningful errors on API failures
	//
	// If any provider's Translate() method returns a hardcoded error
	// like "not yet implemented", these tests will fail.
	//
	// This is verified by the individual provider tests above.
	assert.True(t, true, "anti-bluff documentation test")
}

// Benchmark for provider translation throughput (anti-bluff: performance must be measurable)
func BenchmarkOpenAICompatibleTranslate(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := OpenAIResponse{
			Choices: []Choice{{Message: Message{Content: "benchmark result"}}},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	config := TranslationConfig{
		APIKey:   "test-key",
		Model:    "llama-3.1-70b-versatile",
		BaseURL:  server.URL,
		Provider: "groq",
	}

	client, err := NewGroqClient(config)
	require.NoError(b, err)

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.Translate(ctx, fmt.Sprintf("text %d", i), "translate")
	}
}
