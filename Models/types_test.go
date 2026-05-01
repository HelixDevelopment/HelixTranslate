package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLLMRequestSerialization(t *testing.T) {
	req := LLMRequest{
		Model:       "gpt-4",
		Prompt:      "Hello",
		Temperature: 0.7,
		MaxTokens:   100,
		Options:     map[string]interface{}{"top_p": 0.9},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded LLMRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, req.Model, decoded.Model)
	assert.Equal(t, req.Prompt, decoded.Prompt)
	assert.Equal(t, req.Temperature, decoded.Temperature)
	assert.Equal(t, req.MaxTokens, decoded.MaxTokens)
}

func TestLLMResponseSerialization(t *testing.T) {
	resp := LLMResponse{
		Text:         "Hello world",
		Model:        "gpt-4",
		FinishReason: "stop",
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
		Metadata: map[string]interface{}{"version": "1.0"},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded LLMResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, resp.Text, decoded.Text)
	assert.Equal(t, resp.Usage.TotalTokens, decoded.Usage.TotalTokens)
}

func TestMessageSerialization(t *testing.T) {
	msg := Message{Role: "user", Content: "Hello"}
	data, err := json.Marshal(msg)
	require.NoError(t, err)
	assert.JSONEq(t, `{"role":"user","content":"Hello"}`, string(data))
}

func TestProviderCapabilitiesSerialization(t *testing.T) {
	caps := ProviderCapabilities{
		SupportedModels:         []string{"gpt-4", "gpt-3.5-turbo"},
		SupportedFeatures:       []string{"streaming", "function_calling"},
		SupportsStreaming:       true,
		SupportsFunctionCalling: true,
		SupportsVision:          false,
		Limits: ModelLimits{
			MaxTokens:             4096,
			MaxInputLength:        8192,
			MaxOutputLength:       4096,
			MaxConcurrentRequests: 10,
		},
	}

	data, err := json.Marshal(caps)
	require.NoError(t, err)

	var decoded ProviderCapabilities
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.True(t, decoded.SupportsStreaming)
	assert.False(t, decoded.SupportsVision)
	assert.Equal(t, 4096, decoded.Limits.MaxTokens)
	assert.Len(t, decoded.SupportedModels, 2)
}

func TestUsageDefaults(t *testing.T) {
	u := Usage{}
	assert.Equal(t, 0, u.PromptTokens)
	assert.Equal(t, 0, u.CompletionTokens)
	assert.Equal(t, 0, u.TotalTokens)
}

func TestModelLimitsDefaults(t *testing.T) {
	l := ModelLimits{}
	assert.Equal(t, 0, l.MaxTokens)
	assert.Equal(t, 0, l.MaxInputLength)
	assert.Equal(t, 0, l.MaxOutputLength)
	assert.Equal(t, 0, l.MaxConcurrentRequests)
}

func TestLLMRequestWithMessages(t *testing.T) {
	req := LLMRequest{
		Model: "gpt-4",
		Messages: []Message{
			{Role: "system", Content: "You are a translator"},
			{Role: "user", Content: "Translate this"},
		},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded map[string]interface{}
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	msgs, ok := decoded["messages"].([]interface{})
	require.True(t, ok)
	assert.Len(t, msgs, 2)
}
