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

// captureChatCompletion stands up an OpenAI-compatible endpoint and returns the
// decoded request the client sent.
func captureChatCompletion(t *testing.T) (*httptest.Server, *OpenAIRequest) {
	t.Helper()
	got := &OpenAIRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, got)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"OK"}}]}`))
	}))
	t.Cleanup(srv.Close)
	return srv, got
}

// TestOpenAITranslate_TypedConfigFieldsHonored is the reproduce-first
// (§11.4.146) RED for the typed-field plumbing defect: the unified-translator
// CLI populates TranslationConfig.Temperature / .MaxTokens from the
// -temperature / -max-tokens flags but provides NO Options map. The OpenAI
// client only read Options[...], so the operator's flags were silently dropped
// and the hardcoded defaults (0.3 / 8192) were used instead.
func TestOpenAITranslate_TypedConfigFieldsHonored(t *testing.T) {
	srv, got := captureChatCompletion(t)

	// Exactly what cmd/unified-translator builds on the legacy API path: typed
	// fields set, Options nil.
	config := TranslationConfig{
		Provider:    "openai",
		APIKey:      "k",
		Model:       "gpt-4",
		BaseURL:     srv.URL,
		Temperature: 0.7,
		MaxTokens:   8000,
	}
	client, err := NewOpenAIClient(config)
	require.NoError(t, err)

	_, err = client.Translate(context.Background(), "hi", "p")
	require.NoError(t, err)

	assert.InDelta(t, 0.7, got.Temperature, 1e-9,
		"CLI -temperature must reach the request, not be dropped to the 0.3 default")
	assert.Equal(t, 8000, got.MaxTokens,
		"CLI -max-tokens must reach the request, not be dropped to the 8192 default")
}

// TestOpenAITranslate_OptionsOverrideTypedFields proves Options still wins when
// both are present (Options is the explicit per-call override).
func TestOpenAITranslate_OptionsOverrideTypedFields(t *testing.T) {
	srv, got := captureChatCompletion(t)

	config := TranslationConfig{
		Provider:    "openai",
		APIKey:      "k",
		Model:       "gpt-4",
		BaseURL:     srv.URL,
		Temperature: 0.7,
		MaxTokens:   8000,
		Options:     map[string]interface{}{"temperature": 0.1, "max_tokens": 123},
	}
	client, err := NewOpenAIClient(config)
	require.NoError(t, err)

	_, err = client.Translate(context.Background(), "hi", "p")
	require.NoError(t, err)

	assert.InDelta(t, 0.1, got.Temperature, 1e-9, "Options temperature overrides typed field")
	assert.Equal(t, 123, got.MaxTokens, "Options max_tokens overrides typed field")
}

// TestOpenAITranslate_DefaultsWhenNothingSet proves the documented defaults
// still apply when neither typed fields nor Options carry a value.
func TestOpenAITranslate_DefaultsWhenNothingSet(t *testing.T) {
	srv, got := captureChatCompletion(t)

	config := TranslationConfig{Provider: "openai", APIKey: "k", Model: "gpt-4", BaseURL: srv.URL}
	client, err := NewOpenAIClient(config)
	require.NoError(t, err)

	_, err = client.Translate(context.Background(), "hi", "p")
	require.NoError(t, err)

	assert.InDelta(t, 0.3, got.Temperature, 1e-9)
	assert.Equal(t, 8192, got.MaxTokens)
}

// chatChoicesServer returns an OpenAI/Zhipu/Qwen-style choices response and
// captures the raw request body for field-level assertions.
func chatChoicesServer(t *testing.T) (*httptest.Server, *map[string]json.RawMessage) {
	t.Helper()
	got := &map[string]json.RawMessage{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, got)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"OK"}}]}`))
	}))
	t.Cleanup(srv.Close)
	return srv, got
}

func numField(t *testing.T, m map[string]json.RawMessage, key string) (float64, bool) {
	t.Helper()
	raw, ok := m[key]
	if !ok {
		return 0, false
	}
	var f float64
	require.NoError(t, json.Unmarshal(raw, &f))
	return f, true
}

// TestAnthropicTranslate_TypedConfigFieldsHonored — extend (§11.4.146 STEP 3)
// the typed-field fix to the Anthropic client.
func TestAnthropicTranslate_TypedConfigFieldsHonored(t *testing.T) {
	got := &AnthropicRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, got)
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"OK"}]}`))
	}))
	defer srv.Close()

	config := TranslationConfig{
		Provider: "anthropic", APIKey: "k", Model: "claude-3-haiku-20240307",
		BaseURL: srv.URL, Temperature: 0.7, MaxTokens: 8000,
	}
	client, err := NewAnthropicClient(config)
	require.NoError(t, err)
	_, err = client.Translate(context.Background(), "hi", "p")
	require.NoError(t, err)
	assert.InDelta(t, 0.7, got.Temperature, 1e-9, "anthropic must honor typed -temperature")
	assert.Equal(t, 8000, got.MaxTokens, "anthropic must honor typed -max-tokens")
}

// TestZhipuTranslate_TypedConfigFieldsHonored — extend to Zhipu.
func TestZhipuTranslate_TypedConfigFieldsHonored(t *testing.T) {
	srv, got := chatChoicesServer(t)
	config := TranslationConfig{
		Provider: "zhipu", APIKey: "k", Model: "glm-4",
		BaseURL: srv.URL, Temperature: 0.7, MaxTokens: 8000,
	}
	client, err := NewZhipuClient(config)
	require.NoError(t, err)
	_, err = client.Translate(context.Background(), "hi", "p")
	require.NoError(t, err)
	temp, ok := numField(t, *got, "temperature")
	require.True(t, ok, "zhipu temperature must be present")
	assert.InDelta(t, 0.7, temp, 1e-9, "zhipu must honor typed -temperature")
	mt, ok := numField(t, *got, "max_tokens")
	require.True(t, ok, "zhipu max_tokens must be present")
	assert.Equal(t, 8000.0, mt, "zhipu must honor typed -max-tokens")
}

// TestQwenTranslate_TypedConfigFieldsHonored — extend to Qwen (Bearer-key path).
func TestQwenTranslate_TypedConfigFieldsHonored(t *testing.T) {
	srv, got := chatChoicesServer(t)
	config := TranslationConfig{
		Provider: "qwen", APIKey: "k", Model: "qwen-plus",
		BaseURL: srv.URL, Temperature: 0.7, MaxTokens: 8000,
	}
	client, err := NewQwenClient(config)
	require.NoError(t, err)
	_, err = client.Translate(context.Background(), "hi", "p")
	require.NoError(t, err)
	temp, ok := numField(t, *got, "temperature")
	require.True(t, ok, "qwen temperature must be present")
	assert.InDelta(t, 0.7, temp, 1e-9, "qwen must honor typed -temperature")
	mt, ok := numField(t, *got, "max_tokens")
	require.True(t, ok, "qwen max_tokens must be present")
	assert.Equal(t, 8000.0, mt, "qwen must honor typed -max-tokens")
}

// TestSiblingClients_TemperatureNonFloat64_NoPanic — extend the non-float64
// no-panic + value-propagation guard to the sibling clients that previously
// only accepted a literal float64 (they silently dropped int/json.Number to the
// default rather than panicking, but value-correctness was still broken).
func TestSiblingClients_TemperatureNonFloat64_NoPanic(t *testing.T) {
	t.Run("zhipu_int", func(t *testing.T) {
		srv, got := chatChoicesServer(t)
		client, err := NewZhipuClient(TranslationConfig{
			Provider: "zhipu", APIKey: "k", Model: "glm-4", BaseURL: srv.URL,
			Options: map[string]interface{}{"temperature": 1, "max_tokens": 321},
		})
		require.NoError(t, err)
		_, err = client.Translate(context.Background(), "hi", "p")
		require.NoError(t, err)
		temp, _ := numField(t, *got, "temperature")
		assert.InDelta(t, 1.0, temp, 1e-9)
		mt, _ := numField(t, *got, "max_tokens")
		assert.Equal(t, 321.0, mt)
	})
	t.Run("qwen_jsonnumber", func(t *testing.T) {
		srv, got := chatChoicesServer(t)
		var opts map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(`{"max_tokens":777}`), &opts))
		client, err := NewQwenClient(TranslationConfig{
			Provider: "qwen", APIKey: "k", Model: "qwen-plus", BaseURL: srv.URL, Options: opts,
		})
		require.NoError(t, err)
		_, err = client.Translate(context.Background(), "hi", "p")
		require.NoError(t, err)
		mt, ok := numField(t, *got, "max_tokens")
		require.True(t, ok)
		assert.Equal(t, 777.0, mt, "JSON-decoded max_tokens (float64) must reach the request")
	})
}
