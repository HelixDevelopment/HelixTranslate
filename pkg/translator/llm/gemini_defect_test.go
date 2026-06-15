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

// TestGeminiTranslate_MaxTokensReturnsPartialText is the reproduce-first
// (§11.4.146) RED for the Gemini finishReason defect.
//
// Per the official Gemini API reference
// (https://ai.google.dev/api/generate-content#FinishReason, verified
// 2026-06-13): finishReason == "MAX_TOKENS" means generation stopped at the
// output-token limit and the candidate STILL CONTAINS the usable text generated
// up to that limit. The pre-fix parseResponse() rejected any reason != "STOP",
// so for a book-translation tool (large sections, MaxOutputTokens 4000) a
// truncated-but-valid translation was discarded with a hard error and the user
// lost the whole result.
func TestGeminiTranslate_MaxTokensReturnsPartialText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// A real Gemini truncation response: valid text + finishReason MAX_TOKENS.
		_, _ = w.Write([]byte(`{
			"candidates":[{
				"content":{"parts":[{"text":"переведённый текст"}],"role":"model"},
				"finishReason":"MAX_TOKENS",
				"index":0
			}]
		}`))
	}))
	defer srv.Close()

	config := TranslationConfig{
		Provider: "gemini", APIKey: "k", Model: "gemini-pro", BaseURL: srv.URL,
	}
	client, err := NewGeminiClient(config)
	require.NoError(t, err)

	out, err := client.Translate(context.Background(), "source text", "translate to Russian")
	require.NoError(t, err, "MAX_TOKENS must yield the usable partial text, not a hard error")
	assert.Equal(t, "переведённый текст", out)
}

// TestGeminiTranslate_StopReturnsText keeps the happy path honest.
func TestGeminiTranslate_StopReturnsText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"готово"}]},"finishReason":"STOP"}]}`))
	}))
	defer srv.Close()

	config := TranslationConfig{Provider: "gemini", APIKey: "k", Model: "gemini-pro", BaseURL: srv.URL}
	client, err := NewGeminiClient(config)
	require.NoError(t, err)

	out, err := client.Translate(context.Background(), "x", "p")
	require.NoError(t, err)
	assert.Equal(t, "готово", out)
}

// TestGeminiTranslate_SafetyBlockIsError proves genuine blocks (no usable text)
// still surface as an error — the fix must not blanket-accept every reason.
func TestGeminiTranslate_SafetyBlockIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[]},"finishReason":"SAFETY"}]}`))
	}))
	defer srv.Close()

	config := TranslationConfig{Provider: "gemini", APIKey: "k", Model: "gemini-pro", BaseURL: srv.URL}
	client, err := NewGeminiClient(config)
	require.NoError(t, err)

	out, err := client.Translate(context.Background(), "x", "p")
	require.Error(t, err)
	assert.Empty(t, out)
	assert.Contains(t, err.Error(), "SAFETY")
}

// TestGeminiTranslate_Non200IsError confirms non-200 surfaces as an error.
func TestGeminiTranslate_Non200IsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"API key invalid"}}`))
	}))
	defer srv.Close()

	config := TranslationConfig{Provider: "gemini", APIKey: "bad", Model: "gemini-pro", BaseURL: srv.URL}
	client, err := NewGeminiClient(config)
	require.NoError(t, err)

	out, err := client.Translate(context.Background(), "x", "p")
	require.Error(t, err)
	assert.Empty(t, out)
	assert.Contains(t, err.Error(), "403")
}

// TestGeminiTranslate_TypedConfigFieldsHonored is the reproduce-first (§11.4.146)
// RED for the Gemini config-plumbing defect: the Gemini client hardcodes
// generationConfig.temperature=0.3 and maxOutputTokens=4000, ignoring the
// operator's -temperature / -max-tokens CLI flags (TranslationConfig.Temperature
// / .MaxTokens) AND the Options[...] override entirely. Every sibling client
// (openai/anthropic/zhipu/qwen) honors them per config_field_plumbing_test.go;
// Gemini silently caps large book sections at 4000 tokens and forces temp 0.3.
func TestGeminiTranslate_TypedConfigFieldsHonored(t *testing.T) {
	got := &GeminiRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, got)
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"ok"}]},"finishReason":"STOP"}]}`))
	}))
	defer srv.Close()

	config := TranslationConfig{
		Provider: "gemini", APIKey: "k", Model: "gemini-pro", BaseURL: srv.URL,
		Temperature: 0.7, MaxTokens: 8000,
	}
	client, err := NewGeminiClient(config)
	require.NoError(t, err)
	_, err = client.Translate(context.Background(), "hi", "p")
	require.NoError(t, err)
	require.NotNil(t, got.GenerationConfig, "generationConfig must be present")
	assert.InDelta(t, 0.7, got.GenerationConfig.Temperature, 1e-9,
		"gemini must honor typed -temperature, not hardcode 0.3")
	assert.Equal(t, 8000, got.GenerationConfig.MaxOutputTokens,
		"gemini must honor typed -max-tokens, not hardcode 4000")
}

// TestGeminiTranslate_OptionsOverrideTypedFields proves Options still wins.
func TestGeminiTranslate_OptionsOverrideTypedFields(t *testing.T) {
	got := &GeminiRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, got)
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"ok"}]},"finishReason":"STOP"}]}`))
	}))
	defer srv.Close()

	config := TranslationConfig{
		Provider: "gemini", APIKey: "k", Model: "gemini-pro", BaseURL: srv.URL,
		Temperature: 0.7, MaxTokens: 8000,
		Options: map[string]interface{}{"temperature": 0.1, "max_tokens": 123},
	}
	client, err := NewGeminiClient(config)
	require.NoError(t, err)
	_, err = client.Translate(context.Background(), "hi", "p")
	require.NoError(t, err)
	require.NotNil(t, got.GenerationConfig)
	assert.InDelta(t, 0.1, got.GenerationConfig.Temperature, 1e-9, "Options temperature overrides typed field")
	assert.Equal(t, 123, got.GenerationConfig.MaxOutputTokens, "Options max_tokens overrides typed field")
}
