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

// TestOpenAITranslate_TemperatureNonFloat64_NoPanic is the reproduce-first
// (§11.4.115/§11.4.146) RED test for the unchecked type assertion
// `temperature.(float64)` in OpenAIClient.Translate.
//
// Options is map[string]interface{}; callers (and JSON/config layers) can store
// temperature as an int, float32, or json.Number. The pre-fix code panics on
// any non-float64 value because the assertion is unchecked. Sibling clients
// (anthropic/qwen/zhipu) use the checked `.(float64)` form — openai.go did not.
//
// Because OpenAIClient is embedded by DeepSeek/Groq/Mistral/Cerebras/xAI/Kimi/
// etc., the panic reaches every OpenAI-compatible provider.
func TestOpenAITranslate_TemperatureNonFloat64_NoPanic(t *testing.T) {
	// httptest server captures the request and returns a valid completion so we
	// reach (and pass) the response-decode path once the assertion is fixed.
	var gotTemp json.RawMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]json.RawMessage
		_ = json.Unmarshal(body, &req)
		gotTemp = req["temperature"]
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"привет"}}]}`))
	}))
	defer srv.Close()

	cases := []struct {
		name string
		val  interface{}
		want float64
	}{
		{"int", 1, 1.0},
		{"float32", float32(0.5), 0.5},
		{"json_number", json.Number("0.7"), 0.7},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotTemp = nil
			config := TranslationConfig{
				Provider: "openai",
				APIKey:   "test-key",
				Model:    "gpt-4",
				BaseURL:  srv.URL,
				Options:  map[string]interface{}{"temperature": tc.val},
			}
			client, err := NewOpenAIClient(config)
			require.NoError(t, err)

			// RED on pre-fix artifact: this call panics on the unchecked assertion.
			out, err := client.Translate(context.Background(), "hello", "translate")
			require.NoError(t, err, "Translate must not error on non-float64 temperature")
			assert.Equal(t, "привет", out)
			// The exact temperature must reach the wire — not dropped to default.
			require.NotNil(t, gotTemp)
			var sent float64
			require.NoError(t, json.Unmarshal(gotTemp, &sent))
			assert.InDelta(t, tc.want, sent, 1e-6,
				"caller temperature must propagate to the request body")
		})
	}
}

// TestOpenAITranslate_RequestShapeAndAuth pins the exact wire format the OpenAI
// client sends: POST /chat/completions, Bearer auth, JSON body with model +
// single user message + numeric temperature + max_tokens.
func TestOpenAITranslate_RequestShapeAndAuth(t *testing.T) {
	type capture struct {
		method string
		path   string
		auth   string
		ctype  string
		body   OpenAIRequest
	}
	var cap capture
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.method = r.Method
		cap.path = r.URL.Path
		cap.auth = r.Header.Get("Authorization")
		cap.ctype = r.Header.Get("Content-Type")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &cap.body)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"OK"}}]}`))
	}))
	defer srv.Close()

	config := TranslationConfig{
		Provider: "openai",
		APIKey:   "secret-token",
		Model:    "gpt-4",
		BaseURL:  srv.URL,
		Options:  map[string]interface{}{"temperature": 0.42, "max_tokens": 256},
	}
	client, err := NewOpenAIClient(config)
	require.NoError(t, err)

	out, err := client.Translate(context.Background(), "hello", "translate this prompt")
	require.NoError(t, err)
	assert.Equal(t, "OK", out)

	assert.Equal(t, http.MethodPost, cap.method)
	assert.Equal(t, "/chat/completions", cap.path)
	assert.Equal(t, "Bearer secret-token", cap.auth)
	assert.Equal(t, "application/json", cap.ctype)
	assert.Equal(t, "gpt-4", cap.body.Model)
	require.Len(t, cap.body.Messages, 1)
	assert.Equal(t, "user", cap.body.Messages[0].Role)
	assert.Equal(t, "translate this prompt", cap.body.Messages[0].Content)
	assert.InDelta(t, 0.42, cap.body.Temperature, 1e-9)
	assert.Equal(t, 256, cap.body.MaxTokens)
}

// TestOpenAITranslate_Non200IsError proves a non-200 surfaces as a real error
// carrying the status + body, never a silent empty-string success.
func TestOpenAITranslate_Non200IsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"rate limited"}}`))
	}))
	defer srv.Close()

	config := TranslationConfig{
		Provider: "openai", APIKey: "k", Model: "gpt-4", BaseURL: srv.URL,
	}
	client, err := NewOpenAIClient(config)
	require.NoError(t, err)

	out, err := client.Translate(context.Background(), "hi", "p")
	require.Error(t, err)
	assert.Empty(t, out)
	assert.Contains(t, err.Error(), "429")
	assert.Contains(t, err.Error(), "rate limited")
}

// TestOpenAITranslate_EmptyChoicesIsError proves a 200 with zero choices is an
// error, not an empty-string success.
func TestOpenAITranslate_EmptyChoicesIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer srv.Close()

	config := TranslationConfig{Provider: "openai", APIKey: "k", Model: "gpt-4", BaseURL: srv.URL}
	client, err := NewOpenAIClient(config)
	require.NoError(t, err)

	out, err := client.Translate(context.Background(), "hi", "p")
	require.Error(t, err)
	assert.Empty(t, out)
	assert.Contains(t, err.Error(), "no choices")
}

// TestOpenAITranslate_MaxTokensFloat64FromJSON is the reproduce-first RED for
// the max_tokens plumbing gap: config decoded from JSON yields float64, but the
// client only honored `int`, so a JSON-configured max_tokens was silently
// dropped (default 8192 used instead of the operator's value).
func TestOpenAITranslate_MaxTokensFloat64FromJSON(t *testing.T) {
	var gotMaxTokens int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req OpenAIRequest
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &req)
		gotMaxTokens = req.MaxTokens
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"OK"}}]}`))
	}))
	defer srv.Close()

	// Simulate config decoded by encoding/json: numbers land as float64.
	var opts map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(`{"max_tokens": 512}`), &opts))

	config := TranslationConfig{
		Provider: "openai", APIKey: "k", Model: "gpt-4", BaseURL: srv.URL, Options: opts,
	}
	client, err := NewOpenAIClient(config)
	require.NoError(t, err)

	_, err = client.Translate(context.Background(), "hi", "p")
	require.NoError(t, err)
	assert.Equal(t, 512, gotMaxTokens, "JSON-decoded max_tokens (float64) must reach the request")
}
