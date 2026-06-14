package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOllamaTranslate_SendsOptions is the reproduce-first (§11.4.146) RED for the
// Ollama options-plumbing defect.
//
// Root cause (FACT, pkg/translator/llm/ollama.go:21-25 + 63-67): OllamaRequest
// carries only Model/Prompt/Stream. The /api/generate endpoint takes generation
// parameters (temperature, num_predict, ...) under an "options" object
// (verified 2026-06-14 against the official Ollama API docs
// https://github.com/ollama/ollama/blob/main/docs/api.md). The pre-fix client
// never set options, so the configured / CLI temperature and max_tokens were
// SILENTLY DROPPED on every request. Ollama then used its model-default
// sampling (commonly temperature 0.8) — defeating the translator's deterministic
// 0.3 and ignoring any user -temperature / -max-tokens flag, producing
// non-deterministic translations the user explicitly tried to control.
//
// Every sibling HTTP client (openai/anthropic/qwen/zhipu/gemini) sends these
// params; only Ollama dropped them.
func TestOllamaTranslate_SendsOptions(t *testing.T) {
	var captured map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"response":"Привет","done":true}`))
	}))
	defer srv.Close()

	client := &OllamaClient{
		config: TranslationConfig{
			Provider:    "ollama",
			Model:       "llama3:8b",
			Temperature: 0.1,  // user asked for near-deterministic output
			MaxTokens:   2048, // user-requested generation budget
		},
		httpClient: &http.Client{},
		baseURL:    srv.URL,
	}

	out, err := client.Translate(context.Background(), "Hello", "Translate to Russian")
	require.NoError(t, err)
	assert.Equal(t, "Привет", out)

	// The request body MUST carry an "options" object reaching Ollama.
	require.NotNil(t, captured, "request body must be valid JSON")
	opts, ok := captured["options"].(map[string]any)
	require.True(t, ok, "OllamaRequest must include an \"options\" object so generation params reach the model")

	temp, ok := opts["temperature"].(float64)
	require.True(t, ok, "options.temperature must be sent")
	assert.InDelta(t, 0.1, temp, 1e-9, "user-configured temperature must reach Ollama")

	numPredict, ok := opts["num_predict"].(float64)
	require.True(t, ok, "options.num_predict must be sent")
	assert.Equal(t, 2048, int(numPredict), "user-configured max_tokens must reach Ollama as num_predict")
}

// TestOllamaTranslate_OptionsPrecedenceAndDefault proves the same precedence the
// sibling clients use: Options[...] override > typed config field > default.
func TestOllamaTranslate_OptionsPrecedenceAndDefault(t *testing.T) {
	var captured map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		_, _ = w.Write([]byte(`{"response":"ok","done":true}`))
	}))
	defer srv.Close()

	// No typed fields set -> defaults; Options override wins where present.
	client := &OllamaClient{
		config: TranslationConfig{
			Provider: "ollama",
			Model:    "llama3:8b",
			Options:  map[string]interface{}{"temperature": 0.7},
		},
		httpClient: &http.Client{},
		baseURL:    srv.URL,
	}

	_, err := client.Translate(context.Background(), "Hello", "p")
	require.NoError(t, err)

	opts, ok := captured["options"].(map[string]any)
	require.True(t, ok)
	assert.InDelta(t, 0.7, opts["temperature"].(float64), 1e-9, "Options[temperature] must override the default")
}
